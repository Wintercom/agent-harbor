package main

import (
	"errors"
	"testing"

	"github.com/liyufei916/agent-harbor/internal/contracts"
)

func TestDesktopAPICreateSessionMapsOrchestratorError(t *testing.T) {
	app := NewApp()
	api := NewDesktopAPI(app)
	resp := api.CreateSession(contracts.SessionSpec{
		Workspace: "/tmp/ws", AgentID: "claude", ModelAccountID: "m", MCPIDs: []string{"t"}, Task: "hi",
	})
	if resp.Session != nil || resp.Error == nil || resp.Error.Code != contracts.ErrOrchestratorUnavailable {
		t.Fatalf("%+v", resp)
	}
}

func TestDesktopAPICreateSessionForwardsFake(t *testing.T) {
	app := NewApp()
	rec := &RecordingSession{}
	app.SetSessionService(rec)
	api := NewDesktopAPI(app)
	resp := api.CreateSession(contracts.SessionSpec{
		Workspace: "/demo/project", AgentID: "claude", ModelAccountID: "m1", MCPIDs: []string{"t1"}, Task: "hello",
	})
	if resp.Error != nil || resp.Session == nil || resp.Session.State != "provisioning" {
		t.Fatalf("%+v", resp)
	}
	if rec.LastCreate.Task != "hello" {
		t.Fatalf("forward=%+v", rec.LastCreate)
	}
}

func TestMapErrorSessionBusy(t *testing.T) {
	ae := mapError(errors.New(contracts.ErrSessionBusy))
	if ae.Code != contracts.ErrSessionBusy || ae.Retryable {
		t.Fatalf("%+v", ae)
	}
}
