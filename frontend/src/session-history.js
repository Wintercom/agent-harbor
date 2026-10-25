/**
 * HOME 会话历史行：保留再次开始所需的原工作区与创建规格。
 */

/** @param {object|null} prev @param {object} snapshot @param {object|null|undefined} createSpec @param {string} [lastTask] */
export function mergeHistoryRow(prev, snapshot, createSpec, lastTask = "") {
  if (!snapshot?.sessionId) return null;
  const projectWorkspace =
    createSpec?.workspace ||
    prev?.projectWorkspace ||
    snapshot.sourceWorkspace ||
    snapshot.projectWorkspace ||
    (prev?.workspace &&
    snapshot.workspace &&
    snapshot.workspace !== prev.workspace
      ? prev.projectWorkspace || prev.workspace
      : snapshot.workspace);
  return {
    ...(prev || {}),
    sessionId: snapshot.sessionId,
    state: snapshot.state,
    workspace: snapshot.workspace,
    projectWorkspace,
    agentId: snapshot.agentId || prev?.agentId || createSpec?.agentId,
    cleanupComplete: snapshot.cleanupComplete,
    mountMode:
      snapshot.mountMode ||
      createSpec?.mountMode ||
      prev?.mountMode ||
      "clone",
    mcpIds: snapshot.mcpIds || createSpec?.mcpIds || prev?.mcpIds || [],
    modelAccountId:
      snapshot.modelAccountId ||
      createSpec?.modelAccountId ||
      prev?.modelAccountId ||
      "",
    task: createSpec?.task || prev?.task || lastTask || "",
  };
}

/**
 * 从历史行拼 CreateSession 规格；缺关键字段时返回 null（应回向导）。
 * @param {object} prev
 * @param {{ modelAccountId?: string, lastTask?: string }} [fallbacks]
 */
export function buildRestartCreateSpec(prev, fallbacks = {}) {
  if (!prev) return null;
  const workspace = prev.projectWorkspace || prev.workspace;
  const modelAccountId =
    prev.modelAccountId || fallbacks.modelAccountId || "";
  const task =
    prev.task ||
    fallbacks.lastTask ||
    "继续。请根据工作区当前内容推进任务并写一行摘要。";
  if (!workspace || !prev.agentId || !modelAccountId) return null;
  return {
    workspace,
    agentId: prev.agentId,
    modelAccountId,
    mcpIds: Array.isArray(prev.mcpIds) ? [...prev.mcpIds] : [],
    task,
    mountMode: prev.mountMode === "direct" ? "direct" : "clone",
  };
}
