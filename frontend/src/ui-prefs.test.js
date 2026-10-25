import test from "node:test";
import assert from "node:assert/strict";
import * as mock from "./mock-api.js";

test("mock Get/SetUIPrefs 读写侧栏偏好", async () => {
  mock.resetMock();
  try {
    const first = await mock.getUIPrefs();
    assert.equal(first.prefs.autoCollapseWorkbench, true);
    assert.equal(first.prefs.askClearOnStop, true);
    const saved = await mock.setUIPrefs({
      sidebarCollapsed: true,
      autoCollapseWorkbench: false,
    });
    assert.equal(saved.prefs.sidebarCollapsed, true);
    assert.equal(saved.prefs.autoCollapseWorkbench, false);
    assert.equal(saved.prefs.askClearOnStop, true);
    const again = await mock.getUIPrefs();
    assert.equal(again.prefs.sidebarCollapsed, true);
    assert.equal(again.prefs.autoCollapseWorkbench, false);
    const clear = await mock.setUIPrefs({
      askClearOnStop: false,
      clearOnStopDefault: "clear",
    });
    assert.equal(clear.prefs.askClearOnStop, false);
    assert.equal(clear.prefs.clearOnStopDefault, "clear");
    assert.equal(clear.prefs.sidebarCollapsed, true);
  } finally {
    mock.resetMock();
  }
});
