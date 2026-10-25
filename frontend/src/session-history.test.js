import test from "node:test";
import assert from "node:assert/strict";
import {
  mergeHistoryRow,
  buildRestartCreateSpec,
} from "./session-history.js";

test("mergeHistoryRow：创建时记下 projectWorkspace / 模型 / 任务", () => {
  const row = mergeHistoryRow(
    null,
    {
      sessionId: "s1",
      state: "running",
      workspace: "/proj",
      agentId: "claude",
      mountMode: "clone",
      cleanupComplete: false,
    },
    {
      workspace: "/proj",
      agentId: "claude",
      modelAccountId: "m1",
      mcpIds: ["t1"],
      task: "写 README",
      mountMode: "clone",
    },
  );
  assert.equal(row.projectWorkspace, "/proj");
  assert.equal(row.modelAccountId, "m1");
  assert.equal(row.task, "写 README");
  assert.deepEqual(row.mcpIds, ["t1"]);
});

test("mergeHistoryRow：clone 后快照路径变了仍保留原工作区", () => {
  const first = mergeHistoryRow(
    null,
    {
      sessionId: "s1",
      state: "provisioning",
      workspace: "/proj",
      agentId: "claude",
      cleanupComplete: false,
    },
    { workspace: "/proj", modelAccountId: "m1", task: "t", mcpIds: [] },
  );
  const afterClone = mergeHistoryRow(first, {
    sessionId: "s1",
    state: "running",
    workspace: "/tmp/harbor-clone-s1",
    agentId: "claude",
    cleanupComplete: false,
  });
  assert.equal(afterClone.workspace, "/tmp/harbor-clone-s1");
  assert.equal(afterClone.projectWorkspace, "/proj");
  assert.equal(afterClone.modelAccountId, "m1");
  assert.equal(afterClone.task, "t");
});

test("buildRestartCreateSpec：用 projectWorkspace 再开，不走 clone 路径", () => {
  const spec = buildRestartCreateSpec({
    sessionId: "s1",
    workspace: "/tmp/clone",
    projectWorkspace: "/proj",
    agentId: "claude",
    modelAccountId: "m1",
    mcpIds: ["t1"],
    task: "继续改",
    mountMode: "clone",
  });
  assert.deepEqual(spec, {
    workspace: "/proj",
    agentId: "claude",
    modelAccountId: "m1",
    mcpIds: ["t1"],
    task: "继续改",
    mountMode: "clone",
  });
});

test("buildRestartCreateSpec：缺模型时返回 null", () => {
  assert.equal(
    buildRestartCreateSpec({
      workspace: "/proj",
      agentId: "claude",
      task: "x",
    }),
    null,
  );
});
