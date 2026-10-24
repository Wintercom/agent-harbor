package main

import (
	"context"
	"time"

	"github.com/Wintercom/agent-harbor/internal/contracts"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// DesktopAPI 是唯一绑定到 Wails 的业务服务（研发 B）。
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
	if err != nil || path == "" {
		return contracts.WorkspaceResponse{Cancelled: true}
	}
	return contracts.WorkspaceResponse{Path: path}
}

// ConnectModel 测试并保存模型账号。
func (d *DesktopAPI) ConnectModel(in contracts.ConnectInput) contracts.AccountResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	meta, err := d.app.conn.ConnectModel(ctx, in)
	if err != nil {
		return contracts.AccountResponse{Error: mapError(err)}
	}
	return contracts.AccountResponse{Account: &meta}
}

// ConnectMcp 测试并保存 MCP 账号。
func (d *DesktopAPI) ConnectMcp(in contracts.ConnectInput) contracts.AccountResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	meta, err := d.app.conn.ConnectMcp(ctx, in)
	if err != nil {
		return contracts.AccountResponse{Error: mapError(err)}
	}
	return contracts.AccountResponse{Account: &meta}
}

// CreateSession 转发至编排层。
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

// AppInfo 供诊断显示接线状态（不含密钥）。
func (d *DesktopAPI) AppInfo() map[string]string {
	orch := "unavailable"
	if _, ok := d.app.sessionRef().(UnavailableSession); !ok {
		orch = "injected"
	}
	return map[string]string{
		"product":      "Agent Harbor",
		"role":         "control-plane",
		"gate":         d.app.gate.Health(),
		"orchestrator": orch,
		"framework":    "wails-v3",
	}
}
