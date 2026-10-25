package policy

import (
	"net"
	"net/url"
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
	ID                    string
	Kind                  string // model | mcp
	Origin                string // 例如 https://api.example.com
	Paths                 []string
	Tools                 []string // MCP 工具白名单；模型侧可为空
	Host                  string   // origin 主机名
	Protocol              string   // anthropic 或 streamable-http
	Model                 string   // 固定模型名，与 Kit 一致
	AllowInsecureLoopback bool     // 仅受控测试可显式启用
	TestPath              string   // 连接测试使用的路径
}

// DefaultProfiles 返回两天版固定组合。
// 必须显式登记 origin、模型及工具白名单；没有占位供应商或隐式全工具授权。
func DefaultProfiles() map[string]FixedProfile {
	modelOrigin := envOr("HARBOR_DEMO_MODEL_ORIGIN", "")
	mcpOrigin := envOr("HARBOR_DEMO_MCP_ORIGIN", "")
	tools := []string{}
	for _, tool := range strings.Split(os.Getenv("HARBOR_DEMO_MCP_TOOLS"), ",") {
		if tool = strings.TrimSpace(tool); tool != "" {
			tools = append(tools, tool)
		}
	}
	return map[string]FixedProfile{
		ProfileDemoModel: {ID: ProfileDemoModel, Kind: "model", Protocol: "anthropic", Model: strings.TrimSpace(os.Getenv("HARBOR_CLAUDE_MODEL")), Origin: modelOrigin, Host: hostOfOrigin(modelOrigin), Paths: []string{"/v1/messages"}, TestPath: "/v1/messages"},
		ProfileDemoMCP:   {ID: ProfileDemoMCP, Kind: "mcp", Protocol: "streamable-http", Origin: mcpOrigin, Host: hostOfOrigin(mcpOrigin), Paths: []string{"/mcp"}, TestPath: "/mcp", Tools: tools},
	}
}

// ValidConnection 拒绝未登记、含用户信息或不安全的生产接入配置。
func (p FixedProfile) ValidConnection() bool {
	u, err := url.Parse(p.Origin)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return false
	}
	if u.Scheme != "https" {
		ip := net.ParseIP(u.Hostname())
		if !p.AllowInsecureLoopback || u.Scheme != "http" || ip == nil || !ip.IsLoopback() {
			return false
		}
	}
	if !strings.EqualFold(p.Host, u.Hostname()) || !strings.HasPrefix(p.TestPath, "/") || strings.ContainsAny(p.TestPath, "?#") {
		return false
	}
	if p.Kind == "model" {
		return p.Protocol == "anthropic" && strings.TrimSpace(p.Model) != "" && len(p.Model) <= 256 && !strings.ContainsAny(p.Model, "\x00\r\n")
	}
	return p.Kind == "mcp" && p.Protocol == "streamable-http" && len(p.Tools) > 0
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return strings.TrimRight(v, "/")
	}
	return strings.TrimRight(fallback, "/")
}

func hostOfOrigin(origin string) string {
	u, err := url.Parse(origin)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}
