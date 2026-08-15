# Changelog

## 2026-08-15

- Initial workspace created


## 2026-08-15

Wrote the sessionstream design patterns study (design-doc/01-...): 10 recurring patterns (command/event/projection split; batch-patch-into-delta with preview-vs-authoritative-commit; ordinals; hydration append-log+materialized-view+snapshot; snapshot-before-live transport; pure heartbeat state machine + supervisor; small interfaces + functional adapters; capabilities as data; best-effort observers + trace; schema-vet policy), with prose, bullets, pseudocode, ASCII diagrams, API references, and file references. Related to 17 code/design files.

### Related Files

- ttmp/2026/08/15/SS-DESIGN-PATTERNS--sessionstream-design-patterns-study-command-event-projection-hydration-batch-patch-into-delta-heartbeat-observers/design-doc/01-sessionstream-design-patterns-intern-study.md — The design patterns study


## 2026-08-15

Step 1: diary created; design patterns study committed and related.

### Related Files

- ttmp/2026/08/15/SS-DESIGN-PATTERNS--sessionstream-design-patterns-study-command-event-projection-hydration-batch-patch-into-delta-heartbeat-observers/reference/01-diary.md — Investigation diary


## 2026-08-15

Added reference/02: overlap/expansion/replacement analysis vs. the Architecture Garden sessionstream README. Conclusion: the Study complements (does not replace) the Garden README — intern/code-grounded vs mathematical/research authority. Study adds the batch-patch-into-delta pattern, concrete SQLite mechanics, and post-anchor heartbeat/observer hardening evidence; Garden README adds the math, the hardening laws, the cross-project comparison, and the formal-verification program. Noted the README's commit anchor (fb6b70d, 2026-06-16) predates 12 heartbeat/observer commits.

### Related Files

- ttmp/2026/08/15/SS-DESIGN-PATTERNS--sessionstream-design-patterns-study-command-event-projection-hydration-batch-patch-into-delta-heartbeat-observers/reference/02-overlap-vs-architecture-garden-readme.md — Comparison with the Architecture Garden README


## 2026-08-15

Step 2: Built the Architecture Garden 'Index of Design Patterns' for sessionstream (index + rationale), following the coinvault playbook. 48 ### entries, 4 See redirects, 18-row notation table, cross-reference summary, 47-entry rationale with a 20-situation reader test. Includes the batch-patch-into-delta pattern. Study README back-linked. All links validated (PASS). Committed in go-go-parc (896a381).

### Related Files

- Research/Software Architecture Garden/sessionstream/Index of Design Patterns.md — Garden index derived from the patterns study

