/**
 * 页面对外 API（CONTRACT.md camelCase）
 * 默认 mock transport；B 生成 Wails 绑定后只替换 createLiveTransport()。
 */

import * as mock from "./mock-api.js";

let transport = mock;
let mode = "mock"; // mock | live

function wrapTransportError(e) {
  return {
    code: "INTERNAL_ERROR",
    message: e && e.message ? String(e.message) : "前端调用失败",
    retryable: true,
  };
}

async function call(method, ...args) {
  try {
    const result = await transport[method](...args);
    return result;
  } catch (e) {
    if (method === "listAgents") {
      return { agents: [], error: wrapTransportError(e) };
    }
    if (method === "selectWorkspace") {
      return { path: "", cancelled: false, error: wrapTransportError(e) };
    }
    if (method === "connectModel" || method === "connectMcp") {
      return { account: null, error: wrapTransportError(e) };
    }
    return { session: null, error: wrapTransportError(e) };
  }
}

const LIVE_BINDING =
  "../bindings/github.com/liyufei916/agent-harbor/internal/desktop/desktopapi.js";

function unwrapEvent(ev) {
  if (ev && Object.prototype.hasOwnProperty.call(ev, "data")) {
    return ev.data;
  }
  return ev;
}

/**
 * 真实桌面通道。绑定由 wails3 generate 生成，不要手改。
 * 事件名固定为 harbor:event，回调收到契约 Event 本身。
 */
export async function createLiveTransport() {
  const api = await import(/* @vite-ignore */ LIVE_BINDING);
  const runtime = await import("@wailsio/runtime");
  const Events = runtime.Events;
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
      return Events.On("harbor:event", (ev) => {
        callback(unwrapEvent(ev));
      });
    },
  };
}

export function useMock() {
  transport = mock;
  mode = "mock";
}

export async function useLive() {
  transport = await createLiveTransport();
  mode = "live";
}

export function getMode() {
  return mode;
}

export function isMock() {
  return mode === "mock";
}

export function setMockScenario(name) {
  if (typeof mock.setScenario === "function") mock.setScenario(name);
}

export function getMockScenario() {
  return typeof mock.getScenario === "function" ? mock.getScenario() : "normal";
}

export function listAgents() {
  return call("listAgents");
}
export function selectWorkspace() {
  return call("selectWorkspace");
}
export function connectModel(input) {
  return call("connectModel", input);
}
export function connectMcp(input) {
  return call("connectMcp", input);
}
export function createSession(spec) {
  return call("createSession", spec);
}
export function getSession(sessionId) {
  return call("getSession", sessionId);
}
export function stopSession(sessionId) {
  return call("stopSession", sessionId);
}

/** @returns {() => void} 取消订阅 */
export function onEvent(callback) {
  return transport.onEvent(callback);
}
