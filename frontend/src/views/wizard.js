/**
 * PAGE-DESIGN §5.3 新会话 — 首期复用配置流（工作区/Agent/任务 + 仍可在此连账号）
 * 四步进条视觉先占位；完整拆步待交互打磨。
 */
import { mountSetup } from "./setup.js";

export function mountWizard(root, ctx) {
  root.innerHTML = `
    <h2>新会话</h2>
    <p class="sub">步骤：工作区 → Agent → 模型 → 工具（可先在侧栏「模型/工具」页连好）。当前为合并配置流，与 PAGE-DESIGN 四步同契约。</p>
    <ol class="wizard-steps">
      <li class="on">工作区</li>
      <li class="on">Agent</li>
      <li class="on">模型</li>
      <li class="on">工具 / 任务</li>
    </ol>
    <div id="wizard-setup"></div>
    <div class="actions">
      <button type="button" class="btn" id="btn-back-home">返回 HOME</button>
    </div>
  `;
  const setupMount = mountSetup(root.querySelector("#wizard-setup"), ctx, {
    embedded: true,
  });
  root.querySelector("#btn-back-home").onclick = () => ctx.navigate("home");
  return setupMount;
}
