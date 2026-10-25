package policy

import "testing"

func TestProductionProfilesFailClosedUntilConfigured(t *testing.T) {
	for _, key := range []string{"HARBOR_DEMO_MODEL_ORIGIN", "HARBOR_DEMO_MCP_ORIGIN", "HARBOR_CLAUDE_MODEL", "HARBOR_DEMO_MCP_TOOLS", "HARBOR_DSH_MODEL_ORIGIN", "HARBOR_DSH_MODEL", "HARBOR_DSH_MODEL_BASE_PATH", "HARBOR_CODEX_MODEL_ORIGIN", "HARBOR_CODEX_MODEL", "HARBOR_CODEX_MODEL_BASE_PATH"} {
		t.Setenv(key, "")
	}
	for _, p := range DefaultProfiles() {
		if p.ValidConnection() {
			t.Fatal("未配置的上游可用")
		}
	}
	t.Setenv("HARBOR_DEMO_MODEL_ORIGIN", "https://model.example.test")
	t.Setenv("HARBOR_CLAUDE_MODEL", "approved-model")
	t.Setenv("HARBOR_DSH_MODEL_ORIGIN", "https://dsh.example.test")
	t.Setenv("HARBOR_DSH_MODEL", "approved-dsh-model")
	t.Setenv("HARBOR_DSH_MODEL_BASE_PATH", "/anthropic")
	t.Setenv("HARBOR_CODEX_MODEL_ORIGIN", "https://codex.example.test")
	t.Setenv("HARBOR_CODEX_MODEL", "approved-codex-model")
	t.Setenv("HARBOR_CODEX_MODEL_BASE_PATH", "")
	t.Setenv("HARBOR_DEMO_MCP_ORIGIN", "https://tools.example.test")
	t.Setenv("HARBOR_DEMO_MCP_TOOLS", "read_approved,lookup_approved")
	for _, p := range DefaultProfiles() {
		if !p.ValidConnection() {
			t.Fatalf("配置未通过 %s", p.ID)
		}
	}
	t.Setenv("HARBOR_DEMO_MODEL_ORIGIN", "http://127.0.0.1:1234")
	if DefaultProfiles()[ProfileDemoModel].ValidConnection() {
		t.Fatal("环境变量启用了不安全上游")
	}
}
