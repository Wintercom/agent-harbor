package desktop

import (
	"context"
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
	if agents.Error != nil || len(agents.Agents) != 2 || agents.Agents[0].ID != "claude" || agents.Agents[1].ID != "dsh" || agents.Agents[1].ModelProfileID != "dsh-model" {
		t.Fatalf("%+v", agents)
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

type modelSession struct {
	recordingSession
	input contracts.SwitchModelInput
}

func (s *modelSession) SwitchModel(_ context.Context, input contracts.SwitchModelInput) (contracts.SessionSnapshot, error) {
	s.input = input
	return contracts.SessionSnapshot{SessionID: input.SessionID, State: contracts.StateRunning}, nil
}

func TestSwitchModelForwardsToSessionService(t *testing.T) {
	app := NewApp()
	svc := &modelSession{}
	app.SetSessionService(svc)
	input := contracts.SwitchModelInput{SessionID: "session", ModelAccountID: "new-model"}
	response := NewDesktopAPI(app).SwitchModel(input)
	if svc.input != input || response.Error != nil || response.Session == nil || response.Session.SessionID != input.SessionID {
		t.Fatalf("切换未转交：%+v", response)
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
