/**
 * PAGE-DESIGN §5.7 模型：自由添加 OpenAPI（名 + 地址 + Key）
 * Live 自定义 origin / ListAccounts 依赖 B（FULL-GATE）；本页以 mock 验收 UI。
 */
import * as api from "../api.js";

const PRESETS = [
  { name: "DeepSeek", origin: "https://api.deepseek.com" },
  { name: "Anthropic", origin: "https://api.anthropic.com" },
];

export function mountModels(root, ctx) {
  let editing = null; // account id or "new"

  async function load() {
    const res = await api.listAccounts("model");
    ctx.state.modelAccounts = res.accounts || [];
    // 兼容旧单账号字段
    if (ctx.state.modelAccounts[0]) ctx.state.modelAccount = ctx.state.modelAccounts[0];
    paint();
  }

  function paint() {
    const list = ctx.state.modelAccounts || [];
    root.innerHTML = `
      <div class="page-head row" style="justify-content:space-between">
        <div>
          <h2>模型</h2>
          <p class="sub">给 Agent 一把「嘴」。自行添加：名称、API 地址、API Key。无浏览器登录。</p>
        </div>
        <button type="button" class="btn primary" id="btn-add-model">+ 添加</button>
      </div>
      <div id="model-list" class="session-list"></div>
      ${
        list.length === 0 && editing !== "new"
          ? `<div class="empty-state"><p>还没有模型。自行添加名称、API 地址与 Key；无浏览器登录。</p>
             <div class="actions"><button type="button" class="btn primary" id="empty-add-model">+ 添加第一个</button></div></div>`
          : ""
      }
      <div id="model-form" class="${editing ? "" : "hidden"}"></div>
    `;

    const listEl = root.querySelector("#model-list");
    list.forEach((a) => {
      const row = document.createElement("article");
      row.className = "card session-card";
      row.innerHTML = `
        <div class="row" style="justify-content:space-between">
          <div>
            <strong>${a.name}</strong>
            <span class="meta">${a.host || a.profileId || ""}</span>
            ${a.invalid ? `<span class="bad">▲ 无效</span>` : ""}
          </div>
          <div class="row">
            <button type="button" class="btn" data-edit="${a.id}">编辑</button>
            <button type="button" class="btn danger" data-del="${a.id}">删除</button>
          </div>
        </div>`;
      listEl.appendChild(row);
    });

    const startAdd = () => {
      editing = "new";
      paint();
      paintForm();
    };
    root.querySelector("#btn-add-model").onclick = startAdd;
    root.querySelector("#empty-add-model")?.addEventListener("click", startAdd);
    listEl.querySelectorAll("[data-edit]").forEach((btn) => {
      btn.onclick = () => {
        editing = btn.getAttribute("data-edit");
        paint();
        paintForm();
      };
    });
    listEl.querySelectorAll("[data-del]").forEach((btn) => {
      btn.onclick = async () => {
        const id = btn.getAttribute("data-del");
        const acc = list.find((x) => x.id === id);
        if (!confirm(`删除模型「${acc?.name || id}」及其本机 API Key？`)) return;
        await api.removeAccount(id);
        await load();
      };
    });
    if (editing) paintForm();
  }

  function paintForm() {
    const form = root.querySelector("#model-form");
    const acc =
      editing === "new"
        ? { name: "", host: "", profileId: "custom" }
        : (ctx.state.modelAccounts || []).find((a) => a.id === editing) || {};
    // 占位符勿写成真实 URL，否则深色主题下易被当成已填写
    const originValue =
      acc.origin || (acc.host ? "https://" + acc.host : "");
    form.classList.remove("hidden");
    form.innerHTML = `
      <section class="card">
        <h3>${editing === "new" ? "添加模型" : "编辑模型"}</h3>
        <label class="field">名称<input id="m-name" value="${acc.name || ""}" maxlength="100"/></label>
        <label class="field">API 地址<input id="m-origin" value="${originValue}" placeholder="手输 https://主机 或点下方快捷填入"/></label>
        <p class="meta">须有实际内容（灰色提示不算）。浏览器 mock 可验 UI；桌面 live 的自定义 origin / 列表持久化由 B 接线。</p>
        <label class="field">API Key（提交后清空）<input id="m-key" type="password" autocomplete="off" placeholder="在此粘贴 Key"/></label>
        <p class="meta">快捷填入默认地址（可选）：
          ${PRESETS.map((p) => `<button type="button" class="btn" data-preset="${p.origin}" data-pname="${p.name}">${p.name}</button>`).join(" ")}
        </p>
        <div class="actions">
          <button type="button" class="btn" id="m-cancel">取消</button>
          <button type="button" class="btn primary" id="m-save">保存</button>
          <span class="meta" id="m-status"></span>
        </div>
      </section>`;
    const nameEl = form.querySelector("#m-name");
    const originEl = form.querySelector("#m-origin");
    const keyEl = form.querySelector("#m-key");
    const status = form.querySelector("#m-status");
    const clearStatus = () => {
      status.textContent = "";
    };
    form.querySelectorAll("[data-preset]").forEach((btn) => {
      btn.onclick = () => {
        originEl.value = btn.getAttribute("data-preset") || "";
        if (!nameEl.value.trim()) {
          nameEl.value = "我的 " + (btn.getAttribute("data-pname") || "");
        }
        clearStatus();
        keyEl.focus();
      };
    });
    form.querySelector("#m-cancel").onclick = () => {
      editing = null;
      paint();
    };
    nameEl.addEventListener("input", clearStatus);
    originEl.addEventListener("input", clearStatus);
    keyEl.addEventListener("input", clearStatus);
    form.querySelector("#m-save").onclick = async () => {
      const name = nameEl.value.trim();
      const origin = originEl.value.trim();
      const key = keyEl.value.trim();
      const missing = [];
      if (!name) missing.push("名称");
      if (!origin) missing.push("API 地址");
      if (!key) missing.push("API Key");
      if (missing.length) {
        status.innerHTML = `<span class="bad">请填写：${missing.join("、")}（灰色提示不是已填内容）</span>`;
        if (!origin) originEl.focus();
        else if (!key) keyEl.focus();
        else nameEl.focus();
        return;
      }
      status.innerHTML = `<span class="meta">正在保存…</span>`;
      const saveBtn = form.querySelector("#m-save");
      saveBtn.disabled = true;
      try {
        if (editing !== "new") await api.removeAccount(editing);
        const res = await api.connectModel({
          profileId: "custom",
          name,
          apiKey: key,
          origin,
        });
        if (res.error) {
          // 失败保留 Key，便于改地址后重试
          status.innerHTML = `<span class="bad">${res.error.code}: ${res.error.message}</span>`;
          return;
        }
        keyEl.value = "";
        editing = null;
        ctx.onAccountsChanged?.();
        await load();
      } finally {
        saveBtn.disabled = false;
      }
    };
  }

  load();
}
