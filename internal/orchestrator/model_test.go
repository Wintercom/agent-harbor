package orchestrator

import (
	"context"
	"testing"

	"github.com/liyufei916/agent-harbor/internal/contracts"
)

func TestSwitchModelKeepsSessionAndWorkspace(t *testing.T) {
	f := newFixture()
	o := setup(t, f, Timeouts{}, nil)
	s := create(t, o)
	before := awaitState(t, o, s.SessionID, contracts.StateRunning)
	after, err := o.SwitchModel(context.Background(), contracts.SwitchModelInput{SessionID: s.SessionID, ModelAccountID: "model-next"})
	if err != nil {
		t.Fatal(err)
	}
	if before.SessionID != after.SessionID || before.Workspace != after.Workspace || after.State != contracts.StateRunning {
		t.Fatalf("切换重建会话：%+v", after)
	}
	f.mu.Lock()
	switched := f.switchedModel
	f.mu.Unlock()
	if switched != "model-next" {
		t.Fatalf("未更新 Gate：%s", switched)
	}
}

func TestSwitchModelRejectsUnknownAndWrongKind(t *testing.T) {
	f := newFixture()
	o := setup(t, f, Timeouts{}, nil)
	s := create(t, o)
	awaitState(t, o, s.SessionID, contracts.StateRunning)
	for _, test := range []struct{ account, code string }{{"unknown", "ACCOUNT_NOT_FOUND"}, {"mcp", "INVALID_ARGUMENT"}, {"", "INVALID_ARGUMENT"}} {
		_, err := o.SwitchModel(context.Background(), contracts.SwitchModelInput{SessionID: s.SessionID, ModelAccountID: test.account})
		assertCode(t, err, test.code)
	}
	_, err := o.SwitchModel(context.Background(), contracts.SwitchModelInput{SessionID: "unknown", ModelAccountID: "model-next"})
	assertCode(t, err, "SESSION_NOT_FOUND")
	_, _ = o.Stop(context.Background(), s.SessionID)
	await(t, o, s.SessionID, func(s contracts.SessionSnapshot) bool { return s.CleanupComplete })
	_, err = o.SwitchModel(context.Background(), contracts.SwitchModelInput{SessionID: s.SessionID, ModelAccountID: "model-next"})
	assertCode(t, err, "INVALID_ARGUMENT")
}

func TestSwitchModelGateFailureRetainsOldAccount(t *testing.T) {
	f := newFixture()
	o := setup(t, f, Timeouts{}, nil)
	s := create(t, o)
	awaitState(t, o, s.SessionID, contracts.StateRunning)
	f.mu.Lock()
	f.failAt = "switch-model"
	f.mu.Unlock()
	_, err := o.SwitchModel(context.Background(), contracts.SwitchModelInput{SessionID: s.SessionID, ModelAccountID: "model-next"})
	assertCode(t, err, "INTERNAL_ERROR")
	o.mu.Lock()
	account := o.sessions[s.SessionID].spec.ModelAccountID
	o.mu.Unlock()
	if account != "model" {
		t.Fatalf("失败后账号被改写：%s", account)
	}
}
