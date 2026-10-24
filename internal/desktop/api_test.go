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
	app.forwardEvent(contracts.Event{
		Type:  contracts.EventTypeStateChanged,
		State: &contracts.SessionSnapshot{State: contracts.StateRunning},
	})
	api := NewDesktopAPI(app)
	model := api.ConnectModel(contracts.ConnectInput{ProfileID: "demo-model", Name: "n", APIKey: "secret"})
	mcp := api.ConnectMcp(contracts.ConnectInput{ProfileID: "demo-mcp", Name: "n", APIKey: "secret"})
	if model.Error == nil || model.Error.Code != contracts.ErrSessionBusy || model.Account != nil {
		t.Fatalf("model=%+v", model)
	}
	if mcp.Error == nil || mcp.Error.Code != contracts.ErrSessionBusy {
		t.Fatalf("mcp=%+v", mcp)
	}
	app.forwardEvent(contracts.Event{
		Type:  contracts.EventTypeStateChanged,
		State: &contracts.SessionSnapshot{State: contracts.StateStopped, CleanupComplete: true},
	})
	if app.sessionBusy() {
		t.Fatal("stopped session should release connect")
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
	info := api.AppInfo()
	if info["orchestrator"] != "wired" || info["gate"] != "ok" {
		t.Fatalf("%v", info)
	}
	agents := api.ListAgents()
	if agents.Error != nil || len(agents.Agents) != 1 || agents.Agents[0].ID != "claude" {
		t.Fatalf("%+v", agents)
	}
}

type recordingSession struct {
	last contracts.SessionSpec
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

func (r *recordingSession) Shutdown(context.Context) error { return nil }
