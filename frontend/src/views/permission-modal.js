/**
 * PAGE-DESIGN §5.9 扩权模态
 */
import { showToast } from "../toast.js";

let escHandler = null;

export function mountPermissionModal(root, { sessionLabel, target, action, onDecide }) {
  if (!root) return;
  hidePermissionModal(root);
  root.classList.remove("hidden");
  root.innerHTML = `
    <div class="modal-card" role="dialog" aria-modal="true" aria-labelledby="perm-title">
      <h2 id="perm-title">请求访问新地址</h2>
      <p class="meta">会话「${sessionLabel || "—"}」</p>
      <p class="sub">${action || "请求访问"}</p>
      <p class="target-host">${target || ""}</p>
      <p class="meta">选错可在权限一览或设置里撤销始终允许。没有「全部网站都允许」。</p>
      <div class="actions">
        <button type="button" class="btn danger" data-d="deny">拒绝</button>
        <button type="button" class="btn" data-d="once">仅本次</button>
        <button type="button" class="btn primary" data-d="project">本项目始终允许</button>
      </div>
    </div>`;

  const labels = {
    deny: "已拒绝",
    once: "仅本次允许",
    project: "已为本项目记住允许",
  };

  root.querySelectorAll("[data-d]").forEach((btn) => {
    btn.onclick = async () => {
      root.querySelectorAll("[data-d]").forEach((b) => {
        b.disabled = true;
      });
      const decision = btn.getAttribute("data-d");
      try {
        await onDecide(decision);
        showToast(labels[decision] || "已处理", "ok");
      } catch (_) {
        showToast("处理失败，请重试", "bad");
        root.querySelectorAll("[data-d]").forEach((b) => {
          b.disabled = false;
        });
      }
    };
  });

  escHandler = (e) => {
    if (e.key === "Escape") {
      /* 扩权必须显式选择，Esc 不关闭 */
      e.preventDefault();
    }
  };
  window.addEventListener("keydown", escHandler);
}

export function hidePermissionModal(root) {
  if (escHandler) {
    window.removeEventListener("keydown", escHandler);
    escHandler = null;
  }
  if (!root) return;
  root.classList.add("hidden");
  root.innerHTML = "";
}
