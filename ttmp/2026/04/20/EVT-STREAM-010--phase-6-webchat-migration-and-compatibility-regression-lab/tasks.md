# Tasks

## TODO

### Slice A — Analysis freeze and contract inventory
- [x] Inventory the current `cmd/web-chat` public/runtime behavior that actually matters for migration.
- [x] Write a preserve / change / drop matrix for routes, websocket behavior, hydration behavior, and debug surfaces.
- [x] Define the target `evtstream`-backed `cmd/web-chat` package layout and app/runtime boundaries in enough detail to implement directly.
- [x] Define the new canonical web contract for submit, snapshot, websocket subscribe/live flow, and profile APIs.
- [x] Start a detailed investigation diary for the ticket.
- [ ] Relate the new inventory/matrix docs back to the specific code files they describe.

### Slice B — App-grade evtstream chat package
- [x] Evolve the current chat example into an app-grade `evtstream` chat package suitable for `cmd/web-chat`.
- [x] Decide whether `examples/chat` is replaced by `apps/chat`, wrapped by it, or kept only as a thin transitional shim during the slice.
- [x] Add focused tests for the app-grade package surface.
- [x] Switch Systemlab Phase 4 to consume the app-grade package if that is the cleanest proof of the new package shape.

### Slice C — Canonical backend handlers inside cmd/web-chat
- [x] Introduce app-owned evtstream-backed chat services/handlers inside `cmd/web-chat`.
- [x] Wire canonical submit, snapshot, and websocket handler paths to the new evtstream-backed services.
- [x] Keep the migration seam narrow enough that one end-to-end path can be exercised without a big-bang rewrite.
- [x] Add focused Go tests for the new handler/service layer.

### Slice D — Frontend client-layer cutover
- [x] Update the frontend client layer to the new canonical backend contract.
- [x] Port `ChatWidget.tsx` bootstrap/send flow to the new contract.
- [x] Port `wsManager.ts` to the new websocket subscribe/live model.
- [x] Port snapshot hydration/timeline normalization to the new contract.
- [x] Preserve profile selection and visible status/error behavior through the cutover.
- [x] Run browser validation for real UI flows as slices land, using the `gpt-5-nano-low` inference profile when real inference is required.

### Slice E — Hydration, reconnect, and correctness
- [x] Replace websocket live delivery with the new transport path in the real app flow.
- [x] Replace hydration / restart behavior with the new store and hub path.
- [x] Prove one reload/reconnect scenario end to end against the new stack.
- [x] Capture transcript and/or artifact evidence for the migrated flow.

### Slice F — Sever pkg/webchat from the live path and finalize the playbook
- [x] Remove `pkg/webchat` from the live runtime path for the migrated scenario.
- [x] Capture historical expectation transcripts for the key `cmd/web-chat` flows being replaced.
- [x] Capture new `evtstream`-backed transcripts for the same flows.
- [x] Build Phase 6 Systemlab page as a migration/regression inspection console.
- [x] Tag each difference as preserve / change / drop, intentional / bug / follow-up.
- [x] Document the final cutover recommendation and the removal plan for remaining `pkg/webchat` runtime dependencies.

### Slice G — Real inference backend on the canonical path
- [x] Replace the fake echo-only canonical backend path with runtime-resolved inference wiring.
- [x] Reuse app-edge profile/runtime resolution for the canonical submit path.
- [x] Translate runtime engine events into canonical evtstream chat events.
- [x] Validate a successful real provider-backed canonical run end to end using the normal pinocchio config/profile stack.

### Slice H — User-message projection on the canonical path
- [x] Emit a canonical user-message event when a prompt is accepted.
- [x] Persist user and assistant messages as separate canonical timeline entities.
- [x] Ensure canonical snapshots include the user message as well as the assistant message state.

### Slice I — Frontend role/content alignment for canonical chat entities
- [x] Stop assuming canonical messages are always assistant messages.
- [x] Map canonical `role` and `content` fields from snapshots.
- [x] Map canonical `role` and `content` fields from live websocket UI events.
- [x] Browser-validate that the user message is visible in the migrated UI.

### Slice J — Real browser inference validation on the canonical app path
- [x] Run the migrated canonical browser flow against a real provider-backed profile (`gpt-5-nano-low`) using the normal pinocchio config/profile stack.
- [x] Verify that the browser renders both the user message and the assistant response.
- [x] Verify that the assistant output is real provider output, not the old echo text.

## Exit Criteria

- [x] `cmd/web-chat` runs on top of `evtstream` for at least one end-to-end real scenario.
- [x] The new path preserves the required user-visible behaviors or documents intentional divergences explicitly.
- [x] Systemlab Phase 6 demonstrates the migration with transcript-backed evidence.
- [x] Automated tests and regression fixtures cover the main migration invariants.
- [x] The resulting artifacts form a reusable migration playbook for later application ports.
