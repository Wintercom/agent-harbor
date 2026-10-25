/**
 * 应用内确认框（Wails/WebView 上 window.confirm 常无响应）。
 * @returns {Promise<boolean>}
 */
export function askConfirm(
  {
    title = "确认",
    body = "",
    confirmLabel = "确定",
    cancelLabel = "取消",
    danger = false,
  } = {},
  root = typeof document !== "undefined"
    ? document.getElementById("modal-root")
    : null,
) {
  if (!root) {
    // 无挂载点时退回原生（浏览器 mock）
    try {
      return Promise.resolve(window.confirm([title, body].filter(Boolean).join("\n\n")));
    } catch {
      return Promise.resolve(false);
    }
  }
  return new Promise((resolve) => {
    const prev = root.innerHTML;
    const wasHidden = root.classList.contains("hidden");
    root.classList.remove("hidden");
    root.innerHTML = `
      <div class="modal-card" role="dialog" aria-modal="true" aria-labelledby="confirm-title">
        <h2 id="confirm-title">${escapeHtml(title)}</h2>
        <p class="meta" style="white-space:pre-wrap">${escapeHtml(body)}</p>
        <div class="actions">
          <button type="button" class="btn" id="confirm-cancel">${escapeHtml(cancelLabel)}</button>
          <button type="button" class="btn ${danger ? "danger" : "primary"}" id="confirm-ok">${escapeHtml(confirmLabel)}</button>
        </div>
      </div>`;

    const finish = (ok) => {
      window.removeEventListener("keydown", onKey);
      if (wasHidden) root.classList.add("hidden");
      root.innerHTML = prev;
      resolve(ok);
    };
    const onKey = (e) => {
      if (e.key === "Escape") {
        e.preventDefault();
        finish(false);
      }
    };
    window.addEventListener("keydown", onKey);
    root.querySelector("#confirm-cancel").onclick = () => finish(false);
    root.querySelector("#confirm-ok").onclick = () => finish(true);
  });
}

function escapeHtml(s) {
  return String(s || "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}
