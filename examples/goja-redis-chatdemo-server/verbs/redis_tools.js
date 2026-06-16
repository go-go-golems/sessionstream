__package__({ name: "redis", short: "Redis-backed sessionstream CLI tools" })

const ss = require("sessionstream")
const pb = require("sessionstream.examples.chatdemo.v1")

const sessionId = "demo"
let messageSeq = 0

function nextMessageId(prefix) {
  messageSeq += 1
  return `${prefix}-${messageSeq}`
}

function registerSchemas() {
  return ss.schemas()
    .registerCommand("ChatStartInference", pb.StartInferenceCommand)
    .registerEvent("ChatUserMessageAccepted", pb.UserMessageAcceptedEvent)
    .registerEvent("ChatInferenceStarted", pb.InferenceStartedEvent)
    .registerEvent("ChatTokensDelta", pb.TokensDeltaEvent)
    .registerEvent("ChatInferenceTrace", pb.InferenceTraceEvent)
    .registerEvent("ChatInferenceFinished", pb.InferenceFinishedEvent)
}

function fakeAnswer(prompt) {
  return `CLI fake backend answer: you said "${prompt}". This event sequence was produced by a Redis-backed xgoja jsverb.`
}

function newHub() {
  return ss.hub({ schemas: registerSchemas() })
}

function configureCommandHub() {
  const hub = newHub()
  hub.command("ChatStartInference", async (cmd, _session, pub) => {
    const prompt = String(cmd.payload.prompt || "")
    const userID = nextMessageId("cli-user")
    const assistantID = nextMessageId("cli-assistant")
    const answer = fakeAnswer(prompt)

    await pub.publish("ChatUserMessageAccepted", pb.UserMessageAcceptedEvent.builder()
      .messageId(userID).role("user").content(prompt).streaming(false).build())
    await pub.publish("ChatInferenceStarted", pb.InferenceStartedEvent.builder()
      .messageId(assistantID).prompt(prompt).role("assistant").content("").status("streaming").streaming(true).build())
    await pub.publish("ChatInferenceTrace", pb.InferenceTraceEvent.builder()
      .messageId(assistantID).stage("cli-command").detail("ChatStartInference was submitted from the Redis CLI jsverb").elapsedMs(0).build())
    await pub.publish("ChatTokensDelta", pb.TokensDeltaEvent.builder()
      .messageId(assistantID).role("assistant").chunk(answer).text(answer).content(answer).status("streaming").streaming(true).build())
    await pub.publish("ChatInferenceFinished", pb.InferenceFinishedEvent.builder()
      .messageId(assistantID).role("assistant").text(answer).content(answer).status("done").streaming(false).build())
  })
  return hub
}

async function submitPrompt(options) {
  const sid = String(options.sessionId || sessionId)
  const prompt = String(options.prompt || "hello from the Redis CLI")
  const hub = configureCommandHub()
  await hub.submit(sid, "ChatStartInference", pb.StartInferenceCommand.builder().prompt(prompt).build())
  return `submitted ChatStartInference to session ${sid}`
}

__verb__("submitPrompt", {
  name: "submit-prompt",
  output: "text",
  short: "Submit a ChatStartInference command through the Redis-backed CLI host",
  fields: {
    options: { bind: "all" },
    sessionId: { help: "Session id", default: "demo" },
    prompt: { help: "Prompt text", default: "hello from the Redis CLI" }
  }
})

async function publishUserMessage(options) {
  const sid = String(options.sessionId || sessionId)
  const content = String(options.content || "injected user event from the Redis CLI")
  const messageID = String(options.messageId || nextMessageId("cli-user"))
  const hub = newHub()
  await hub.publish(sid, "ChatUserMessageAccepted", pb.UserMessageAcceptedEvent.builder()
    .messageId(messageID).role("user").content(content).streaming(false).build())
  return `published ChatUserMessageAccepted ${messageID} to session ${sid}`
}

__verb__("publishUserMessage", {
  name: "publish-user-message",
  output: "text",
  short: "Inject a typed user-message event through the Redis-backed CLI host",
  fields: {
    options: { bind: "all" },
    sessionId: { help: "Session id", default: "demo" },
    messageId: { help: "Message id; defaults to a generated cli-user id" },
    content: { help: "Message content", default: "injected user event from the Redis CLI" }
  }
})

async function publishTrace(options) {
  const sid = String(options.sessionId || sessionId)
  const messageID = String(options.messageId || "cli-trace")
  const stage = String(options.stage || "cli")
  const detail = String(options.detail || "custom trace event injected from the Redis CLI")
  const elapsedMs = Number(options.elapsedMs || 0)
  const hub = newHub()
  await hub.publish(sid, "ChatInferenceTrace", pb.InferenceTraceEvent.builder()
    .messageId(messageID).stage(stage).detail(detail).elapsedMs(elapsedMs).build())
  return `published ChatInferenceTrace ${stage} for ${messageID} to session ${sid}`
}

__verb__("publishTrace", {
  name: "publish-trace",
  output: "text",
  short: "Inject a custom protobuf trace event through the Redis-backed CLI host",
  fields: {
    options: { bind: "all" },
    sessionId: { help: "Session id", default: "demo" },
    messageId: { help: "Assistant message id", default: "cli-trace" },
    stage: { help: "Trace stage", default: "cli" },
    detail: { help: "Trace detail", default: "custom trace event injected from the Redis CLI" },
    elapsedMs: { type: "int", help: "Elapsed milliseconds", default: 0 }
  }
})
