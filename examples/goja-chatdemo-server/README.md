# xgoja sessionstream chatbot server

This example is a real xgoja application:

- JavaScript owns the application setup, HTTP routes, command handler, projections, and fake chatbot behavior.
- Go provides xgoja, the Express-style HTTP host, generated protobuf builders, sessionstream Hub/runtime, hydration, and the Go WebSocket handler.
- The JavaScript app mounts the Go-backed WebSocket server with `app.mount("/ws", ss.webSocket.server(hub))`.
- The browser HTML, CSS, and client JavaScript are embedded into the generated xgoja binary as an `assets` source and served from `require("fs:assets")`.

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

The browser connects to `/ws`, subscribes to the `demo` session, sends prompts to `/api/chat`, and receives live UI events over the WebSocket. The fake backend intentionally waits between published events with the xgoja `timer` module so the browser shows the assistant message progressing through streaming states instead of completing immediately.

Pass a custom session id in the URL if you want separate conversations:

```text
http://127.0.0.1:18789/?sessionId=alice
```

The frontend loads `/api/config` for the server default session id and lets the query parameter override it.
