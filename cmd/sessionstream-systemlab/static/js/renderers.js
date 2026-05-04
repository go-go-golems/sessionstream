import { setJSON } from "./dom.js";

export function renderError(el, error) {
  el.className = "";
  setJSON(el, { error: error.message || String(error) });
}

export function renderTrace(el, trace, empty = "No trace yet.") {
  el.className = "trace-rendered";
  if (!Array.isArray(trace) || trace.length === 0) {
    el.innerHTML = `<span class="empty">${escapeHTML(empty)}</span>`;
    return;
  }
  el.innerHTML = trace.map((step) => `
    <div class="trace-step">
      <span class="trace-step-num">${escapeHTML(step.step)}</span>
      <span class="trace-step-kind ${escapeAttr(step.kind || "unknown")}">${escapeHTML(step.kind || "unknown")}</span>
      <span class="trace-step-message">
        ${escapeHTML(step.message || "")}
        ${step.details ? `<span class="trace-step-detail">${compactDetails(step.details)}</span>` : ""}
      </span>
    </div>
  `).join("");
}

export function renderClientFrames(el, value) {
  el.className = "frame-rendered";
  const frames = value?.frames || [];
  const status = value?.status || value?.error || "idle";
  el.innerHTML = `
    <div class="session-header">
      <div class="session-header-label">Status</div>
      <div class="session-header-value">${escapeHTML(status)}</div>
    </div>
    ${frames.length ? frames.map(renderFrame).join("") : '<span class="empty">No frames yet.</span>'}
  `;
}

export function renderConnectionSnapshot(el, value) {
  el.className = "snapshot-rendered rendered-state";
  const connections = value?.connections || [];
  const snapshot = value?.snapshot || {};
  el.innerHTML = `
    ${renderConnections(connections)}
    ${renderSnapshotBlock("Snapshot", snapshot)}
  `;
}

export function renderRestartState(el, value) {
  el.className = "snapshot-rendered rendered-state";
  el.innerHTML = `
    ${renderConnections(value?.connections || [])}
    <div class="state-compare-grid">
      ${renderSnapshotBlock("Pre Restart", value?.preRestart || {})}
      ${renderSnapshotBlock("Post Restart", value?.postRestart || {})}
    </div>
  `;
}

export function renderReplayState(el, value) {
  el.className = "snapshot-rendered rendered-state";
  const errors = value?.errors || [];
  el.innerHTML = `
    <div class="metric-row">
      <span class="metric-chip">event cursor <strong>${escapeHTML(value?.eventCursor ?? 0)}</strong></span>
      <span class="metric-chip">timeline cursor <strong>${escapeHTML(value?.timelineCursor ?? 0)}</strong></span>
      <span class="metric-chip">errors <strong>${escapeHTML(errors.length)}</strong></span>
    </div>
    ${errors.length ? renderErrorTable(errors) : '<div class="empty compact-empty">No persisted errors.</div>'}
  `;
}

function renderConnections(connections) {
  if (!Array.isArray(connections) || connections.length === 0) {
    return '<div class="empty compact-empty">No active connections.</div>';
  }
  return `
    <table class="data-table compact-table connection-table">
      <thead><tr><th>Connection</th><th>Subscriptions</th></tr></thead>
      <tbody>
        ${connections.map((conn) => `
          <tr>
            <td>${escapeHTML(conn.connectionId)}</td>
            <td>${(conn.subscriptions || []).map((sid) => `<span class="ordinal-chip">${escapeHTML(sid)}</span>`).join("") || "—"}</td>
          </tr>
        `).join("")}
      </tbody>
    </table>
  `;
}

function renderSnapshotBlock(title, snapshot) {
  const entities = snapshot?.entities || [];
  return `
    <section class="snapshot-card compact-snapshot-card">
      <header class="snapshot-card-header">
        <span class="snapshot-card-title">${escapeHTML(title)}</span>
        <span class="ordinal-chip">session ${escapeHTML(snapshot?.sessionId || "unknown")}</span>
        <span class="ordinal-chip">snapshot ${escapeHTML(snapshot?.snapshotOrdinal ?? snapshot?.ordinal ?? 0)}</span>
        <span class="ordinal-chip">${entities.length} entities</span>
      </header>
      ${renderEntities(entities)}
    </section>
  `;
}

function renderEntities(entities) {
  if (!Array.isArray(entities) || entities.length === 0) return '<div class="empty compact-empty">No entities.</div>';
  return `
    <table class="data-table compact-table snapshot-table">
      <thead><tr><th>Kind</th><th>ID</th><th>Ordinals</th><th>Payload</th></tr></thead>
      <tbody>
        ${entities.map((entity) => `
          <tr>
            <td>${escapeHTML(entity.kind)}</td>
            <td>${escapeHTML(entity.id)}</td>
            <td><code>created: ${escapeHTML(entity.createdOrdinal ?? "")} · last: ${escapeHTML(entity.lastEventOrdinal ?? "")}</code></td>
            <td><code>${escapeHTML(inlineObject(entity.payload))}</code></td>
          </tr>
        `).join("")}
      </tbody>
    </table>
  `;
}

function renderFrame(frame, index) {
  const normalized = normalizeFrame(frame);
  return `
    <div class="frame-card">
      <span class="ordinal-chip">${index + 1}</span>
      <span class="frame-type">${escapeHTML(normalized.type || "frame")}</span>
      ${normalized.sessionId ? `<span class="frame-detail">session ${escapeHTML(normalized.sessionId)}</span>` : ""}
      ${normalized.snapshotOrdinal ? `<span class="frame-detail">snapshot ${escapeHTML(normalized.snapshotOrdinal)}</span>` : ""}
      ${normalized.eventOrdinal ? `<span class="frame-detail">event ${escapeHTML(normalized.eventOrdinal)}</span>` : ""}
      ${normalized.sinceSnapshotOrdinal ? `<span class="frame-detail">since ${escapeHTML(normalized.sinceSnapshotOrdinal)}</span>` : ""}
      ${normalized.name ? `<span class="frame-detail">${escapeHTML(normalized.name)}</span>` : ""}
      ${normalized.payload ? `<code>${escapeHTML(inlineObject(normalized.payload))}</code>` : ""}
      ${normalized.entities ? `<code>${escapeHTML(`${normalized.entities.length} entities`)}</code>` : ""}
    </div>
  `;
}

function normalizeFrame(frame) {
  if (!frame || typeof frame !== "object") return { type: "frame" };
  const oneof = ["hello", "snapshot", "subscribed", "unsubscribed", "uiEvent", "error", "ping", "pong"].find((key) => frame[key]);
  if (!oneof) return frame;
  const value = frame[oneof] || {};
  return { type: oneof, ...value };
}

function renderErrorTable(errors) {
  return `
    <table class="data-table compact-table">
      <thead><tr><th>Kind</th><th>Session</th><th>Ordinal</th><th>Event</th><th>Error</th></tr></thead>
      <tbody>
        ${errors.map((err) => `
          <tr><td>${escapeHTML(err.kind)}</td><td>${escapeHTML(err.sessionId)}</td><td class="num">${escapeHTML(err.ordinal)}</td><td>${escapeHTML(err.eventName)}</td><td>${escapeHTML(err.error)}</td></tr>
        `).join("")}
      </tbody>
    </table>
  `;
}

function compactDetails(details) {
  const preferred = ["connectionId", "sessionId", "snapshotOrdinal", "eventOrdinal", "createdOrdinal", "lastEventOrdinal", "ordinal", "uiEvent", "entityId", "sourceEvent", "action", "sinceSnapshotOrdinal", "sinceOrdinal", "text", "status"];
  const parts = [];
  for (const key of preferred) if (details[key] !== undefined && details[key] !== "") parts.push(`${key}: ${formatScalar(details[key])}`);
  for (const [key, value] of Object.entries(details || {})) {
    if (preferred.includes(key) || value === undefined || value === "") continue;
    if (parts.length >= 5) break;
    parts.push(`${key}: ${formatScalar(value)}`);
  }
  return parts.map(escapeHTML).join(" · ");
}

function inlineObject(value) {
  if (value === undefined || value === null) return "";
  if (typeof value !== "object") return String(value);
  return Object.entries(value).map(([key, val]) => `${key}: ${formatScalar(val)}`).join(" · ");
}

function formatScalar(value) {
  if (value === null || value === undefined) return "";
  if (typeof value === "object") return inlineObject(value);
  return String(value);
}

function escapeHTML(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function escapeAttr(value) {
  return escapeHTML(value).replaceAll(" ", "-");
}
