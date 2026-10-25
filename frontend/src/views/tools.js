/**
 * PAGE-DESIGN：工具页 — MCP；两天版 Token 轨 + 固定 demo-mcp
 */
import * as api from "../api.js";

export function mountTools(root, ctx) {
  root.innerHTML = `
    <h2>工具</h2>
    <p class="sub">给 Agent 一双手（MCP）。OAuth 轨待 P0.5；当前仅 API Key/Token。</p>
    <section class="card">
      <h3>添加 / 连接</h3>
      <label class="field">显示名
        <input id="mcp-name" value="演示 MCP" maxlength="100" />
      </label>
      <label class="field">profileId
        <input id="mcp-profile" value="demo-mcp" readonly />
      </label>
      <label class="field">API Key / Token（提交后清空）
        <input id="mcp-key" type="password" placeholder="粘贴 Token" autocomplete="off" />
      </label>
      <div class="row">
        <button type="button" class="btn primary" id="btn-connect-mcp">连接 MCP</button>
        <span class="meta" id="mcp-status"></span>
      </div>
    </section>
    <section class="card" style="margin-top:12px">
      <h3>已添加</h3>
      <div id="mcp-list"></div>
    </section>
  `;

  const status = root.querySelector("#mcp-status");
  const list = root.querySelector("#mcp-list");
  function renderList() {
    if (!ctx.state.mcpAccount) {
      list.innerHTML = `<p class="meta">列表为空。</p>`;
      return;
    }
    const a = ctx.state.mcpAccount;
    list.innerHTML = `
      <div class="row session-card">
        <span class="ok">● ${a.name || a.id}</span>
        <span class="meta">${a.profileId} · ${a.host || ""} · ${a.id}</span>
      </div>
    `;
  }
  renderList();

  root.querySelector("#btn-connect-mcp").onclick = async () => {
    const name = root.querySelector("#mcp-name").value.trim();
    const key = root.querySelector("#mcp-key").value;
    if (!name || !key) {
      status.innerHTML = `<span class="bad">请填写显示名与 Token</span>`;
      return;
    }
    const res = await api.connectMcp({
      profileId: "demo-mcp",
      name,
      apiKey: key,
    });
    root.querySelector("#mcp-key").value = "";
    if (res.error) {
      ctx.state.mcpAccount = null;
      status.innerHTML = `<span class="bad">${res.error.code}: ${res.error.message}</span>`;
    } else {
      ctx.state.mcpAccount = res.account;
      status.innerHTML = `<span class="ok">已连接 · ${res.account.id}</span>`;
    }
    renderList();
    ctx.onAccountsChanged?.();
  };
}
