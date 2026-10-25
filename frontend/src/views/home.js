/**
 * PAGE-DESIGN §5.2 HOME — 会话中心（单会话时代先展示当前/历史摘要）
 */
export function mountHome(root, ctx) {
  const running = (ctx.state.sessionHistory || []).filter((s) =>
    ["provisioning", "running", "stopping"].includes(s.state),
  ).length;
  const healthOk = ctx.state.health?.available !== false;
  const list = ctx.state.sessionHistory || [];

  root.innerHTML = `
    <div class="page-head row" style="justify-content:space-between">
      <div>
        <h2>我的会话</h2>
        <p class="sub">运行中 ${running}/2（CONTRACT v1.1 上限；实现未就绪时 mock 仍可能单会话）</p>
      </div>
      <button type="button" class="btn primary" id="btn-new-session" ${healthOk ? "" : "disabled"}>
        + 新会话
      </button>
    </div>
    ${
      !healthOk
        ? `<p class="meta bad">隔离不可用，已禁用新会话。请打开设置查看检测结果或见阻断说明。</p>`
        : ""
    }
    <div id="home-list" class="session-list"></div>
    ${
      list.length === 0
        ? `<p class="meta">暂无会话。先在「模型」「工具」连接账号，再点「新会话」。</p>`
        : ""
    }
  `;

  const listEl = root.querySelector("#home-list");
  list.forEach((s) => {
    const card = document.createElement("article");
    card.className = "card session-card";
    card.innerHTML = `
      <div class="row" style="justify-content:space-between">
        <div>
          <strong>${s.agentId || "—"}</strong>
          <span class="status-pill ${s.state || ""}">${s.state || "unknown"}</span>
          <p class="meta">${s.workspace || ""} · ${s.sessionId || ""}</p>
        </div>
        <div class="row">
          <button type="button" class="btn" data-open="${s.sessionId}">打开工作台</button>
          <button type="button" class="btn danger" data-stop="${s.sessionId}" ${
            ["stopped", "failed"].includes(s.state) && s.cleanupComplete ? "disabled" : ""
          }>停止</button>
        </div>
      </div>
    `;
    listEl.appendChild(card);
  });

  root.querySelector("#btn-new-session").onclick = () => ctx.navigate("wizard");
  listEl.querySelectorAll("[data-open]").forEach((btn) => {
    btn.onclick = async () => {
      const id = btn.getAttribute("data-open");
      if (ctx.openSession) {
        await ctx.openSession(id);
        return;
      }
      ctx.state.sessionId = id;
      ctx.navigate("session");
    };
  });
  listEl.querySelectorAll("[data-stop]").forEach((btn) => {
    btn.onclick = async () => {
      const id = btn.getAttribute("data-stop");
      btn.disabled = true;
      await ctx.stopSession(id);
      ctx.refreshHome?.();
    };
  });
}
