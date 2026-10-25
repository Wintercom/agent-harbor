//go:build !darwin

package claude

import (
	"context"

	"github.com/liyufei916/agent-harbor/internal/contracts"
)

func inspectExecutable(context.Context, string, []string, []string) (string, error) {
	return "", &contracts.AppError{Code: "AGENT_UNSUPPORTED", Message: "当前平台不支持 Claude 受限检测。"}
}
