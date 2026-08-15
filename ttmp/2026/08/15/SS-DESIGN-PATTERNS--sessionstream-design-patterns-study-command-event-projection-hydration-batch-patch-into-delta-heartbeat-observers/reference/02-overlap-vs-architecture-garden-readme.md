---
Title: Overlap, expansion, and replacement vs. the Architecture Garden README
Ticket: SS-DESIGN-PATTERNS
Status: draft
Topics:
    - architecture
    - framework
    - event-streaming
    - sessionstream
    - onboarding
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://examples/chatdemo/chat.go
      Note: Source of the batch-patch-into-delta pattern absent from the Garden README (Pattern 2)
    - Path: repo://pkg/sessionstream/transport/ws/internal/heartbeat/machine.go
      Note: Post-anchor heartbeat kernel the Garden README predates (Pattern 6 vs Garden design 03)
    - Path: repo://pkg/sessionstream/transport/ws/observer.go
      Note: Post-anchor hardened observer dispatcher the Garden README predates (Pattern 9 vs Garden design 01)
ExternalSources: []
Summary: 'Maps the 10 patterns in the SS-DESIGN-PATTERNS study to the existing Architecture Garden sessionstream README, identifies overlap, what each side expands, and confirms the study does not replace the Garden README but complements it (intern/code-grounded vs. mathematical/research). Notes the commit-anchor gap: the Garden README predates the heartbeat kernel and observer-trace hardening.'
LastUpdated: 2026-08-15T00:00:00-04:00
WhatFor: Reconcile the intern patterns study with the Garden's mathematical/research study so the two cross-reference instead of diverge.
WhenToUse: Read when deciding which study to cite, or before updating the Garden README to reflect post-anchor hardening.
---


# Overlap, expansion, and replacement vs. the Architecture Garden README

This note compares `design-doc/01-sessionstream-design-patterns-intern-study.md`
(the **Study**, 10 intern-facing patterns) with
`Research/Software Architecture Garden/sessionstream/README.md` (the
**Garden README**, the mathematical/research study). The headline:

> **The Study does not replace the Garden README.** The two are complementary.
> The Garden README is the *mathematical/research authority*: it states the
> laws, the open correctness obligations, the Pattern-Zoo correlations, the
> cross-project comparison, and the formal-verification program. The Study is
> the *engineering/onboarding companion*: it expresses the same ideas as
> concrete, code-anchored patterns an intern can navigate, and adds the
> **batch-patch-into-delta** pattern the Garden README does not cover.

## Snapshot-identity gap (important)

| | Garden README | Study |
|---|---|---|
| Anchor commit | `fb6b70d` (2026-06-16) | current HEAD `d62dca9` / `004652e` (2026-08-15) |
| Post-anchor hardening reflected? | No | Yes |

The Garden README predates 12 observer/heartbeat commits, including
`d0693bf feat(ws): add heartbeat failure detector kernel`,
`dbfbf02 refactor(ws): supervise heartbeat state machine`, and the
observer-trace lifecycle fixes (`957c906`, `229a47e`, `4754f78`, `ed50601`).
Concretely:

- The Garden's design entry 03 (*Effect-Acknowledged State Machines*) says
  chat startup *lacks* the pure-reducer + serialized-supervisor that heartbeat
  "currently has." At the README's anchor the heartbeat kernel did not yet
  exist as a pure machine; the Study's **Pattern 6** describes the *now-built*
  pure `Machine.Step` + supervisor split. So the Study is newer evidence the
  Garden README should fold in.
- The Garden's design entry 01 (*Bounded Asynchronous Observer Dispatcher*)
  says the generic mechanism "remains unextracted because there is still only
  one retained delivery use." The Study's **Pattern 9** reflects the
  post-anchor hardened observer/trace dispatcher (drop-reporting, in-order,
  synctest-tested). Again, newer evidence.

## Pattern-by-pattern mapping

| Study pattern | Garden README counterpart | Relationship |
|---|---|---|
| 1. Command → Event → Projection split | "Command → canonical event → projection separation" (maturity row); candidate patterns #1 *Typed Intent, Host-Owned Effect* + #2 *Canonical Event, Multiple Projections*; §2 stateful event algebras (`δ_T`, `δ_U`) | **Overlap.** Study = code-grounded intern version; Garden = mathematical framing + the vocabulary discipline (command ≠ event ≠ entity ≠ snapshot). |
| 2. **Batch-patch-into-delta (preview vs authoritative commit)** | §7 "Streaming work as a labeled transition system" (`γ: W → E×W + O`) — coalgebraic streaming only; **no delta-event shape, no preview/commit discipline** | **Expansion (Study adds).** The Study is the only place the delta-carrying-accumulated-state event and the live-only-preview / authoritative-commit discipline are described. The closest Garden concept is the abstract LTS; the concrete pattern is new. |
| 3. Ordinals as universal ordering key | "Sequence coordinate"/"Ordinal", "Prefix cut"/"SnapshotOrdinal", "Projection checkpoint" vocabulary; §4 prefix order | **Overlap + mild expansion.** Study adds the two assignment paths (local counter vs bus `DeriveOrdinalFromStreamID`) and the browser-`uint64`-as-string sharp edge the README only alludes to. |
| 4. Hydration: append-log + materialized view + snapshot | Candidate pattern #5 *Durable Evidence plus Rebuildable Projection*; §5 temporal materialization (`entityAt`); maturity rows "Per-session event append/replay", "Timeline rebuild", "Consistent SQLite snapshot cut" (open) | **Overlap, complementary.** Garden states the *laws and open obligations* (consistent-cut, atomic projection progress, stable retry identity, deterministic replay); Study describes the *actual SQLite schema and `Apply` upsert mechanics*. Laws vs mechanics — neither alone is enough. |
| 5. Snapshot-before-live transport | Candidate pattern #4 *Snapshot Cut plus Live Suffix*; §4 prefix-cut math; WS adapter description | **Overlap.** Study frames it as the transport contract with the ordinal join; Garden frames it as prefix-cut protocol + buffering/discard mechanics. |
| 6. Pure state machine + supervisor (timed failure detector) | Design entry 03 *Effect-Acknowledged State Machines*; §7 LTS; verification research 02 (*Constraining the Go Binary*) | **Overlap, Garden expands.** Study is the intern intro; Garden has the formal refinement model (commit-before-concurrency, linearization points, trace inclusion, synctest, state-aware fuzzing) and the Coq/Lean/TLA+ program. Study does not replace; it's a teaser for design entry 03. |
| 7. Small interfaces + functional adapters (`UIProjectionFunc`, `HubOption`) | — (not present) | **Expansion (Study adds).** A Go-idiom angle the Garden README, being mathematical, doesn't cover. |
| 8. Capabilities/behavior as data, not branches | "Command as Data" (PBUI 5), "Registry and Module Boundary" (PBUI 9), candidate #6 *Small Schema Admission Kernel*; §6 typed sums | **Overlap.** Study shows concrete registry mechanics + typed decode; Garden frames it as the admission boundary with the typed-sum math. |
| 9. Observers + trace as best-effort side channel | Design entries 01 *Bounded Asynchronous Observer Dispatcher* + 04 *Observer as Diagnostic Projection*; verification research 01 (*Proving the Dispatcher* — TLA+/Alloy/Coq/Lean) | **Overlap, Garden expands massively.** Study is a one-paragraph summary; Garden has the bounded/lossy delivery laws (FIFO, drop, panic, close, drain, wait) and formal proofs. But the Study reflects *post-anchor hardened state* (drop-reporting, synctest tests) the README predates. |
| 10. Schema-vet as build-time policy | Candidate #6 *Small Schema Admission Kernel*; §6 typed sums; RAG Pattern 10 (small trusted validators) | **Overlap.** Study shows it is a Go analyzer (compile-time policy); Garden frames it as the admission boundary and explicitly flags the open gap (transport uses `Any`, names are strings, `Session.Metadata` is `any` — the contract is not statically closed end-to-end). |

## What each side adds (neither replaces the other)

### The Study adds (not in the Garden README)
- **The batch-patch-into-delta pattern** (the user's specific ask): the delta
  event carrying both `chunk` and accumulated `text`, the timeline projection's
  *whole-state batch upsert* each tick, and the preview-vs-authoritative-commit
  discipline. This is the single biggest gap the Study fills.
- **Concrete SQLite mechanics**: the `sessionstream_entities` /
  `_entity_versions` / `_events` / `_projection_cursors` schema, the idempotent
  `Apply` upsert, the `Snapshot(asOf)` point-in-time query. The Garden gives
  laws; the Study gives the implementation.
- **The two ordinal assignment paths** (local counter vs bus stream-ID
  derivation) and the browser `uint64`-as-string rule.
- **Go-idiom patterns** (small interfaces + functional adapters) — an
  engineering angle the mathematical README doesn't take.
- **Newer commit state**: reflects the heartbeat kernel + supervisor and the
  hardened observer/trace dispatcher, all post-dating the README's anchor.

### The Garden README adds (not in the Study — the Study does NOT replace it)
- **Mathematical foundations**: free monoids + fold associativity, stateful
  event algebras (`δ_T`, `δ_U`), product decomposition / noninterference,
  prefix-order cuts, temporal materialization (`entityAt`), typed sums at
  trust boundaries, coalgebraic streaming.
- **The "Laws that should guide hardening"** — the Study's most important
  *omission*: per-session serializability, consistent-cut snapshots, stable
  retry identity, atomic projection progress, deterministic replay,
  snapshot-plus-suffix completeness. These are open correctness obligations the
  Study does not flag.
- **Maturity assessment** (candidate / established-locally / open-obligation /
  intentionally-external) per pattern.
- **Cross-project comparison** (devctl, upwork, publish-vault, rag-ttc,
  rag-evaluation, go-go-datadrop, zitadel).
- **Pattern Zoo correlations** (RAG-MATHS, PBUI-MATHS) with strength/boundary.
- **Candidate ecosystem vocabulary** (scope key, intent value, canonical
  event, projection, materialized entity, sequence coordinate, prefix cut,
  projection checkpoint, admission registry, live suffix).
- **Four design entries + two verification research entries** (formal proofs
  in TLA+/Alloy/Coq/Lean; the Go-binary refinement program).
- **Implications for elegant JavaScript APIs** (branded typed values,
  `AsyncIterable` history, pure-by-default projections, lawful combinators).
- **The "contract not statically closed" caveat** (`Any`, string names, `any`
  metadata) requiring descriptor distribution and a schema-evolution policy.

## Recommendation

1. **Cross-link the two.** The Study should cite the Garden README as the
   mathematical/law authority for shared patterns; the Garden README should
   cite the Study as the code-anchored companion and as the source for the
   batch-patch-into-delta pattern.
2. **Refresh the Garden README's anchor** (or add a postscript) to reflect the
   heartbeat kernel + supervisor and the hardened observer/trace dispatcher
   the Study describes — the README currently predates that work.
3. **Port the Study's Pattern 2 (batch-patch-into-delta) into the Garden** as a
   new design entry or a section under §7 (streaming LTS), since the Garden's
   streaming model is currently abstract and the delta+accumulated-state+
   preview/commit pattern is concrete and reusable.
4. **Keep the Garden's "Laws that should guide hardening" as the authority.**
   The Study is deliberately silent on them; interns should be sent from the
   Study's Pattern 4 to the Garden's "Laws" section for the open obligations.
