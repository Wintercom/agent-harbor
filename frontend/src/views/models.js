/**
 * PAGE-DESIGN：模型页 — 本机录入；两天版仍固定 profileId
 */
import * as api from "../api.js";

export function mountModels(root, ctx) {
  const profile =
    ctx.state.agentId === "dsh" ? "dsh-model" : "demo-model";
  const acc = ctx.state.modelAccount;

  root.innerHTML = `
    <h2>模型</h2>
    <p class="sub">给 Agent 一把「嘴」。密钥只在宿主机；提交后清空。当前契约固定 profileId（自由添加 OpenAPI 待 P0.5）。</p>
    <section class="card">
      <h3>添加 / 连接</h3>
      <label class="field">显示名
        <input id="model-name" value="演示模型" maxlength="100" />
      </label>
      <label class="field">profileId
        <input id="model-profile" value="${profile}" readonly />
      </label>
      <label class="field">API Key（提交后清空）
        <input id="model-key" type="password" placeholder="粘贴 Key" autocomplete="off" />
      </label>
      <div class="row">
        <button type="button" class="btn primary" id="btn-connect-model">连接模型</button>
        <span class="meta" id="model-status"></span>
      </div>
    </section>
    <section class="card" style="margin-top:12px">
      <h3>已添加</h3>
      <div id="model-list"></div>
    </section>
  `;

  const status = root.querySelector("#model-status");
  const list = root.querySelector("#model-list");
  function renderList() {
    if (!ctx.state.modelAccount) {
      list.innerHTML = `<p class="meta">列表为空。添加成功后显示账号引用（无 Key）。</p>`;
      return;
    }
    const a = ctx.state.modelAccount;
    list.innerHTML = `
      <div class="row session-card">
        <span class="ok">● ${a.name || a.id}</span>
        <span class="meta">${a.profileId} · ${a.host || ""} · ${a.id}</span>
      </div>
    `;
  }
  renderList();
  if (acc) status.innerHTML = `<span class="ok">当前已连接</span>`;

  root.querySelector("#btn-connect-model").onclick = async () => {
    const name = root.querySelector("#model-name").value.trim();
    const key = root.querySelector("#model-key").value;
    const profileId = root.querySelector("#model-profile").value;
    if (!name || !key) {
      status.innerHTML = `<span class="bad">请填写显示名与 API Key</span>`;
      return;
    }
    const res = await api.connectModel({ profileId, name, apiKey: key });
    root.querySelector("#model-key").value = "";
    if (res.error) {
      ctx.state.modelAccount = null;
      status.innerHTML = `<span class="bad">${res.error.code}: ${res.error.message}</span>`;
    } else {
      ctx.state.modelAccount = res.account;
      status.innerHTML = `<span class="ok">已连接 · ${res.account.id}</span>`;
    }
    renderList();
    ctx.onAccountsChanged?.();
  };
}
