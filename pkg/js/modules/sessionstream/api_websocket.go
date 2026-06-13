package sessionstream

import (
	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	ws "github.com/go-go-golems/sessionstream/pkg/sessionstream/transport/ws"
)

func (m *moduleRuntime) webSocketServerBuilder(call goja.FunctionCall) goja.Value {
	hub, ok := m.hubRef(call.Argument(0))
	if !ok {
		panic(m.vm.NewTypeError("webSocket.server expects a sessionstream Hub"))
	}
	server, err := ws.NewServer(hub.hub)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	obj := m.vm.NewObject()
	m.attachRef(obj, &websocketRef{server: server})
	if err := gojahttp.AttachHTTPHandler(m.vm, obj, server); err != nil {
		panic(m.vm.NewGoError(err))
	}
	m.mustSet(obj, "connections", func() any { return server.Connections() })
	return obj
}
