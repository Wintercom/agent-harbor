import test from "node:test";
import assert from "node:assert/strict";
import * as mock from "./mock-api.js";

async function connectBasics() {
  mock.setScenario("normal");
  await mock.connectModel({
    profileId: "demo-model",
    name: "fixture",
    apiKey: "fixture",
  });
  await mock.connectMcp({
    profileId: "demo-mcp",
    name: "fixture",
    apiKey: "fixture",
  });
}

test("mock 双会话：同工作区 WORKSPACE_BUSY，不同路径可开两条，第三条 SESSION_LIMIT", async () => {
  mock.resetMock();
  try {
    await connectBasics();
    const a = await mock.createSession({
      workspace: "/demo/project",
      agentId: "claude",
      modelAccountId: "mock-model",
      mcpIds: [],
      task: "one",
      mountMode: "clone",
    });
    assert.ok(a.session);
    assert.equal(a.session.mountMode, "clone");

    const same = await mock.createSession({
      workspace: "/demo/project",
      agentId: "claude",
      modelAccountId: "mock-model",
      mcpIds: [],
      task: "dup",
    });
    assert.equal(same.error?.code, "WORKSPACE_BUSY");

    const b = await mock.createSession({
      workspace: "/demo/project-b",
      agentId: "claude",
      modelAccountId: "mock-model",
      mcpIds: [],
      task: "two",
    });
    assert.ok(b.session);
    assert.notEqual(a.session.sessionId, b.session.sessionId);

    const listed = await mock.listSessions();
    assert.equal(listed.sessions.filter((s) => ["provisioning", "running"].includes(s.state) || s.state === "running").length >= 1, true);
    assert.ok(listed.sessions.length >= 2);

    const third = await mock.createSession({
      workspace: "/demo/project-c",
      agentId: "claude",
      modelAccountId: "mock-model",
      mcpIds: [],
      task: "three",
    });
    assert.equal(third.error?.code, "SESSION_LIMIT");

    // 扩权：project 写入记住表；状态保持 running（无 awaiting_permission）
    let sawPerm = false;
    const off = mock.onEvent((ev) => {
      if (ev.type === "permission_needed") sawPerm = true;
    });
    await new Promise((r) => setTimeout(r, 1200));
    off();
    assert.equal(sawPerm, true);
    assert.equal((await mock.getSession(a.session.sessionId)).session.state, "running");
    await mock.resolvePermission({
      sessionId: a.session.sessionId,
      target: "api.npmjs.org",
      decision: "project",
    });
    const hosts = await mock.listRememberedHosts(a.session.workspace);
    assert.ok(hosts.hosts.some((h) => h.host === "api.npmjs.org"));

    const pty = await mock.writePty({
      sessionId: a.session.sessionId,
      data: "hello",
    });
    assert.equal(pty.error, null);
  } finally {
    mock.resetMock();
  }
});

test("mock 多模型/多 MCP 账号可并存，auth 场景列表标无效", async () => {
  mock.resetMock();
  try {
    const a = await mock.connectModel({
      profileId: "custom",
      name: "模型甲",
      apiKey: "k1",
      origin: "https://a.example.com",
    });
    const b = await mock.connectModel({
      profileId: "custom",
      name: "模型乙",
      apiKey: "k2",
      origin: "https://b.example.com",
    });
    assert.equal(a.account.id, "mock-model");
    assert.notEqual(a.account.id, b.account.id);
    let models = await mock.listAccounts("model");
    assert.equal(models.accounts.length, 2);

    await mock.connectMcp({
      profileId: "demo-mcp",
      name: "MCP甲",
      apiKey: "t1",
    });
    await mock.connectMcp({
      profileId: "demo-mcp",
      name: "MCP乙",
      apiKey: "t2",
    });
    const mcps = await mock.listAccounts("mcp");
    assert.equal(mcps.accounts.length, 2);

    mock.setScenario("auth");
    models = await mock.listAccounts("model");
    assert.ok(models.accounts.every((x) => x.invalid));
  } finally {
    mock.resetMock();
  }
});

test("mock OAuth pending 完成后出现 MCP 账号，可取消", async () => {
  mock.resetMock();
  try {
    const start = await mock.startMcpOAuth({ name: "OAuth MCP", profileId: "demo-mcp" });
    assert.ok(start.pendingId);
    assert.ok(start.authUrl);
    await new Promise((r) => setTimeout(r, 1000));
    const list = await mock.listAccounts("mcp");
    assert.equal(list.accounts.length, 1);
    assert.equal(list.accounts[0].authMode, "oauth");

    mock.resetMock();
    const again = await mock.startMcpOAuth({ name: "X", profileId: "demo-mcp" });
    await mock.cancelMcpOAuth(again.pendingId);
    await new Promise((r) => setTimeout(r, 1000));
    const empty = await mock.listAccounts("mcp");
    assert.equal(empty.accounts.length, 0);
  } finally {
    mock.resetMock();
  }
});
