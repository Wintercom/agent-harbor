/**
 * PAGE-DESIGN §5.10：隔离不可用阻断层
 * 全页说明；无「仍要继续」；与侧栏红灯一致。
 */
export function mountBlock(root, { reason, checks, onRecheck, onSettings }) {
  const items =
    Array.isArray(checks) && checks.length
      ? checks
      : [
          {
            ok: false,
            name: "隔离环境",
            detail: reason || "当前无法建立隔离环境",
          },
          {
            ok: false,
            name: "安全闸口",
            detail: "依赖隔离就绪后才能开新会话",
          },
        ];

  root.innerHTML = `
    <div class="block-card" role="alertdialog" aria-labelledby="block-title" aria-describedby="block-desc">
      <h1 id="block-title">无法安全启动</h1>
      <p id="block-desc" class="sub">这台电脑当前无法建立隔离环境。Agent Harbor 不会在无隔离时启动 Agent，也没有「仍要继续」。</p>
      <ul class="block-checks">
        ${items
          .map(
            (c) => `
          <li class="${c.ok ? "ok" : "bad"}">
            <span class="block-check-mark" aria-hidden="true">${c.ok ? "●" : "○"}</span>
            <span>
              <strong>${c.name || "检测项"}</strong>
              <span class="meta">${c.detail || (c.ok ? "通过" : "未通过")}</span>
            </span>
          </li>`,
          )
          .join("")}
      </ul>
      <p class="meta bad">${reason || "ISOLATION_UNAVAILABLE"}</p>
      <div class="actions">
        <button type="button" class="btn primary" id="btn-block-recheck">重新检测</button>
        <button type="button" class="btn" id="btn-block-settings">前往设置</button>
      </div>
    </div>
  `;

  const recheck = root.querySelector("#btn-block-recheck");
  recheck.onclick = async () => {
    recheck.disabled = true;
    recheck.textContent = "检测中…";
    try {
      await onRecheck?.();
    } finally {
      recheck.disabled = false;
      recheck.textContent = "重新检测";
    }
  };
  root.querySelector("#btn-block-settings").onclick = () => onSettings?.();
}
