package sessionstream

import "context"

// UIFanout is the consumer-side output seam used to publish projected UI events
// to fanout-only transports such as the websocket snapshot/fanout adapter.
type UIFanout interface {
	PublishUI(ctx context.Context, sid SessionId, ord uint64, events []UIEvent) error
}

// UIFanoutFunc adapts a function to UIFanout.
type UIFanoutFunc func(ctx context.Context, sid SessionId, ord uint64, events []UIEvent) error

func (f UIFanoutFunc) PublishUI(ctx context.Context, sid SessionId, ord uint64, events []UIEvent) error {
	return f(ctx, sid, ord, events)
}
