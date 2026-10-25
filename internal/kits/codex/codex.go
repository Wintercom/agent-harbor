// Package codex 适配 OpenAI Codex CLI，执行面只接收 Harbor 的代理地址和会话 Sentinel。
package codex

import (
	"context"
	"fmt"
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

type Kit struct {
	executable string
	model      string
	inspect    func(context.Context, string, []string) (string, error)
}

var _ kits.Kit = (*Kit)(nil)

func New(executable, model string) (*Kit, error) {
	model = strings.TrimSpace(model)
	if model == "" || len(model) > 256 || strings.ContainsAny(model, "\x00\r\n") {
		return nil, kitError("INVALID_ARGUMENT")
	}
	if executable == "" {
		executable = "codex"
	}
	return &Kit{executable: executable, model: model, inspect: inspectExecutable}, nil
}

func kitError(code string) *contracts.AppError {
	message := "当前 Codex CLI 未安装或不满足受限适配要求。"
	if code == "INVALID_ARGUMENT" {
		message = "Agent 配置参数不合法。"
	}
	if code == "CANCELLED" {
		message = "操作已取消。"
	}
	return contracts.NewAppError(code, message, false)
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
	info := kits.KitInfo{Agent: contracts.AgentInfo{ID: "codex", Name: "OpenAI Codex", ModelProfileID: "codex-model", Reason: "未安装或受限检测未通过。"}, ReadOnlyPaths: []string{}, NeedsChildProcesses: true}
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
	version = strings.TrimSpace(version)
	if version == "" || !strings.Contains(strings.ToLower(version), "codex") {
		return info, nil
	}
	help, err := k.inspect(ctx, path, []string{"exec", "--help"})
	if err != nil {
		return info, nil
	}
	for _, flag := range []string{"--json", "--ephemeral", "--skip-git-repo-check"} {
		if !strings.Contains(help, flag) {
			return info, nil
		}
	}
	info.Agent.Available, info.Agent.Version, info.Agent.Reason = true, version, ""
	info.ReadOnlyPaths = []string{path}
	return info, nil
}

func gateURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	host, port, err := net.SplitHostPort(u.Host)
	n, portErr := strconv.Atoi(port)
	return err == nil && portErr == nil && host == "127.0.0.1" && n > 0 && n < 65536 && strconv.Itoa(n) == port
}

func validSentinel(value string) bool {
	if len(value) < 32 || len(value) > 512 {
		return false
	}
	for _, r := range value {
		if r < 33 || r > 126 {
			return false
		}
	}
	return true
}

// Build 使用 Codex 的 Responses provider 和 HTTP MCP 配置；配置中只写入本会话 Sentinel。
func (k *Kit) Build(ctx context.Context, input kits.KitInput) (runtime.LaunchSpec, error) {
	if ctx.Err() != nil {
		return runtime.LaunchSpec{}, kitError("CANCELLED")
	}
	if !utf8.ValidString(input.Task) || strings.TrimSpace(input.Task) == "" || len(input.Task) > 32768 || !filepath.IsAbs(input.PrivateHome) || filepath.Clean(input.PrivateHome) != input.PrivateHome || !gateURL(input.ModelBaseURL) || !gateURL(input.MCPURL) || !validSentinel(input.Sentinel) {
		return runtime.LaunchSpec{}, kitError("INVALID_ARGUMENT")
	}
	command, err := k.resolve()
	if err != nil {
		return runtime.LaunchSpec{}, err
	}
	providerURL := strconv.Quote(input.ModelBaseURL)
	mcpURL := strconv.Quote(input.MCPURL)
	model := strconv.Quote(k.model)
	sentinel := strconv.Quote("Bearer " + input.Sentinel)
	config := []byte(fmt.Sprintf("model = %s\nmodel_provider = \"harbor\"\napproval_policy = \"never\"\nsandbox_mode = \"workspace-write\"\n\n[model_providers.harbor]\nname = \"Agent Harbor\"\nbase_url = %s\nwire_api = \"responses\"\nenv_key = \"OPENAI_API_KEY\"\n\n[mcp_servers.harbor]\nurl = %s\nhttp_headers = { Authorization = %s }\n", model, providerURL, mcpURL, sentinel))
	return runtime.LaunchSpec{
		Command:     command,
		Args:        []string{"exec", "--json", "--ephemeral", "--skip-git-repo-check", "--model", k.model, "-"},
		Env:         map[string]string{"CODEX_HOME": filepath.Join(input.PrivateHome, "codex"), "OPENAI_API_KEY": input.Sentinel},
		Stdin:       []byte(input.Task),
		ConfigFiles: []runtime.ConfigFile{{RelativePath: "codex/config.toml", Content: config}},
	}, nil
}

func inspectExecutable(ctx context.Context, command string, args []string) (string, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}
