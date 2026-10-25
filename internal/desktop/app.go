// Package desktop 是桌面控制面装配与 Wails 绑定（研发 B）。
// 会话状态机在 orchestrator；本包只做校验、错误转换和组件注入。
package desktop

import (
	"context"
	"sync"

	"github.com/liyufei916/agent-harbor/internal/contracts"
	"github.com/liyufei916/agent-harbor/internal/gate"
	"github.com/liyufei916/agent-harbor/internal/policy"
	"github.com/liyufei916/agent-harbor/internal/secrets"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// SessionService 与研发 A 的 orchestrator.SessionService 方法一致。
type SessionService interface {
	WithConnection(context.Context, func(context.Context) error) error
	Create(context.Context, contracts.SessionSpec) (contracts.SessionSnapshot, error)
	Get(context.Context, string) (contracts.SessionSnapshot, error)
	Stop(context.Context, string) (contracts.SessionSnapshot, error)
	Shutdown(context.Context) error
}

// AgentDetector 由 Kit 注册表实现。
type AgentDetector interface {
	ListAgents(context.Context) ([]contracts.AgentInfo, error)
}

// App 持有控制面组件。前端只看到 DesktopAPI。
type App struct {
	secrets *secrets.Store
	policy  *policy.Builder
	gate    *gate.Harbor
	conn    *gate.Connections

	mu       sync.Mutex
	sessions SessionService
	agents   AgentDetector
	ready    bool
}

// NewApp 创建 Gate、Policy 与内存凭据。编排在 AttachOrchestrator 中注入。
func NewApp() *App {
	store := secrets.NewStore()
	pol := policy.NewBuilder(nil)
	app := &App{
		secrets:  store,
		policy:   pol,
		sessions: unavailableSession{},
	}
	app.gate = gate.New(store, pol)
	app.conn = gate.NewConnections(store, pol)
	return app
}

// SetSessionService 替换会话服务。nil 表示尚未接入编排。
func (a *App) SetSessionService(s SessionService) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if s == nil {
		a.sessions = unavailableSession{}
		a.ready = false
		return
	}
	a.sessions = s
	a.ready = true
}

// SetAgentDetector 替换 Agent 检测。
func (a *App) SetAgentDetector(d AgentDetector) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.agents = d
}

func (a *App) sessionRef() SessionService {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.sessions
}

func (a *App) agentRef() AgentDetector {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.agents
}

func (a *App) orchestratorReady() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.ready
}

// forwardEvent 不缓存或推导状态；准入由编排统一管理。
func (a *App) forwardEvent(ev contracts.Event) { a.EmitHarborEvent(ev) }

// EmitHarborEvent 把契约事件发到前端固定通道。
func (a *App) EmitHarborEvent(ev contracts.Event) {
	if app := application.Get(); app != nil {
		app.Event.Emit(contracts.HarborEventName, ev)
	}
}

// Shutdown 先停止编排，再清空内存凭据。
func (a *App) Shutdown(ctx context.Context) error {
	err := a.sessionRef().Shutdown(ctx)
	if err == nil {
		a.secrets.Clear()
	}
	return err
}
