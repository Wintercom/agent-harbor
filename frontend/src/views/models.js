/**
 * PAGE-DESIGN §5.7 模型：自由添加 OpenAPI（名 + 地址 + Key）
 * Live 自定义 origin 须 Anthropic 兼容 /v1/messages。
 */
import * as api from "../api.js";

const PRESETS = [
  { name: "Anthropic", origin: "https://api.anthropic.com" },
  { name: "DeepSeek", origin: "https://api.deepseek.com" },
];

export function mountModels(root, ctx, opts = {}) {
  const returning = ctx.state.returnAfterAccount?.kind === "model";
  let editing = opts.intent === "add" || returning ? "new" : null;

  function clearReturn() {
    if (ctx.state.returnAfterAccount?.kind === "model") {
      ctx.state.returnAfterAccount = null;
    }
  }

  function resumeWizard(accountId) {
    const ret = ctx.state.returnAfterAccount;
    if (!ret || ret.kind !== "model" || ret.route !== "wizard") return false;
    const prev = ctx.state.wizardResume || { step: 2, draft: {} };
    ctx.state.wizardResume = {
      step: 2,
      draft: {
        ...(prev.draft || {}),
        modelAccountId: accountId || prev.draft?.modelAccountId || "",
      },
    };
    ctx.state.returnAfterAccount = null;
    ctx.navigate("wizard");
    return true;
  }

  async function load() {
    const res = await api.listAccounts("model");
    ctx.state.modelAccounts = res.accounts || [];
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
      ${
        returning
          ? `<p class="meta ok" id="model-return-banner">添加成功后将回到「新会话」模型步骤。
             <button type="button" class="btn" id="m-back-wizard">直接返回</button></p>`
          : ""
      }
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
    root.querySelector("#m-back-wizard")?.addEventListener("click", () => {
      clearReturn();
      ctx.navigate("wizard");
    });
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
    const originValue = acc.origin || (acc.host ? "https://" + acc.host : "");
    form.classList.remove("hidden");
    form.innerHTML = `
      <section class="card">
        <h3>${editing === "new" ? "添加模型" : "编辑模型"}</h3>
        <label class="field">名称<input id="m-name" value="${acc.name || ""}" maxlength="100"/></label>
        <label class="field">API 地址<input id="m-origin" value="${originValue}" placeholder="https://主机（勿带 path；探测走 /v1/messages）"/></label>
        <p class="meta">须 Anthropic 兼容：HTTPS 主机 + Key。探测会 POST <code>/v1/messages</code>（同时带 X-Api-Key 与 Bearer）。</p>
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
      if (returning) {
        clearReturn();
        ctx.navigate("wizard");
        return;
      }
      paint();
    };
    nameEl.addEventListener("input", clearStatus);
    originEl.addEventListener("input", clearStatus);
    keyEl.addEventListener("input", clearStatus);
    form.querySelector("#m-save").onclick = async () => {
      const name = nameEl.value.trim();
      let origin = originEl.value.trim().replace(/\/+$/, "");
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
      // 契约 Origin 仅 scheme+host[:port]；用户若贴了 /v1 等 path，剥掉再提交。
      try {
        const u = new URL(origin);
        if (u.pathname && u.pathname !== "/") {
          origin = `${u.protocol}//${u.host}`;
          originEl.value = origin;
        }
      } catch (_) {
        /* 交给后端校验 */
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
          let hint = "";
          if (res.error.code === "UPSTREAM_AUTH_FAILED") {
            hint =
              " 请确认 Key 有效，且地址为 Anthropic 兼容上游（非纯 OpenAI /chat/completions）。";
          }
          status.innerHTML = `<span class="bad">${res.error.code}: ${res.error.message}</span><span class="meta">${hint}</span>`;
          return;
        }
        keyEl.value = "";
        editing = null;
        ctx.onAccountsChanged?.();
        if (resumeWizard(res.account?.id)) return;
        await load();
      } finally {
        saveBtn.disabled = false;
      }
    };
  }

  load();
}
