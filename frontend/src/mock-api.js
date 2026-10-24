/**
 * 浏览器独立 mock transport（CONTRACT.md / MODULE-DESIGN §5）
 * 固定数据：claude / mock-model+demo-model / mock-mcp+demo-mcp / /demo/project
 * 场景：normal | auth | busy | isolation | cleanup
 * 不导入 wailsjs；不含真实 Key。
 */

const FIXED = {
  agent: {
    id: "claude",
    name: "Claude Code",
    version: "2.1.292",
    available: true,
    reason: "",
  },
  modelProfileId: "demo-model",
  mcpProfileId: "demo-mcp",
  workspace: "/demo/project",
};

let scenario = "normal";
let seq = 0;
let session = null;
let modelAccount = null;
let mcpAccount = null;
let busy = false;
let timers = [];
const listeners = new Set();

function err(code, message, retryable = false) {
  return { code, message, retryable };
}

function nextSeq() {
  seq += 1;
  return seq;
}

function clearTimers() {
  timers.forEach((t) => clearTimeout(t));
  timers = [];
}

function emit(partial) {
  const event = {
    sessionId: partial.sessionId || (session && session.sessionId) || "",
    seq: nextSeq(),
    type: partial.type,
    state: partial.state || null,
    output: partial.output || null,
    audit: partial.audit || null,
    error: partial.error || null,
  };
  listeners.forEach((cb) => {
    try {
      cb(event);
    } catch (_) {
      /* UI 错误不打断 mock */
    }
  });
  return event;
}

function snapshot(extra = {}) {
  if (!session) return null;
  session = { ...session, ...extra, seq };
  return { ...session, error: session.error ? { ...session.error } : null };
}

function schedule(ms, fn) {
  const id = setTimeout(fn, ms);
  timers.push(id);
  return id;
}

export function setScenario(name) {
  scenario = name || "normal";
}

export function getScenario() {
  return scenario;
}

export function isMockTransport() {
  return true;
}

export async function listAgents() {
  if (scenario === "isolation") {
    return {
      agents: [
        {
          ...FIXED.agent,
          available: false,
          reason: "隔离能力不可用（模拟）",
        },
      ],
      error: null,
    };
  }
  return { agents: [{ ...FIXED.agent }], error: null };
}

export async function selectWorkspace() {
  return { path: FIXED.workspace, cancelled: false, error: null };
}

export async function connectModel(input) {
  if (scenario === "auth") {
    return {
      account: null,
      error: err("AUTH_FAILED", "模型认证失败（模拟）", true),
    };
  }
  if (!input?.profileId || !input?.name || !input?.apiKey) {
    return {
      account: null,
      error: err("INVALID_ARGUMENT", "profileId、name、apiKey 均必填"),
    };
  }
  if (input.profileId !== FIXED.modelProfileId) {
    return {
      account: null,
      error: err("INVALID_ARGUMENT", "仅支持固定 profileId=demo-model"),
    };
  }
  if (busy || (session && !["stopped", "failed"].includes(session.state))) {
    return {
      account: null,
      error: err("SESSION_BUSY", "会话进行中，拒绝重新连接"),
    };
  }
  modelAccount = {
    id: "mock-model",
    kind: "model",
    name: input.name,
    profileId: input.profileId,
    host: "demo-model.local",
  };
  return { account: { ...modelAccount }, error: null };
}

export async function connectMcp(input) {
  if (scenario === "auth") {
    return {
      account: null,
      error: err("AUTH_FAILED", "MCP 认证失败（模拟）", true),
    };
  }
  if (!input?.profileId || !input?.name || !input?.apiKey) {
    return {
      account: null,
      error: err("INVALID_ARGUMENT", "profileId、name、apiKey 均必填"),
    };
  }
  if (input.profileId !== FIXED.mcpProfileId) {
    return {
      account: null,
      error: err("INVALID_ARGUMENT", "仅支持固定 profileId=demo-mcp"),
    };
  }
  if (busy || (session && !["stopped", "failed"].includes(session.state))) {
    return {
      account: null,
      error: err("SESSION_BUSY", "会话进行中，拒绝重新连接"),
    };
  }
  mcpAccount = {
    id: "mock-mcp",
    kind: "mcp",
    name: input.name,
    profileId: input.profileId,
    host: "demo-mcp.local",
  };
  return { account: { ...mcpAccount }, error: null };
}

export async function createSession(spec) {
  if (scenario === "isolation") {
    return {
      session: null,
      error: err("ISOLATION_UNAVAILABLE", "当前无法建立隔离环境（模拟）"),
    };
  }
  if (scenario === "busy" || busy) {
    return {
      session: null,
      error: err("SESSION_BUSY", "已有会话占用（模拟）"),
    };
  }
  if (!spec?.workspace || !spec?.agentId || !spec?.modelAccountId || !spec?.task) {
    return {
      session: null,
      error: err("INVALID_ARGUMENT", "workspace、agentId、modelAccountId、task 必填"),
    };
  }
  if (!Array.isArray(spec.mcpIds) || spec.mcpIds.length !== 1) {
    return {
      session: null,
      error: err("INVALID_ARGUMENT", "mcpIds 长度必须为 1"),
    };
  }
  if (!modelAccount || spec.modelAccountId !== modelAccount.id) {
    return {
      session: null,
      error: err("INVALID_ARGUMENT", "请先连接模型账号"),
    };
  }
  if (!mcpAccount || spec.mcpIds[0] !== mcpAccount.id) {
    return {
      session: null,
      error: err("INVALID_ARGUMENT", "请先连接 MCP 账号"),
    };
  }
  if (spec.agentId !== FIXED.agent.id) {
    return {
      session: null,
      error: err("INVALID_ARGUMENT", "仅支持 agentId=claude"),
    };
  }

  clearTimers();
  busy = true;
  const sessionId = "sess-mock-1";
  session = {
    sessionId,
    state: "provisioning",
    seq: 0,
    workspace: spec.workspace,
    agentId: spec.agentId,
    cleanupComplete: false,
    error: null,
  };
  const first = snapshot();
  schedule(200, () => {
    if (!session || session.sessionId !== sessionId) return;
    if (session.state === "stopping") return;
    snapshot({ state: "running" });
    emit({ type: "state_changed", state: snapshot() });
    emit({
      type: "output",
      output: { stream: "stdout", text: "（模拟）开始执行任务…\n" },
    });
  });
  schedule(700, () => {
    if (!session || session.state !== "running") return;
    emit({
      type: "output",
      output: {
        stream: "stdout",
        text: "（模拟）调用模型完成摘要。\n",
      },
    });
    emit({
      type: "audit",
      audit: {
        target: "demo-mcp.local",
        tool: "search",
        result: "allowed",
      },
    });
  });
  schedule(1400, () => {
    if (!session || session.state !== "running") return;
    emit({
      type: "output",
      output: {
        stream: "stdout",
        text: "（模拟）已写入工作区示例文件。\n任务完成。\n",
      },
    });
    beginStop(false);
  });

  return { session: first, error: null };
}

function beginStop(fromUser) {
  if (!session) return;
  if (session.state === "stopped" || (session.state === "failed" && session.cleanupComplete)) {
    return;
  }
  clearTimers();
  snapshot({ state: "stopping", cleanupComplete: false });
  emit({ type: "state_changed", state: snapshot() });
  schedule(400, () => {
    if (!session) return;
    if (scenario === "cleanup" && fromUser) {
      snapshot({
        state: "failed",
        cleanupComplete: false,
        error: err("CLEANUP_FAILED", "清理未确认（模拟）", true),
      });
      emit({ type: "state_changed", state: snapshot() });
      emit({
        type: "error",
        error: err("CLEANUP_FAILED", "清理未确认，锁仍占用（模拟）", true),
      });
      // 保留 busy，允许 Stop 重试
      return;
    }
    snapshot({
      state: "stopped",
      cleanupComplete: true,
      error: null,
    });
    emit({ type: "state_changed", state: snapshot() });
    busy = false;
  });
}

export async function getSession(sessionId) {
  if (!session || session.sessionId !== sessionId) {
    return {
      session: null,
      error: err("SESSION_NOT_FOUND", "未知会话"),
    };
  }
  return { session: snapshot(), error: null };
}

export async function stopSession(sessionId) {
  if (!session || session.sessionId !== sessionId) {
    return {
      session: null,
      error: err("SESSION_NOT_FOUND", "未知会话"),
    };
  }
  if (session.state === "stopped") {
    return { session: snapshot(), error: null };
  }
  if (session.state === "failed" && session.cleanupComplete) {
    return { session: snapshot(), error: null };
  }
  // cleanup 失败场景：再次 Stop 视为重试清理成功
  if (session.state === "failed" && !session.cleanupComplete && scenario === "cleanup") {
    snapshot({ state: "stopping", error: null, cleanupComplete: false });
    emit({ type: "state_changed", state: snapshot() });
    schedule(300, () => {
      snapshot({ state: "stopped", cleanupComplete: true, error: null });
      emit({ type: "state_changed", state: snapshot() });
      busy = false;
    });
    return { session: snapshot(), error: null };
  }
  beginStop(true);
  return { session: snapshot(), error: null };
}

export function onEvent(callback) {
  listeners.add(callback);
  return () => listeners.delete(callback);
}

export function resetMock() {
  clearTimers();
  listeners.clear();
  seq = 0;
  session = null;
  modelAccount = null;
  mcpAccount = null;
  busy = false;
}
