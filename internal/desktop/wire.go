package desktop

import (
	"context"
	"os"
	"strings"

	"github.com/liyufei916/agent-harbor/internal/contracts"
	"github.com/liyufei916/agent-harbor/internal/kits"
	"github.com/liyufei916/agent-harbor/internal/kits/claude"
	"github.com/liyufei916/agent-harbor/internal/orchestrator"
	"github.com/liyufei916/agent-harbor/internal/runtime"
)

// AttachOrchestrator 用研发 A 的 Runtime、Claude Kit 与 Orchestrator 接上本进程的 Gate/Policy/Secrets。
// appCtx 是应用级 context，会话任务从它派生；不要传入已取消的请求 context。
// 模型名来自 HARBOR_CLAUDE_MODEL，缺省为宿主机固定占位，不从任务正文推导。
func (a *App) AttachOrchestrator(appCtx context.Context) error {
	if appCtx == nil {
		appCtx = context.Background()
	}
	model := strings.TrimSpace(os.Getenv("HARBOR_CLAUDE_MODEL"))
	if model == "" {
		model = "claude-sonnet-4-5"
	}
	kit, err := claude.New(os.Getenv("HARBOR_CLAUDE_BIN"), model)
	if err != nil {
		return err
	}
	reg, err := kits.NewRegistry(map[string]kits.Kit{"claude": kit})
	if err != nil {
		return err
	}
	svc, err := orchestrator.New(appCtx, orchestrator.Dependencies{
		Runtime: runtime.NewMacOS(),
		Kits:    map[string]kits.Kit{"claude": kit},
		Catalog: a.secrets,
		Policy:  a.policy,
		Gate:    a.gate,
	}, a.forwardEvent, orchestrator.Timeouts{})
	if err != nil {
		return err
	}
	a.SetAgentDetector(registryDetector{reg: reg})
	a.SetSessionService(svc)
	return nil
}

type registryDetector struct {
	reg *kits.Registry
}

func (d registryDetector) ListAgents(ctx context.Context) ([]contracts.AgentInfo, error) {
	infos, err := d.reg.DetectAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]contracts.AgentInfo, 0, len(infos))
	for _, info := range infos {
		out = append(out, info.Agent)
	}
	return out, nil
}
