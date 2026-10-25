import test from "node:test";
import assert from "node:assert/strict";
import * as api from "./api.js";
import * as mock from "./mock-api.js";
import { mountSetup } from "./views/setup.js";

test("配置页切换 dsh 使用对应 profile 并清空旧模型，刷新保留选择", async () => {
  mock.resetMock(); mock.setScenario("normal"); api.useMock();
  const previous = globalThis.document;
  globalThis.document = { createElement() { return {}; } };
  const nodes = new Map();
  const root = {innerHTML: "", querySelector(id) {
    if (!nodes.has(id)) nodes.set(id, {value: "", textContent: "", innerHTML: "", disabled: false, querySelector() {return null;}, replaceChildren(...children) {this.children=children;}});
    return nodes.get(id);
  }};
  const ctx = {state: {agentId:"claude", modelAccount:{id:"old"}, mcpAccount:{id:"tools"}, workspace:"/demo/project"}};
  try {
    const view=mountSetup(root,ctx);
    await view.refreshAgents();
    const select=nodes.get("#agent-select");
    assert.equal(select.children.length,2);
    select.value="dsh";select.onchange();
    assert.equal(ctx.state.agentId,"dsh");
    assert.equal(ctx.state.modelAccount,null);
    assert.equal(ctx.state.mcpAccount.id,"tools");
    assert.equal(nodes.get("#model-profile").value,"dsh-model");
    nodes.get("#model-name").value="fixture";
    nodes.get("#model-key").value="fixture";
    await nodes.get("#btn-connect-model").onclick();
    assert.equal(ctx.state.modelAccount.profileId,"dsh-model");
    assert.equal(nodes.get("#model-key").value,"");
    await view.refreshAgents();
    assert.equal(select.value,"dsh");
    assert.equal(ctx.state.modelAccount.profileId,"dsh-model");
    mock.setScenario("isolation");await view.refreshAgents();
    assert.equal(ctx.state.agentAvailable,false);
  } finally {globalThis.document=previous;mock.resetMock(); mock.setScenario("normal");}
});

test("mock dsh 会话拒绝 Claude 账号并允许匹配的 dsh 账号", async () => {
  mock.resetMock(); mock.setScenario("normal");
  try {
    await mock.connectModel({profileId:"demo-model",name:"fixture",apiKey:"fixture"});
    await mock.connectMcp({profileId:"demo-mcp",name:"fixture",apiKey:"fixture"});
    const spec={agentId:"dsh",workspace:"/demo/project",task:"fixture",modelAccountId:"mock-model",mcpIds:["mock-mcp"]};
    assert.equal((await mock.createSession(spec)).error.code,"INVALID_ARGUMENT");
    await mock.connectModel({profileId:"dsh-model",name:"fixture",apiKey:"fixture"});
    const created=await mock.createSession(spec);
    assert.equal(created.session.agentId,"dsh");
    assert.equal(created.error,null);
  } finally {mock.resetMock(); mock.setScenario("normal");}
});
