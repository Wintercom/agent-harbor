package main

import (
	"context"
	"errors"
	"sync"

	"github.com/liyufei916/agent-harbor/internal/contracts"
	"github.com/liyufei916/agent-harbor/internal/gate"
	"github.com/liyufei916/agent-harbor/internal/policy"
	"github.com/liyufei916/agent-harbor/internal/secrets"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// SessionService 由研发 A 的 Orchestrator 实现；B 只注入与转发。
type SessionService interface {
	Create(context.Context, contracts.SessionSpec) (contracts.SessionSnapshot, error)
	Get(context.Context, string) (contracts.SessionSnapshot, error)
	Stop(context.Context, string) (contracts.SessionSnapshot, error)
	Shutdown(context.Context) error
}

// AgentDetector 由研发 A 的 Kit 注册表实现；未注入时 ListAgents 返回空可用列表。
type AgentDetector interface {
	ListAgents(context.Context) ([]contracts.AgentInfo, error)
}

// App 拥有控制面组件（研发 B）。前端只看到 DesktopAPI。
type App struct {
	secrets *secrets.Store
	policy  *policy.Builder
	gate    *gate.Harbor
	conn    *gate.Connections

	mu       sync.Mutex
	sessions SessionService
	agents   AgentDetector
}

// NewApp 装配 B 负责的控制面服务。
func NewApp() *App {
	store := secrets.NewStore()
	pol := policy.NewBuilder(nil)
	app := &App{
		secrets:  store,
		policy:   pol,
		sessions: UnavailableSession{},
	}
	app.gate = gate.New(store, pol)
	app.conn = gate.NewConnections(store, pol)
	return app
}

// SetSessionService 注入 A 的编排实现。
func (a *App) SetSessionService(s SessionService) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if s == nil {
		a.sessions = UnavailableSession{}
		return
	}
	a.sessions = s
}

// SetAgentDetector 注入 A 的 Agent 检测。
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

// EmitHarborEvent 将契约事件发到前端固定通道。
func (a *App) EmitHarborEvent(ev contracts.Event) {
	if app := application.Get(); app != nil {
		app.Event.Emit(contracts.HarborEventName, ev)
	}
}

// HostDeps 供 A 构造 Orchestrator，不绑定到前端。
type HostDeps struct {
	Secrets     *secrets.Store
	Policy      *policy.Builder
	Gate        *gate.Harbor
	Connections *gate.Connections
	EmitEvent   func(contracts.Event)
}

// HostDepsOf 返回 B 拥有的控制面依赖。
func HostDepsOf(app *App) HostDeps {
	return HostDeps{
		Secrets:     app.secrets,
		Policy:      app.policy,
		Gate:        app.gate,
		Connections: app.conn,
		EmitEvent:   app.EmitHarborEvent,
	}
}

// WireSessionService 在 wails 启动前由 A 的工厂注入编排。
func WireSessionService(app *App, factory func(HostDeps) SessionService) {
	if factory == nil {
		app.SetSessionService(UnavailableSession{})
		return
	}
	app.SetSessionService(factory(HostDepsOf(app)))
}

func (a *App) shutdown(ctx context.Context) error {
	err := a.sessionRef().Shutdown(ctx)
	a.secrets.Clear()
	return err
}

func mapError(err error) *contracts.AppError {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, gate.ErrInvalidConnectInput):
		return contracts.NewAppError(contracts.ErrInvalidArgument, "输入不合法", false)
	case errors.Is(err, gate.ErrUpstreamAuth):
		return contracts.NewAppError(contracts.ErrUpstreamAuthFailed, "上游认证失败", false)
	case errors.Is(err, gate.ErrUpstreamTimeout):
		return contracts.NewAppError(contracts.ErrUpstreamTimeout, "上游连接超时", true)
	case errors.Is(err, gate.ErrUpstreamDown):
		return contracts.NewAppError(contracts.ErrUpstreamUnavailable, "上游不可用", true)
	case errors.Is(err, gate.ErrCancelled):
		return contracts.NewAppError(contracts.ErrCancelled, "操作已取消", false)
	case errors.Is(err, secrets.ErrNotFound):
		return contracts.NewAppError(contracts.ErrAccountNotFound, "账号不存在", false)
	case errors.Is(err, ErrOrchestratorDown):
		return contracts.NewAppError(contracts.ErrOrchestratorUnavailable, "编排服务尚未就绪", false)
	case errors.Is(err, policy.ErrInvalidWorkspace):
		return contracts.NewAppError(contracts.ErrWorkspaceDenied, "工作区不被允许", false)
	default:
		msg := err.Error()
		switch msg {
		case contracts.ErrSessionBusy:
			return contracts.NewAppError(contracts.ErrSessionBusy, "已有会话进行中", false)
		case contracts.ErrSessionNotFound:
			return contracts.NewAppError(contracts.ErrSessionNotFound, "会话不存在", false)
		case contracts.ErrAgentUnsupported:
			return contracts.NewAppError(contracts.ErrAgentUnsupported, "Agent 不可用", false)
		case contracts.ErrIsolationUnavailable:
			return contracts.NewAppError(contracts.ErrIsolationUnavailable, "隔离能力不可用", false)
		case contracts.ErrCleanupFailed:
			return contracts.NewAppError(contracts.ErrCleanupFailed, "清理未完成", false)
		default:
			return contracts.NewAppError(contracts.ErrInternal, "内部错误", false)
		}
	}
}
