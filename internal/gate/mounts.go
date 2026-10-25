package gate

import (
	"context"
	"fmt"
	"github.com/liyufei916/agent-harbor/internal/policy"
)

// 只接收编排已授权的挂载，不从执行面请求构造上游或账号。
func (h *Harbor) mcpMounts(ctx context.Context, spec GateSpec) ([]policy.MCPMount, error) {
	mounts := spec.Policy.MCPMounts
	if len(mounts) == 0 && spec.MCPAccountID != "" {
		if spec.Policy.MCPID != "" && spec.Policy.MCPID != spec.MCPAccountID {
			return nil, fmt.Errorf("gate: inconsistent mcp")
		}
		mounts = []policy.MCPMount{{AccountID: spec.MCPAccountID, Endpoint: spec.Policy.MCP, Tools: spec.Policy.Tools}}
	} else if spec.MCPAccountID != "" && (len(mounts) != 1 || mounts[0].AccountID != spec.MCPAccountID) {
		return nil, fmt.Errorf("gate: ambiguous mcp")
	}
	if len(mounts) > 8 {
		return nil, fmt.Errorf("gate: too many mcp mounts")
	}
	seen := map[string]bool{}
	result := make([]policy.MCPMount, 0, len(mounts))
	for _, mount := range mounts {
		if mount.AccountID == "" || seen[mount.AccountID] {
			return nil, fmt.Errorf("gate: invalid mcp mount")
		}
		seen[mount.AccountID] = true
		account, err := h.secrets.Lookup(ctx, mount.AccountID)
		if err != nil {
			return nil, err
		}
		if account.Kind != "mcp" {
			return nil, fmt.Errorf("gate: invalid mcp account")
		}
		mount.Endpoint.Paths = append([]string{}, mount.Endpoint.Paths...)
		mount.Tools = append([]string{}, mount.Tools...)
		result = append(result, mount)
	}
	return result, nil
}
