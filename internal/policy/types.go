// 本文件按 docs/CONTRACT.md v1 定义，不包含业务实现。
package policy

import (
	"github.com/liyufei916/agent-harbor/internal/contracts"
)

type EndpointRule struct {
	Origin string
	Paths  []string
}

type PolicyRequest struct {
	Workspace string
	AgentID   string
	Model     contracts.AccountInfo
	MCP       contracts.AccountInfo
}

type PolicySnapshot struct {
	Workspace string
	MountMode string
	Model     EndpointRule
	MCP       EndpointRule
	MCPID     string
	Tools     []string
}
