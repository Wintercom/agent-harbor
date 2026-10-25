package desktop

import (
	"context"
	"github.com/liyufei916/agent-harbor/internal/contracts"
	"testing"
)

type fullSession struct {
	recordingSession
	err         error
	destroyed   string
	healthCalls int
}

func (s *fullSession) List(context.Context) ([]contracts.SessionSummary, error) {
	return []contracts.SessionSummary{{SessionID: "one"}, {SessionID: "two"}}, s.err
}
func (s *fullSession) Destroy(_ context.Context, id string) (contracts.SessionSnapshot, error) {
	s.destroyed = id
	return contracts.SessionSnapshot{SessionID: id, CleanupComplete: true}, s.err
}
func (s *fullSession) Health(context.Context) error { s.healthCalls++; return s.err }
func TestSessionCapabilitiesForwardToOrchestrator(t *testing.T) {
	app := NewApp()
	svc := &fullSession{}
	app.SetSessionService(svc)
	api := NewDesktopAPI(app)
	list := api.ListSessions()
	if list.Error != nil || len(list.Sessions) != 2 {
		t.Fatalf("%+v", list)
	}
	destroyed := api.DestroySession("two")
	if destroyed.Error != nil || destroyed.Session == nil || svc.destroyed != "two" {
		t.Fatalf("%+v", destroyed)
	}
	health := api.GetRuntimeHealth()
	if health.Error != nil || !health.Available || svc.healthCalls != 1 {
		t.Fatalf("%+v", health)
	}
	svc.err = contracts.NewAppError(contracts.ErrIsolationUnavailable, "隔离不可用", false)
	health = api.GetRuntimeHealth()
	if health.Available || health.Reason == "" || health.Error == nil || health.Error.Code != contracts.ErrIsolationUnavailable {
		t.Fatalf("%+v", health)
	}
	if api.ListSessions().Error == nil || api.DestroySession("two").Error == nil {
		t.Fatal("服务错误丢失")
	}
}
