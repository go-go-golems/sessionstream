# Phase 5 — Persistence and Restart Correctness

## What this chapter is about

Phase 3 showed you how the framework handles reconnect in the same process. Phase 5 shows you what happens when the process restarts. The system must prove that it can stop, come back, and resume from the right place without losing state or duplicating work.

By the end of this chapter, you should understand why snapshot cursor and timeline state must survive restart together, how the SQL hydration store can run either durably on disk or ephemerally in memory, and what restart correctness means for reconnect.

---

## 1. Why restart changes everything

A system can be convincing while it stays warm. It can stream, project, and reconnect in the same process. But it carries a hidden fragility: if the process dies, everything is lost.

Phase 5 replaces fragility with durability. Once the hydration store becomes SQL-backed, the framework can survive restart and resume correctly.

---

## 2. The central rule

> Snapshot cursor state and timeline state must survive restart together, or the framework cannot promise reconnect correctness.

This means:

- If timeline entities survive but the snapshot cursor does not, reconnects may receive an old `snapshotOrdinal` for newer entity state.
- If the snapshot cursor survives but entities do not, the framework believes it is further along than state suggests.
- If they survive independently without transactional discipline, the restart story becomes ambiguous.

Phase 5 is not just "store stuff in SQL." It is about making `Apply(...)`, `Snapshot(...)`, `View(...)`, and `Cursor(...)` mean the same thing durably that they meant in memory.

---

## 3. What the hydration store promises

The hydration store interface does not change when you switch from memory to SQL. What changes is what survives restart.

**`Apply(sessionId, ordinal, entities)`** must advance state and the snapshot cursor atomically. A crash in the middle must not leave them out of sync. The applied ordinal becomes the snapshot cursor when timeline materialization succeeds.

**`Snapshot(sessionId, asOf)`** must return a coherent view of current or historical state. This is what a reconnecting client receives as a protobuf `SnapshotFrame` with `snapshotOrdinal`.

**`View(sessionId)`** must give projections a read-only view of current state for computing the next result.

**`Cursor(sessionId)`** must return the latest applied timeline snapshot ordinal. It is the reconnect snapshot cursor and the timeline projection progress point after restart.

---

## 4. Why transactional Apply matters

Consider what happens if `Apply` is not atomic:

```go
// Wrong: two separate operations
store.ApplyEntities(sessionId, ordinal, entities)
store.AdvanceSnapshotOrdinal(sessionId, ordinal)
```

A crash happens between these two calls. Now:
- The snapshot cursor has not advanced.
- Entities have been applied.
- On restart, the consumer resumes from the old snapshot cursor.
- It may reapply entities, or skip ahead past them.

```go
// Right: one atomic operation
store.Apply(sessionId, ordinal, entities)  // atomic
```

The SQL implementation must use transactions to guarantee this.

---

## 5. What restart looks like

Here is what happens when the backend restarts:

```text
Backend starts
         ↓
Consumer resumes
         ↓
Consumer calls Cursor(sessionId) or ProjectionCursor("timeline", sessionId)
         ↓
SQL store returns: snapshot/timeline ordinal 7
         ↓
Consumer asks bus for events after ordinal 7
         ↓
Events are replayed
         ↓
Ordinals and state catch up
```

The consumer resumes from where it left off. No duplicate work. No skipped state.

---

## 6. What the snapshot cursor does

The snapshot cursor looks like a small field compared to timeline entities. But it is the framework's memory of where materialized timeline state stands.

After restart, the consumer and transport need to know:

- What event ordinal has already been applied to the timeline?
- What `snapshotOrdinal` should be attached to the next hydration snapshot?
- Where should timeline replay or projection repair continue?
- Is the system resuming cleanly?

Without a durable snapshot cursor, restart is guessing. With it, restart is exact.

---

## 7. Memory vs SQL: semantic equivalence

The Phase 5 page lets you compare memory and SQL modes side by side.

The question is not whether SQL stores data. The question is whether SQL preserves the same semantics as memory:

| Store behavior | Memory | SQL |
|----------------|--------|-----|
| `Apply` atomic | ✓ | ✓ |
| `Cursor` / `snapshotOrdinal` after `Apply` | correct | correct |
| `Snapshot` reflects state | yes | yes |
| Survives restart | no | yes |

The framework contract must hold in both implementations. If the implementations disagree, the framework's semantics are no longer stable.

---

## 8. Things to try

**Seed in memory mode.** See the snapshot cursor and snapshot. This is your baseline.

**Seed in SQL mode.** See the same snapshot cursor and snapshot. Notice: the semantics look identical. Only the implementation changed.

**Restart the backend (SQL mode).** After restart, the store still reports the same session state and snapshot cursor. The system still believes the same things about this session.

**Reconnect a client after restart.** The client hydrates from current state and continues without duplicate or skipped ordinals. This is where persistence and transport connect.

**Compare memory and SQL.** If they diverge, the framework's semantics are no longer implementation-stable.

---

## 9. How restart connects to reconnect

Phase 3 showed reconnect within a running process. Phase 5 shows reconnect across process restarts.

In both cases, the client needs:

1. A snapshot of current state, labeled by `snapshotOrdinal`.
2. Future live UI events, each labeled by `eventOrdinal`.

The difference is what "current state" means:

- **Within a process**: current state is in memory.
- **After restart**: current state is in SQL.

The SQL hydration store is what makes restart behave the same as in-process reconnect.

---

## 10. Replay inspection: event cursor vs timeline cursor

Phase 5 also exposes a read-only replay inspection panel. It shows two cursors for the selected session:

- **Event cursor** — the highest backend event ordinal stored in the event log.
- **Timeline cursor / snapshot cursor** — the highest backend event ordinal successfully materialized by the timeline projection and exposed as `Snapshot.snapshotOrdinal`.

When everything is healthy, these cursors usually match:

```text
event cursor:    4
timeline cursor: 4
```

If timeline projection fails, the event can still be durably stored while the timeline cursor stays behind:

```text
event cursor:    7
timeline cursor: 4
```

That state means: "events 5 through 7 exist, but the timeline projection has not successfully applied them yet." The replay panel also lists persisted error records so you can connect cursor gaps to concrete projection, decode, fanout, or storage errors.

The panel is deliberately read-only. Systemlab uses it to teach replay state; it does not expose retry or rebuild controls.

The SQL store also persists `created_ordinal` and `last_event_ordinal` for every current entity and versioned entity row. That is separate from the session snapshot cursor: it lets hydrated snapshots preserve logical timeline order and update position across restarts.

---

## 11. Common mistakes

**Apply is not atomic.** State and snapshot cursor get out of sync on crash.

**Testing restart manually only.** Restart correctness should be in repeatable tests, not just human observation.

**Memory is truth, SQL approximates it.** This used to be a risk when the project had a separate map-backed memory store. The current local mode uses in-memory SQLite so local and disk-backed SQL share one implementation.

**Underestimating cursor correctness.** Snapshot cursor and projection cursor handling is where restart bugs quietly originate.

**Reconnect and persistence are separate stories.** In this framework, they are the same story told in different contexts.

---

## Key Points

- Snapshot cursor state and timeline state must survive restart together. Neither is sufficient alone.
- `Apply` must be atomic. A crash must not leave state and snapshot cursor out of sync.
- After restart, the consumer resumes from the cursor/projection cursor. No duplicate work. No skipped state.
- The disk-backed SQL store must preserve the same semantics as the in-memory SQLite mode. Implementation changes are allowed; semantic drift is not.
- Persistence and reconnect are the same story in different contexts. SQL makes restart behave like in-process reconnect.
- Event cursor and timeline/snapshot cursor are intentionally separate so failed projections can be detected and repaired instead of hidden.

---

## API Reference

### HydrationStore methods

- **`Apply(sessionId, ordinal, entities)`**: Atomically advance state and snapshot cursor.
- **`Snapshot(sessionId, asOf)`**: Return current or historical state for a session with `snapshotOrdinal`.
- **`View(sessionId)`**: Return read-only view for projections.
- **`Cursor(sessionId)`**: Return latest applied timeline snapshot ordinal for a session.
- **`EventCursor(sessionId)`**: Return latest stored backend event ordinal when the store supports replay.
- **`ProjectionCursor("timeline", sessionId)`**: Return latest timeline projection ordinal when the store supports projection cursors.

### Implementation notes

- SQL store uses transactions for atomic `Apply`.
- In-memory mode uses SQLite's named in-memory database support.
- Current entities and entity versions store `created_ordinal` and `last_event_ordinal` for stable snapshot ordering.
- The event log, entity versions, projection cursors, and error records all live in the SQLite backend.

---

## File References

### Framework files

- `pkg/sessionstream/hydration.go` — store interface
- `pkg/sessionstream/hydration/sqlite/store.go` — SQLite implementation, including in-memory SQLite mode
- `pkg/sessionstream/projection.go` — timeline entity ordinal fields
- `pkg/sessionstream/transport/ws/server.go` — protobuf snapshot transport
- `proto/sessionstream/v1/transport.proto` — `SnapshotFrame` and `SnapshotEntity` schema

### Systemlab files

- `cmd/sessionstream-systemlab/static/partials/phase5.html`
- `cmd/sessionstream-systemlab/static/js/pages/phase5.js`
