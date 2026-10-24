// 本文件按 docs/CONTRACT.md v1 定义，不包含业务实现。
package gate

import (
	"github.com/liyufei916/agent-harbor/internal/policy"
)

type GateSpec struct {
	SessionID      string
	Policy         policy.PolicySnapshot
	ModelAccountID string
	MCPAccountID   string
}

type GateBinding struct {
	ModelBaseURL string
	MCPURL       string
	Sentinel     string
	AllowedTCP   []string // 精确的 127.0.0.1:port，供 Runtime 强制限制
}
