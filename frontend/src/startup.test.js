import assert from "node:assert/strict";
import test from "node:test";
import { initializeTransport } from "./startup.js";

function fixture(fail = false) {
  const calls = [];
  return {
    calls,
    api: {
      useMock() { calls.push("mock"); },
      setMockScenario(value) { calls.push(value); },
      async useLive() {
        calls.push("live");
        if (fail) throw new Error("fixture load failure");
      },
    },
  };
}

test("桌面默认使用 live，即使 URL 带有模拟参数", async () => {
  const f = fixture();
  await initializeTransport(f.api, { shell: true, search: "?live=0&scenario=auth" });
  assert.deepEqual(f.calls, ["live"]);
});

test("live 初始化失败不会回退 mock", async () => {
  for (const options of [{ shell: true }, { search: "?live=1" }]) {
    const f = fixture(true);
    await assert.rejects(initializeTransport(f.api, options));
    assert.deepEqual(f.calls, ["live"]);
  }
});

test("普通浏览器保留显式模拟场景", async () => {
  const f = fixture();
  await initializeTransport(f.api, { search: "?scenario=auth" });
  assert.deepEqual(f.calls, ["mock", "auth"]);
});
