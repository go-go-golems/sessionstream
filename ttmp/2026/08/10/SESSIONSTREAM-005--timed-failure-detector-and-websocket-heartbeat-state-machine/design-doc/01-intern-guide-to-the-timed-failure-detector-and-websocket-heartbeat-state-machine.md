---
Title: Intern Guide to the Timed Failure Detector and WebSocket Heartbeat State Machine
Ticket: SESSIONSTREAM-005
Status: active
Topics:
    - websocket
    - architecture
    - event-streaming
    - onboarding
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/sessionstream-systemlab/static/js/websocket.js
      Note: Shared browser nonce-echo implementation
    - Path: repo://pkg/sessionstream/hub.go
      Note: Domain pipeline and UI fanout ownership boundary
    - Path: repo://pkg/sessionstream/hydration.go
      Note: Snapshot contract whose blocking behavior must not starve heartbeat processing
    - Path: repo://pkg/sessionstream/transport/ws/heartbeat.go
      Note: Production supervisor, timer, nonce, event queue, and action adapter
    - Path: repo://pkg/sessionstream/transport/ws/internal/heartbeat/machine.go
      Note: Pure timed failure-detector reducer and state-event-action contract
    - Path: repo://pkg/sessionstream/transport/ws/internal/heartbeat/machine_test.go
      Note: Deterministic transition matrix, boundary, stale-event, and fuzz coverage
    - Path: repo://pkg/sessionstream/transport/ws/observer.go
      Note: Current synchronous transport observation contract and control-path interference boundary
    - Path: repo://pkg/sessionstream/transport/ws/server.go
      Note: Current connection lifecycle, queues, reader, writer, heartbeat loop, and shutdown implementation
    - Path: repo://pkg/sessionstream/transport/ws/server_test.go
      Note: Existing heartbeat, lifecycle, hydration, shutdown, and concurrency regression coverage
    - Path: repo://proto/sessionstream/v1/transport.proto
      Note: Authoritative application-level ping/pong and client/server frame schema
ExternalSources:
    - https://datatracker.ietf.org/doc/html/rfc6455
    - https://pkg.go.dev/github.com/gorilla/websocket
    - https://dl.acm.org/doi/10.1145/226643.226647
    - https://classes.cs.uchicago.edu/archive/2026/spring/23380-1/papers/hayashibara_phi.pdf
Summary: Evidence-backed design and intern implementation guide for replacing ad hoc WebSocket ping/pong coordination with a pure timed failure-detector state machine and a single connection supervisor.
LastUpdated: 2026-08-10T20:10:00-04:00
WhatFor: Designing, implementing, reviewing, and testing Sessionstream WebSocket heartbeat behavior without relying on timing-sensitive channel interactions.
WhenToUse: Before changing heartbeat, connection lifecycle, observer dispatch, shutdown, or WebSocket control-frame handling in pkg/sessionstream/transport/ws.
---




# Intern Guide to the Timed Failure Detector and WebSocket Heartbeat State Machine

## Executive summary

Sessionstream's WebSocket server needs to notice clients that disappear without completing a clean close handshake. It currently does this with application-level protobuf-JSON `PingFrame` and `PongFrame` messages, a ticker, a one-slot pong channel, and several cooperating goroutines. That implementation works in ordinary cases, but its correctness depends on subtle ordering among the socket reader, socket writer, heartbeat loop, request worker, observers, timers, and shutdown path.

This document proposes replacing the ad hoc heartbeat coordination with a small, pure **timed failure-detector state machine** and integrating that machine through one per-connection supervisor. The state machine does not perform network I/O, start goroutines, read clocks, or invoke observers. It receives explicit events and returns explicit actions. This makes its behavior deterministic, exhaustively testable, and explainable to a new contributor.

The core idea is:

```text
(current state, input event) -> (next state, ordered actions)
```

A timeout does not mathematically prove that a client failed. In an asynchronous system, silence may also mean delay, queue congestion, scheduler starvation, or a blocked local callback. The detector therefore establishes **suspicion under a configured timing assumption**. Sessionstream's policy may close a suspected connection, but the distinction remains important for naming, metrics, tests, and future adaptive policies.

The proposed implementation keeps the existing public wire contract and `ConnectionConfig` fields. It introduces an internal package, a typed event/action model, a monotonic generation number, a single outstanding challenge, timer-generation validation, and deterministic tests. The migration is intentionally internal-first: callers should not be forced onto a new public API while the model is being proven.

### Implementation status

Phases 0 through 6 were implemented on `task/sessionstream-005-heartbeat-machine` in commits `d0693bf` and `dbfbf02`. The implementation added the pure reducer under `internal/heartbeat`, the runtime adapter in `heartbeat.go`, generation-safe timer and nonce seams, a bounded observer dispatcher, and deterministic fake-time tests. It removed the old pong channel, latest-pong replacement helper, free-running ticker, and legacy heartbeat loops.

Implementation uncovered one scheduler-order refinement not explicit in the initial proposal: a real client can return a pong after the socket write succeeds but before the supervisor selects the writer acknowledgement. `PhaseWriting` therefore retains the earliest matching pong timestamp as pending. When `PingWritten` arrives, the reducer accepts that pending response only if it is at or after the successful write timestamp and before the derived deadline. This keeps correctness independent of Go `select` ordering.

## 1. Audience and learning goals

This guide is written for an intern who knows basic Go and HTTP but has not yet worked on distributed failure detection or Sessionstream. By the end, the reader should be able to:

1. Explain why heartbeat timeout means “suspected,” not “proven dead.”
2. Trace a WebSocket connection from HTTP upgrade through hello, heartbeat, subscriptions, fanout, and shutdown.
3. Explain why the timeout starts after the ping is written rather than queued.
4. Identify which goroutine owns socket reads and socket writes.
5. Implement the pure heartbeat reducer without timers or goroutines.
6. Integrate the reducer with a per-connection supervisor and fake clock.
7. Test safety, liveness-under-assumptions, stale messages, shutdown, and observer interference.
8. Review the implementation against explicit invariants rather than anecdotal test cases.

## 2. Scope

### 2.1 In scope

This ticket designs and plans:

- a formal state model for Sessionstream heartbeat detection;
- an internal Go API for events, states, and actions;
- one outstanding nonce-bearing challenge per connection;
- actual-write-based timeout semantics;
- timer generation and stale-event handling;
- supervisor integration with existing reader and writer ownership;
- observer isolation from the heartbeat critical path;
- deterministic, model-based, integration, browser, race, and shutdown tests;
- migration from the current pong channel and heartbeat loop;
- documentation of compatibility and operational behavior.

### 2.2 Out of scope

The first implementation should not:

- add distributed consensus;
- claim perfect crash detection;
- replace protobuf-JSON transport framing;
- change subscription, hydration, or UI fanout semantics;
- add a public pluggable failure-detector API before internal behavior stabilizes;
- implement phi-accrual detection immediately;
- use heartbeat as authentication or authorization;
- interpret latency as application health;
- hide event replay inside subscription semantics.

## 3. System orientation

### 3.1 What Sessionstream does

Sessionstream provides session-keyed command handling, event projection, durable hydration snapshots, and live UI fanout. The WebSocket adapter is not the command-ingress path. It allows a browser or other client to:

- establish a connection;
- receive a connection ID in a hello frame;
- subscribe and unsubscribe to session IDs;
- receive a snapshot during subscription hydration;
- receive subsequent projected UI events;
- exchange application-level ping and pong messages.

The package-level `Hub` owns the domain pipeline and can attach a `UIFanout`. The WebSocket `Server` implements that fanout boundary. See `pkg/sessionstream/hub.go:34-54`, `pkg/sessionstream/hub.go:104-125`, and `pkg/sessionstream/transport/ws/server.go:108-129`.

### 3.2 Current transport protocol

The protocol schema is `proto/sessionstream/v1/transport.proto`. `ClientFrame` has `subscribe`, `unsubscribe`, `ping`, and `pong` alternatives at lines 9-16. `ServerFrame` has hello, snapshot, subscription acknowledgements, UI events, errors, ping, and pong alternatives at lines 18-29. The heartbeat payload is deliberately tiny:

```protobuf
message PingFrame {
  string nonce = 1;
}

message PongFrame {
  string nonce = 1;
}
```

Both client and server can send application-level pings. A recipient echoes the nonce in a pong. This is separate from RFC 6455 protocol-level Ping and Pong control frames.

### 3.3 Why application-level heartbeat exists

RFC 6455 Sections 5.5.2 and 5.5.3 define WebSocket Ping and Pong control frames. Gorilla WebSocket supports protocol-level control handling, but browser JavaScript does not expose an API for sending arbitrary control frames. Sessionstream therefore uses normal protobuf-JSON messages so the same visible contract can be implemented by browser clients.

The shipped Systemlab helper at `cmd/sessionstream-systemlab/static/js/websocket.js:1-8` recognizes `frame.ping` and replies with `{pong: {nonce}}`. The Goja chat client has equivalent handling at `examples/goja-chatdemo-server/assets/public/app.js:90-92`. Generated Redis-demo assets mirror that source.

The distinction matters:

```text
RFC control Ping/Pong
  - handled at the WebSocket protocol layer
  - not directly available to browser JavaScript

Sessionstream PingFrame/PongFrame
  - ordinary application messages
  - decoded by Sessionstream
  - visible to clients, observers, queues, and tests
```

### 3.4 Existing connection configuration

`ConnectionConfig` currently defines five bounds in `pkg/sessionstream/transport/ws/server.go:28-46`:

- `MaxReadBytes`: maximum inbound message size;
- `SendQueueSize`: bounded outbound and request queue capacity;
- `WriteTimeout`: maximum time for one socket write;
- `HeartbeatInterval`: idle time between heartbeat attempts;
- `PongTimeout`: allowed time after a ping is written.

`WithConnectionConfig` validates and installs this configuration at `server.go:67-76`. These fields can remain public and source-compatible while their internal implementation changes.

### 3.5 Existing goroutine topology

After upgrade and registration, `ServeHTTP` starts three goroutines and runs the reader in the handler goroutine (`server.go:222-299`):

```text
HTTP handler goroutine
  `-- readLoop

per-connection goroutines
  |-- writeLoop
  |-- heartbeatLoopAfterReady -> heartbeatLoop
  `-- requestLoopAfterReady -> requestLoop
```

Important ownership rules already exist:

- `readLoop` is the only code that calls `ReadMessage` (`server.go:341-394`).
- `writeLoop` is the only code that calls `SetWriteDeadline` and `WriteMessage` (`server.go:537-561`).
- request processing is serialized so blocking snapshot hydration does not stop socket reads (`server.go:415-535`).
- hello is queued before other producers start, and `ready` opens after hello is actually written (`server.go:260-297`).
- `closeConnection` is idempotent through `sync.Once` (`server.go:641-668`).

These are good boundaries and should be preserved.

### 3.6 Existing heartbeat mechanics

The current heartbeat loop at `server.go:574-638` performs this sequence:

1. Wait for a ticker tick.
2. Generate a nonce from connection ID and `UnixNano`.
3. Enqueue a ping with a write-completion channel.
4. Wait until the writer reports the ping was written.
5. Start a `PongTimeout` timer.
6. Read nonce strings from `c.pongs` until one matches.
7. Close the connection if the timer fires.

The reader treats ping and pong as control messages before ordinary request queuing (`server.go:341-385`). The current `offerLatestPong` helper replaces a stale value in a one-slot channel (`server.go:397-413`).

This design has accumulated correct local protections, but heartbeat state remains distributed across:

- `c.ready`;
- `c.done`;
- `c.pongs`;
- ticker state;
- a per-cycle timer;
- the outbound frame's `written` acknowledgement;
- the writer goroutine;
- the reader goroutine;
- synchronous observer callbacks;
- connection close state.

The proposed state machine makes those relationships explicit.

## 4. Theoretical foundation

### 4.1 Failure detection is epistemic

A detector observes messages and local time. It does not inspect the remote process directly. Suppose no pong arrives before a deadline. The following worlds are observationally equivalent to the server:

1. The browser process crashed.
2. The browser is alive but its event loop is blocked.
3. A network path dropped the ping.
4. A network path delayed the pong.
5. Sessionstream queued the ping but did not write it promptly.
6. Sessionstream received the pong but blocked before processing it.
7. The host scheduler paused the process.

Because the detector receives the same evidence—silence—in each world, it cannot prove which world occurred. This is why distributed-systems literature calls such components **unreliable failure detectors**. Chandra and Toueg characterize detectors through completeness and accuracy rather than omniscience.

For Sessionstream:

- **Completeness goal:** a permanently unreachable client is eventually suspected.
- **Accuracy goal:** a responsive client is not suspected when the configured timing assumptions hold.
- **Reality:** temporary false suspicion remains possible when assumptions do not hold.

### 4.2 Partial synchrony assumption

A fixed timeout becomes useful under a partial-synchrony assumption: during healthy operation, relevant delays are eventually bounded. Define:

```text
D_total = D_network_out
        + D_client_dispatch
        + D_client_response
        + D_network_in
        + D_server_read
```

The timeout should satisfy:

```text
PongTimeout > expected upper bound of D_total + safety margin
```

The server's outbound queue delay is intentionally excluded because the timeout starts only after successful ping write acknowledgement. Blocking observer delay must also be excluded by processing control state before invoking, or independently of, observer callbacks.

### 4.3 Fixed-threshold detector

The first design uses a binary fixed-threshold detector:

```text
trusted   --deadline expires--> suspected
suspected --policy action-----> connection closed
```

This is appropriate because Sessionstream needs a simple connection-liveness policy, not cluster membership or consensus. An adaptive phi-accrual detector can be considered later if deployments show high latency variance.

### 4.4 Phi-accrual alternative

A phi-accrual detector reports a continuously increasing suspicion level instead of a Boolean answer. Informally:

```text
phi(t) = -log10(P(next heartbeat arrives later than t))
```

A deployment chooses a threshold such as `phi >= k`. This can adapt to observed timing distributions, but it requires a sample window, minimum standard deviation, warm-up policy, and careful behavior after idle periods. It is not justified for the first refactor because the current problem is ownership and determinism, not timeout calibration.

### 4.5 Safety and conditional liveness

The state machine should satisfy two classes of properties.

**Safety properties** say something bad never happens:

- no heartbeat begins before hello is written;
- at most one challenge is outstanding;
- a stale nonce never acknowledges the current challenge;
- a stale timeout from an old generation never closes a healthy connection;
- observer latency cannot delay pong acceptance;
- no transition occurs after the machine is stopped.

**Conditional liveness properties** say something good eventually happens if assumptions hold:

- after a tick in idle state, a ping is eventually offered to the writer;
- after a ping is written and its matching pong arrives before the deadline, the machine returns to idle;
- after a written ping receives no matching pong by its deadline, the machine becomes suspected;
- after stop, the supervisor and timer worker eventually terminate.

Liveness depends on fair scheduling, functioning local I/O, and delivery within the configured bound. Tests must state these assumptions rather than claiming unconditional guarantees.

## 5. Current-state gap analysis

### 5.1 State is implicit

The current code has no value that answers “what heartbeat phase is this connection in?” A reviewer must infer phase from which goroutine is blocked on which channel. This makes illegal transitions difficult to reject and difficult to observe.

### 5.2 Timer identity is implicit

A timer belongs to the nonce in the current loop iteration, but that relationship is lexical rather than represented as data. Once scheduling is moved or timers are reused, a stale timeout event could be mistaken for the active deadline unless every timer carries a generation.

### 5.3 Pong buffering encodes policy accidentally

A one-slot channel is both transport and state. Whether it keeps the oldest or newest value changes correctness. The recent `offerLatestPong` fix chooses newest, but direct state-machine nonce comparison is clearer: pongs are events, not durable queue entries.

### 5.4 Synchronous observers can interfere

`observe` invokes `TransportObserver.OnTransport` synchronously at `observer.go:119-126`. Panics are recovered, but blocking is permitted in practice. Heartbeat control processing was moved before observer calls in the reader, yet the heartbeat loop itself still invokes observers synchronously. A state-machine supervisor must never invoke user callbacks on its event-processing path.

### 5.5 Ticker semantics are not named

The current ticker continues while the loop waits for pong. A tick can remain buffered and trigger a new ping immediately after a delayed successful pong. The desired semantic should be explicit: **wait `HeartbeatInterval` after a successful cycle before beginning the next challenge**. This avoids catch-up bursts and makes interval behavior easy to test.

### 5.6 Failure policy and detector mechanics are fused

The current timeout branch both detects a missed deadline and closes the connection. Separating these concepts allows tests to prove the detector transition independently from the WebSocket policy:

```text
Detector: Awaiting -> Suspected
Policy:   Suspected -> close connection
```

The initial policy remains “close immediately,” preserving behavior.

## 6. Proposed architecture

### 6.1 Component diagram

```text
                         application-level frames
        +---------------------------------------------------+
        |                                                   |
        v                                                   |
+----------------+      HeartbeatEvent      +------------------------+
| socket reader  | -----------------------> | connection supervisor  |
| one goroutine  |                          | owns heartbeat machine |
+----------------+                          +-----------+------------+
                                                        |
                                               HeartbeatAction
                                                        |
                           +----------------------------+------------------+
                           |                            |                  |
                           v                            v                  v
                    +-------------+              +------------+    +------------+
                    | write queue |              | timer port |    | close port |
                    +------+------+              +-----+------+    +------+-----+
                           |                           |                    |
                           v                           |                    |
                    +-------------+                    |                    |
                    | socket      | -- PingWritten -->+                    |
                    | writer      |                                         |
                    +-------------+                                         |
                                                                          v
                                                               idempotent closeConnection

Observer records are copied to a separate bounded dispatcher after critical
state transitions. Observer callbacks never run in the supervisor.
```

### 6.2 Package layout

Start with an internal package:

```text
pkg/sessionstream/transport/ws/
  server.go
  observer.go
  heartbeat.go
  internal/
    heartbeat/
      doc.go
      machine.go
      machine_test.go
      model_test.go
      clock.go          # test/runtime scheduling seam if needed
```

Why internal?

- It prevents premature public compatibility obligations.
- The first consumer is the WebSocket transport.
- The reducer can later be promoted if another transport demonstrates the same semantics.
- It keeps protobuf and Gorilla types out of the mathematical kernel.

### 6.3 Pure kernel

The kernel owns only protocol-neutral heartbeat state:

```go
type Phase uint8

const (
    PhaseBooting Phase = iota
    PhaseIdle
    PhaseWriting
    PhaseAwaiting
    PhaseSuspected
    PhaseStopped
)

type State struct {
    Phase         Phase
    Generation    uint64
    Nonce         string
    WrittenAt     time.Time
    Deadline      time.Time
    PendingPongAt time.Time
}

type Machine struct {
    state       State
    pongTimeout time.Duration
}
```

The state should not include a `time.Timer`, channel, context, WebSocket connection, logger, or observer.

### 6.4 Events

Use a closed set of typed events. An interface with unexported marker methods or a tagged struct are both viable. A tagged struct makes table-driven tests compact:

```go
type EventKind uint8

const (
    EventReady EventKind = iota
    EventTick
    EventPingWritten
    EventPingWriteFailed
    EventPongReceived
    EventDeadlineElapsed
    EventStop
)

type Event struct {
    Kind       EventKind
    At         time.Time
    Generation uint64
    Nonce      string
    Err        error
}
```

Event meanings:

- `Ready`: hello has been successfully written; heartbeat may begin.
- `Tick`: the configured idle interval elapsed.
- `PingWritten`: the sole writer successfully put the challenge on the wire.
- `PingWriteFailed`: the writer could not send the challenge.
- `PongReceived`: the reader decoded an application-level pong.
- `DeadlineElapsed`: a timer for a specific generation fired.
- `Stop`: connection shutdown began.

### 6.5 Actions

The reducer returns ordered actions that the supervisor executes:

```go
type ActionKind uint8

const (
    ActionScheduleTick ActionKind = iota
    ActionSendPing
    ActionArmDeadline
    ActionCancelDeadline
    ActionRecordPong
    ActionRecordStalePong
    ActionRecordSuspected
    ActionCloseConnection
    ActionStop
)

type Action struct {
    Kind       ActionKind
    At         time.Time
    Generation uint64
    Nonce      string
    Deadline   time.Time
    Reason     error
}
```

`ActionSendPing` should include generation and nonce. The supervisor translates it to the existing protobuf `PingFrame`, submits it through the sole writer, and converts write completion into `EventPingWritten` or `EventPingWriteFailed`.

### 6.6 Transition table

| Current phase | Event | Guard | Next phase | Actions |
|---|---|---|---|---|
| Booting | Ready | — | Idle | ScheduleTick |
| Booting | PongReceived | — | Booting | RecordStalePong |
| Idle | Tick | — | Writing | SendPing |
| Idle | PongReceived | — | Idle | RecordStalePong |
| Writing | PingWritten | event generation and nonce match | Awaiting | ArmDeadline |
| Writing | PingWriteFailed | generation matches | Suspected | RecordSuspected, CloseConnection |
| Writing | PongReceived | nonce matches | Writing | retain earliest pending pong timestamp |
| Writing | PongReceived | nonce differs | Writing | RecordStalePong |
| Awaiting | PongReceived | nonce matches | Idle | CancelDeadline, RecordPong, ScheduleTick |
| Awaiting | PongReceived | nonce differs | Awaiting | RecordStalePong |
| Awaiting | DeadlineElapsed | generation matches and time is at/after deadline | Suspected | RecordSuspected, CloseConnection |
| Awaiting | DeadlineElapsed | generation stale | Awaiting | no action or diagnostic |
| Suspected | any except Stop | — | Suspected | no action |
| any | Stop | — | Stopped | CancelDeadline, Stop |
| Stopped | any | — | Stopped | no action |

Illegal or impossible event combinations should return a typed invariant error in tests and development. Runtime adapters may choose to record and close on invariant violation.

### 6.7 State diagram

```text
                             matching Pong
                        +-----------------------+
                        |                       |
                        v                       |
Booting --Ready--> Idle --Tick--> Writing --PingWritten--> Awaiting
                    ^              |                         |
                    |              | PingWriteFailed         | matching deadline
                    |              v                         v
                    +--------- Suspected <---------------- Suspected
                                  |
                                  | close policy
                                  v
                               Stopped

Stale pong or stale deadline: state does not advance.
Stop from every phase: Stopped.
```

### 6.8 Generation and nonce

Generation is local machine identity; nonce is wire identity.

```text
generation = monotonically increasing uint64
nonce       = opaque string carried by PingFrame and PongFrame
```

Every new challenge increments generation. Timer events carry generation so an old timer cannot affect a later challenge. A nonce can encode connection ID and generation for diagnostics, but correctness should not parse the nonce.

Recommended generator interface in the adapter:

```go
type NonceSource interface {
    Next(connectionID sessionstream.ConnectionId, generation uint64) (string, error)
}
```

The default can use cryptographically random bytes plus a generation suffix. Cryptographic unpredictability is not required for liveness, but `crypto/rand` avoids accidental collisions and makes clear that wall-clock timestamps are not identity authorities. Tests use a deterministic source.

Bound generated nonce length, for example to 64 ASCII characters. An inbound pong remains covered by `MaxReadBytes`; the reducer should reject or classify an empty/oversized nonce before retaining it.

### 6.9 Supervisor

The supervisor is the only goroutine that mutates the heartbeat machine. Reader, writer acknowledgements, and timers send it events.

```go
func runHeartbeatSupervisor(ctx context.Context, ports Ports, machine *heartbeat.Machine) {
    events := ports.Events()
    dispatch(Event{Kind: EventReady, At: ports.Clock.Now()})

    for {
        select {
        case <-ctx.Done():
            dispatch(Event{Kind: EventStop, At: ports.Clock.Now()})
            return
        case event := <-events:
            actions, err := machine.Step(event)
            if err != nil {
                ports.Close(err)
                return
            }
            for _, action := range actions {
                execute(action)
            }
        }
    }
}
```

The pseudocode is intentionally incomplete: production code must avoid blocking the supervisor while an outbound queue, observer, or timer subsystem is stalled. Action execution rules are therefore part of the design.

### 6.10 Action execution rules

#### SendPing

- Construct `ServerFrame_Ping`.
- Submit it to the existing bounded writer queue.
- Do not start `PongTimeout` yet.
- Arrange for write completion to become an event.
- If queue admission fails, treat it as a transport failure and close.

#### ArmDeadline

- Start or reset a timer only after `PingWritten`.
- Capture generation in the timer callback/event.
- Never let a timer callback mutate the machine directly.

#### CancelDeadline

- Stop and safely drain/reset the current timer in the timer owner.
- It is still valid for a stale timer event to race with cancellation; generation validation makes that harmless.

#### ScheduleTick

- Schedule one idle-delay event after successful readiness or matching pong.
- Do not use a free-running ticker that accumulates catch-up ticks.

#### Record actions

- Convert to immutable `TransportRecord` values.
- Offer records to a bounded best-effort dispatcher.
- Never invoke `OnTransport` from the supervisor.

#### CloseConnection

- Call the existing idempotent `closeConnection` path.
- Do not close shared event channels from arbitrary producers.
- Rely on connection context/done signals to stop reader, writer, request worker, supervisor, and dispatcher ownership loops.

### 6.11 Ports and dependency injection

A practical adapter API could be:

```go
type heartbeatPorts struct {
    clock       Clock
    nonceSource NonceSource
    sendPing    func(generation uint64, nonce string)
    close       func(error)
    observe     func(TransportRecord)
}

type Clock interface {
    Now() time.Time
    AfterFunc(time.Duration, func()) Timer
}

type Timer interface {
    Stop() bool
}
```

An alternative is for the supervisor itself to own one `time.Timer` and expose only `Now`. That is simpler and preferable if fake-time tests can drive the supervisor deterministically. Do not introduce a large generic clock framework solely for this ticket.

### 6.12 Writer acknowledgement contract

The current `outboundFrame` has `written chan error` (`server.go:175-179`) and `sendFrameTracked` supplies that acknowledgement. Preserve the concept but define its contract:

```text
nil acknowledgement
  means WriteMessage returned nil for this exact frame

non-nil acknowledgement
  means write setup or WriteMessage failed

connection done before acknowledgement
  means the challenge was not accepted as written
```

The writer must send the acknowledgement before invoking a potentially blocking observer. Current `writeLoop` observes `ServerFrameWritten` before `notifyFrameWritten` at `server.go:552-558`; the implementation phase should reverse that order or dispatch observation asynchronously.

### 6.13 Observer dispatch

`TransportObserver` is documented as best-effort but currently synchronous. Introduce a bounded dispatcher owned by the server or connection:

```go
type observerDispatcher struct {
    records chan TransportRecord
    done    chan struct{}
}

func (d *observerDispatcher) Offer(rec TransportRecord) {
    safe := cloneTransportRecord(rec)
    select {
    case d.records <- safe:
    default:
        // Increment an internal dropped-observation counter.
    }
}
```

The dispatcher invokes callbacks with panic recovery. Its queue must be bounded. Shutdown must not wait forever for a blocking observer; either observer callbacks receive a cancelable context and are excluded from hard shutdown waits after deadline, or the API explicitly requires nonblocking callbacks and tests enforce only transport independence. This requires a focused decision during implementation because Go cannot forcibly stop a blocked callback.

## 7. Formal invariants

Give each invariant a stable identifier so code comments and tests can cite it.

### HB-1: Readiness precedes heartbeat

```text
SendPing implies hello has been written successfully.
```

### HB-2: Single outstanding challenge

```text
phase in {Writing, Awaiting} implies exactly one current generation and nonce.
```

### HB-3: Timeout begins at actual write

```text
phase = Awaiting implies WrittenAt is set and Deadline = WrittenAt + PongTimeout.
```

### HB-4: Exact nonce acceptance

```text
PongReceived(m) acknowledges the challenge if and only if m = current nonce.
```

### HB-5: Generation-safe timers

```text
DeadlineElapsed(g) changes state only if g = current generation.
```

### HB-6: Control-path independence

```text
Reader delivery, write acknowledgement, timeout handling, and close do not wait for observers, hydration, or ordinary request processing.
```

### HB-7: One socket reader and one socket writer

Gorilla WebSocket permits one concurrent reader and one concurrent writer. No heartbeat refactor may call `ReadMessage`, `WriteMessage`, `SetWriteDeadline`, or equivalent write methods from an additional goroutine.

### HB-8: Stop is absorbing

```text
Step(Stopped, event) = (Stopped, no actions)
```

### HB-9: Bounded resources

Per connection:

- event queues are bounded;
- outbound queue remains bounded;
- there is at most one heartbeat timer;
- there is at most one heartbeat supervisor;
- stale pong input does not allocate unbounded storage.

### HB-10: Detector and policy are distinct

```text
Missed deadline -> Suspected
Sessionstream policy -> close connection
```

Tests should be able to assert the first without requiring a real socket.

## 8. API details and pseudocode

### 8.1 Reducer skeleton

```go
func (m *Machine) Step(event Event) ([]Action, error) {
    if m.state.Phase == PhaseStopped {
        return nil, nil
    }
    if event.Kind == EventStop {
        actions := []Action{{Kind: ActionStop}}
        if m.state.Phase == PhaseAwaiting {
            actions = append([]Action{{Kind: ActionCancelDeadline}}, actions...)
        }
        m.state = State{Phase: PhaseStopped}
        return actions, nil
    }

    switch m.state.Phase {
    case PhaseBooting:
        return m.stepBooting(event)
    case PhaseIdle:
        return m.stepIdle(event)
    case PhaseWriting:
        return m.stepWriting(event)
    case PhaseAwaiting:
        return m.stepAwaiting(event)
    case PhaseSuspected:
        return nil, nil
    default:
        return nil, ErrInvalidPhase
    }
}
```

### 8.2 Tick transition

```go
func (m *Machine) onTick(event Event) ([]Action, error) {
    if m.state.Phase != PhaseIdle {
        return nil, nil // stale or duplicate scheduler event
    }

    generation := m.state.Generation + 1
    nonce, err := m.nonces.Next(generation)
    if err != nil {
        return m.suspect(err)
    }

    m.state = State{
        Phase:      PhaseWriting,
        Generation: generation,
        Nonce:      nonce,
    }
    return []Action{{
        Kind:       ActionSendPing,
        Generation: generation,
        Nonce:      nonce,
    }}, nil
}
```

If nonce generation lives outside the pure reducer, split this into `Tick -> NeedNonce` and `ChallengePrepared`. The implementation should choose one approach and keep random I/O outside deterministic state mutation.

### 8.3 Write acknowledgement transition

```go
func (m *Machine) onPingWritten(event Event) ([]Action, error) {
    if m.state.Phase != PhaseWriting {
        return nil, nil // stale acknowledgement
    }
    if event.Generation != m.state.Generation || event.Nonce != m.state.Nonce {
        return nil, nil
    }

    deadline := event.At.Add(m.pongTimeout)
    m.state.Phase = PhaseAwaiting
    m.state.WrittenAt = event.At
    m.state.Deadline = deadline

    return []Action{{
        Kind:       ActionArmDeadline,
        Generation: event.Generation,
        Deadline:   deadline,
    }}, nil
}
```

### 8.4 Pong transition

```go
func (m *Machine) onPong(event Event) ([]Action, error) {
    if m.state.Phase != PhaseAwaiting || event.Nonce != m.state.Nonce {
        return []Action{{Kind: ActionRecordStalePong, Nonce: event.Nonce}}, nil
    }

    generation := m.state.Generation
    m.state = State{Phase: PhaseIdle, Generation: generation}
    return []Action{
        {Kind: ActionCancelDeadline, Generation: generation},
        {Kind: ActionRecordPong, Generation: generation, Nonce: event.Nonce},
        {Kind: ActionScheduleTick, Generation: generation},
    }, nil
}
```

A pong that arrives after the nominal deadline but before the timer event is processed needs an explicit policy. Recommended policy: compare `event.At` with stored deadline and classify it as late if `At` is at or after the deadline. This avoids scheduler ordering deciding correctness.

### 8.5 Deadline transition

```go
func (m *Machine) onDeadline(event Event) ([]Action, error) {
    if m.state.Phase != PhaseAwaiting {
        return nil, nil
    }
    if event.Generation != m.state.Generation {
        return nil, nil
    }
    if event.At.Before(m.state.Deadline) {
        return nil, ErrEarlyDeadline
    }

    reason := ErrPongDeadlineExceeded
    m.state.Phase = PhaseSuspected
    return []Action{
        {Kind: ActionRecordSuspected, Reason: reason},
        {Kind: ActionCloseConnection, Reason: reason},
    }, nil
}
```

### 8.6 Reader integration

```go
switch typed := frame.GetFrame().(type) {
case *sessionstreamv1.ClientFrame_Ping:
    // Protocol response remains on the sole writer path.
    enqueuePong(typed.Ping.GetNonce())

case *sessionstreamv1.ClientFrame_Pong:
    // Offer event before any observer callback.
    heartbeatEvents.Offer(Event{
        Kind:  EventPongReceived,
        At:    clock.Now(),
        Nonce: typed.Pong.GetNonce(),
    })

case *sessionstreamv1.ClientFrame_Subscribe,
     *sessionstreamv1.ClientFrame_Unsubscribe:
    requestQueue.Offer(frame)
}
```

If the bounded heartbeat event queue is full, close the connection rather than silently dropping the matching pong. Under the intended topology, one supervisor and a small event rate should make overflow exceptional and diagnostic.

### 8.7 End-to-end sequence

```text
Client               readLoop          supervisor         writeLoop          timer
  |                      |                  |                  |                |
  |--- HTTP upgrade ---->|                  |                  |                |
  |<----- hello ------------------------------ write ---------|                |
  |                      |--- Ready -------->|                  |                |
  |                      |                  |--- schedule --------------------->|
  |                      |                  |<------------- Tick --------------|
  |                      |                  |--- SendPing ---->|                |
  |<------ ping n -------------------------------- write -----|                |
  |                      |                  |<-- PingWritten --|                |
  |                      |                  |--- arm deadline ---------------->|
  |------ pong n ------->|                  |                  |                |
  |                      |--- Pong(n) ----->|                  |                |
  |                      |                  |--- cancel ----------------------->|
  |                      |                  |--- schedule next tick ---------->|
```

Timeout sequence:

```text
write ping n -> PingWritten(n, t) -> arm deadline(n, t + timeout)
                                        |
                                        +-- no matching pong
                                        |
deadline(n) -> Awaiting to Suspected -> record -> closeConnection
```

## 9. Design decisions

### Decision: Model heartbeat as an unreliable timed failure detector

- **Context:** Silence cannot distinguish crash, network delay, client event-loop delay, or local scheduling delay.
- **Options considered:** Treat timeout as proof of death; treat it as a health check; model it as suspicion under timing assumptions.
- **Decision:** Use failure-detector terminology internally and expose timeout as a suspicion-triggered close policy.
- **Rationale:** This matches distributed-systems theory and prevents misleading guarantees.
- **Consequences:** Metrics and docs should say “heartbeat timeout” or “suspected,” not “proved dead.” False positives remain possible.
- **Status:** proposed

### Decision: Use a pure reducer with explicit events and actions

- **Context:** Current correctness is spread across goroutines, channels, timers, and callbacks.
- **Options considered:** Continue patching loops; create a mutex-protected heartbeat object; use a pure state machine plus supervisor.
- **Decision:** Implement a pure deterministic reducer and an I/O adapter.
- **Rationale:** Transition tests can cover every state/event pair without sleeping or opening sockets.
- **Consequences:** More types are introduced, but concurrency logic becomes smaller and reviewable.
- **Status:** proposed

### Decision: Keep the first implementation internal to the WebSocket transport

- **Context:** No second transport currently needs the abstraction.
- **Options considered:** Public `sessionstream.FailureDetector`; package-level WebSocket API; internal package.
- **Decision:** Use `pkg/sessionstream/transport/ws/internal/heartbeat`.
- **Rationale:** Avoids premature public API commitment while preserving promotion later.
- **Consequences:** External callers configure existing durations rather than constructing detectors.
- **Status:** proposed

### Decision: Permit one outstanding challenge

- **Context:** Multiple overlapping nonces require sets, independent deadlines, and ambiguous liveness interpretation.
- **Options considered:** Free-running periodic pings; bounded overlapping window; one challenge followed by idle delay.
- **Decision:** At most one challenge may be writing or awaiting pong.
- **Rationale:** It is sufficient for connection liveness and produces a compact state space.
- **Consequences:** Detection cadence is `interval + round trip`, not a rigid wall-clock period.
- **Status:** proposed

### Decision: Start timeout after successful socket write

- **Context:** Outbound queue delay is controlled by the server, not the client.
- **Options considered:** Start at schedule, queue admission, writer dequeue, or successful write.
- **Decision:** Start from successful `WriteMessage` completion.
- **Rationale:** Prevents local backpressure from falsely consuming client response budget.
- **Consequences:** A blocked write is governed by `WriteTimeout`; writer acknowledgement must precede observer callbacks.
- **Status:** proposed

### Decision: Add generation identity in addition to wire nonce

- **Context:** Timer cancellation races can deliver old timeout events.
- **Options considered:** Trust timer stop; compare nonce only; use monotonic generation plus nonce.
- **Decision:** Every challenge and timer carries a local generation.
- **Rationale:** Stale events become harmless by construction.
- **Consequences:** Tests should intentionally deliver old generation events.
- **Status:** proposed

### Decision: Keep the existing application-level wire frames

- **Context:** Browser JavaScript cannot directly originate RFC 6455 control frames, and shipped clients already implement protobuf-JSON ping/pong.
- **Options considered:** Protocol-level control frames only; application frames only; both.
- **Decision:** Preserve application frames for compatibility; protocol-level keepalive may be separately evaluated later.
- **Rationale:** No wire migration is required, and browser behavior remains visible and testable.
- **Consequences:** Heartbeat messages traverse application decoding and must remain prioritized over hydration and observers.
- **Status:** proposed

### Decision: Keep observers outside the control path

- **Context:** `TransportObserver` is extension code and may block or panic.
- **Options considered:** Synchronous callbacks; one goroutine per record; bounded dispatcher.
- **Decision:** Use a bounded best-effort dispatcher and transition before observation.
- **Rationale:** Unbounded goroutines leak under blocked callbacks; synchronous calls violate liveness isolation.
- **Consequences:** Observation records may be dropped under overload and need a drop counter or log.
- **Status:** proposed

### Decision: Defer phi-accrual detection

- **Context:** Adaptive suspicion can reduce false positives in variable-latency deployments.
- **Options considered:** Fixed deadline; phi accrual immediately; configurable detector interface.
- **Decision:** First implement a fixed threshold behind an internal reducer.
- **Rationale:** The observed defects concern concurrency and state ownership, not statistical calibration.
- **Consequences:** A future implementation can reuse supervisor events if operational evidence justifies adaptation.
- **Status:** proposed

## 10. Implementation plan

### Phase 0: Freeze behavior and invariants

1. Add this document's HB-1 through HB-10 identifiers to test names or comments.
2. Record current defaults and public API behavior.
3. Freeze protobuf JSON fixtures for ping and pong.
4. Freeze hello-first, matching-pong, missed-pong, and shutdown behavior.
5. Confirm all shipped JavaScript clients echo nonces.

Deliverable: characterization tests pass before refactoring.

### Phase 1: Build the pure kernel

Create `internal/heartbeat/machine.go` with:

- phase enum;
- immutable-readable state snapshot;
- event kinds;
- action kinds;
- reducer entry point;
- typed invariant errors;
- no goroutines, timers, channels, contexts, sockets, observers, or protobuf imports.

Create exhaustive table tests for every `(phase, event kind)` pair. A matrix makes omissions visible:

```text
             Ready Tick Written Failed Pong Deadline Stop
Booting        X    X      X       X     X      X      X
Idle           X    X      X       X     X      X      X
Writing        X    X      X       X     X      X      X
Awaiting       X    X      X       X     X      X      X
Suspected      X    X      X       X     X      X      X
Stopped        X    X      X       X     X      X      X
```

Here `X` means the behavior is asserted, even when it is “ignore safely.”

### Phase 2: Add model and property tests

Generate event sequences and assert invariants after every step:

```go
func FuzzMachine(f *testing.F) {
    f.Fuzz(func(t *testing.T, encoded []byte) {
        m := New(...)
        for _, event := range decodeEvents(encoded) {
            actions, _ := m.Step(event)
            assertStateWellFormed(t, m.State())
            assertActionOrder(t, actions)
            assertSingleOutstanding(t, m.State())
        }
    })
}
```

Add targeted properties:

- matching pong before deadline returns to idle;
- every nonmatching pong preserves the active challenge;
- stale deadline generation never suspects;
- no action after stop;
- deadline cannot exist before write acknowledgement;
- generation never decreases;
- at most one arm action exists per generation.

Optionally create a small TLA+ or PlusCal model under the ticket's `scripts/` directory if review uncovers ambiguity. The Go reducer and tests remain the implementation authority.

### Phase 3: Introduce the supervisor adapter

1. Add a per-connection heartbeat event queue.
2. Start one supervisor only after connection registration.
3. Send `Ready` after hello write acknowledgement.
4. Translate reducer actions into writer, timer, observation, and close operations.
5. Ensure event queue admission is bounded and closure-aware.
6. Add deterministic nonce and fake-clock seams for tests.
7. Keep existing heartbeat loop behind a temporary test-only comparison if needed; do not ship duplicate detectors.

### Phase 4: Integrate reader and writer

Reader changes:

- decode ping/pong before ordinary request dispatch;
- send pong events to the supervisor before observation;
- never block on hydration or observer work;
- close on heartbeat event queue overflow.

Writer changes:

- preserve sole writer ownership;
- notify frame completion immediately after `WriteMessage` returns;
- enqueue observation after acknowledgement;
- include generation metadata in internal tracked frames without changing wire payload.

### Phase 5: Isolate observer dispatch

1. Add bounded observation dispatch.
2. Clone records before queue admission.
3. Recover callback panics in the dispatcher.
4. define drop metrics/logging.
5. Verify blocked observers cannot delay hello acknowledgement, pong processing, deadlines, or `Close(ctx)` beyond documented policy.

This phase may be split into a dedicated follow-up if general observer lifecycle changes exceed heartbeat scope. If split, heartbeat-critical records must still avoid synchronous callbacks.

### Phase 6: Remove old mechanics

Delete:

- `connection.pongs`;
- `offerLatestPong`;
- `heartbeatLoopAfterReady`;
- `heartbeatLoop`;
- ticker-based catch-up semantics;
- tests that inspect pong-channel implementation details.

Replace them with behavior-level tests. Do not leave compatibility shims or duplicate heartbeat goroutines.

### Phase 7: Validate and document

Run:

```bash
GOWORK=off go test ./pkg/sessionstream/transport/ws/... -count=1
GOWORK=off go test -race ./pkg/sessionstream/transport/ws/... -count=20
GOWORK=off go test ./... -count=1
GOWORK=off go vet ./...
GOWORK=off go build ./...
make lint
GOWORK=off govulncheck ./...
```

Also:

- run focused state-machine tests hundreds of times without wall-clock sleeps;
- run browser JavaScript syntax checks;
- run Systemlab/manual reconnect scenarios;
- run repository hooks and release snapshot checks;
- document any unrelated flaky test rather than weakening heartbeat assertions.

## 11. Testing strategy

### 11.1 Pure transition tests

These are the primary correctness tests. They must not call `time.Sleep`. Use fixed timestamps and explicit events.

Example:

```go
func TestMatchingPongBeforeDeadlineReturnsIdle(t *testing.T) {
    t0 := time.Unix(0, 0)
    m := newReadyMachine(t0)

    actions := stepTick(t, m, t0)
    challenge := requireSendPing(t, actions)

    stepWritten(t, m, challenge, t0.Add(time.Millisecond))
    actions = stepPong(t, m, challenge.Nonce, t0.Add(2*time.Millisecond))

    require.Equal(t, PhaseIdle, m.State().Phase)
    requireActionKinds(t, actions,
        ActionCancelDeadline,
        ActionRecordPong,
        ActionScheduleTick,
    )
}
```

### 11.2 Boundary-time tests

Specify exact policy for:

- pong one nanosecond before deadline;
- pong exactly at deadline;
- pong one nanosecond after deadline;
- deadline event delivered early;
- pong and deadline events delivered in either scheduler order.

Recommended rule: event timestamps determine semantics, not channel selection order. A pong at or after deadline is late.

### 11.3 Stale-event tests

Cover:

- duplicate current pong;
- previous generation pong during current challenge;
- unsolicited pong while idle;
- old write acknowledgement;
- old timer after cancellation;
- tick while writing or awaiting;
- events after stop.

### 11.4 Integration tests with fake time

A fake clock should advance explicitly:

```text
connect -> receive hello -> advance interval -> receive ping
respond matching -> advance timeout -> connection remains open
advance interval -> receive next ping
withhold pong -> advance timeout -> connection closes
```

Avoid millisecond sleeps as the main assertion mechanism. Real-time smoke tests can remain as secondary coverage.

### 11.5 Interference tests

Block each neighboring subsystem independently:

- snapshot hydration;
- subscription authorization;
- transport observer callback;
- outbound non-heartbeat fanout;
- request queue worker;
- server close observer;
- writer queue before ping write.

Then verify which delays are allowed:

| Blocked component | Socket reads continue | Matching pong accepted | Timeout starts | Close remains bounded |
|---|---:|---:|---:|---:|
| Hydration | yes | yes | after write | yes |
| Authorization | yes | yes | after write | yes |
| Observer | yes | yes | after write | policy-defined, never indefinite |
| Writer before ping write | yes | not yet relevant | no | yes |
| Client event loop | server reads other frames | no pong | yes | closes after deadline |

### 11.6 Race and leak tests

Use race tests to overlap:

- pong with deadline cancellation;
- stop with tick;
- stop with write acknowledgement;
- close with event queue admission;
- connection upgrade with server close;
- observer dispatch with shutdown.

Count goroutines only with a stable leak-testing library or explicit completion channels; raw global goroutine counts are noisy.

### 11.7 Wire compatibility tests

Freeze JSON:

```json
{"ping":{"nonce":"n-1"}}
{"pong":{"nonce":"n-1"}}
```

Confirm existing browser helpers still respond. No schema migration is required.

### 11.8 Operational tests

Expose or observe enough data to answer:

- How many ping cycles completed?
- How many stale pongs arrived?
- How many timeouts caused suspicion?
- How much time elapsed from write to matching pong?
- Were observer records dropped?
- Did outbound queue pressure delay ping write?

Do not put unbounded cardinality such as raw nonce into metrics labels.

## 12. Review guide

Review in this order:

1. `proto/sessionstream/v1/transport.proto` for wire meaning.
2. `pkg/sessionstream/transport/ws/internal/heartbeat/machine.go` for transition correctness.
3. `machine_test.go` for exhaustive state/event coverage.
4. `heartbeat.go` for timer, queue, nonce, and action execution.
5. `server.go` reader/writer ownership and lifecycle integration.
6. `observer.go` for noninterference.
7. browser clients for nonce echo behavior.
8. race and shutdown tests.

Ask these questions:

- Can any path create two outstanding challenges?
- Can a timeout exist before a successful ping write?
- Can an old generation close a current connection?
- Can an observer or hydration callback prevent pong acceptance?
- Can any goroutine besides `writeLoop` call Gorilla write methods?
- Does every goroutine have a cancellation and completion path?
- Are queues and timers bounded?
- Does shutdown remain idempotent?
- Are protocol and product compatibility unchanged?

## 13. Risks and mitigations

### Risk: State-machine abstraction adds ceremony

Mitigation: keep the reducer small and transport-neutral; avoid generic framework machinery.

### Risk: Supervisor event queue introduces another overflow mode

Mitigation: size it for bounded control traffic, prioritize control admission, test overflow, and close rather than silently lose the current pong.

### Risk: Fake clock diverges from runtime timer behavior

Mitigation: keep a small adapter, retain a few real-timer integration tests, and test stale timer delivery explicitly.

### Risk: Observer isolation changes delivery timing

Mitigation: preserve record contents and relative per-dispatcher order where documented; explicitly classify delivery as best-effort; add drop diagnostics.

### Risk: Interval semantics change

Mitigation: document that the next interval begins after successful pong. If strict current ticker behavior is required, model periodic ticks explicitly and still reject overlap.

### Risk: Close waits on blocked extension code

Mitigation: define whether observer callbacks are included in graceful waits. Never invoke callbacks while holding lifecycle locks. Honor `Close(ctx)` even if an extension fails to return.

### Risk: Random nonce generation failure

Mitigation: treat it as local transport failure, record internally, and close. Tests inject deterministic success and failure.

## 14. Alternatives considered

### Continue with channels and local patches

This has low immediate code movement but preserves implicit state and review difficulty. Rejected as the long-term design.

### Mutex-protected heartbeat object

A mutex can prevent races but does not define legal transitions or action ordering. It remains possible to arm a timer in the wrong phase. Rejected as the primary abstraction, though the adapter may use a mutex where ownership cannot remain single-threaded.

### RFC 6455 control frames only

This would use protocol-native ping/pong but would not expose the same client behavior to browser JavaScript. It may be useful as an additional lower-level keepalive, not as a transparent replacement. Deferred.

### Phi-accrual detector immediately

This addresses variable latency but increases policy and testing complexity before the ownership problem is solved. Deferred.

### One goroutine per timer or observation

This appears nonblocking but can create unbounded goroutines under overload or blocked callbacks. Rejected.

### Public generic detector framework

No second consumer currently proves the abstraction. Rejected until internal use establishes a stable contract.

## 15. Open questions

1. Should the observer dispatcher be per server or per connection?
2. Should a pong timestamp be captured immediately after JSON decode or immediately after socket read?
3. Is a pong exactly at the deadline accepted or late? This guide recommends late.
4. Should malformed or oversized nonce values close the connection or produce a protocol error?
5. Should protocol-level RFC ping/pong be added in addition to application heartbeat for nonbrowser clients?
6. Is `HeartbeatInterval` measured after matching pong, after ping write, or from a fixed epoch? This guide recommends after matching pong.
7. Should timeout observation retain a separate `suspected` stage before disconnection?
8. How should observer-drop counts be exposed without adding a metrics dependency?
9. Should write failure be classified as detector suspicion or direct transport failure? This guide recommends direct transport failure with the same close action but distinct telemetry.
10. At what point, if ever, should the internal kernel become a public package?

## 16. Definition of done

Implementation is complete when:

- HB-1 through HB-10 have direct test evidence;
- all phase/event combinations have deterministic tests;
- the reducer has no I/O or concurrency dependencies;
- one supervisor owns reducer mutation;
- reader and writer ownership still satisfy Gorilla's concurrency contract;
- timeout begins only after successful ping write;
- stale pong, write, tick, and timer events cannot break the current cycle;
- blocked hydration and observers cannot cause false timeout;
- old pong-channel and ticker-loop mechanics are removed;
- public configuration and wire fixtures remain compatible;
- workspace and `GOWORK=off` tests, race tests, vet, build, lint, vulnerability scan, hooks, and release snapshot pass;
- package documentation explains suspicion semantics and operational tuning;
- no TODO, shim, duplicate detector, unbounded queue, or leaked goroutine remains.

## 17. File reference map

| File | Why it matters |
|---|---|
| `proto/sessionstream/v1/transport.proto` | Authoritative client/server frame schema and ping/pong nonce fields. |
| `pkg/sessionstream/transport/ws/server.go` | Current lifecycle, queues, reader, writer, heartbeat, subscription, fanout, and shutdown implementation. |
| `pkg/sessionstream/transport/ws/server_test.go` | Existing heartbeat, hello ordering, hydration interference, close, and protocol tests. |
| `pkg/sessionstream/transport/ws/observer.go` | Transport stages, records, cloning, panic recovery, and synchronous callback behavior. |
| `pkg/sessionstream/hydration.go` | Snapshot contract that may block during subscription without blocking heartbeat reads. |
| `pkg/sessionstream/hub.go` | Domain pipeline and UI fanout attachment boundary. |
| `cmd/sessionstream-systemlab/static/js/websocket.js` | Shared browser application-heartbeat response helper. |
| `cmd/sessionstream-systemlab/static/js/pages/phase3.js` | Browser integration during hydration/reconnect lab. |
| `cmd/sessionstream-systemlab/static/js/pages/phase4.js` | Browser integration during chat lab. |
| `cmd/sessionstream-systemlab/static/js/pages/phase5.js` | Browser integration during persisted hydration lab. |
| `examples/goja-chatdemo-server/assets/public/app.js` | Shipped standalone browser client heartbeat handling. |
| `Makefile` | Repository validation and lint entry points. |
| `lefthook.yml` | Commit and push quality gates. |

## 18. External references

1. T. D. Chandra and S. Toueg, “Unreliable Failure Detectors for Reliable Distributed Systems,” Journal of the ACM, 1996: <https://dl.acm.org/doi/10.1145/226643.226647>.
2. N. Hayashibara et al., “The Phi Accrual Failure Detector,” SRDS 2004: <https://classes.cs.uchicago.edu/archive/2026/spring/23380-1/papers/hayashibara_phi.pdf>.
3. RFC 6455, “The WebSocket Protocol,” especially Sections 5.5.2 and 5.5.3: <https://datatracker.ietf.org/doc/html/rfc6455>.
4. Gorilla WebSocket package documentation, especially its concurrency section: <https://pkg.go.dev/github.com/gorilla/websocket>.
5. Sessionstream PR #10, lifecycle hardening context: <https://github.com/go-go-golems/sessionstream/pull/10>.
