import { fetchChapterHTML, phase2ExportURL, runPhase2 } from "../api.js";
import { byId, renderChecks, setHTML, setJSON } from "../dom.js";

export async function initPhase2Page() {
  const chapter = byId("phase2-chapter");
  if (chapter) {
    try {
      setHTML(chapter, await fetchChapterHTML("phase-2-ordering-and-ordinals"));
    } catch (error) {
      chapter.textContent = error.message;
    }
  }
  const sessionAInput = byId("phase2-session-a");
  const sessionBInput = byId("phase2-session-b");
  const burstCountInput = byId("phase2-burst-count");
  const streamModeInput = byId("phase2-stream-mode");
  const traceOutput = byId("phase2-trace-output");
  const messagesOutput = byId("phase2-messages-output");
  const ordinalsOutput = byId("phase2-ordinals-output");
  const snapshotsOutput = byId("phase2-snapshots-output");
  const checksOutput = byId("phase2-checks");

  bindAction('[data-action="phase2-publish-a"]', () => submitAction("publish-a"));
  bindAction('[data-action="phase2-publish-b"]', () => submitAction("publish-b"));
  bindAction('[data-action="phase2-burst-a"]', () => submitAction("burst-a"));
  bindAction('[data-action="phase2-restart-consumer"]', () => submitAction("restart-consumer"));
  bindAction('[data-action="phase2-reset"]', () => submitAction("reset-phase2"));
  bindAction('[data-action="phase2-export-json"]', () => window.open(phase2ExportURL("json"), "_blank"));
  bindAction('[data-action="phase2-export-markdown"]', () => window.open(phase2ExportURL("markdown"), "_blank"));

  async function submitAction(action) {
    try {
      const data = await runPhase2({
        action,
        sessionA: sessionAInput?.value,
        sessionB: sessionBInput?.value,
        burstCount: Number.parseInt(burstCountInput?.value || "4", 10),
        streamMode: streamModeInput?.value,
      });
      renderPhase2Trace(traceOutput, data.trace || []);
      renderPhase2Messages(messagesOutput, data.messageHistory || []);
      renderPhase2Ordinals(ordinalsOutput, data.perSessionOrdinals || {});
      renderPhase2Snapshots(snapshotsOutput, data.snapshots || {});
      renderChecks(checksOutput, data.checks);
    } catch (error) {
      const value = { error: error.message };
      renderPhase2Error(traceOutput, value);
      renderPhase2Error(messagesOutput, value);
      renderPhase2Error(ordinalsOutput, value);
      renderPhase2Error(snapshotsOutput, value);
      renderChecks(checksOutput, {});
    }
  }
}

function renderPhase2Trace(el, trace) {
  el.className = "trace-rendered phase2-trace-rendered";
  if (!Array.isArray(trace) || trace.length === 0) {
    el.innerHTML = '<span class="empty">No trace yet.</span>';
    return;
  }
  el.innerHTML = trace.map((step) => {
    const details = compactDetails(step.details);
    return `
      <div class="trace-step phase2-trace-step">
        <span class="trace-step-num">${escapeHTML(step.step)}</span>
        <span class="trace-step-kind ${escapeAttr(step.kind || "unknown")}">${escapeHTML(step.kind || "unknown")}</span>
        <span class="trace-step-message">
          ${escapeHTML(step.message || "")}
          ${details ? `<span class="trace-step-detail">${details}</span>` : ""}
        </span>
      </div>
    `;
  }).join("");
}

function renderPhase2Messages(el, messages) {
  el.className = "table-rendered phase2-messages-rendered";
  if (!Array.isArray(messages) || messages.length === 0) {
    el.innerHTML = '<span class="empty">No messages yet.</span>';
    return;
  }
  const rows = messages.map((msg) => `
    <tr>
      <td>${escapeHTML(msg.sessionId)}</td>
      <td>${escapeHTML(msg.label)}</td>
      <td>${escapeHTML(msg.eventName)}</td>
      <td class="num">${escapeHTML(msg.publishedOrdinal)}</td>
      <td class="num">${escapeHTML(msg.assignedOrdinal)}</td>
      <td>${escapeHTML(msg.topic)}</td>
      <td><code>${escapeHTML(metadataSummary(msg.publishMetadata))}</code></td>
      <td><code>${escapeHTML(metadataSummary(msg.consumeMetadata))}</code></td>
    </tr>
  `).join("");
  el.innerHTML = `
    <table class="data-table compact-table">
      <thead>
        <tr>
          <th>Session</th>
          <th>Label</th>
          <th>Event</th>
          <th>Pub Ord</th>
          <th>Assigned</th>
          <th>Topic</th>
          <th>Published Metadata</th>
          <th>Consumed Metadata</th>
        </tr>
      </thead>
      <tbody>${rows}</tbody>
    </table>
  `;
}

function renderPhase2Ordinals(el, ordinals) {
  el.className = "table-rendered phase2-ordinals-rendered";
  const entries = Object.entries(ordinals || {});
  if (entries.length === 0) {
    el.innerHTML = '<span class="empty">No ordinals yet.</span>';
    return;
  }
  el.innerHTML = `
    <table class="data-table compact-table ordinal-table">
      <thead><tr><th>Session</th><th>Assigned Ordinals</th></tr></thead>
      <tbody>
        ${entries.map(([sid, values]) => `
          <tr>
            <td>${escapeHTML(sid)}</td>
            <td>${(values || []).map((value) => `<span class="ordinal-chip">${escapeHTML(value)}</span>`).join("")}</td>
          </tr>
        `).join("")}
      </tbody>
    </table>
  `;
}

function renderPhase2Snapshots(el, snapshots) {
  el.className = "snapshot-rendered phase2-snapshots-rendered";
  const entries = Object.entries(snapshots || {});
  if (entries.length === 0) {
    el.innerHTML = '<span class="empty">No snapshots yet.</span>';
    return;
  }
  el.innerHTML = entries.map(([sid, snapshot]) => {
    const entities = snapshot?.entities || [];
    return `
      <section class="snapshot-card compact-snapshot-card">
        <header class="snapshot-card-header">
          <span class="snapshot-card-title">${escapeHTML(sid)}</span>
          <span class="ordinal-chip">snapshot ${escapeHTML(snapshot?.snapshotOrdinal ?? snapshot?.ordinal ?? 0)}</span>
          <span class="ordinal-chip">${entities.length} entities</span>
        </header>
        ${renderSnapshotEntities(entities)}
      </section>
    `;
  }).join("");
}

function renderSnapshotEntities(entities) {
  if (!Array.isArray(entities) || entities.length === 0) {
    return '<div class="empty compact-empty">No entities.</div>';
  }
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

function renderPhase2Error(el, value) {
  el.className = "";
  setJSON(el, value);
}

function compactDetails(details) {
  if (!details || typeof details !== "object") return "";
  const preferred = ["sessionId", "label", "ordinal", "assignedOrdinal", "publishedOrdinal", "topic", "action", "streamMode"];
  const parts = [];
  for (const key of preferred) {
    if (details[key] !== undefined && details[key] !== "") parts.push(`${key}: ${details[key]}`);
  }
  for (const [key, value] of Object.entries(details)) {
    if (preferred.includes(key) || value === undefined || value === "") continue;
    if (parts.length >= 4) break;
    parts.push(`${key}: ${formatScalar(value)}`);
  }
  return parts.map(escapeHTML).join(" · ");
}

function metadataSummary(metadata) {
  if (!metadata || typeof metadata !== "object") return "";
  const keys = ["sessionstream_stream_id", "sessionstream_partition_key", "sessionstream_published_ordinal"];
  const parts = [];
  for (const key of keys) {
    if (metadata[key]) parts.push(`${shortMetadataKey(key)}=${metadata[key]}`);
  }
  return parts.join(" · ") || inlineObject(metadata);
}

function shortMetadataKey(key) {
  return key.replace(/^sessionstream_/, "").replace(/_/g, "-");
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

function bindAction(selector, handler) {
  document.querySelector(selector)?.addEventListener("click", handler);
}
