# Goja chatdemo protobuf builder example

This example is the first sessionstream-specific integration step for the generic
`go-go-goja` protobuf builder generator. It proves JavaScript can import the
chatdemo schema module, build a typed `StartInferenceCommand`, recover the Go
protobuf value with `protogoja.MessageFromValue`, and submit it to the existing
chatdemo `Hub`/`Engine`.

```js
const pb = require("sessionstream.examples.chatdemo.v1");

exports.command = pb.StartInferenceCommand.builder()
  .prompt("Explain ordinals")
  .build();
```

Run the smoke tests from the repository root:

```bash
go test ./examples/goja-chatdemo/... -count=1
```

The `provider` package exposes the generated protobuf module as an xgoja provider
with TypeScript declarations, using package id `sessionstream-chatdemo` and module
name `sessionstream.examples.chatdemo.v1`.
