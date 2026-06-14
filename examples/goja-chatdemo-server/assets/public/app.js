const messages = new Map();
const el = document.getElementById("messages");
const statusEl = document.getElementById("status");
const promptEl = document.getElementById("prompt");

function render() {
  el.innerHTML = "";
  for (const msg of messages.values()) {
    const div = document.createElement("div");
    div.className = "msg " + (msg.role === "user" ? "user" : "assistant");
    const meta = document.createElement("div");
    meta.className = "meta";
    meta.textContent = msg.role + " · " + msg.id + (msg.status ? " · " + msg.status : "") + (msg.streaming ? " · streaming" : "");
    div.appendChild(meta);
    div.appendChild(document.createTextNode(msg.content || msg.text || ""));
    el.appendChild(div);
  }
  el.scrollTop = el.scrollHeight;
}

function payloadOf(frame) {
  const p = frame.payload || {};
  return p.value || p;
}

function upsertFromPayload(p) {
  if (!p || !p.messageId) return;
  messages.set(p.messageId, {
    id: p.messageId,
    role: p.role || "assistant",
    content: p.content || p.text || "",
    text: p.text || p.content || "",
    streaming: Boolean(p.streaming),
    status: p.status || "",
  });
  render();
}

async function loadConfig() {
  const response = await fetch("/api/config");
  if (!response.ok) throw new Error("config request failed: " + response.status);
  return response.json();
}

function resolveSessionId(config) {
  const fromURL = new URLSearchParams(location.search).get("sessionId");
  if (fromURL) return fromURL;
  return config.defaultSessionId || "demo";
}

function connectWebSocket(sessionId) {
  const ws = new WebSocket("ws://" + location.host + "/ws");
  ws.onopen = () => {
    statusEl.textContent = "websocket connected · session " + sessionId;
    ws.send(JSON.stringify({ subscribe: { sessionId, sinceSnapshotOrdinal: "0" } }));
  };
  ws.onmessage = (ev) => {
    const frame = JSON.parse(ev.data);
    if (frame.hello) statusEl.textContent = "connected as " + frame.hello.connectionId + " · session " + sessionId;
    if (frame.snapshot) {
      for (const ent of frame.snapshot.entities || []) upsertFromPayload(payloadOf(ent));
    }
    if (frame.uiEvent) upsertFromPayload(payloadOf(frame.uiEvent));
    if (frame.error) console.error(frame.error);
  };
  ws.onclose = () => { statusEl.textContent = "websocket closed"; };
  return ws;
}

async function main() {
  const config = await loadConfig();
  const sessionId = resolveSessionId(config);
  connectWebSocket(sessionId);

  document.getElementById("form").addEventListener("submit", async (e) => {
    e.preventDefault();
    const prompt = promptEl.value;
    const response = await fetch("/api/chat", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ sessionId, prompt }),
    });
    if (!response.ok) {
      statusEl.textContent = "chat request failed: " + response.status;
    }
  });
}

main().catch((err) => {
  console.error(err);
  statusEl.textContent = String(err && err.message ? err.message : err);
});
