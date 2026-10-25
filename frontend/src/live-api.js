// Wails v3 live transport。生成绑定来自 frontend/bindings，不要手改生成文件。
// 不写扩展名，以同时兼容 wails3 的 JS 与 TS 绑定输出模式。
import * as api from "../bindings/github.com/liyufei916/agent-harbor/internal/desktop/desktopapi";
import { Events } from "@wailsio/runtime";

function unwrapEvent(ev) {
  if (ev && Object.prototype.hasOwnProperty.call(ev, "data")) {
    return ev.data;
  }
  return ev;
}

export function createLiveTransport() {
  if (!api.ListAgents || !Events || typeof Events.On !== "function") {
    throw new Error("Wails 绑定不完整");
  }
  return {
    listAgents: () => api.ListAgents(),
    selectWorkspace: () => api.SelectWorkspace(),
    connectModel: (input) => api.ConnectModel(input),
    connectMcp: (input) => api.ConnectMcp(input),
    createSession: (spec) => api.CreateSession(spec),
    getSession: (id) => api.GetSession(id),
    stopSession: (id) => api.StopSession(id),
    onEvent(callback) {
      return Events.On("harbor:event", (ev) => callback(unwrapEvent(ev)));
    },
  };
}
