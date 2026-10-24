/**
 * 页面装配入口（产品 C）。
 * 默认 mock；Wails v3 绑定目录为 frontend/bindings/（B 生成，勿手改）。
 */
import * as api from "./api.js";
import { mountSetup } from "./views/setup.js";
import { mountSession } from "./views/session.js";

const badge = document.getElementById("mode-badge");
const setupRoot = document.getElementById("view-setup");
const sessionRoot = document.getElementById("view-session");

const ctx = {
  state: {
    agentId: "claude",
    agentAvailable: false,
    modelAccount: null,
    mcpAccount: null,
    workspace: "",
    sessionId: "",
    lastSnapshot: null,
  },
  showTab(name) {
    document.querySelectorAll(".tab").forEach((t) => {
      t.classList.toggle("on", t.getAttribute("data-tab") === name);
    });
    setupRoot.classList.toggle("hidden", name !== "setup");
    sessionRoot.classList.toggle("hidden", name !== "session");
  },
  onSessionCreated(snapshot) {
    ctx.state.sessionId = snapshot.sessionId;
    ctx.state.lastSnapshot = snapshot;
    sessionView.resetForNewSession(snapshot);
    ctx.showTab("session");
  },
};

let sessionView;

function syncBadge() {
  if (api.isMock()) {
    badge.textContent = "模拟模式";
    badge.className = "badge mock";
  } else {
    badge.textContent = "真实绑定";
    badge.className = "badge live";
  }
}

function isWailsShell() {
  if (location.protocol === "wails:" || location.hostname === "wails.localhost") {
    return true;
  }
  return typeof window !== "undefined" && !!window._wails;
}

async function boot() {
  api.useMock();
  const params = new URLSearchParams(location.search);
  if (params.get("scenario")) {
    api.setMockScenario(params.get("scenario"));
  }
  const wantLive = params.get("live") === "1" || (params.get("live") !== "0" && !params.get("scenario") && isWailsShell());
  if (wantLive) {
    try {
      await api.useLive();
    } catch (e) {
      console.warn("live transport 不可用，保持 mock:", e.message || e);
      api.useMock();
    }
  }
  syncBadge();

  document.querySelectorAll(".tab").forEach((btn) => {
    btn.addEventListener("click", () => ctx.showTab(btn.getAttribute("data-tab")));
  });

  mountSetup(setupRoot, ctx);
  sessionView = mountSession(sessionRoot, ctx);
  ctx.showTab("setup");
}

boot();
