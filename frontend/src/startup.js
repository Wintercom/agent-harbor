// 桌面只能使用真实绑定；模拟场景只供普通浏览器独立验收。
export async function initializeTransport(api, { shell = false, search = "" } = {}) {
  const params = new URLSearchParams(search);
  if (shell || params.get("live") === "1") {
    await api.useLive();
    return;
  }
  api.useMock();
  if (params.get("scenario")) api.setMockScenario(params.get("scenario"));
}
