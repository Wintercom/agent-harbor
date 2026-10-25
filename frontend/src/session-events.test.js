import test from "node:test";
import assert from "node:assert/strict";
import { SessionEvents, OutputTail } from "./session-events.js";
const snapshot = (state, seq, id = "s") => ({
  sessionId: id,
  state,
  seq,
  cleanupComplete: state === "stopped",
});
const event = (type, seq, data, id = "s") => ({
  sessionId: id,
  seq,
  type,
  [type === "state_changed" ? "state" : type]: data,
});

test("早到事件在激活后按 seq 合并，快照不能吞输出", () => {
  const events = new SessionEvents();
  const got = [];
  events.subscribe((e) => got.push(e));
  events.beginCreate();
  events.receive(event("output", 2, { text: "早到" }));
  events.receive(event("state_changed", 3, snapshot("stopped", 3)));
  assert.equal(got.length, 0);
  events.activate(snapshot("running", 4));
  assert.deepEqual(
    got.filter((e) => e.output).map((e) => e.output.text),
    ["早到"],
  );
  assert.equal(got.filter((e) => e.state).at(-1).state.state, "running");
});
test("旧快照不覆盖新状态、输出去重且不被快照水位过滤", () => {
  const events = new SessionEvents();
  const got = [];
  events.subscribe((e) => got.push(e));
  events.activate(snapshot("running", 1));
  events.receive(event("state_changed", 10, snapshot("stopped", 10)));
  events.applySnapshot(snapshot("running", 2));
  events.receive(event("output", 9, { text: "最终输出" }));
  events.receive(event("output", 9, { text: "最终输出" }));
  assert.equal(got.filter((e) => e.state).at(-1).state.state, "stopped");
  assert.equal(got.filter((e) => e.output).length, 1);
});
test("第二次创建能接收新会话早到事件，旧会话不污染", () => {
  const events = new SessionEvents();
  const got = [];
  events.subscribe((e) => got.push(e));
  events.activate(snapshot("stopped", 5, "old"));
  events.beginCreate();
  events.receive(event("output", 2, { text: "新输出" }, "new"));
  events.receive(event("output", 6, { text: "旧输出" }, "old"));
  events.activate(snapshot("provisioning", 1, "new"));
  assert.deepEqual(
    got.filter((e) => e.output).map((e) => e.output.text),
    ["新输出"],
  );
});
test("取消创建保留原会话，取消订阅有效", () => {
  const events = new SessionEvents();
  let count = 0;
  const off = events.subscribe(() => count++);
  events.activate(snapshot("running", 1));
  events.beginCreate();
  events.cancelCreate();
  events.receive(event("output", 2, { text: "继续" }));
  assert.equal(count, 2);
  off();
  events.receive(event("output", 3, { text: "停止监听" }));
  assert.equal(count, 2);
});
test("输出按序去重并按 UTF-8 字节保留最新尾部", () => {
  const tail = new OutputTail(12);
  tail.append(2, "中文");
  tail.append(1, "旧内容");
  tail.append(2, "中文");
  tail.append(3, "NEW");
  assert.ok(new TextEncoder().encode(tail.text).length <= 12);
  assert.ok(tail.text.endsWith("中文NEW"));
  assert.equal(tail.truncated, true);
  assert.equal(tail.text.includes("�"), false);
});

test("早到缓冲截断保留合法水位、最新输出及终态", () => {
  const events = new SessionEvents();
  const got = [];
  events.subscribe((event) => got.push(event));
  events.beginCreate();
  for (let seq = 1; seq <= 70; seq++)
    events.receive(event("output", seq, { text: "x".repeat(16 * 1024) }));
  events.receive(event("output", 71, { text: "LATEST" }));
  events.receive(event("state_changed", 72, snapshot("stopped", 72)));
  events.activate(snapshot("provisioning", 1));
  assert.ok(got.every((event) => event.seq > 0));
  assert.ok(got.some((event) => event.output?.text.includes("已截断")));
  assert.equal(
    got.filter((event) => event.output).at(-1).output.text,
    "LATEST",
  );
  assert.equal(got.at(-1).state.state, "stopped");
});
