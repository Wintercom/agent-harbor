package runtime

import (
	"bytes"
	_ "embed"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

//go:embed sandbox.sb
var profileTemplate string

func sbQuote(value string) (string, error) {
	for _, r := range value {
		if r < 32 || r == 127 {
			return "", runtimeError("INVALID_ARGUMENT")
		}
	}
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value) + `"`, nil
}

func endpointPort(address string) (int, error) {
	host, raw, err := net.SplitHostPort(address)
	port, portErr := strconv.Atoi(raw)
	if err != nil || portErr != nil || host != "127.0.0.1" || port < 1 || port > 65535 || strconv.Itoa(port) != raw {
		return 0, runtimeError("INVALID_ARGUMENT")
	}
	return port, nil
}

var controlledExecutablePaths = []string{
	"/bin/cat", "/bin/cp", "/bin/ls", "/bin/mkdir", "/bin/mv", "/bin/rm", "/bin/sh", "/bin/echo",
	"/bin/zsh", "/bin/bash", "/usr/bin/awk", "/usr/bin/env", "/usr/bin/false", "/usr/bin/find",
	"/usr/bin/git", "/usr/bin/grep", "/usr/bin/head", "/usr/bin/node", "/usr/bin/printf",
	"/usr/bin/pwd", "/usr/bin/sed", "/usr/bin/tail", "/usr/bin/true", "/usr/bin/which", "/usr/bin/xargs",
	"/usr/local/bin/node", "/opt/homebrew/bin/node", "/opt/homebrew/bin/npm", "/opt/homebrew/bin/npx",
	"/opt/homebrew/bin/go", "/usr/local/go/bin/go",
}

// Codex 启动时会探测系统级 managed requirements；只允许读取固定策略文件，不能读取整个 /etc。
var codexManagedConfigPaths = []string{
	"/etc/codex/requirements.toml",
	"/etc/codex/managed_config.toml",
}

func makeProfile(workspace, home string, readOnly, addresses []string, allowChildren bool) (string, error) {
	var rules strings.Builder
	for _, path := range []string{workspace, home} {
		quoted, err := sbQuote(path)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&rules, "(allow file-read* file-write* (subpath %s))\n", quoted)
		fmt.Fprintf(&rules, "(allow file-read-metadata (path-ancestors %s))\n", quoted)
	}
	// 预检 true 也在明确的执行白名单内，不开放整个 /usr/bin。
	paths := append([]string{"/usr/bin/true"}, readOnly...)
	if allowChildren {
		paths = append(paths, controlledExecutablePaths...)
	}
	for _, path := range paths {
		quoted, err := sbQuote(path)
		if err != nil {
			return "", err
		}
		info, err := os.Stat(path)
		if err != nil {
			if allowChildren && containsPath(controlledExecutablePaths, path) {
				continue
			}
			return "", runtimeError("AGENT_UNSUPPORTED")
		}
		if info.Mode()&(os.ModeSetuid|os.ModeSetgid) != 0 {
			return "", runtimeError("AGENT_UNSUPPORTED")
		}
		if info.IsDir() {
			fmt.Fprintf(&rules, "(allow file-read* file-map-executable (subpath %s))\n", quoted)
			fmt.Fprintf(&rules, "(allow file-read-metadata (path-ancestors %s))\n", quoted)
			continue
		}
		fmt.Fprintf(&rules, "(allow file-read* file-map-executable (literal %s))\n", quoted)
		if info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0 {
			fmt.Fprintf(&rules, "(allow process-exec (literal %s))\n", quoted)
		}
	}
	if allowChildren {
		for _, path := range codexManagedConfigPaths {
			quoted, err := sbQuote(path)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&rules, "(allow file-read* (literal %s))\n", quoted)
			fmt.Fprintf(&rules, "(allow file-read-metadata (path-ancestors %s))\n", quoted)
		}
		// Codex 启动时通过 cfprefsd 同步系统 managed preferences；只放行查询服务，
		// 不开放用户配置目录、钥匙串或其它宿主 IPC。
		rules.WriteString("(allow mach-lookup (global-name \"com.apple.cfprefsd.daemon\"))\n")
		rules.WriteString("(allow mach-lookup (global-name \"com.apple.cfprefsd.agent\"))\n")
	}
	for _, address := range addresses {
		port, err := endpointPort(address)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&rules, "(allow network-outbound (remote tcp4 \"localhost:%d\"))\n", port)
	}
	tmpl, err := template.New("seatbelt").Parse(profileTemplate)
	if err != nil {
		return "", err
	}
	var result bytes.Buffer
	err = tmpl.Execute(&result, struct {
		Rules         string
		AllowChildren bool
	}{Rules: rules.String(), AllowChildren: allowChildren})
	return result.String(), err
}

func containsPath(paths []string, target string) bool {
	for _, path := range paths {
		if path == target {
			return true
		}
	}
	return false
}

func canonicalPath(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", runtimeError("INVALID_ARGUMENT")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", runtimeError("WORKSPACE_DENIED")
	}
	return filepath.Clean(resolved), nil
}

func within(parent, path string) bool {
	rel, err := filepath.Rel(parent, path)
	return err == nil && filepath.IsLocal(rel)
}
