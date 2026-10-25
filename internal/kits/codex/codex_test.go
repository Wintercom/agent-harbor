package codex

import (
	"context"
	"strings"
	"testing"

	"github.com/liyufei916/agent-harbor/internal/kits"
)

func validInput() kits.KitInput {
	return kits.KitInput{
		Task:         "读取 README 并总结",
		PrivateHome:  "/private/tmp/harbor-home",
		ModelBaseURL: "http://127.0.0.1:41001",
		MCPURL:       "http://127.0.0.1:41002/mcp",
		Sentinel:     strings.Repeat("s", 64),
	}
}

func TestBuildUsesResponsesProviderAndPrivateCodexHome(t *testing.T) {
	k, err := New("/bin/sh", "gpt-5-codex")
	if err != nil {
		t.Fatal(err)
	}
	launch, err := k.Build(context.Background(), validInput())
	if err != nil {
		t.Fatal(err)
	}
	if launch.Command != "/bin/sh" || launch.Env["CODEX_HOME"] != "/private/tmp/harbor-home/codex" || launch.Env["OPENAI_API_KEY"] != strings.Repeat("s", 64) {
		t.Fatalf("Codex 启动环境错误: %+v", launch)
	}
	if len(launch.ConfigFiles) != 1 || launch.ConfigFiles[0].RelativePath != "codex/config.toml" {
		t.Fatalf("Codex 配置路径错误: %+v", launch.ConfigFiles)
	}
	config := string(launch.ConfigFiles[0].Content)
	for _, required := range []string{"wire_api = \"responses\"", "env_key = \"OPENAI_API_KEY\"", "[mcp_servers.harbor]", "http://127.0.0.1:41002/mcp"} {
		if !strings.Contains(config, required) {
			t.Fatalf("Codex 配置缺少 %q: %s", required, config)
		}
	}
	if strings.Contains(config, "api_key") {
		t.Fatal("Codex 配置写入了 api_key 字段")
	}
}

func TestDetectRequiresCodexExecFlags(t *testing.T) {
	k, err := New("/bin/sh", "gpt-5-codex")
	if err != nil {
		t.Fatal(err)
	}
	k.inspect = func(_ context.Context, _ string, args []string) (string, error) {
		if len(args) == 1 && args[0] == "--version" {
			return "codex-cli 0.163.0", nil
		}
		return "--json --ephemeral --skip-git-repo-check", nil
	}
	info, err := k.Detect(context.Background())
	if err != nil || !info.Agent.Available || info.Agent.ID != "codex" || info.Agent.ModelProfileID != "codex-model" || !info.NeedsChildProcesses {
		t.Fatalf("Codex 检测失败: %+v %v", info, err)
	}
}

func TestBuildRejectsNonGateURLs(t *testing.T) {
	k, _ := New("/bin/sh", "gpt-5-codex")
	input := validInput()
	input.ModelBaseURL = "https://api.openai.com/v1"
	if _, err := k.Build(context.Background(), input); err == nil {
		t.Fatal("Codex 接受了非 Gate 地址")
	}
}
