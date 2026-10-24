import * as api from "../api.js";

/**
 * 配置页：连接模型/MCP、选工作区、任务、启动
 */
export function mountSetup(root, ctx) {
  root.innerHTML = `
    <h2>配置</h2>
    <p class="sub">凭据提交后立即清空；成功只保留账号引用。profileId 固定为 demo-model / demo-mcp。</p>
    <div class="grid">
      <section class="card">
        <h3>Agent</h3>
        <p class="meta" id="agent-status">检测中…</p>
        <div class="actions">
          <button type="button" class="btn" id="btn-refresh-agents">重新检测</button>
        </div>
      </section>
      <section class="card">
        <h3>模拟场景（仅 mock）</h3>
        <label class="field">场景
          <select id="scenario">
            <option value="normal">正常闭环</option>
            <option value="auth">认证失败</option>
            <option value="busy">SESSION_BUSY</option>
            <option value="isolation">ISOLATION_UNAVAILABLE</option>
            <option value="cleanup">cleanupComplete=false</option>
          </select>
        </label>
        <p class="meta">对应验收 D4 四个失败场景 + 正常流。</p>
      </section>
      <section class="card">
        <h3>模型连接</h3>
        <label class="field">显示名
          <input id="model-name" value="演示模型" maxlength="100" />
        </label>
        <label class="field">profileId
          <input id="model-profile" value="demo-model" readonly />
        </label>
        <label class="field">API Key（提交后清空）
          <input id="model-key" type="password" placeholder="粘贴测试 Key" autocomplete="off" />
        </label>
        <div class="row">
          <button type="button" class="btn primary" id="btn-connect-model">连接模型</button>
          <span class="meta" id="model-status">未连接</span>
        </div>
      </section>
      <section class="card">
        <h3>MCP 连接</h3>
        <label class="field">显示名
          <input id="mcp-name" value="演示 MCP" maxlength="100" />
        </label>
        <label class="field">profileId
          <input id="mcp-profile" value="demo-mcp" readonly />
        </label>
        <label class="field">API Key / Token（提交后清空）
          <input id="mcp-key" type="password" placeholder="粘贴测试 Token" autocomplete="off" />
        </label>
        <div class="row">
          <button type="button" class="btn primary" id="btn-connect-mcp">连接 MCP</button>
          <span class="meta" id="mcp-status">未连接</span>
        </div>
      </section>
    </div>
    <section class="card" style="margin-top:12px">
      <h3>工作区与任务</h3>
      <p class="meta warn">direct 模式会直接修改所选文件夹中的文件。</p>
      <div class="row" style="margin:10px 0">
        <button type="button" class="btn" id="btn-pick-ws">选择工作区</button>
        <span class="meta" id="ws-path">未选择</span>
      </div>
      <label class="field">任务（将发给 Agent）
        <textarea id="task">请在工作区写一个 hello.txt，并用 MCP 做一次只读查询后总结结果。</textarea>
      </label>
      <div class="actions">
        <button type="button" class="btn primary" id="btn-start" disabled>开始运行</button>
        <span class="meta" id="setup-msg"></span>
      </div>
    </section>
  `;

  const els = {
    agentStatus: root.querySelector("#agent-status"),
    scenario: root.querySelector("#scenario"),
    modelName: root.querySelector("#model-name"),
    modelKey: root.querySelector("#model-key"),
    modelStatus: root.querySelector("#model-status"),
    mcpName: root.querySelector("#mcp-name"),
    mcpKey: root.querySelector("#mcp-key"),
    mcpStatus: root.querySelector("#mcp-status"),
    wsPath: root.querySelector("#ws-path"),
    task: root.querySelector("#task"),
    start: root.querySelector("#btn-start"),
    msg: root.querySelector("#setup-msg"),
  };

  function refreshStart() {
    const ok =
      ctx.state.agentAvailable &&
      ctx.state.modelAccount &&
      ctx.state.mcpAccount &&
      ctx.state.workspace &&
      els.task.value.trim().length > 0;
    els.start.disabled = !ok;
  }

  async function refreshAgents() {
    const res = await api.listAgents();
    if (res.error) {
      els.agentStatus.innerHTML = `<span class="bad">${res.error.code}: ${res.error.message}</span>`;
      ctx.state.agentAvailable = false;
      refreshStart();
      return;
    }
    const a = (res.agents || [])[0];
    if (!a) {
      els.agentStatus.textContent = "无可用 Agent";
      ctx.state.agentAvailable = false;
    } else {
      ctx.state.agentId = a.id;
      ctx.state.agentAvailable = !!a.available;
      els.agentStatus.innerHTML = a.available
        ? `<span class="ok">● ${a.name} ${a.version}</span>`
        : `<span class="bad">○ ${a.name} 不可用 · ${a.reason || ""}</span>`;
    }
    refreshStart();
  }

  els.scenario.value = api.getMockScenario();
  els.scenario.onchange = () => {
    api.setMockScenario(els.scenario.value);
    els.msg.textContent = `已切换场景：${els.scenario.value}`;
    refreshAgents();
  };

  root.querySelector("#btn-refresh-agents").onclick = refreshAgents;

  root.querySelector("#btn-connect-model").onclick = async () => {
    const key = els.modelKey.value;
    const name = els.modelName.value.trim();
    els.modelKey.value = "";
    if (!name || !key) {
      els.modelStatus.innerHTML = `<span class="bad">请填写显示名与 API Key</span>`;
      return;
    }
    const res = await api.connectModel({
      profileId: "demo-model",
      name,
      apiKey: key,
    });
    if (res.error) {
      ctx.state.modelAccount = null;
      els.modelStatus.innerHTML = `<span class="bad">${res.error.code}: ${res.error.message}</span>`;
    } else {
      ctx.state.modelAccount = res.account;
      els.modelStatus.innerHTML = `<span class="ok">已连接 · ${res.account.id} · ${res.account.host}</span>`;
    }
    refreshStart();
  };

  root.querySelector("#btn-connect-mcp").onclick = async () => {
    const key = els.mcpKey.value;
    const name = els.mcpName.value.trim();
    els.mcpKey.value = "";
    if (!name || !key) {
      els.mcpStatus.innerHTML = `<span class="bad">请填写显示名与 Token</span>`;
      return;
    }
    const res = await api.connectMcp({
      profileId: "demo-mcp",
      name,
      apiKey: key,
    });
    if (res.error) {
      ctx.state.mcpAccount = null;
      els.mcpStatus.innerHTML = `<span class="bad">${res.error.code}: ${res.error.message}</span>`;
    } else {
      ctx.state.mcpAccount = res.account;
      els.mcpStatus.innerHTML = `<span class="ok">已连接 · ${res.account.id} · ${res.account.host}</span>`;
    }
    refreshStart();
  };

  root.querySelector("#btn-pick-ws").onclick = async () => {
    const res = await api.selectWorkspace();
    if (res.error) {
      els.wsPath.innerHTML = `<span class="bad">${res.error.message}</span>`;
      ctx.state.workspace = "";
    } else if (res.cancelled) {
      els.wsPath.textContent = "已取消";
    } else {
      ctx.state.workspace = res.path;
      els.wsPath.textContent = res.path;
    }
    refreshStart();
  };

  els.task.oninput = refreshStart;

  els.start.onclick = async () => {
    els.msg.textContent = "创建会话…";
    const res = await api.createSession({
      workspace: ctx.state.workspace,
      agentId: ctx.state.agentId || "claude",
      modelAccountId: ctx.state.modelAccount.id,
      mcpIds: [ctx.state.mcpAccount.id],
      task: els.task.value.trim(),
    });
    if (res.error) {
      els.msg.innerHTML = `<span class="bad">${res.error.code}: ${res.error.message}</span>`;
      return;
    }
    ctx.state.sessionId = res.session.sessionId;
    ctx.onSessionCreated(res.session);
    els.msg.innerHTML = `<span class="ok">已进入 ${res.session.state}</span>`;
  };

  refreshAgents();
  return { refreshStart };
}
