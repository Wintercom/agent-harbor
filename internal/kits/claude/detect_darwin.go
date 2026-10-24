package claude

import (
	"context"

	"github.com/liyufei916/agent-harbor/internal/runtime"
)

func inspectExecutable(ctx context.Context, command string, args []string) (string, error) {
	return runtime.ProbeExecutable(ctx, command, args)
}
