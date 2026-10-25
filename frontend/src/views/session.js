/**
 * PAGE-DESIGN §5.4 工作台：边界条 + PTY + 摘要；扩权 / 权限一览
 */
import * as api from "../api.js";
import { OutputTail } from "../session-events.js";
import {
  mountPermissionModal,
  hidePermissionModal,
} from "./permission-modal.js";

export function mountSession(root, ctx) {
  const healthOk = ctx.state.health?.available !== false;

  root.innerHTML = `
    <div id="sess-empty" class="empty-state">
      <p>当前没有打开的会话。从 HOME 进入工作台，或创建新会话。</p>
      <div class="actions">
        <button type="button" class="btn" id="sess-empty-home">回 HOME</button>
        <button type="button" class="btn primary" id="sess-empty-wizard" ${healthOk ? "" : "disabled"}>新会话</button>
      </div>
    </div>
    <div id="sess-workbench" class="hidden">
      <div class="boundary" id="boundary">
        <div class="row" style="justify-content:space-between;width:100%">
          <div>
            <span id="sess-state" class="status-pill">无会话</span>
            <span class="meta" id="sess-meta"></span>
          </div>
          <div class="row">
            <label class="field inline">当前模型
              <select id="sess-model"></select>
            </label>
            <button type="button" class="btn" id="btn-perm">权限一览</button>
            <button type="button" class="btn danger" id="btn-stop" disabled>停止</button>
            <button type="button" class="btn" id="btn-to-home">回 HOME</button>
          </div>
        </div>
        <p class="meta" id="sess-hint-line"></p>
      </div>
      <div class="session-grid">
        <div class="pty-panel">
          <div class="pty-toolbar meta">终端 / Agent 输出（PTY）</div>
          <div id="sess-pty" class="pty" tabindex="0"></div>
          <textarea id="sess-pty-input" class="pty-input" rows="2" placeholder="同会话后续轮次：输入后 Enter 发送（mock 回显；live 走 WritePty）"></textarea>
        </div>
        <aside class="session-side">
          <h3>本趟摘要</h3>
          <ul class="meta" id="sess-stats">
            <li>输出块：<span id="stat-out">0</span></li>
            <li>审计：<span id="stat-audit">0</span></li>
          </ul>
          <h3>最近活动</h3>
          <ul id="sess-audit" class="audit-list"></ul>
          <p class="meta warn hidden" id="sess-hint"></p>
          <p class="meta" id="sess-error"></p>
        </aside>
      </div>
      <div id="perm-drawer" class="perm-drawer hidden"></div>
    </div>
  `;

  const els = {
    empty: root.querySelector("#sess-empty"),
    workbench: root.querySelector("#sess-workbench"),
    boundary: root.querySelector("#boundary"),
    state: root.querySelector("#sess-state"),
    meta: root.querySelector("#sess-meta"),
    hintLine: root.querySelector("#sess-hint-line"),
    model: root.querySelector("#sess-model"),
    stop: root.querySelector("#btn-stop"),
    pty: root.querySelector("#sess-pty"),
    ptyInput: root.querySelector("#sess-pty-input"),
    audit: root.querySelector("#sess-audit"),
    error: root.querySelector("#sess-error"),
    hint: root.querySelector("#sess-hint"),
    statOut: root.querySelector("#stat-out"),
    statAudit: root.querySelector("#stat-audit"),
    drawer: root.querySelector("#perm-drawer"),
  };

  let tail = new OutputTail();
  let outCount = 0;
  let auditCount = 0;
  const modalRoot =
    typeof document !== "undefined"
      ? document.getElementById("modal-root")
      : null;

  function showEmpty() {
    els.empty?.classList.remove("hidden");
    els.workbench?.classList.add("hidden");
  }

  function showWorkbench() {
    els.empty?.classList.add("hidden");
    els.workbench?.classList.remove("hidden");
  }

  const emptyHome = root.querySelector("#sess-empty-home");
  if (emptyHome && typeof emptyHome.addEventListener === "function") {
    emptyHome.addEventListener("click", () => ctx.navigate?.("home"));
  }
  const emptyWizard = root.querySelector("#sess-empty-wizard");
  if (emptyWizard && typeof emptyWizard.addEventListener === "function") {
    emptyWizard.addEventListener("click", () => {
      if (ctx.state.health?.available === false) {
        ctx.showBlock?.(ctx.state.health.reason);
        return;
      }
      ctx.navigate?.("wizard");
    });
  }

  function setAwaiting(on) {
    const list = els.boundary?.classList;
    if (list && typeof list.toggle === "function") {
      list.toggle("awaiting", !!on);
    }
  }

  function notifyResize() {
    if (!ctx.state.sessionId || !els.pty) return;
    const cols = Math.max(40, Math.floor((els.pty.clientWidth || 640) / 8));
    const rows = Math.max(10, Math.floor((els.pty.clientHeight || 240) / 16));
    api.resizePty({ sessionId: ctx.state.sessionId, cols, rows }).catch(() => {});
  }

  async function fillModels(selectedId) {
    if (!els.model) return;
    if (typeof document === "undefined" || !document.createElement) return;
    const res = await api.listAccounts("model");
    const accounts = res.accounts || [];
    const want =
      selectedId ||
      ctx.state.lastSnapshot?.modelAccountId ||
      ctx.state.modelAccount?.id ||
      "";
    const options = accounts.map((a) => {
      const o = document.createElement("option");
      o.value = a.id;
      o.textContent = `${a.name} · ${a.host || ""}`;
      if (a.id === want) o.selected = true;
      return o;
    });
    if (typeof els.model.replaceChildren === "function") {
      els.model.replaceChildren(...options);
    } else {
      els.model.innerHTML = "";
      options.forEach((o) => els.model.appendChild?.(o));
    }
  }

  function renderSnapshot(s) {
    if (!s) {
      showEmpty();
      setAwaiting(false);
      if (els.state) {
        els.state.textContent = "无会话";
        els.state.className = "status-pill";
      }
      if (els.meta) els.meta.textContent = "";
      if (els.stop) els.stop.disabled = true;
      return;
    }
    showWorkbench();
    els.state.textContent = s.state;
    els.state.className = `status-pill ${s.state}`;
    const toolN = (s.mcpIds || []).length;
    els.meta.textContent = `${s.agentId || ""} · ${s.workspace || ""} · 文件:${s.mountMode === "clone" ? "安全副本" : "直接编辑"} · 工具:${toolN}`;
    els.hintLine.textContent = `● 隔离运行中 · ${s.sessionId || ""}`;
    const active =
      ["provisioning", "running", "stopping"].includes(s.state) ||
      (s.state === "failed" && !s.cleanupComplete);
    els.stop.disabled = !active;
    if (s.error) els.error.textContent = `${s.error.code}: ${s.error.message}`;
    else els.error.textContent = "";
    if (s.state === "failed" && !s.cleanupComplete) {
      els.hint.classList.remove("hidden");
      els.hint.textContent = "清理未完成，请再点停止。";
    } else if (s.state === "stopped") {
      els.hint.classList.remove("hidden");
      els.hint.textContent = "已停止。可回 HOME 再开。";
    } else {
      els.hint.classList.add("hidden");
    }
    if (s.modelAccountId) fillModels(s.modelAccountId);
  }

  function appendPty(text) {
    if (!els.pty) return;
    els.pty.textContent += text || "";
    els.pty.scrollTop = els.pty.scrollHeight;
  }

  function onEvent(ev) {
    if (ctx.state.sessionId && ev.sessionId && ev.sessionId !== ctx.state.sessionId)
      return;
    if (ev.type === "state_changed" && ev.state) {
      ctx.state.lastSnapshot = ev.state;
      renderSnapshot(ev.state);
    } else if (ev.type === "output" && ev.output) {
      outCount += 1;
      els.statOut.textContent = String(outCount);
      tail.append(ev.seq, ev.output.text || "");
      appendPty(ev.output.text || "");
    } else if (ev.type === "pty" && (ev.pty || ev.output)) {
      const t = (ev.pty || ev.output).text || "";
      appendPty(t);
    } else if (ev.type === "audit" && ev.audit) {
      auditCount += 1;
      els.statAudit.textContent = String(auditCount);
      const li = document.createElement("li");
      li.textContent = `${ev.audit.result} · ${ev.audit.tool || "-"} @ ${ev.audit.target || "-"}`;
      els.audit.prepend(li);
    } else if (ev.type === "error" && ev.error) {
      els.error.textContent = `${ev.error.code}: ${ev.error.message}`;
    } else if (ev.type === "permission_needed" && ev.permission) {
      const snap = ctx.state.lastSnapshot;
      const sid = ctx.state.sessionId || snap?.sessionId || "";
      setAwaiting(true);
      mountPermissionModal(modalRoot, {
        sessionLabel: `${snap?.agentId || "—"} · ${snap?.workspace || "—"} · ${sid}`,
        target: ev.permission.target,
        action: ev.permission.action,
        async onDecide(decision) {
          const res = await api.resolvePermission({
            sessionId: sid,
            target: ev.permission.target,
            decision,
          });
          if (res?.error) {
            throw new Error(`${res.error.code}: ${res.error.message}`);
          }
          hidePermissionModal(modalRoot);
          setAwaiting(false);
          const li = document.createElement("li");
          li.textContent = `扩权 ${decision} · ${ev.permission.target} · ${sid}`;
          els.audit?.prepend(li);
        },
      });
    }
  }

  const unsub = api.onEvent(onEvent);
  fillModels();

  if (els.model) {
    els.model.onchange = async () => {
      if (!ctx.state.sessionId || !els.model.value) return;
      const res = await api.switchModel({
        sessionId: ctx.state.sessionId,
        modelAccountId: els.model.value,
      });
      if (res.error) {
        els.error.textContent = `${res.error.code}: ${res.error.message}`;
      }
    };
  }

  if (els.stop) {
    els.stop.onclick = async () => {
      if (!ctx.state.sessionId) return;
      const prefs = ctx.state.uiPrefs || {};
      if (!window.confirm("停止此会话？")) return;
      let destroyAfter = false;
      if (prefs.askClearOnStop !== false) {
        destroyAfter = window.confirm(
          "清除隔离环境数据？本机密钥保留。\n确定 = 清除并停止；取消 = 仅停止。",
        );
      } else {
        destroyAfter = prefs.clearOnStopDefault === "clear";
      }
      els.stop.disabled = true;
      const res = await api.stopSession(ctx.state.sessionId);
      if (res.error) {
        els.error.textContent = `${res.error.code}: ${res.error.message}`;
        els.stop.disabled = false;
        return;
      }
      if (destroyAfter && ctx.state.sessionId) {
        await ctx.destroySession?.(ctx.state.sessionId);
      }
    };
  }

  const toHome = root.querySelector("#btn-to-home");
  if (toHome) toHome.onclick = () => ctx.navigate?.("home") || ctx.showTab?.("setup");

  const permBtn = root.querySelector("#btn-perm");
  if (permBtn) permBtn.onclick = async () => {
    if (!els.drawer) return;
    const sid = ctx.state.sessionId || "";
    const res = await api.getPermissionOverview(sid);
    const hostsRes = await api.listRememberedHosts(sid);
    const o = res.overview;
    const remembered = hostsRes.hosts || [];
    els.drawer.classList.remove("hidden");
    els.drawer.innerHTML = `
      <div class="card">
        <div class="row" style="justify-content:space-between">
          <h3>当前边界</h3>
          <button type="button" class="btn" id="perm-close">关闭</button>
        </div>
        ${
          res.error
            ? `<p class="bad">${res.error.code}: ${res.error.message}</p>`
            : `
          <p class="meta">会话 ${sid || "—"} · 工作区：${o?.workspace || "—"}</p>
          <h4>文件</h4>
          <ul class="audit-list">
            <li>可读写：${o?.workspace || "—"}（${ctx.state.lastSnapshot?.mountMode === "clone" ? "安全副本" : "直接编辑"}）</li>
            ${(o?.deniedPathCategories || []).map((m) => `<li>默认不可见：${m}</li>`).join("")}
          </ul>
          <h4>网络（白名单）</h4>
          <ul class="audit-list" id="perm-hosts">
            ${(o?.allowedHosts || [])
              .map((h) => `<li>${h} · 模型/系统</li>`)
              .join("") || "<li class='meta'>无</li>"}
            ${remembered
              .map(
                (h) =>
                  `<li class="row" style="justify-content:space-between">
                    <span>${h.host} · ${h.scope === "project" ? "本项目始终允许" : "本会话"}</span>
                    <button type="button" class="btn" data-rev="${h.host}" data-ws="${h.workspace || o?.workspace || ""}">撤销</button>
                  </li>`,
              )
              .join("")}
          </ul>
          <h4>工具</h4>
          <ul class="audit-list">${(o?.mcpSummary || []).map((m) => `<li>${m}</li>`).join("") || "<li class='meta'>无</li>"}</ul>
        `
        }
      </div>`;
    els.drawer.querySelector("#perm-close").onclick = () => {
      els.drawer.classList.add("hidden");
      els.drawer.innerHTML = "";
    };
    els.drawer.querySelectorAll("[data-rev]").forEach((btn) => {
      btn.onclick = async () => {
        await api.revokeRememberedHost({
          host: btn.getAttribute("data-rev"),
          workspace: btn.getAttribute("data-ws"),
        });
        permBtn.click();
      };
    });
  };

  if (els.ptyInput?.addEventListener) {
    els.ptyInput.addEventListener("keydown", async (e) => {
      if (e.key !== "Enter" || e.shiftKey) return;
      e.preventDefault();
      const data = els.ptyInput.value;
      els.ptyInput.value = "";
      if (!ctx.state.sessionId || !data) return;
      appendPty(`\n$ ${data}\n`);
      await api.writePty({ sessionId: ctx.state.sessionId, data: data + "\n" });
    });
  }

  const side = root.querySelector(".session-side");
  if (side && typeof side.appendChild === "function" && typeof document !== "undefined") {
    const demo = document.createElement("button");
    demo.type = "button";
    demo.className = "btn";
    demo.textContent = "演示扩权（mock）";
    demo.style.marginTop = "8px";
    side.appendChild(demo);
    demo.onclick = () => {
      if (!ctx.state.sessionId) {
        els.error.textContent = "请先打开会话再演示扩权";
        return;
      }
      onEvent({
        sessionId: ctx.state.sessionId,
        seq: Date.now(),
        type: "permission_needed",
        permission: { target: "api.npmjs.org", action: "请求访问新地址" },
      });
    };
  }

  if (typeof window !== "undefined") {
    window.addEventListener("resize", notifyResize);
  }

  // 初始无会话
  showEmpty();

  return {
    resetForNewSession(snapshot) {
      tail = new OutputTail();
      outCount = 0;
      auditCount = 0;
      if (els.statOut) els.statOut.textContent = "0";
      if (els.statAudit) els.statAudit.textContent = "0";
      if (els.pty) els.pty.textContent = "";
      if (els.audit) els.audit.innerHTML = "";
      if (els.error) els.error.textContent = "";
      els.hint?.classList.add("hidden");
      hidePermissionModal(modalRoot);
      setAwaiting(false);
      if (snapshot) {
        ctx.state.lastSnapshot = snapshot;
        api.activateSession(snapshot);
        renderSnapshot(snapshot);
        notifyResize();
      } else {
        renderSnapshot(null);
      }
      fillModels(snapshot?.modelAccountId);
    },
    showEmpty,
    destroy() {
      unsub();
      hidePermissionModal(modalRoot);
      setAwaiting(false);
      if (typeof window !== "undefined") {
        window.removeEventListener("resize", notifyResize);
      }
    },
  };
}
