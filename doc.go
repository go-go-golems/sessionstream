// Package sessionstream provides a reusable substrate for session-based streaming applications.
//
// The framework is intended to own generic concepts such as:
//   - session identity,
//   - typed commands and backend events,
//   - UI and timeline projections,
//   - hydration stores,
//   - transport adapters.
//
// Product-specific runtime policy, middleware behavior, and HTTP edge compatibility
// belong in downstream consumer applications rather than in this repository.
package sessionstream
