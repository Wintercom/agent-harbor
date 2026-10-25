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
    modelProfileId: "demo-model",
  },
  modelProfileId: "demo-model",
  mcpProfileId: "demo-mcp",
  workspace: "/demo/project",
};

let scenario = "normal";
let seq = 0;
let sessionNumber = 0;
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
    permission: partial.permission || null,
    pty: partial.pty || null,
  };
  if (session && event.sessionId === session.sessionId) session.seq = event.seq;
  if (event.state) event.state = { ...event.state, seq: event.seq };
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
  const agents = [
    { ...FIXED.agent },
    { id: "dsh", name: "DeepSeek dsh", version: "0.2.1-rc.2", available: true, reason: "", modelProfileId: "dsh-model" },
  ];
  if (scenario === "isolation") {
    return {
      agents: agents.map((agent) => ({...agent, available: false, reason: "隔离能力不可用（模拟）"})),
      error: null,
    };
  }
  return { agents, error: null };
}

/** CONTRACT v1.2 GetRuntimeHealth */
export async function getRuntimeHealth() {
  if (scenario === "isolation") {
    return {
      available: false,
      reason: "隔离能力不可用（模拟）",
      isolationLevel: "",
      error: null,
    };
  }
  return {
    available: true,
    reason: "",
    isolationLevel: "mock-seatbelt",
    error: null,
  };
}

/** CONTRACT v1.2 ListSessions（mock 单会话表） */
export async function listSessions() {
  if (!session) return { sessions: [], error: null };
  return {
    sessions: [
      {
        sessionId: session.sessionId,
        state: session.state,
        workspace: session.workspace,
        agentId: session.agentId,
        cleanupComplete: session.cleanupComplete,
      },
    ],
    error: null,
  };
}

export async function destroySession(sessionId) {
  if (!session || session.sessionId !== sessionId) {
    return {
      session: null,
      error: err("SESSION_NOT_FOUND", "会话不存在（模拟）"),
    };
  }
  if (!["stopped", "failed"].includes(session.state) || !session.cleanupComplete) {
    return {
      session: snapshot(),
      error: err("INVALID_ARGUMENT", "仅终态且清理完成的会话可销毁（模拟）"),
    };
  }
  const last = snapshot();
  session = null;
  busy = false;
  return { session: last, error: null };
}

export async function listAccounts(kind) {
  const accounts = [];
  if ((!kind || kind === "model") && modelAccount) accounts.push({ ...modelAccount });
  if ((!kind || kind === "mcp") && mcpAccount) accounts.push({ ...mcpAccount });
  return { accounts, error: null };
}

export async function removeAccount(accountId) {
  if (modelAccount?.id === accountId) {
    if (session && !["stopped", "failed"].includes(session.state)) {
      return { error: err("SESSION_BUSY", "会话仍引用该账号（模拟）") };
    }
    modelAccount = null;
    return { error: null };
  }
  if (mcpAccount?.id === accountId) {
    if (session && !["stopped", "failed"].includes(session.state)) {
      return { error: err("SESSION_BUSY", "会话仍引用该账号（模拟）") };
    }
    mcpAccount = null;
    return { error: null };
  }
  return { error: err("ACCOUNT_NOT_FOUND", "账号不存在（模拟）") };
}

export async function getPermissionOverview(sessionId) {
  return {
    overview: {
      sessionId: sessionId || session?.sessionId || "",
      workspace: session?.workspace || FIXED.workspace,
      allowedHosts: ["api.example.com"],
      mcpSummary: mcpAccount ? [mcpAccount.name] : [],
      deniedPathCategories: ["系统目录"],
    },
    error: null,
  };
}

export async function listRememberedHosts() {
  return { hosts: [], error: null };
}

export async function revokeRememberedHost() {
  return { error: null };
}

export async function resolvePermission(input) {
  if (!input?.sessionId || !input?.target || !input?.decision) {
    return { error: err("INVALID_ARGUMENT", "扩权参数不完整（模拟）") };
  }
  if (!["deny", "once", "project"].includes(input.decision)) {
    return { error: err("INVALID_ARGUMENT", "非法决策（模拟）") };
  }
  return { error: null };
}

export async function getUIPrefs() {
  return {
    prefs: { sidebarCollapsed: false, autoCollapseWorkbench: true },
    error: null,
  };
}

export async function setUIPrefs(prefs) {
  return {
    prefs: {
      sidebarCollapsed: !!prefs?.sidebarCollapsed,
      autoCollapseWorkbench: prefs?.autoCollapseWorkbench !== false,
    },
    error: null,
  };
}

export async function selectWorkspace() {
  return { path: FIXED.workspace, cancelled: false, error: null };
}

export async function connectModel(input) {
  if (scenario === "auth") {
    return {
      account: null,
      error: err("UPSTREAM_AUTH_FAILED", "模型认证失败（模拟）", false),
    };
  }
  if (!input?.name || !input?.apiKey) {
    return {
      account: null,
      error: err("INVALID_ARGUMENT", "name、apiKey 均必填"),
    };
  }
  const profileId = input.profileId || "custom";
  const allowed = [FIXED.modelProfileId, "dsh-model", "custom"];
  if (!allowed.includes(profileId)) {
    return {
      account: null,
      error: err("INVALID_ARGUMENT", "模型 profile 不受支持"),
    };
  }
  if (profileId === "custom" && !input.origin) {
    return {
      account: null,
      error: err("INVALID_ARGUMENT", "自定义模型须提供 origin"),
    };
  }
  if (busy || (session && !["stopped", "failed"].includes(session.state))) {
    return {
      account: null,
      error: err("SESSION_BUSY", "会话进行中，拒绝重新连接"),
    };
  }
  let host = "demo-model.local";
  if (input.origin) {
    try {
      host = new URL(input.origin).host;
    } catch {
      return {
        account: null,
        error: err("INVALID_ARGUMENT", "origin 非法"),
      };
    }
  }
  modelAccount = {
    id: "mock-model",
    kind: "model",
    name: input.name,
    profileId,
    host,
    origin: input.origin || "",
  };
  return { account: { ...modelAccount }, error: null };
}

export async function startMcpOAuth(input) {
  const pendingId = `oauth-${Date.now()}`;
  schedule(400, () => {
    mcpAccount = {
      id: "mock-mcp",
      kind: "mcp",
      name: input?.name || "OAuth MCP",
      profileId: input?.profileId || FIXED.mcpProfileId,
      host: "oauth.mcp.local",
      authMode: "oauth",
    };
  });
  return {
    authUrl: "https://example.com/oauth/mock",
    pendingId,
    error: null,
  };
}

export async function cancelMcpOAuth() {
  return { error: null };
}

export async function switchModel(input) {
  if (!session || session.sessionId !== input?.sessionId) {
    return {
      session: null,
      error: err("SESSION_NOT_FOUND", "会话不存在（模拟）"),
    };
  }
  if (!modelAccount || modelAccount.id !== input.modelAccountId) {
    return {
      session: null,
      error: err("ACCOUNT_NOT_FOUND", "模型账号不存在（模拟）"),
    };
  }
  return { session: snapshot(), error: null };
}

export async function writePty(input) {
  if (!session || session.sessionId !== input?.sessionId) {
    return { error: err("INVALID_ARGUMENT", "无可用 PTY（模拟）") };
  }
  emit({
    type: "pty",
    sessionId: session.sessionId,
    pty: { stream: "stdout", text: `[pty echo] ${input.data || ""}` },
  });
  return { error: null };
}

export async function resizePty() {
  return { error: null };
}

export async function connectMcp(input) {
  if (scenario === "auth") {
    return {
      account: null,
      error: err("UPSTREAM_AUTH_FAILED", "MCP 认证失败（模拟）", false),
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
  if (
    !spec?.workspace ||
    !spec?.agentId ||
    !spec?.modelAccountId ||
    !spec?.task
  ) {
    return {
      session: null,
      error: err(
        "INVALID_ARGUMENT",
        "workspace、agentId、modelAccountId、task 必填",
      ),
    };
  }
  if (!Array.isArray(spec.mcpIds) || spec.mcpIds.length > 8) {
    return {
      session: null,
      error: err("INVALID_ARGUMENT", "mcpIds 长度须为 0..8"),
    };
  }
  if (!modelAccount || spec.modelAccountId !== modelAccount.id) {
    return {
      session: null,
      error: err("INVALID_ARGUMENT", "请先连接模型账号"),
    };
  }
  const expected =
    spec.agentId === "dsh" ? "dsh-model" : "demo-model";
  if (
    modelAccount.profileId !== "custom" &&
    modelAccount.profileId !== expected
  ) {
    return {
      session: null,
      error: err("INVALID_ARGUMENT", "Agent 与模型 profile 不匹配"),
    };
  }
  if (spec.mcpIds.length > 0) {
    if (!mcpAccount || !spec.mcpIds.every((id) => id === mcpAccount.id)) {
      return {
        session: null,
        error: err("INVALID_ARGUMENT", "请先连接 MCP 账号，且 mcpIds 须匹配"),
      };
    }
  }
  if (![FIXED.agent.id, "dsh"].includes(spec.agentId)) {
    return {
      session: null,
      error: err("INVALID_ARGUMENT", "Agent 不受支持"),
    };
  }

  clearTimers();
  busy = true;
  const sessionId = `sess-mock-${++sessionNumber}`;
  seq = 0;
  session = {
    sessionId,
    state: "provisioning",
    mountMode: spec.mountMode === "clone" ? "clone" : "direct",
    mcpIds: [...(spec.mcpIds || [])],
    seq: 0,
    workspace: spec.workspace,
    agentId: spec.agentId,
    cleanupComplete: false,
    error: null,
  };
  emit({ type: "state_changed", state: snapshot() });
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
  if (
    session.state === "stopped" ||
    (session.state === "failed" && session.cleanupComplete)
  ) {
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
        error: err("CLEANUP_FAILED", "清理未确认（模拟）", false),
      });
      emit({ type: "state_changed", state: snapshot() });
      emit({
        type: "error",
        error: err("CLEANUP_FAILED", "清理未确认，锁仍占用（模拟）", false),
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
  if (["stopped", "stopping"].includes(session.state)) {
    return { session: snapshot(), error: null };
  }
  if (session.state === "failed" && session.cleanupComplete) {
    return { session: snapshot(), error: null };
  }
  // cleanup 失败场景：再次 Stop 视为重试清理成功
  if (
    session.state === "failed" &&
    !session.cleanupComplete &&
    scenario === "cleanup"
  ) {
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
