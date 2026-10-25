import test from "node:test";
import assert from "node:assert/strict";
import { mountBlock } from "./views/block.js";

function fakeRoot() {
  const el = {
    innerHTML: "",
    querySelector(sel) {
      const id = sel.startsWith("#") ? sel.slice(1) : null;
      if (!id) return null;
      const buttons = {};
      return {
        onclick: null,
        get disabled() {
          return !!buttons[id]?.disabled;
        },
        set disabled(v) {
          buttons[id] = { ...(buttons[id] || {}), disabled: v };
        },
        get textContent() {
          return buttons[id]?.textContent || "";
        },
        set textContent(v) {
          buttons[id] = { ...(buttons[id] || {}), textContent: v };
        },
      };
    },
  };
  // mountBlock uses querySelector twice; share handlers via map
  const handlers = {};
  el.querySelector = (sel) => {
    const id = sel.slice(1);
    if (!handlers[id]) {
      handlers[id] = {
        disabled: false,
        textContent: "",
        onclick: null,
      };
    }
    return handlers[id];
  };
  el._handlers = handlers;
  return el;
}

test("阻断页无「仍要继续」，有重新检测与设置", async () => {
  const root = fakeRoot();
  let rechecked = false;
  let toSettings = false;
  mountBlock(root, {
    reason: "隔离能力不可用（模拟）",
    checks: [{ ok: false, name: "隔离环境", detail: "失败" }],
    async onRecheck() {
      rechecked = true;
    },
    onSettings() {
      toSettings = true;
    },
  });
  assert.match(root.innerHTML, /无法安全启动/);
  assert.match(root.innerHTML, /不会在无隔离时启动/);
  assert.match(root.innerHTML, /也没有「仍要继续」/);
  assert.doesNotMatch(root.innerHTML, /id="btn-.*continue"/);
  assert.match(root.innerHTML, /隔离环境/);
  assert.ok(root._handlers["btn-block-recheck"]);
  assert.ok(root._handlers["btn-block-settings"]);
  await root._handlers["btn-block-recheck"].onclick();
  assert.equal(rechecked, true);
  root._handlers["btn-block-settings"].onclick();
  assert.equal(toSettings, true);
});
