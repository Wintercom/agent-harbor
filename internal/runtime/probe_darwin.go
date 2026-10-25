package runtime

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/liyufei916/agent-harbor/internal/contracts"
)

// ProbeExecutable 在无网络、无子进程限制下检测 CLI，不借用宿主登录态。
// 仅供受信 Kit 使用，不绑定到桌面或暴露给 Agent。
func ProbeExecutable(ctx context.Context, command string, args []string) (result string, resultErr error) {
	return ProbeInstallation(ctx, command, args, nil)
}

// ProbeInstallation 用明确的只读运行文件检测解释型 CLI，不加载宿主配置。
func ProbeInstallation(ctx context.Context, command string, args, readOnly []string) (result string, resultErr error) {
	workspace, err := os.MkdirTemp("/private/tmp", "agent-harbor-inspect-")
	if err != nil {
		return "", runtimeError("AGENT_UNSUPPORTED")
	}
	defer os.RemoveAll(workspace)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	r := NewMacOS()
	handle, err := r.create(ctx, RuntimeSpec{SessionID: "inspect", Workspace: workspace, MountMode: "direct", ReadOnlyPaths: append([]string{command}, readOnly...)}, true)
	if err != nil {
		return "", err
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := r.Destroy(cleanup, handle); err != nil {
			result, resultErr = "", err
		}
	}()
	var mu sync.Mutex
	var text strings.Builder
	truncated := false
	process, err := r.Exec(ctx, handle, LaunchSpec{Command: command, Args: args}, func(data contracts.OutputData) {
		mu.Lock()
		defer mu.Unlock()
		if text.Len()+len(data.Text) > 64<<10 {
			truncated = true
			return
		}
		text.WriteString(data.Text)
	})
	if err != nil {
		return "", err
	}
	exit := process.Wait()
	mu.Lock()
	defer mu.Unlock()
	if exit.Code != 0 || exit.Err != nil || exit.Cancelled || truncated {
		return "", runtimeError("AGENT_UNSUPPORTED")
	}
	return text.String(), nil
}
