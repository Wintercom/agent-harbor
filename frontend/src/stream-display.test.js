import test from "node:test";
import assert from "node:assert/strict";
import { createStreamDisplay } from "./stream-display.js";

test("stream-display 提取 assistant 文本并折叠协议噪声", () => {
  const d = createStreamDisplay();
  const chunk =
    JSON.stringify({
      type: "system",
      subtype: "init",
      model: "claude-sonnet-4-5",
      tools: ["Read", "Write"],
    }) +
    "\n" +
    JSON.stringify({
      type: "assistant",
      message: {
        content: [
          { type: "thinking", thinking: "secret" },
          { type: "text", text: "你好" },
          {
            type: "tool_use",
            name: "Read",
            input: { file_path: "/tmp/a.md" },
          },
        ],
      },
    }) +
    "\n" +
    JSON.stringify({
      type: "user",
      message: {
        content: [
          { type: "tool_result", content: "big file...", is_error: false },
        ],
      },
    }) +
    "\n" +
    JSON.stringify({
      type: "result",
      subtype: "success",
      result: "短结论",
    }) +
    "\n";

  const out = d.push(chunk);
  assert.match(out, /会话就绪/);
  assert.match(out, /你好/);
  assert.match(out, /⚙ Read/);
  assert.doesNotMatch(out, /secret/);
  assert.doesNotMatch(out, /big file/);
  assert.match(out, /短结论/);
});

test("stream-display 跨 chunk 拼完整 JSON 行", () => {
  const d = createStreamDisplay();
  const line =
    JSON.stringify({
      type: "assistant",
      message: { content: [{ type: "text", text: "跨行" }] },
    }) + "\n";
  assert.equal(d.push(line.slice(0, 10)), "");
  const out = d.push(line.slice(10));
  assert.match(out, /跨行/);
});
