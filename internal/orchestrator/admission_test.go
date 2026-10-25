package orchestrator

import (
	"context"
	"github.com/liyufei916/agent-harbor/internal/contracts"
	"testing"
)

func TestConnectionAndSessionAdmissionIsAtomic(t *testing.T) {
	o := setup(t, newFixture(), Timeouts{}, nil)
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- o.WithConnection(context.Background(), func(context.Context) error { close(entered); <-release; return nil })
	}()
	<-entered
	_, err := o.Create(context.Background(), spec())
	assertCode(t, err, "SESSION_BUSY")
	err = o.WithConnection(context.Background(), func(context.Context) error { t.Error("不应开始第二连接"); return nil })
	assertCode(t, err, "SESSION_BUSY")
	close(release)
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	old := create(t, o)
	_, _ = o.Stop(context.Background(), old.SessionID)
	await(t, o, old.SessionID, func(s contracts.SessionSnapshot) bool { return s.CleanupComplete })
	current := create(t, o)
	awaitState(t, o, current.SessionID, contracts.StateRunning)
	_, _ = o.Get(context.Background(), old.SessionID)
	// running 期间允许新增账号（不影响已挂载工具）
	if err = o.WithConnection(context.Background(), func(context.Context) error { return nil }); err != nil {
		t.Fatalf("running 时应允许 Connect：%v", err)
	}
	_, _ = o.Stop(context.Background(), current.SessionID)
}
