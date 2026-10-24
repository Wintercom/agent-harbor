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

func makeProfile(workspace, home string, readOnly, addresses []string) (string, error) {
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
	for _, path := range append([]string{"/usr/bin/true"}, readOnly...) {
		quoted, err := sbQuote(path)
		if err != nil {
			return "", err
		}
		info, err := os.Stat(path)
		if err != nil {
			return "", runtimeError("AGENT_UNSUPPORTED")
		}
		if info.Mode()&(os.ModeSetuid|os.ModeSetgid) != 0 {
			return "", runtimeError("AGENT_UNSUPPORTED")
		}
		if info.IsDir() {
			return "", runtimeError("INVALID_ARGUMENT")
		}
		fmt.Fprintf(&rules, "(allow file-read* file-map-executable (literal %s))\n", quoted)
		if info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0 {
			fmt.Fprintf(&rules, "(allow process-exec (literal %s))\n", quoted)
		}
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
	err = tmpl.Execute(&result, struct{ Rules string }{rules.String()})
	return result.String(), err
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
