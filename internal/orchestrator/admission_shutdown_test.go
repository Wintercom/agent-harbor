package orchestrator

import (
	"context"
	"github.com/liyufei916/agent-harbor/internal/contracts"
	"testing"
	"time"
)

func TestShutdownCancelsAndWaitsForConnection(t *testing.T) {
	o := setup(t, newFixture(), Timeouts{}, nil)
	entered, canceled, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- o.WithConnection(context.Background(), func(ctx context.Context) error {
			close(entered)
			<-ctx.Done()
			close(canceled)
			<-release
			return ctx.Err()
		})
	}()
	<-entered
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	err := o.Shutdown(ctx)
	assertCode(t, err, "CLEANUP_FAILED")
	<-canceled
	close(release)
	<-done
	if err = o.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, err = o.Create(context.Background(), spec())
	assertCode(t, err, "CANCELLED")
}

func TestShutdownCancelsEverySessionBeforeWaiting(t *testing.T) {
	o := setup(t, newFixture(), Timeouts{}, nil)
	firstCtx, firstCancel := context.WithCancel(context.Background())
	secondCtx, secondCancel := context.WithCancel(context.Background())
	defer firstCancel()
	defer secondCancel()
	for id, call := range map[string]struct {
		ctx    context.Context
		cancel context.CancelFunc
	}{"first": {firstCtx, firstCancel}, "second": {secondCtx, secondCancel}} {
		o.sessions[id] = &session{snapshot: contracts.SessionSnapshot{SessionID: id, State: contracts.StateRunning}, ctx: call.ctx, cancel: call.cancel, cleanupDone: make(chan struct{})}
	}
	defer func() {
		for _, s := range o.sessions {
			s.snapshot.CleanupComplete = true
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	assertCode(t, o.Shutdown(ctx), "CLEANUP_FAILED")
	if firstCtx.Err() == nil || secondCtx.Err() == nil {
		t.Fatal("一个会话清理阻塞时，其余会话仍在运行")
	}
	// 本测试没有运行清理器，手工完成，供 setup 的退出钩子收尾。
	for _, s := range o.sessions {
		s.snapshot.CleanupComplete = true
	}
}
