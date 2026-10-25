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
	_, _ = o.Get(context.Background(), old.SessionID)
	err = o.WithConnection(context.Background(), func(context.Context) error { t.Error("活动会话期间连接获准"); return nil })
	assertCode(t, err, "SESSION_BUSY")
	_, _ = o.Stop(context.Background(), current.SessionID)
}
