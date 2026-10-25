// Package claude 为固定 Claude Code 版本生成无真实凭据的启动规格。
package claude

import (
	"context"
	"encoding/json"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/liyufei916/agent-harbor/internal/contracts"
	"github.com/liyufei916/agent-harbor/internal/kits"
	"github.com/liyufei916/agent-harbor/internal/runtime"
)

const SupportedVersion = "2.1.292"

type Kit struct {
	executable, model string
	inspect           func(context.Context, string, []string) (string, error)
}

var _ kits.Kit = (*Kit)(nil)

// New 的 model 来自宿主机固定接入配置，不从任务正文推导或自动切换。
func New(executable, model string) (*Kit, error) {
	if strings.TrimSpace(model) == "" || len(model) > 256 || strings.ContainsAny(model, "\x00\r\n") {
		return nil, kitError("INVALID_ARGUMENT")
	}
	if executable == "" {
		executable = "claude"
	}
	return &Kit{executable: executable, model: model, inspect: inspectExecutable}, nil
}

func kitError(code string) *contracts.AppError {
	message := "当前 Claude Code 未安装或不满足受限适配要求。"
	if code == "INVALID_ARGUMENT" {
		message = "Agent 配置参数不合法。"
	}
	if code == "CANCELLED" {
		message = "操作已取消。"
	}
	return &contracts.AppError{Code: code, Message: message}
}

func (k *Kit) resolve() (string, error) {
	path, err := exec.LookPath(k.executable)
	if err != nil {
		return "", kitError("AGENT_UNSUPPORTED")
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil || !filepath.IsAbs(path) {
		return "", kitError("AGENT_UNSUPPORTED")
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 || info.Mode()&(os.ModeSetuid|os.ModeSetgid) != 0 {
		return "", kitError("AGENT_UNSUPPORTED")
	}
	return path, nil
}

func (k *Kit) Detect(ctx context.Context) (kits.KitInfo, error) {
	info := kits.KitInfo{Agent: contracts.AgentInfo{ID: "claude", Name: "Claude Code", ModelProfileID: "demo-model", Reason: "未安装、版本不支持或受限检测未通过。"}, ReadOnlyPaths: []string{}}
	if ctx.Err() != nil {
		return info, kitError("CANCELLED")
	}
	path, err := k.resolve()
	if err != nil {
		return info, nil
	}
	version, err := k.inspect(ctx, path, []string{"--version"})
	if err != nil {
		if ctx.Err() != nil {
			return info, kitError("CANCELLED")
		}
		return info, nil
	}
	if strings.TrimSpace(version) != SupportedVersion+" (Claude Code)" {
		return info, nil
	}
	info.Agent.Version = SupportedVersion
	help, err := k.inspect(ctx, path, []string{"--bare", "--help"})
	if err != nil {
		return info, nil
	}
	for _, flag := range []string{"--bare", "--print", "--input-format", "stream-json", "--strict-mcp-config", "--mcp-config", "--setting-sources", "--no-session-persistence", "--permission-prompts", "--permission-mode", "--disable-slash-commands", "--tools"} {
		if !strings.Contains(help, flag) {
			return info, nil
		}
	}
	info.Agent.Available, info.Agent.Reason = true, ""
	info.ReadOnlyPaths = []string{path}
	return info, nil
}

func gateURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	host, rawPort, err := net.SplitHostPort(u.Host)
	port, portErr := strconv.Atoi(rawPort)
	return err == nil && portErr == nil && host == "127.0.0.1" && port > 0 && port < 65536 && strconv.Itoa(port) == rawPort
}

func (k *Kit) Build(ctx context.Context, input kits.KitInput) (runtime.LaunchSpec, error) {
	if ctx.Err() != nil {
		return runtime.LaunchSpec{}, kitError("CANCELLED")
	}
	if !utf8.ValidString(input.Task) || strings.TrimSpace(input.Task) == "" || len(input.Task) > 32768 || !filepath.IsAbs(input.PrivateHome) || filepath.Clean(input.PrivateHome) != input.PrivateHome || !gateURL(input.ModelBaseURL) || len(input.Sentinel) < 32 || len(input.Sentinel) > 512 {
		return runtime.LaunchSpec{}, kitError("INVALID_ARGUMENT")
	}
	urls, urlErr := kits.MCPGateURLs(input)
	if urlErr != nil {
		return runtime.LaunchSpec{}, kitError("INVALID_ARGUMENT")
	}
	for _, r := range input.Sentinel {
		if r < 33 || r > 126 {
			return runtime.LaunchSpec{}, kitError("INVALID_ARGUMENT")
		}
	}
	command, err := k.resolve()
	if err != nil {
		return runtime.LaunchSpec{}, err
	}
	mcpPath := filepath.Join(input.PrivateHome, "harbor-mcp.json")
	settingsPath := filepath.Join(input.PrivateHome, "harbor-settings.json")
	servers := map[string]any{}
	permissions := []string{"Read", "Write", "Edit"}
	for i, endpoint := range urls {
		name := kits.MCPName(i)
		servers[name] = map[string]any{"type": "http", "url": endpoint, "headers": map[string]string{"Authorization": "Bearer " + input.Sentinel}}
		permissions = append(permissions, "mcp__"+name+"__*")
	}
	mcp, err := json.Marshal(map[string]any{"mcpServers": servers})
	if err != nil {
		return runtime.LaunchSpec{}, kitError("INVALID_ARGUMENT")
	}
	settings, err := json.Marshal(map[string]any{"disableAllHooks": true, "permissions": map[string]any{"allow": permissions}})
	if err != nil {
		return runtime.LaunchSpec{}, kitError("INVALID_ARGUMENT")
	}
	return runtime.LaunchSpec{
		Command:     command,
		Args:        []string{"--bare", "--print", "--output-format", "stream-json", "--verbose", "--input-format", "stream-json", "--no-session-persistence", "--strict-mcp-config", "--mcp-config", mcpPath, "--settings", settingsPath, "--setting-sources", "", "--permission-mode", "dontAsk", "--permission-prompts", "none", "--disable-slash-commands", "--no-chrome", "--tools", "Read,Write,Edit", "--model", k.model},
		Env:         map[string]string{"ANTHROPIC_BASE_URL": input.ModelBaseURL, "ANTHROPIC_API_KEY": input.Sentinel, "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1", "DISABLE_AUTOUPDATER": "1", "DISABLE_TELEMETRY": "1", "DISABLE_ERROR_REPORTING": "1"},
		Stdin:       []byte(input.Task),
		Interactive: true,
		EncodeInput: encodeMessage,
		ConfigFiles: []runtime.ConfigFile{{RelativePath: "harbor-mcp.json", Content: mcp}, {RelativePath: "harbor-settings.json", Content: settings}},
	}, nil
}

// 每个用户轮次是一条 JSONL，正文中的换行和引号不能注入协议记录。
func encodeMessage(data []byte) ([]byte, error) {
	if !utf8.Valid(data) || len(data) == 0 || len(data) > 32768 || strings.TrimSpace(string(data)) == "" || strings.ContainsRune(string(data), 0) {
		return nil, kitError("INVALID_ARGUMENT")
	}
	message := map[string]any{"type": "user", "message": map[string]string{"role": "user", "content": string(data)}}
	encoded, err := json.Marshal(message)
	if err != nil {
		return nil, kitError("INVALID_ARGUMENT")
	}
	return append(encoded, '\n'), nil
}
