package policy

import (
	"net"
	"net/url"
	"os"
	"strings"
)

// 固定 profileId：首小时确认后不得由前端随意改写 origin。
const (
	ProfileDemoModel  = "demo-model"
	ProfileDemoMCP    = "demo-mcp"
	ProfileDSHModel   = "dsh-model"
	ProfileCodexModel = "codex-model"
)

// FixedProfile 描述受信后端配置中的固定上游。
type FixedProfile struct {
	ID                    string
	Kind                  string // model | mcp
	Origin                string // 例如 https://api.example.com
	BasePath              string // 受信模型路径前缀，不由执行面选择
	Paths                 []string
	Tools                 []string // MCP 工具白名单；模型侧可为空
	Host                  string   // origin 主机名
	Protocol              string   // anthropic、openai-responses 或 streamable-http
	Model                 string   // 固定模型名，与 Kit 一致
	AllowInsecureLoopback bool     // 仅受控测试可显式启用
	TestPath              string   // 连接测试使用的路径
}

// DefaultProfiles 返回两天版固定组合。
// 必须显式登记 origin、模型及工具白名单；没有占位供应商或隐式全工具授权。
func DefaultProfiles() map[string]FixedProfile {
	// demo-model 由 Claude Kit 使用。优先采用语义更明确的 Claude 变量，
	// 同时保留旧 DEMO 变量，避免已有启动脚本在升级后失效。
	modelOrigin := envFirst("HARBOR_CLAUDE_MODEL_ORIGIN", "HARBOR_DEMO_MODEL_ORIGIN")
	modelBasePath := envFirst("HARBOR_CLAUDE_MODEL_BASE_PATH", "HARBOR_DEMO_MODEL_BASE_PATH")
	mcpOrigin := envOr("HARBOR_DEMO_MCP_ORIGIN", "")
	tools := []string{}
	for _, tool := range strings.Split(os.Getenv("HARBOR_DEMO_MCP_TOOLS"), ",") {
		if tool = strings.TrimSpace(tool); tool != "" {
			tools = append(tools, tool)
		}
	}
	return map[string]FixedProfile{
		ProfileDSHModel:   {ID: ProfileDSHModel, Kind: "model", Protocol: "anthropic", Model: strings.TrimSpace(os.Getenv("HARBOR_DSH_MODEL")), Origin: envOr("HARBOR_DSH_MODEL_ORIGIN", ""), Host: hostOfOrigin(envOr("HARBOR_DSH_MODEL_ORIGIN", "")), BasePath: strings.TrimRight(strings.TrimSpace(os.Getenv("HARBOR_DSH_MODEL_BASE_PATH")), "/"), Paths: []string{"/v1/messages"}, TestPath: "/v1/messages"},
		ProfileCodexModel: {ID: ProfileCodexModel, Kind: "model", Protocol: "openai-responses", Model: strings.TrimSpace(os.Getenv("HARBOR_CODEX_MODEL")), Origin: envOr("HARBOR_CODEX_MODEL_ORIGIN", ""), Host: hostOfOrigin(envOr("HARBOR_CODEX_MODEL_ORIGIN", "")), BasePath: strings.TrimRight(strings.TrimSpace(os.Getenv("HARBOR_CODEX_MODEL_BASE_PATH")), "/"), Paths: []string{"/v1/responses"}, TestPath: "/v1/responses"},
		ProfileDemoModel:  {ID: ProfileDemoModel, Kind: "model", Protocol: "anthropic", Model: strings.TrimSpace(os.Getenv("HARBOR_CLAUDE_MODEL")), Origin: modelOrigin, Host: hostOfOrigin(modelOrigin), BasePath: modelBasePath, Paths: []string{"/v1/messages"}, TestPath: "/v1/messages"},
		ProfileDemoMCP:    {ID: ProfileDemoMCP, Kind: "mcp", Protocol: "streamable-http", Origin: mcpOrigin, Host: hostOfOrigin(mcpOrigin), Paths: []string{"/mcp"}, TestPath: "/mcp", Tools: tools},
	}
}

// ValidConnection 拒绝未登记、含用户信息或不安全的生产接入配置。
func (p FixedProfile) ValidConnection() bool {
	if !ValidBasePath(p.BasePath) {
		return false
	}
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
		return (p.Protocol == "anthropic" || p.Protocol == "openai-responses") && strings.TrimSpace(p.Model) != "" && len(p.Model) <= 256 && !strings.ContainsAny(p.Model, "\x00\r\n")
	}
	return p.Kind == "mcp" && p.Protocol == "streamable-http" && len(p.Tools) > 0
}

// 路径前缀只能来自宿主配置，拒绝编码、路径穿越或 URL 组成部分。
func ValidBasePath(path string) bool {
	if path == "" {
		return true
	}
	if !strings.HasPrefix(path, "/") || strings.HasSuffix(path, "/") || strings.Contains(path, "//") || strings.ContainsAny(path, "%?#\\\x00\r\n") {
		return false
	}
	for _, part := range strings.Split(path, "/")[1:] {
		if part == "." || part == ".." || strings.TrimSpace(part) != part {
			return false
		}
	}
	return true
}

// JoinBasePath 将受信路径前缀与请求路径拼接一次，避免 provider 已把 /v1 放在请求路径中时重复拼接。
func JoinBasePath(base, path string) string {
	base = strings.TrimRight(base, "/")
	if base == "" || path == base || strings.HasPrefix(path, base+"/") {
		return path
	}
	return base + "/" + strings.TrimLeft(path, "/")
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return strings.TrimRight(v, "/")
	}
	return strings.TrimRight(fallback, "/")
}

// envFirst 返回第一个非空环境变量，并按配置路径规则去掉末尾斜杠。
func envFirst(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return strings.TrimRight(value, "/")
		}
	}
	return ""
}

func hostOfOrigin(origin string) string {
	u, err := url.Parse(origin)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}
