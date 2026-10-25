package desktop

import (
	"os/exec"
	"runtime"
	"strings"
)

// openSystemBrowser 尽力打开系统浏览器；失败不影响 OAuth pending。
func openSystemBrowser(rawURL string) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return
	}
	go func() {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", rawURL)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
		default:
			cmd = exec.Command("xdg-open", rawURL)
		}
		_ = cmd.Start()
	}()
}
