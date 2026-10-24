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

/**
 * B 接入：Wails v3 生成目录为 frontend/bindings/（勿手改）。
 * 生成后在此 import DesktopAPI，并用 EventsOn('harbor:event', ...) 转发。
 */
export function createLiveTransport() {
  throw new Error(
    "请接入 frontend/bindings/.../desktopapi.js 后实现 createLiveTransport()；当前请用 mock 或等 B 生成绑定"
  );
}

export function useMock() {
  transport = mock;
  mode = "mock";
}

export function useLive() {
  transport = createLiveTransport();
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
