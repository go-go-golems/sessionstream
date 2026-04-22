package transport

import (
	"context"

	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
)

// Transport is the public wire seam used by the substrate.
type Transport interface {
	Start(ctx context.Context, in chan<- IncomingCommand, out <-chan OutgoingMessage) error
}

// IncomingCommand is the transport-neutral payload entering dispatch.
type IncomingCommand struct {
	ConnectionId sessionstream.ConnectionId
	SessionId    sessionstream.SessionId
	Name         string
	PayloadBytes []byte
}

// OutgoingMessage is the transport-neutral payload leaving projection fan-out.
type OutgoingMessage struct {
	SessionId     sessionstream.SessionId
	ConnectionIds []sessionstream.ConnectionId
	UIEvent       sessionstream.UIEvent
}
