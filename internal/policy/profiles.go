package policy

import (
	"os"
	"strings"
)

// 固定 profileId：首小时确认后不得由前端随意改写 origin。
const (
	ProfileDemoModel = "demo-model"
	ProfileDemoMCP   = "demo-mcp"
)

// FixedProfile 描述受信后端配置中的固定上游。
type FixedProfile struct {
	ID       string
	Kind     string // model | mcp
	Origin   string // 例如 https://api.example.com
	Paths    []string
	Tools    []string // MCP 工具白名单；模型侧可为空
	Host     string   // origin 主机名
	TestPath string   // 连接测试使用的路径
}

// DefaultProfiles 返回两天版固定组合。
// 可通过环境变量覆盖 origin，便于本地联调（不提交真实凭据）：
// HARBOR_DEMO_MODEL_ORIGIN、HARBOR_DEMO_MCP_ORIGIN。
func DefaultProfiles() map[string]FixedProfile {
	modelOrigin := envOr("HARBOR_DEMO_MODEL_ORIGIN", "https://api.deepseek.com")
	mcpOrigin := envOr("HARBOR_DEMO_MCP_ORIGIN", "https://mcp.example.test")
	return map[string]FixedProfile{
		ProfileDemoModel: {
			ID:       ProfileDemoModel,
			Kind:     "model",
			Origin:   modelOrigin,
			Paths:    []string{"/v1/", "/chat/"},
			Host:     hostOfOrigin(modelOrigin),
			TestPath: "/v1/models",
		},
		ProfileDemoMCP: {
			ID:       ProfileDemoMCP,
			Kind:     "mcp",
			Origin:   mcpOrigin,
			Paths:    []string{"/mcp"},
			Tools:    []string{"list_resources", "read_resource"},
			Host:     hostOfOrigin(mcpOrigin),
			TestPath: "/mcp",
		},
	}
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return strings.TrimRight(v, "/")
	}
	return strings.TrimRight(fallback, "/")
}

func hostOfOrigin(origin string) string {
	origin = strings.TrimSpace(origin)
	origin = strings.TrimPrefix(origin, "https://")
	origin = strings.TrimPrefix(origin, "http://")
	if i := strings.IndexByte(origin, '/'); i >= 0 {
		origin = origin[:i]
	}
	if i := strings.IndexByte(origin, ':'); i >= 0 {
		origin = origin[:i]
	}
	return strings.ToLower(origin)
}
