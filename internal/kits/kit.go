// 本文件按 docs/CONTRACT.md v1 定义，不包含业务实现。
package kits

import (
	"context"

	"github.com/liyufei916/agent-harbor/internal/contracts"
	"github.com/liyufei916/agent-harbor/internal/runtime"
)

type KitInfo struct {
	Agent         contracts.AgentInfo
	ReadOnlyPaths []string // 固定 CLI 与必要运行文件的真实路径
}

type KitInput struct {
	Task         string
	PrivateHome  string
	ModelBaseURL string
	MCPURL       string
	Sentinel     string
}

type Kit interface {
	Detect(context.Context) (KitInfo, error)
	Build(context.Context, KitInput) (runtime.LaunchSpec, error)
}
