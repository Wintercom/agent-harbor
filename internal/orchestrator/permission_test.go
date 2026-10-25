package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/liyufei916/agent-harbor/internal/contracts"
)

func waitPermission(t *testing.T, f *fixture, target string) {
	t.Helper()
	deadline := time.After(time.Second)
	for {
		f.mu.Lock()
		found := false
		for _, event := range f.events {
			if event.Permission != nil && event.Permission.Target == target {
				found = true
				break
			}
		}
		f.mu.Unlock()
		if found {
			return
		}
		select {
		case <-deadline:
			t.Fatal("未收到扩权事件")
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

func TestPermissionDecisionKeepsSessionRunning(t *testing.T) {
	for _, decision := range []string{"deny", "once", "project"} {
		t.Run(decision, func(t *testing.T) {
			f := newFixture()
			o := setup(t, f, Timeouts{}, nil)
			s := create(t, o)
			awaitState(t, o, s.SessionID, contracts.StateRunning)
			result := make(chan string, 1)
			go func() {
				answer, err := o.RequestPermission(context.Background(), s.SessionID, contracts.PermissionNeeded{Target: "docs.example.test", Action: "connect"})
				if err != nil {
					result <- "error"
					return
				}
				result <- answer
			}()
			waitPermission(t, f, "docs.example.test")
			current, _ := o.Get(context.Background(), s.SessionID)
			if current.State != contracts.StateRunning {
				t.Fatalf("扩权改变了状态：%+v", current)
			}
			input := contracts.ResolvePermissionInput{SessionID: s.SessionID, Target: "docs.example.test", Decision: decision}
			if err := o.ResolvePermission(context.Background(), input); err != nil {
				t.Fatal(err)
			}
			select {
			case answer := <-result:
				if answer != decision {
					t.Fatalf("决策=%s", answer)
				}
			case <-time.After(time.Second):
				t.Fatal("决策未解除等待")
			}
			assertCode(t, o.ResolvePermission(context.Background(), input), "INVALID_ARGUMENT")
		})
	}
}

func TestPermissionStopAndRequestCancellation(t *testing.T) {
	for _, stop := range []bool{false, true} {
		f := newFixture()
		o := setup(t, f, Timeouts{}, nil)
		s := create(t, o)
		awaitState(t, o, s.SessionID, contracts.StateRunning)
		ctx, cancel := context.WithCancel(context.Background())
		finished := make(chan error, 1)
		go func() {
			_, err := o.RequestPermission(ctx, s.SessionID, contracts.PermissionNeeded{Target: "docs.example.test", Action: "connect"})
			finished <- err
		}()
		waitPermission(t, f, "docs.example.test")
		if stop {
			_, _ = o.Stop(context.Background(), s.SessionID)
		} else {
			cancel()
		}
		select {
		case err := <-finished:
			assertCode(t, err, "CANCELLED")
		case <-time.After(time.Second):
			t.Fatal("取消后仍挂起")
		}
		cancel()
		assertCode(t, o.ResolvePermission(context.Background(), contracts.ResolvePermissionInput{SessionID: s.SessionID, Target: "docs.example.test", Decision: "once"}), "INVALID_ARGUMENT")
	}
}

func TestPermissionRejectsBroadTargetsAndUnknownDecisions(t *testing.T) {
	f := newFixture()
	o := setup(t, f, Timeouts{}, nil)
	s := create(t, o)
	awaitState(t, o, s.SessionID, contracts.StateRunning)
	for _, target := range []string{"*", "*.example.test", "https://example.test/path", "example.test:443", "localhost", "127.0.0.1", "192.168.1.1", "example.test\n", "Example.test", "a..test", sentinel + ".test"} {
		_, err := o.RequestPermission(context.Background(), s.SessionID, contracts.PermissionNeeded{Target: target, Action: "connect"})
		assertCode(t, err, "INVALID_ARGUMENT")
	}
	assertCode(t, o.ResolvePermission(context.Background(), contracts.ResolvePermissionInput{SessionID: s.SessionID, Target: "docs.example.test", Decision: "all"}), "INVALID_ARGUMENT")
}

func TestPermissionDuplicateAndOtherSessionCannotResolve(t *testing.T) {
	f := newFixture()
	o := setup(t, f, Timeouts{}, nil)
	s := create(t, o)
	awaitState(t, o, s.SessionID, contracts.StateRunning)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	request := contracts.PermissionNeeded{Target: "docs.example.test", Action: "connect"}
	go func() { _, err := o.RequestPermission(ctx, s.SessionID, request); result <- err }()
	waitPermission(t, f, request.Target)
	_, err := o.RequestPermission(ctx, s.SessionID, request)
	assertCode(t, err, "INVALID_ARGUMENT")
	assertCode(t, o.ResolvePermission(ctx, contracts.ResolvePermissionInput{SessionID: "another-session", Target: request.Target, Decision: "once"}), "SESSION_NOT_FOUND")
	select {
	case <-result:
		t.Fatal("其他会话影响了本次等待")
	default:
	}
	cancel()
	select {
	case err := <-result:
		assertCode(t, err, "CANCELLED")
	case <-time.After(time.Second):
		t.Fatal("请求未取消")
	}
}
