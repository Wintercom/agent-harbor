package desktop

import (
	"context"
	"time"

	"github.com/liyufei916/agent-harbor/internal/contracts"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// DesktopAPI 是唯一绑定到 Wails 的业务服务。
// 方法返回统一 Response，不把原始 Go error 交给前端。
type DesktopAPI struct {
	app *App
}

// NewDesktopAPI 创建前端绑定对象。
func NewDesktopAPI(app *App) *DesktopAPI {
	return &DesktopAPI{app: app}
}

// ListAgents 返回本机 Agent 检测结果。
func (d *DesktopAPI) ListAgents() contracts.AgentsResponse {
	det := d.app.agentRef()
	if det == nil {
		return contracts.AgentsResponse{Agents: []contracts.AgentInfo{}}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	list, err := det.ListAgents(ctx)
	if err != nil {
		return contracts.AgentsResponse{Agents: []contracts.AgentInfo{}, Error: mapError(err)}
	}
	if list == nil {
		list = []contracts.AgentInfo{}
	}
	return contracts.AgentsResponse{Agents: list}
}

// SelectWorkspace 打开系统目录面板。
func (d *DesktopAPI) SelectWorkspace() contracts.WorkspaceResponse {
	wailsApp := application.Get()
	if wailsApp == nil {
		return contracts.WorkspaceResponse{Error: contracts.NewAppError(contracts.ErrInternal, "应用未就绪", false)}
	}
	path, err := wailsApp.Dialog.OpenFile().
		SetTitle("选择工作区").
		CanChooseDirectories(true).
		CanChooseFiles(false).
		PromptForSingleSelection()
	return workspaceResult(path, err)
}

func workspaceResult(path string, err error) contracts.WorkspaceResponse {
	if err != nil {
		return contracts.WorkspaceResponse{Error: mapError(err)}
	}
	if path == "" {
		return contracts.WorkspaceResponse{Cancelled: true}
	}
	return contracts.WorkspaceResponse{Path: path}
}

// ConnectModel 测试并保存模型账号。
func (d *DesktopAPI) ConnectModel(in contracts.ConnectInput) contracts.AccountResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	var meta contracts.AccountInfo
	err := d.app.sessionRef().WithConnection(ctx, func(call context.Context) error {
		var err error
		meta, err = d.app.conn.ConnectModel(call, in)
		return err
	})
	if err != nil {
		return contracts.AccountResponse{Error: mapError(err)}
	}
	return contracts.AccountResponse{Account: &meta}
}

// ConnectMcp 测试并保存 MCP 账号。
func (d *DesktopAPI) ConnectMcp(in contracts.ConnectInput) contracts.AccountResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	var meta contracts.AccountInfo
	err := d.app.sessionRef().WithConnection(ctx, func(call context.Context) error {
		var err error
		meta, err = d.app.conn.ConnectMcp(call, in)
		return err
	})
	if err != nil {
		return contracts.AccountResponse{Error: mapError(err)}
	}
	return contracts.AccountResponse{Account: &meta}
}

// CreateSession 转发给编排层。
func (d *DesktopAPI) CreateSession(spec contracts.SessionSpec) contracts.SessionResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	snap, err := d.app.sessionRef().Create(ctx, spec)
	if err != nil {
		return contracts.SessionResponse{Error: mapError(err)}
	}
	return contracts.SessionResponse{Session: &snap}
}

// GetSession 返回会话快照。
func (d *DesktopAPI) GetSession(id string) contracts.SessionResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	snap, err := d.app.sessionRef().Get(ctx, id)
	if err != nil {
		return contracts.SessionResponse{Error: mapError(err)}
	}
	return contracts.SessionResponse{Session: &snap}
}

// StopSession 请求停止会话。
func (d *DesktopAPI) StopSession(id string) contracts.SessionResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	snap, err := d.app.sessionRef().Stop(ctx, id)
	if err != nil {
		return contracts.SessionResponse{Error: mapError(err)}
	}
	return contracts.SessionResponse{Session: &snap}
}

func unimplemented() *contracts.AppError {
	return contracts.NewAppError(contracts.ErrUnimplemented, "该方法尚未实现，形状已按契约占位。", false)
}

// ListAccounts 列出已保存账号（V1.1；未接线时 UNIMPLEMENTED）。
func (d *DesktopAPI) ListAccounts(kind string) contracts.AccountsResponse {
	_ = kind
	return contracts.AccountsResponse{Accounts: []contracts.AccountInfo{}, Error: unimplemented()}
}

// RemoveAccount 删除账号（V1.1 占位）。
func (d *DesktopAPI) RemoveAccount(accountID string) contracts.EmptyResponse {
	_ = accountID
	return contracts.EmptyResponse{Error: unimplemented()}
}

// ListSessions 读取编排记录，不在桌面层重建会话状态。
func (d *DesktopAPI) ListSessions() contracts.SessionsResponse {
	service, ok := d.app.sessionRef().(interface {
		List(context.Context) ([]contracts.SessionSummary, error)
	})
	if !ok {
		return contracts.SessionsResponse{Sessions: []contracts.SessionSummary{}, Error: unimplemented()}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	sessions, err := service.List(ctx)
	if err != nil {
		return contracts.SessionsResponse{Sessions: []contracts.SessionSummary{}, Error: mapError(err)}
	}
	if sessions == nil {
		sessions = []contracts.SessionSummary{}
	}
	return contracts.SessionsResponse{Sessions: sessions}
}

// DestroySession 仅转发请求；终态和清理检查由编排负责。
func (d *DesktopAPI) DestroySession(sessionID string) contracts.SessionResponse {
	service, ok := d.app.sessionRef().(interface {
		Destroy(context.Context, string) (contracts.SessionSnapshot, error)
	})
	if !ok {
		return contracts.SessionResponse{Error: unimplemented()}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	snapshot, err := service.Destroy(ctx, sessionID)
	if err != nil {
		return contracts.SessionResponse{Error: mapError(err)}
	}
	return contracts.SessionResponse{Session: &snapshot}
}

// GetRuntimeHealth 请求编排执行实际隔离探针；失败只返回固定公开错误。
func (d *DesktopAPI) GetRuntimeHealth() contracts.RuntimeHealthResponse {
	service, ok := d.app.sessionRef().(interface{ Health(context.Context) error })
	if !ok {
		return contracts.RuntimeHealthResponse{Reason: "隔离探测服务尚未就绪", Error: unimplemented()}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := service.Health(ctx); err != nil {
		public := mapError(err)
		return contracts.RuntimeHealthResponse{Reason: public.Message, Error: public}
	}
	return contracts.RuntimeHealthResponse{Available: true}
}

// GetPermissionOverview 权限一览（V1.1 占位）。
func (d *DesktopAPI) GetPermissionOverview(sessionID string) contracts.PermissionOverviewResponse {
	_ = sessionID
	return contracts.PermissionOverviewResponse{Error: unimplemented()}
}

// ListRememberedHosts 已记住主机（V1.1 占位）。
func (d *DesktopAPI) ListRememberedHosts(workspace string) contracts.RememberedHostsResponse {
	_ = workspace
	return contracts.RememberedHostsResponse{Hosts: []contracts.RememberedHost{}, Error: unimplemented()}
}

// RevokeRememberedHost 撤销记住主机（V1.1 占位）。
func (d *DesktopAPI) RevokeRememberedHost(in contracts.RevokeHostInput) contracts.EmptyResponse {
	_ = in
	return contracts.EmptyResponse{Error: unimplemented()}
}

// ResolvePermission 将扩权决策转交给唯一的会话状态机。
func (d *DesktopAPI) ResolvePermission(in contracts.ResolvePermissionInput) contracts.EmptyResponse {
	resolver, ok := d.app.sessionRef().(interface {
		ResolvePermission(context.Context, contracts.ResolvePermissionInput) error
	})
	if !ok {
		return contracts.EmptyResponse{Error: unimplemented()}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return contracts.EmptyResponse{Error: mapError(resolver.ResolvePermission(ctx, in))}
}

// GetUIPrefs 读取 UI 偏好（V1.1 占位，返回默认值形状）。
func (d *DesktopAPI) GetUIPrefs() contracts.UIPrefsResponse {
	prefs := contracts.UIPrefs{SidebarCollapsed: false, AutoCollapseWorkbench: true}
	return contracts.UIPrefsResponse{Prefs: &prefs, Error: unimplemented()}
}

// SetUIPrefs 写入 UI 偏好（V1.1 占位）。
func (d *DesktopAPI) SetUIPrefs(prefs contracts.UIPrefs) contracts.UIPrefsResponse {
	return contracts.UIPrefsResponse{Prefs: &prefs, Error: unimplemented()}
}

// SwitchModel 将运行中模型账号切换转交编排，不自行管理会话状态。
func (d *DesktopAPI) SwitchModel(in contracts.SwitchModelInput) contracts.SessionResponse {
	switcher, ok := d.app.sessionRef().(interface {
		SwitchModel(context.Context, contracts.SwitchModelInput) (contracts.SessionSnapshot, error)
	})
	if !ok {
		return contracts.SessionResponse{Error: unimplemented()}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	snapshot, err := switcher.SwitchModel(ctx, in)
	if err != nil {
		return contracts.SessionResponse{Error: mapError(err)}
	}
	return contracts.SessionResponse{Session: &snapshot}
}

// StartMcpOAuth 启动 MCP OAuth（占位）。
func (d *DesktopAPI) StartMcpOAuth(in contracts.StartMcpOAuthInput) contracts.McpOAuthResponse {
	_ = in
	return contracts.McpOAuthResponse{Error: unimplemented()}
}

// CancelMcpOAuth 取消 OAuth（占位）。
func (d *DesktopAPI) CancelMcpOAuth(pendingID string) contracts.EmptyResponse {
	_ = pendingID
	return contracts.EmptyResponse{Error: unimplemented()}
}

// WritePty 写入 PTY（占位）。
func (d *DesktopAPI) WritePty(in contracts.PtyInput) contracts.EmptyResponse {
	writer, ok := d.app.sessionRef().(interface {
		WritePty(context.Context, contracts.PtyInput) error
	})
	if !ok {
		return contracts.EmptyResponse{Error: unimplemented()}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := writer.WritePty(ctx, in); err != nil {
		return contracts.EmptyResponse{Error: mapError(err)}
	}
	return contracts.EmptyResponse{}
}

// ResizePty 调整 PTY（占位）。
func (d *DesktopAPI) ResizePty(in contracts.PtyInput) contracts.EmptyResponse {
	resizer, ok := d.app.sessionRef().(interface {
		ResizePty(context.Context, contracts.PtyInput) error
	})
	if !ok {
		return contracts.EmptyResponse{Error: unimplemented()}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := resizer.ResizePty(ctx, in); err != nil {
		return contracts.EmptyResponse{Error: mapError(err)}
	}
	return contracts.EmptyResponse{}
}
