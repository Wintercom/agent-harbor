package desktop

import (
	"context"
	"strings"
	"testing"

	"github.com/liyufei916/agent-harbor/internal/contracts"
)

func TestCreateSessionBeforeWire(t *testing.T) {
	app := NewApp()
	api := NewDesktopAPI(app)
	resp := api.CreateSession(contracts.SessionSpec{
		Workspace: "/tmp/ws", AgentID: "claude", ModelAccountID: "m", MCPIDs: []string{"t"}, Task: "hi",
	})
	if resp.Session != nil || resp.Error == nil || resp.Error.Code != contracts.ErrOrchestratorUnavailable {
		t.Fatalf("%+v", resp)
	}
}

func TestCreateSessionForwardsInjectedService(t *testing.T) {
	app := NewApp()
	rec := &recordingSession{}
	app.SetSessionService(rec)
	api := NewDesktopAPI(app)
	resp := api.CreateSession(contracts.SessionSpec{
		Workspace: "/demo/project", AgentID: "claude", ModelAccountID: "m1", MCPIDs: []string{"t1"}, Task: "hello",
	})
	if resp.Error != nil || resp.Session == nil || resp.Session.State != "provisioning" {
		t.Fatalf("%+v", resp)
	}
	if rec.last.Task != "hello" {
		t.Fatalf("forward=%+v", rec.last)
	}
}

func TestConnectRejectedWhileSessionBusy(t *testing.T) {
	app := NewApp()
	app.SetSessionService(&recordingSession{connectionErr: contracts.NewAppError(contracts.ErrSessionBusy, "已有会话进行中", false)})
	api := NewDesktopAPI(app)
	model := api.ConnectModel(contracts.ConnectInput{ProfileID: "demo-model", Name: "n", APIKey: "secret"})
	mcp := api.ConnectMcp(contracts.ConnectInput{ProfileID: "demo-mcp", Name: "n", APIKey: "secret"})
	if model.Error == nil || model.Error.Code != contracts.ErrSessionBusy || model.Account != nil {
		t.Fatalf("model=%+v", model)
	}
	if mcp.Error == nil || mcp.Error.Code != contracts.ErrSessionBusy {
		t.Fatalf("mcp=%+v", mcp)
	}

}

func TestMapErrorPassesPublicAppError(t *testing.T) {
	src := contracts.NewAppError(contracts.ErrSessionBusy, "已有会话正在准备、运行或等待清理。", false)
	got := mapError(src)
	if got.Code != contracts.ErrSessionBusy || got.Message != src.Message {
		t.Fatalf("%+v", got)
	}
}

func TestAttachOrchestratorWiresRealService(t *testing.T) {
	app := NewApp()
	if err := app.AttachOrchestrator(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !app.orchestratorReady() {
		t.Fatal("expected orchestrator")
	}
	api := NewDesktopAPI(app)
	agents := api.ListAgents()
	if agents.Error != nil || len(agents.Agents) != 3 || agents.Agents[0].ID != "claude" || agents.Agents[1].ID != "codex" || agents.Agents[2].ID != "dsh" || agents.Agents[1].ModelProfileID != "codex-model" {
		t.Fatalf("%+v", agents)
	}
	health := api.GetRuntimeHealth()
	if health.Error != nil && health.Error.Code == contracts.ErrUnimplemented {
		t.Fatalf("Health 仍未接线: %+v", health)
	}
	// 非 Darwin25/arm64 时 Available=false 且带隔离错误，属预期；不得再报「尚未接线」
	if health.Reason == "健康检查尚未接线" {
		t.Fatalf("仍返回占位文案: %+v", health)
	}
}

func TestStartCancelMcpOAuth(t *testing.T) {
	app := NewApp()
	api := NewDesktopAPI(app)
	started := api.StartMcpOAuth(contracts.StartMcpOAuthInput{
		Name: "Notion", Origin: "https://auth.example.test",
	})
	if started.Error != nil || started.PendingID == "" || !strings.Contains(started.AuthURL, "127.0.0.1") {
		t.Fatalf("%+v", started)
	}
	cancelled := api.CancelMcpOAuth(started.PendingID)
	if cancelled.Error != nil {
		t.Fatalf("%+v", cancelled)
	}
	again := api.CancelMcpOAuth(started.PendingID)
	if again.Error == nil {
		t.Fatal("重复取消应失败")
	}
}

func TestListAccountsAndPrefs(t *testing.T) {
	app := NewApp()
	api := NewDesktopAPI(app)
	_, err := app.secrets.Put(context.Background(), contracts.AccountInfo{
		Kind: "model", Name: "demo", ProfileID: "demo-model", Host: "api.example.test",
	}, []byte("secret-key"))
	if err != nil {
		t.Fatal(err)
	}
	listed := api.ListAccounts("model")
	if listed.Error != nil || len(listed.Accounts) != 1 || listed.Accounts[0].Host != "api.example.test" {
		t.Fatalf("%+v", listed)
	}
	prefs := api.SetUIPrefs(contracts.UIPrefs{SidebarCollapsed: true, AutoCollapseWorkbench: false})
	if prefs.Error != nil || prefs.Prefs == nil || !prefs.Prefs.SidebarCollapsed {
		t.Fatalf("%+v", prefs)
	}
	again := api.GetUIPrefs()
	if again.Error != nil || again.Prefs == nil || again.Prefs.AutoCollapseWorkbench {
		t.Fatalf("%+v", again)
	}
	sessions := api.ListSessions()
	if sessions.Error == nil || sessions.Error.Code != contracts.ErrOrchestratorUnavailable {
		t.Fatalf("未装配编排时应失败: %+v", sessions)
	}
	app.SetSessionService(&recordingSession{})
	sessions = api.ListSessions()
	if sessions.Error != nil || len(sessions.Sessions) != 1 {
		t.Fatalf("%+v", sessions)
	}
}

type recordingSession struct {
	last          contracts.SessionSpec
	connectionErr error
	shutdownErr   error
}

type permissionSession struct {
	recordingSession
	input contracts.ResolvePermissionInput
}

func (s *permissionSession) ResolvePermission(_ context.Context, input contracts.ResolvePermissionInput) error {
	s.input = input
	return contracts.NewAppError(contracts.ErrInvalidArgument, "无挂起的扩权请求。", false)
}

func TestResolvePermissionForwardsToSessionService(t *testing.T) {
	app := NewApp()
	svc := &permissionSession{}
	app.SetSessionService(svc)
	input := contracts.ResolvePermissionInput{SessionID: "session", Target: "docs.example.test", Decision: "once"}
	response := NewDesktopAPI(app).ResolvePermission(input)
	if svc.input != input || response.Error == nil || response.Error.Code != contracts.ErrInvalidArgument {
		t.Fatalf("决策未转交：%+v %+v", svc.input, response)
	}
}

func (r *recordingSession) Create(_ context.Context, spec contracts.SessionSpec) (contracts.SessionSnapshot, error) {
	r.last = spec
	return contracts.SessionSnapshot{
		SessionID: "fake-session", State: "provisioning", Seq: 1,
		Workspace: spec.Workspace, AgentID: spec.AgentID,
	}, nil
}

func (r *recordingSession) Get(context.Context, string) (contracts.SessionSnapshot, error) {
	return contracts.SessionSnapshot{}, nil
}

func (r *recordingSession) Stop(context.Context, string) (contracts.SessionSnapshot, error) {
	return contracts.SessionSnapshot{State: "stopping"}, nil
}

func (r *recordingSession) List(context.Context) ([]contracts.SessionSummary, error) {
	return []contracts.SessionSummary{{SessionID: "fake-session", State: "running", Workspace: "/demo", AgentID: "claude"}}, nil
}

func (r *recordingSession) Destroy(_ context.Context, id string) (contracts.SessionSnapshot, error) {
	return contracts.SessionSnapshot{SessionID: id, State: "stopped", CleanupComplete: true}, nil
}

func (r *recordingSession) Shutdown(context.Context) error { return r.shutdownErr }

func (r *recordingSession) WithConnection(ctx context.Context, action func(context.Context) error) error {
	if r.connectionErr != nil {
		return r.connectionErr
	}
	return action(ctx)
}

func TestShutdownRetainsCredentialsUntilCleanupConfirmed(t *testing.T) {
	app := NewApp()
	svc := &recordingSession{shutdownErr: contracts.NewAppError(contracts.ErrCleanupFailed, "清理未确认完成", false)}
	app.SetSessionService(svc)
	_, _ = app.secrets.Put(context.Background(), contracts.AccountInfo{Kind: "model", Name: "fixture", ProfileID: "m", Host: "example.test"}, []byte("fixture"))
	if err := app.Shutdown(context.Background()); err == nil || app.secrets.Count() != 1 {
		t.Fatal("失败后无法保留资源重试")
	}
	svc.shutdownErr = nil
	if err := app.Shutdown(context.Background()); err != nil || app.secrets.Count() != 0 {
		t.Fatal("确认清理后没有清空凭据")
	}
}
