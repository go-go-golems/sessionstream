const pb = require("sessionstream.examples.chatdemo.v1");

exports.command = pb.StartInferenceCommand.builder()
  .prompt("Explain ordinals")
  .build();
