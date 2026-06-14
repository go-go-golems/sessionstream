# xgoja sessionstream chatbot server

This example is a real xgoja application:

- JavaScript owns the application setup, HTTP routes, command handler, projections, and fake chatbot behavior.
- Go provides xgoja, the Express-style HTTP host, generated protobuf builders, sessionstream Hub/runtime, hydration, and the Go WebSocket handler.
- The JavaScript app mounts the Go-backed WebSocket server with `app.mount("/ws", ss.webSocket.server(hub))`.

Run the smoke test:

```bash
make -C examples/goja-chatdemo-server smoke
```

Run manually:

```bash
make -C examples/goja-chatdemo-server build
examples/goja-chatdemo-server/dist/goja-chatdemo-server serve chatbot serve --http-listen 127.0.0.1:18789
```

Open:

```text
http://127.0.0.1:18789/
```

The browser connects to `/ws`, subscribes to the `demo` session, sends prompts to `/api/chat`, and receives live UI events over the WebSocket.
