package orchestrator

import (
	"context"
	"os"
	"path/filepath"

	"github.com/liyufei916/agent-harbor/internal/runtime"
)

// retainCloneAside 在 Runtime.Destroy 前把会话副本挪出沙箱根，便于停止后仍可 SyncBack。
func retainCloneAside(sessionID, clonePath string) (string, error) {
	if sessionID == "" || clonePath == "" {
		return "", publicError("INVALID_ARGUMENT")
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dest := filepath.Join(base, "AgentHarbor", "retained", sessionID)
	if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
		return "", err
	}
	_ = os.RemoveAll(dest)
	if err := os.Rename(clonePath, dest); err == nil {
		return dest, nil
	}
	if err := runtime.SyncWorkspace(context.Background(), clonePath, dest); err != nil {
		_ = os.RemoveAll(dest)
		return "", err
	}
	_ = os.RemoveAll(clonePath)
	return dest, nil
}
