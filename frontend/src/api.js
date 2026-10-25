/**
 * 页面对外 API（CONTRACT.md camelCase）
 * 默认 mock transport；Wails v3 live transport 延迟加载，不影响浏览器 mock。
 */

import * as mock from "./mock-api.js";
import { SessionEvents } from "./session-events.js";
const events = new SessionEvents();
let stopEvents;
function subscribeTransport() {
  stopEvents?.();
  stopEvents = transport.onEvent((event) => events.receive(event));
}

let transport = mock;
let mode = "mock"; // mock | live

function wrapTransportError() {
  return {
    code: "INTERNAL_ERROR",
    message: "桌面调用失败，请重试。",
    retryable: false,
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
 * 真实桌面通道。绑定由 wails3 generate 生成，不要手改。
 * 事件名固定为 harbor:event，回调收到契约 Event 本身。
 */
export async function createLiveTransport() {
  const { createLiveTransport: create } = await import("./live-api.js");
  return create();
}

export function useMock() {
  transport = mock;
  mode = "mock";
  subscribeTransport();
}

export async function useLive() {
  transport = await createLiveTransport();
  mode = "live";
  subscribeTransport();
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
export async function createSession(spec) {
  if (!events.beginCreate())
    return {
      session: null,
      error: {
        code: "SESSION_BUSY",
        message: "会话创建正在进行中。",
        retryable: false,
      },
    };
  const response = await call("createSession", spec);
  if (response.error || !response.session) events.cancelCreate();
  return response;
}
export function activateSession(snapshot) {
  events.activate(snapshot);
}
export async function getSession(sessionId) {
  const response = await call("getSession", sessionId);
  if (response.session) events.applySnapshot(response.session);
  return response;
}
export async function stopSession(sessionId) {
  const response = await call("stopSession", sessionId);
  if (response.session) events.applySnapshot(response.session);
  return response;
}

/** @returns {() => void} 取消订阅 */
export function onEvent(callback) {
  if (!stopEvents) subscribeTransport();
  return events.subscribe(callback);
}
