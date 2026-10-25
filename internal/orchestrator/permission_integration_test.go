package orchestrator

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/liyufei916/agent-harbor/internal/contracts"
	"github.com/liyufei916/agent-harbor/internal/gate"
	"github.com/liyufei916/agent-harbor/internal/kits"
	"github.com/liyufei916/agent-harbor/internal/policy"
	"github.com/liyufei916/agent-harbor/internal/secrets"
)

type mountedGate struct {
	*gate.Harbor
	bindings chan gate.GateBinding
}

func (g mountedGate) Start(ctx context.Context, spec gate.GateSpec, audit func(contracts.AuditData), fatal func(error)) (gate.GateBinding, error) {
	binding, err := g.Harbor.Start(ctx, spec, audit, fatal)
	if err == nil {
		g.bindings <- binding
	}
	return binding, err
}

type redirectPolicy struct{ snapshot policy.PolicySnapshot }

func (p redirectPolicy) Build(context.Context, policy.PolicyRequest) (policy.PolicySnapshot, error) {
	return p.snapshot, nil
}

// 实际 Gateway HTTP 请求须在编排层挂起，并由该会话的决策解除等待。
// 跨域重定向仍禁止转发凭据；批准记录不能替代受信路由及 DNS 检查。
func TestGateRedirectPermissionReachesOrchestrator(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://docs.example.test/next")
		w.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer upstream.Close()
	store := secrets.NewStore()
	model, err := store.Put(context.Background(), contracts.AccountInfo{Kind: "model", Name: "fixture", ProfileID: "fixture", Host: "127.0.0.1"}, []byte("fixture-model-key"))
	if err != nil {
		t.Fatal(err)
	}
	f := newFixture()
	g := mountedGate{Harbor: gate.New(store, policy.NewBuilder(nil)), bindings: make(chan gate.GateBinding, 1)}
	p := redirectPolicy{policy.PolicySnapshot{Workspace: t.TempDir(), MountMode: "direct", Model: policy.EndpointRule{Origin: upstream.URL, Paths: []string{"/v1/messages"}}}}
	o, err := New(context.Background(), Dependencies{Runtime: f, Kits: map[string]kits.Kit{"claude": f}, Catalog: store, Policy: p, Gate: g}, f.record, Timeouts{})
	if err != nil {
		t.Fatal(err)
	}
	defer o.Shutdown(context.Background())
	input := spec()
	input.ModelAccountID = model
	input.MCPIDs = nil
	s, err := o.Create(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	awaitState(t, o, s.SessionID, contracts.StateRunning)
	binding := <-g.bindings
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	request, _ := http.NewRequestWithContext(ctx, "POST", binding.ModelBaseURL+"/v1/messages", strings.NewReader("{}"))
	request.Header.Set("Authorization", "Bearer "+binding.Sentinel)
	result := make(chan int, 1)
	go func() {
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			result <- 0
			return
		}
		io.Copy(io.Discard, response.Body)
		response.Body.Close()
		result <- response.StatusCode
	}()
	waitPermission(t, f, "docs.example.test")
	select {
	case <-result:
		t.Fatal("尚未决策就结束了扩权等待")
	default:
	}
	if err := o.ResolvePermission(ctx, contracts.ResolvePermissionInput{SessionID: s.SessionID, Target: "docs.example.test", Decision: "deny"}); err != nil {
		t.Fatal(err)
	}
	select {
	case status := <-result:
		if status != http.StatusBadGateway {
			t.Fatalf("status=%d", status)
		}
	case <-ctx.Done():
		t.Fatal("拒绝后请求未恢复")
	}
}
