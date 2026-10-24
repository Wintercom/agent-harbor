import * as api from "../api.js";

const MAX_OUTPUT_CHARS = 1024 * 1024;

/**
 * 工作台：状态、输出、审计、停止
 */
export function mountSession(root, ctx) {
  root.innerHTML = `
    <h2>工作台</h2>
    <p class="sub">状态以事件为准；输出按不可信纯文本展示。停止后可回配置页再次开始。</p>
    <div class="card">
      <div class="row" style="justify-content:space-between">
        <div>
          <span id="sess-state" class="status-pill">无会话</span>
          <span class="meta" id="sess-meta" style="margin-left:10px"></span>
        </div>
        <div class="row">
          <button type="button" class="btn danger" id="btn-stop" disabled>停止</button>
          <button type="button" class="btn" id="btn-to-setup">回配置</button>
        </div>
      </div>
      <p class="meta" id="sess-error" style="margin-top:10px"></p>
      <p class="meta warn hidden" id="sess-hint"></p>
      <div id="sess-output" class="output"></div>
      <ul id="sess-audit" class="audit-list"></ul>
    </div>
  `;

  const els = {
    state: root.querySelector("#sess-state"),
    meta: root.querySelector("#sess-meta"),
    error: root.querySelector("#sess-error"),
    hint: root.querySelector("#sess-hint"),
    output: root.querySelector("#sess-output"),
    audit: root.querySelector("#sess-audit"),
    stop: root.querySelector("#btn-stop"),
  };

  let outputLen = 0;
  let truncated = false;

  function renderSnapshot(s) {
    if (!s) {
      els.state.textContent = "无会话";
      els.state.className = "status-pill";
      els.meta.textContent = "";
      els.stop.disabled = true;
      return;
    }
    els.state.textContent = s.state;
    els.state.className = `status-pill ${s.state}`;
    els.meta.textContent = `${s.sessionId} · seq ${s.seq} · ${s.workspace} · cleanup=${s.cleanupComplete}`;
    const active = ["provisioning", "running", "stopping"].includes(s.state) ||
      (s.state === "failed" && !s.cleanupComplete);
    els.stop.disabled = !active;
    if (s.error) {
      els.error.innerHTML = `<span class="bad">${s.error.code}: ${s.error.message}</span>`;
    } else {
      els.error.textContent = "";
    }
    if (s.state === "failed" && !s.cleanupComplete) {
      els.hint.classList.remove("hidden");
      els.hint.textContent = "清理未完成，会话锁仍占用。请再点「停止」重试清理；成功前不能开新会话。";
    } else if (s.state === "stopped") {
      els.hint.classList.remove("hidden");
      els.hint.textContent = "已停止。可回配置页修改任务后再次开始。";
    } else {
      els.hint.classList.add("hidden");
      els.hint.textContent = "";
    }
  }

  function appendOutput(text) {
    if (truncated) return;
    const next = text || "";
    if (outputLen + next.length > MAX_OUTPUT_CHARS) {
      const remain = Math.max(0, MAX_OUTPUT_CHARS - outputLen);
      els.output.textContent += next.slice(0, remain) + "\n…（前端输出已截断至 1 MiB）\n";
      outputLen = MAX_OUTPUT_CHARS;
      truncated = true;
      return;
    }
    els.output.textContent += next;
    outputLen += next.length;
    els.output.scrollTop = els.output.scrollHeight;
  }

  function onEvent(ev) {
    if (ctx.state.sessionId && ev.sessionId && ev.sessionId !== ctx.state.sessionId) return;
    if (ev.type === "state_changed" && ev.state) {
      ctx.state.lastSnapshot = ev.state;
      renderSnapshot(ev.state);
      if (ev.state.state === "stopped" || (ev.state.state === "failed" && ev.state.cleanupComplete)) {
        /* 可再次创建 */
      }
    } else if (ev.type === "output" && ev.output) {
      appendOutput(ev.output.text || "");
    } else if (ev.type === "audit" && ev.audit) {
      const li = document.createElement("li");
      li.textContent = `${ev.audit.result} · ${ev.audit.tool || "-"} @ ${ev.audit.target || "-"}`;
      els.audit.prepend(li);
    } else if (ev.type === "error" && ev.error) {
      els.error.innerHTML = `<span class="bad">${ev.error.code}: ${ev.error.message}</span>`;
    }
  }

  const unsub = api.onEvent(onEvent);

  els.stop.onclick = async () => {
    if (!ctx.state.sessionId) return;
    els.stop.disabled = true;
    const res = await api.stopSession(ctx.state.sessionId);
    if (res.error) {
      els.error.innerHTML = `<span class="bad">${res.error.code}: ${res.error.message}</span>`;
      els.stop.disabled = false;
      return;
    }
    if (res.session) renderSnapshot(res.session);
  };

  root.querySelector("#btn-to-setup").onclick = () => ctx.showTab("setup");

  return {
    resetForNewSession(snapshot) {
      outputLen = 0;
      truncated = false;
      els.output.textContent = "";
      els.audit.innerHTML = "";
      els.error.textContent = "";
      renderSnapshot(snapshot);
    },
    destroy() {
      unsub();
    },
  };
}
