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

// SupportedVersion 是文档/联调基线；Detect 默认按能力放行，仅当设置
// HARBOR_CLAUDE_REQUIRE_VERSION 时才要求精确匹配。
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
	candidates := []string{k.executable}
	base := filepath.Base(k.executable)
	if base == "claude" || k.executable == "claude" {
		// GUI/Wails 常缺 Homebrew PATH；补常见绝对路径。
		candidates = append(candidates,
			"/opt/homebrew/bin/claude",
			"/usr/local/bin/claude",
		)
	}
	seen := map[string]bool{}
	for _, cand := range candidates {
		cand = strings.TrimSpace(cand)
		if cand == "" || seen[cand] {
			continue
		}
		seen[cand] = true
		path := cand
		if !filepath.IsAbs(path) {
			p, err := exec.LookPath(path)
			if err != nil {
				continue
			}
			path = p
		}
		path, err := filepath.EvalSymlinks(path)
		if err != nil || !filepath.IsAbs(path) {
			continue
		}
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 || info.Mode()&(os.ModeSetuid|os.ModeSetgid) != 0 {
			continue
		}
		return path, nil
	}
	return "", kitError("AGENT_UNSUPPORTED")
}

func detectProbeReason(path, step string, err error) string {
	if ae, ok := err.(*contracts.AppError); ok && ae != nil {
		if ae.Code == "ISOLATION_UNAVAILABLE" {
			return "本机隔离不可用，无法在受限环境中检测 Agent（先看侧栏隔离状态 / 重新检测隔离）。"
		}
		if ae.Code == "CANCELLED" {
			return "检测已取消。"
		}
		// 只用固定错误码文案，不回传可能含内部细节的 Message。
		return "可执行文件 " + path + " 在受限环境执行 " + step + " 未通过（" + ae.Code + "）。"
	}
	return "可执行文件 " + path + " 在受限环境执行 " + step + " 未通过。"
}

func parseClaudeVersion(raw string) (string, bool) {
	line := strings.TrimSpace(strings.Split(raw, "\n")[0])
	const suffix = " (Claude Code)"
	if !strings.HasSuffix(line, suffix) {
		return "", false
	}
	ver := strings.TrimSpace(strings.TrimSuffix(line, suffix))
	if ver == "" || strings.ContainsAny(ver, " \t") {
		return "", false
	}
	return ver, true
}

// requiredHelpFlags 是启动规格依赖、且须出现在 --help 中的参数。
// --bare 以能成功执行 `--bare --help` 为准（帮助文案可能不列出）。
// --permission-prompts 在较新 CLI 中已移除，改用 --permission-mode。
var requiredHelpFlags = []string{
	"--print", "--input-format", "stream-json",
	"--strict-mcp-config", "--mcp-config", "--setting-sources",
	"--no-session-persistence", "--permission-mode",
	"--disable-slash-commands", "--tools",
}

func (k *Kit) Detect(ctx context.Context) (kits.KitInfo, error) {
	info := kits.KitInfo{
		Agent: contracts.AgentInfo{
			ID: "claude", Name: "Claude Code", ModelProfileID: "demo-model",
			Reason: "未安装 Claude Code，或 PATH/HARBOR_CLAUDE_BIN 未指向可执行文件。",
		},
		ReadOnlyPaths: []string{},
	}
	if ctx.Err() != nil {
		return info, kitError("CANCELLED")
	}
	path, err := k.resolve()
	if err != nil {
		return info, nil
	}
	versionOut, err := k.inspect(ctx, path, []string{"--version"})
	if err != nil {
		if ctx.Err() != nil {
			return info, kitError("CANCELLED")
		}
		info.Agent.Reason = detectProbeReason(path, "--version", err)
		return info, nil
	}
	version, ok := parseClaudeVersion(versionOut)
	if !ok {
		info.Agent.Reason = "版本输出无法识别（期望形如「x.y.z (Claude Code)」）。"
		return info, nil
	}
	info.Agent.Version = version
	if want := strings.TrimSpace(os.Getenv("HARBOR_CLAUDE_REQUIRE_VERSION")); want != "" && version != want {
		info.Agent.Reason = "已安装 " + version + "，当前要求 " + want + "（HARBOR_CLAUDE_REQUIRE_VERSION）。"
		return info, nil
	}
	help, err := k.inspect(ctx, path, []string{"--bare", "--help"})
	if err != nil {
		if ctx.Err() != nil {
			return info, kitError("CANCELLED")
		}
		info.Agent.Reason = "已安装 " + version + "。" + detectProbeReason(path, "--bare --help", err)
		return info, nil
	}
	var missing []string
	for _, flag := range requiredHelpFlags {
		if !strings.Contains(help, flag) {
			missing = append(missing, flag)
		}
	}
	if len(missing) > 0 {
		info.Agent.Reason = "已安装 " + version + "，缺少受限启动所需参数：" + strings.Join(missing, ", ") + "。"
		return info, nil
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
	if !utf8.ValidString(input.Task) || strings.TrimSpace(input.Task) == "" || len(input.Task) > 32768 || !filepath.IsAbs(input.PrivateHome) || filepath.Clean(input.PrivateHome) != input.PrivateHome || !gateURL(input.ModelBaseURL) || !gateURL(input.MCPURL) || len(input.Sentinel) < 32 || len(input.Sentinel) > 512 {
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
	mcp, err := json.Marshal(map[string]any{"mcpServers": map[string]any{"harbor": map[string]any{"type": "http", "url": input.MCPURL, "headers": map[string]string{"Authorization": "Bearer " + input.Sentinel}}}})
	if err != nil {
		return runtime.LaunchSpec{}, kitError("INVALID_ARGUMENT")
	}
	settings, err := json.Marshal(map[string]any{"disableAllHooks": true, "permissions": map[string]any{"allow": []string{"Read", "Write", "Edit", "mcp__harbor__*"}}})
	if err != nil {
		return runtime.LaunchSpec{}, kitError("INVALID_ARGUMENT")
	}
	return runtime.LaunchSpec{
		Command: command,
		// 不再传 --permission-prompts（新 CLI 已移除）；dontAsk 由 --permission-mode 承担。
		Args:        []string{"--bare", "--print", "--output-format", "stream-json", "--verbose", "--input-format", "stream-json", "--no-session-persistence", "--strict-mcp-config", "--mcp-config", mcpPath, "--settings", settingsPath, "--setting-sources", "", "--permission-mode", "dontAsk", "--disable-slash-commands", "--no-chrome", "--tools", "Read,Write,Edit", "--model", k.model},
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
