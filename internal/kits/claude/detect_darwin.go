package claude

import (
	"context"

	"github.com/liyufei916/agent-harbor/internal/runtime"
)

func inspectExecutable(ctx context.Context, command string, args, readOnly []string) (string, error) {
	return runtime.ProbeInstallation(ctx, command, args, readOnly)
}
