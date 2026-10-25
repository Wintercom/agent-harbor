// Package desktop 是桌面控制面装配与 Wails 绑定（研发 B）。
// 会话状态机在 orchestrator；本包只做校验、错误转换和组件注入。
package desktop

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/liyufei916/agent-harbor/internal/contracts"
	"github.com/liyufei916/agent-harbor/internal/gate"
	"github.com/liyufei916/agent-harbor/internal/policy"
	"github.com/liyufei916/agent-harbor/internal/runtime"
	"github.com/liyufei916/agent-harbor/internal/secrets"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// SessionService 与研发 A 的 orchestrator.SessionService 方法一致。
type SessionService interface {
	WithConnection(context.Context, func(context.Context) error) error
	Create(context.Context, contracts.SessionSpec) (contracts.SessionSnapshot, error)
	Get(context.Context, string) (contracts.SessionSnapshot, error)
	Stop(context.Context, string) (contracts.SessionSnapshot, error)
	List(context.Context) ([]contracts.SessionSummary, error)
	Destroy(context.Context, string) (contracts.SessionSnapshot, error)
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
	dataDir string

	mu       sync.Mutex
	sessions SessionService
	agents   AgentDetector
	rt       runtime.Runtime
	prefs    contracts.UIPrefs
	hosts    map[string]contracts.RememberedHost // key: workspace\x00host
	oauth    map[string]*oauthPending
	ready    bool
}

// NewApp 创建纯内存控制面（单测默认）。桌面进程请用 NewDesktopApp。
func NewApp() *App {
	return NewAppWithDataDir("")
}

// NewDesktopApp 使用本机 Application Support 落盘偏好/记住主机，并持久化凭据。
func NewDesktopApp() *App {
	return NewAppWithDataDir(defaultDataDir())
}

// NewAppWithDataDir 供测试注入临时目录；dataDir 空则纯内存凭据与状态。
func NewAppWithDataDir(dataDir string) *App {
	var store *secrets.Store
	if dataDir != "" {
		_ = os.MkdirAll(dataDir, 0700)
		if s, err := secrets.NewPersistentStore(filepath.Join(dataDir, "accounts")); err == nil {
			store = s
		}
	}
	if store == nil {
		store = secrets.NewStore()
	}
	pol := policy.NewBuilder(nil)
	app := &App{
		secrets:  store,
		policy:   pol,
		dataDir:  dataDir,
		sessions: unavailableSession{},
		prefs:    contracts.UIPrefs{SidebarCollapsed: false, AutoCollapseWorkbench: true},
		hosts:    make(map[string]contracts.RememberedHost),
	}
	app.loadPersistedState()
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

// SetRuntime 注入隔离 Runtime，供 GetRuntimeHealth 探测。
func (a *App) SetRuntime(rt runtime.Runtime) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.rt = rt
}

func (a *App) runtimeRef() runtime.Runtime {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.rt
}

func (a *App) orchestratorReady() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.ready
}

func hostMapKey(workspace, host string) string {
	return strings.TrimSpace(workspace) + "\x00" + strings.ToLower(strings.TrimSpace(host))
}

func (a *App) listRemembered(workspace string) []contracts.RememberedHost {
	a.mu.Lock()
	defer a.mu.Unlock()
	want := strings.TrimSpace(workspace)
	out := make([]contracts.RememberedHost, 0, len(a.hosts))
	for _, h := range a.hosts {
		if want != "" && h.Workspace != want {
			continue
		}
		out = append(out, h)
	}
	return out
}

func (a *App) rememberHost(host contracts.RememberedHost) {
	a.mu.Lock()
	if a.hosts == nil {
		a.hosts = make(map[string]contracts.RememberedHost)
	}
	a.hosts[hostMapKey(host.Workspace, host.Host)] = host
	a.mu.Unlock()
	a.savePersistedState()
}

func (a *App) revokeHost(workspace, host string) bool {
	a.mu.Lock()
	key := hostMapKey(workspace, host)
	if _, ok := a.hosts[key]; !ok {
		a.mu.Unlock()
		return false
	}
	delete(a.hosts, key)
	a.mu.Unlock()
	a.savePersistedState()
	return true
}

func (a *App) getPrefs() contracts.UIPrefs {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.prefs
}

func (a *App) setPrefs(p contracts.UIPrefs) contracts.UIPrefs {
	a.mu.Lock()
	a.prefs = p
	out := a.prefs
	a.mu.Unlock()
	a.savePersistedState()
	return out
}

// forwardEvent 不缓存或推导状态；准入由编排统一管理。
func (a *App) forwardEvent(ev contracts.Event) { a.EmitHarborEvent(ev) }

// EmitHarborEvent 把契约事件发到前端固定通道。
func (a *App) EmitHarborEvent(ev contracts.Event) {
	if app := application.Get(); app != nil {
		app.Event.Emit(contracts.HarborEventName, ev)
	}
}

// Shutdown 先停止编排。纯内存凭据在退出时清空；持久化 Store（Keychain/文件）保留跨重启。
func (a *App) Shutdown(ctx context.Context) error {
	err := a.sessionRef().Shutdown(ctx)
	if err == nil && a.dataDir == "" {
		a.secrets.Clear()
	}
	return err
}
