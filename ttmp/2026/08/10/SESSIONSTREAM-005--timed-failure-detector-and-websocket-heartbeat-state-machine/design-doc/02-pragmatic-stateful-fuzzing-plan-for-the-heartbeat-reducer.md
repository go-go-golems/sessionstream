---
Title: Pragmatic Stateful Fuzzing Plan for the Heartbeat Reducer
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
    - Path: repo://pkg/sessionstream/transport/ws/internal/heartbeat/machine.go
      Note: Reducer whose state, events, actions, and errors define fuzz invariants
    - Path: repo://pkg/sessionstream/transport/ws/internal/heartbeat/machine_test.go
      Note: Current fuzzer and target implementation file
ExternalSources:
    - https://go.dev/doc/security/fuzz/
Summary: A deliberately bounded plan to make the heartbeat reducer fuzzer state-aware, deadline-aware, and useful across multiple challenge generations without building a second formal verification system.
LastUpdated: 2026-08-10T22:05:00-04:00
WhatFor: Implementing and reviewing the practical fuzz coverage added to the pure heartbeat reducer.
WhenToUse: Before changing FuzzMachinePreservesInvariants, its byte decoder, seed corpus, or reducer action invariants.
---


# Pragmatic Stateful Fuzzing Plan for the Heartbeat Reducer

## Executive summary

The heartbeat reducer already has deterministic transition-table and boundary tests plus a small native Go fuzzer. The existing fuzzer is useful for panic and malformed-order detection, but its generated nonces rarely match the active challenge. Many random traces therefore enter `Writing` and remain there until `Stop`, leaving successful pong cycles and deadline transitions underexplored.

This plan makes the existing fuzzer **state-aware** without introducing a second reference implementation, external property-testing framework, formal-language code generator, or CI-long campaign. Each fuzz byte encodes an operation, identity mode, and time mode. One operation is a state-sensitive “advance validly” command that gives random mutation a reliable path through repeated healthy cycles. The remaining operations inject stale identities, boundary timestamps, failures, duplicate events, and stop.

The implementation remains one fuzz target in `machine_test.go`. It adds a small decoder, stronger action/state assertions, and a compact seed corpus. A bounded local campaign proves that the harness executes and produces reusable corpus artifacts if it finds a failure. This is intentionally the last fuzzing scope for shipment.

## Goals

1. Reach `Booting`, `Idle`, `Writing`, `Awaiting`, `Suspected`, and `Stopped` through generated traces.
2. Complete multiple successful heartbeat generations.
3. Exercise current, stale, future, and empty identity combinations.
4. Exercise timestamps immediately before, exactly at, and immediately after deadlines.
5. Check observable action contracts in addition to state shape.
6. Preserve native `go test -fuzz` shrinking and corpus behavior.
7. Keep ordinary unit-test execution fast.

## Non-goals

This work will not add:

- TLA+, Quint, PlusCal, Alloy, or model-checker infrastructure;
- generated production code;
- an independent reference reducer;
- supervisor/socket fuzzing;
- network packet mutation;
- an external property-testing dependency;
- a long-running mandatory CI fuzz job;
- probabilistic timing or sleeps.

Those techniques may be interesting later, but they are not required to ship the current state-machine refactor.

## Current fuzzer

`FuzzMachinePreservesInvariants` currently maps every input byte directly to an `EventKind`. It uses the machine's current generation, assigns nonce `"next"` to ticks and `"nonce"` to most other events, advances time by whole seconds, ignores reducer errors, and checks generation monotonicity plus state shape.

The key weakness is identity mismatch:

```text
Tick creates challenge nonce "next"
PingWritten normally carries nonce "nonce"
Pong normally carries nonce "nonce"
```

Consequently, many traces cannot move from `Writing` to `Awaiting` or back to `Idle`. Deterministic tests cover those paths, but the fuzzer does not combine them deeply with arbitrary stale events.

## Proposed byte encoding

One byte is divided into three fields:

```text
bits 0..2: operation     (0 through 7)
bits 3..4: identity mode (0 through 3)
bits 5..7: time mode     (0 through 7)
```

Pseudocode:

```go
operation := raw & 0b00000111
identity  := (raw >> 3) & 0b00000011
timeMode  := (raw >> 5) & 0b00000111
```

This format is compact, mutation-friendly, and automatically shrinkable by Go's fuzz engine.

## Operations

| Value | Operation | Purpose |
|---:|---|---|
| 0 | Advance | Produce the normal next event for the current phase. |
| 1 | Ready | Duplicate, early, or normal readiness. |
| 2 | Tick | Start or duplicate a challenge tick. |
| 3 | PingWritten | Deliver matching or stale write completion. |
| 4 | PingWriteFailed | Deliver matching or stale write failure. |
| 5 | PongReceived | Deliver matching, stale, duplicate, early, or late pong. |
| 6 | DeadlineElapsed | Deliver matching, stale, early, exact, or late deadline. |
| 7 | Stop | Enter or repeat the absorbing terminal state. |

### State-sensitive Advance

`Advance` deliberately follows a healthy path:

```text
Booting  -> Ready
Idle     -> Tick with fresh nonce
Writing  -> matching PingWritten
Awaiting -> matching Pong one nanosecond before deadline
Suspected -> Stop
Stopped   -> Stop
```

This operation is important because random bytes can repeatedly traverse:

```text
Idle -> Writing -> Awaiting -> Idle -> Writing -> ...
```

without requiring three independently matching random choices.

## Identity modes

Identity applies where generation or nonce is meaningful.

| Value | Generation | Nonce |
|---:|---|---|
| 0 | current | current |
| 1 | previous/stale | current |
| 2 | next/future | stale |
| 3 | current | empty |

For `Tick`, current/fresh mode supplies a deterministic new nonce such as `fuzz-<next-generation>`. Empty mode intentionally exercises `ErrMissingNonce`.

For a machine with generation zero, “previous” remains zero rather than underflowing.

## Time modes

The decoder maintains a logical cursor and derives timestamps from current state.

| Value | Timestamp |
|---:|---|
| 0 | cursor plus one millisecond |
| 1 | active deadline minus one nanosecond |
| 2 | active deadline exactly |
| 3 | active deadline plus one nanosecond |
| 4 | active write-completion timestamp |
| 5 | unchanged cursor |
| 6 | cursor plus one pong timeout |
| 7 | cursor plus one second |

When no deadline or write timestamp exists, the decoder falls back to the cursor. The cursor advances to the maximum timestamp generated so ordinary traces tend forward while explicit modes can still produce duplicate or boundary times.

## Properties checked after every step

### State shape

Preserve the existing assertions:

```text
Writing  => generation > 0, nonce nonempty, no deadline
Awaiting => generation > 0, nonce nonempty,
            deadline = writtenAt + pongTimeout
```

Also assert:

```text
Idle and Booting have no active nonce or deadline
Stopped is absorbing
Generation never decreases
```

### Error atomicity

The reducer has two expected input errors:

- `ErrMissingNonce`;
- `ErrEarlyDeadline`.

For either error:

```text
state after = state before
actions = empty
```

Any other error fails the fuzz case.

### Action contracts

Check only stable, high-value contracts:

- `ActionSendPing` leaves the machine in `Writing` and carries the state's generation and nonce.
- `ActionArmDeadline` leaves the machine in `Awaiting` and matches the state's deadline.
- `ActionRecordPong` occurs only when the resulting machine is `Idle`.
- `ActionRecordSuspected` and `ActionCloseConnection` occur together and leave the machine `Suspected`.
- `ActionStop` leaves the machine `Stopped`.
- No action contains a generation greater than the machine's resulting generation, except zero-valued lifecycle actions.

Do not duplicate every transition-table assertion in the fuzzer. Deterministic tests remain the precise specification for action ordering.

## Seed corpus

Add readable byte sequences representing:

1. Two healthy cycles.
2. Pong before write acknowledgement.
3. Stale pong followed by matching pong.
4. Early deadline followed by exact deadline.
5. Matching write failure.
6. Stop from `Awaiting`.
7. Duplicate events after `Stopped`.

Seeds should be created through a tiny `fuzzByte(operation, identity, timeMode)` helper so reviewers can understand them. They remain byte slices at the `f.Add` boundary.

## Implementation sketch

```go
func FuzzMachinePreservesInvariants(f *testing.F) {
    addHeartbeatFuzzSeeds(f)

    f.Fuzz(func(t *testing.T, input []byte) {
        machine := mustNewMachine(t)
        cursor := epoch
        lastGeneration := uint64(0)

        for _, raw := range input {
            before := machine.State()
            event, nextCursor := decodeFuzzEvent(raw, before, cursor)
            actions, err := machine.Step(event)
            after := machine.State()

            assertExpectedFuzzErrorIsAtomic(t, before, after, actions, err)
            assertStateWellFormed(t, after)
            assertGenerationMonotonic(t, lastGeneration, after.Generation)
            assertFuzzActions(t, after, actions)

            cursor = nextCursor
            lastGeneration = after.Generation
        }
    })
}
```

The decoder and assertion helpers remain in `_test.go`; no production API changes are needed.

## Validation

Run ordinary seed regression:

```bash
GOWORK=off go test ./pkg/sessionstream/transport/ws/internal/heartbeat -count=100
```

Run race-enabled seed regression:

```bash
GOWORK=off go test -race ./pkg/sessionstream/transport/ws/internal/heartbeat -count=100
```

Run one bounded campaign:

```bash
GOWORK=off go test ./pkg/sessionstream/transport/ws/internal/heartbeat \
  -run='^$' \
  -fuzz=FuzzMachinePreservesInvariants \
  -fuzztime=60s
```

A 60-second local campaign is enough for this shipment. If it finds a failure, keep the generated corpus input and add a named deterministic regression when the scenario deserves explanation.

## Risks and controls

### Risk: State-aware generation hides malformed traces

Control: seven direct event operations remain available; only operation zero is the happy-path shortcut.

### Risk: Assertions reproduce implementation logic

Control: check coarse observable invariants and leave exact transition expectations in independent table tests.

### Risk: Fuzzer becomes hard to understand

Control: one-byte documented encoding, small helpers, named constants, and readable seeds.

### Risk: CI becomes slow or flaky

Control: ordinary tests run only seeds. The timed campaign remains an explicit validation command rather than a mandatory long CI job.

### Risk: Time generation produces meaningless overflow

Control: use bounded durations around a fixed epoch and cap input processing if necessary. Go's default fuzz byte slices are already practical, but the harness may limit itself to the first 4,096 operations to bound one execution.

## Definition of done

- The fuzzer can traverse at least two successful generations through a seed.
- Random operations can select matching, stale, future, and empty identities.
- Random operations can select before/exact/after deadline timestamps.
- Expected reducer errors are atomic.
- Stable action contracts are asserted.
- Existing deterministic and race tests pass.
- A bounded 60-second campaign completes without failure.
- The implementation adds no production dependency or API.
