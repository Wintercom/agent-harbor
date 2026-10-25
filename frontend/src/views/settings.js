/**
 * PAGE-DESIGN：设置 — Agent 检测、mock 场景、关于
 */
import * as api from "../api.js";

export function mountSettings(root, ctx) {
  root.innerHTML = `
    <h2>设置</h2>
    <p class="sub">Agent 检测与演示选项。扩权/清理策略等待契约。</p>
    <section class="card">
      <h3>Agent 检测</h3>
      <p class="meta" id="settings-agent">—</p>
      <div class="actions">
        <button type="button" class="btn" id="btn-refresh-agents">重新检测</button>
      </div>
    </section>
    <section class="card ${api.isMock() ? "" : "hidden"}" style="margin-top:12px" id="settings-mock">
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
      <p class="meta">对应 D4；切换后请回模型/工具页按需重连。</p>
    </section>
    <section class="card" style="margin-top:12px">
      <h3>关于</h3>
      <p class="meta">页面壳按 PAGE-DESIGN；两天版仍单会话、固定 profile。任务拆分见 docs/TASK-SPLIT-ABC.md。</p>
      <div class="actions">
        <button type="button" class="btn" id="btn-reset-onboard">下次启动重新显示引导</button>
      </div>
    </section>
  `;

  async function refreshAgents() {
    const el = root.querySelector("#settings-agent");
    const res = await api.listAgents();
    if (res.error) {
      el.innerHTML = `<span class="bad">${res.error.code}: ${res.error.message}</span>`;
      return;
    }
    el.innerHTML = (res.agents || [])
      .map(
        (a) =>
          `${a.available ? "●" : "○"} ${a.name} ${a.version || ""} ${
            a.available ? "" : "· " + (a.reason || "")
          }`,
      )
      .join("<br/>");
    ctx.state.agents = res.agents || [];
    const cur = res.agents?.find((a) => a.id === ctx.state.agentId) || res.agents?.[0];
    if (cur) {
      ctx.state.agentId = cur.id;
      ctx.state.agentAvailable = !!cur.available;
    }
    ctx.refreshHealth?.();
  }

  root.querySelector("#btn-refresh-agents").onclick = refreshAgents;
  const sel = root.querySelector("#scenario");
  if (sel) {
    sel.value = api.getMockScenario();
    sel.onchange = () => {
      api.setMockScenario(sel.value);
      ctx.refreshHealth?.();
      refreshAgents();
    };
  }
  root.querySelector("#btn-reset-onboard").onclick = () => {
    try {
      localStorage.removeItem("harbor-onboarded");
    } catch {}
    alert("已清除。刷新页面将再次显示引导。");
  };
  refreshAgents();
}
