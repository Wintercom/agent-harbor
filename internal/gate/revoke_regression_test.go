package gate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/liyufei916/agent-harbor/internal/contracts"
	"github.com/liyufei916/agent-harbor/internal/policy"
	"github.com/liyufei916/agent-harbor/internal/secrets"
)

func TestRevokeCancelsAndWaitsForInflightHTTP(t *testing.T) {
	entered, canceled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		select {
		case <-r.Context().Done():
			close(canceled)
		case <-release:
		}
	}))
	defer up.Close()
	var once sync.Once
	releaseNow := func() { once.Do(func() { close(release) }) }
	defer releaseNow()
	store := secrets.NewStore()
	ctx := context.Background()
	model, _ := store.Put(ctx, contracts.AccountInfo{Kind: "model", Name: "m", ProfileID: "m", Host: "127.0.0.1"}, []byte("fixture-model"))
	mcp, _ := store.Put(ctx, contracts.AccountInfo{Kind: "mcp", Name: "t", ProfileID: "t", Host: "127.0.0.1"}, []byte("fixture-mcp"))
	h := New(store, policy.NewBuilder(nil))
	bind, err := h.Start(ctx, GateSpec{SessionID: "revoke", ModelAccountID: model, MCPAccountID: mcp, Policy: policy.PolicySnapshot{Model: policy.EndpointRule{Origin: up.URL, Paths: []string{"/v1/"}}}}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Revoke(ctx, "revoke")
	requestDone := make(chan struct{})
	go func() {
		defer close(requestDone)
		req, _ := http.NewRequest("POST", bind.ModelBaseURL+"/v1/messages", nil)
		req.Header.Set("Authorization", "Bearer "+bind.Sentinel)
		if resp, e := http.DefaultClient.Do(req); e == nil {
			resp.Body.Close()
		}
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("请求未到达受控上游")
	}
	budget, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	started := time.Now()
	err = h.Revoke(budget, "revoke")
	if err != nil {
		t.Errorf("撤销失败：%v", err)
	}
	if time.Since(started) > time.Second {
		t.Error("撤销忽略调用预算")
	}
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Error("撤销返回后真实上游仍未取消")
	}
	releaseNow()
	select {
	case <-requestDone:
	case <-time.After(time.Second):
		t.Error("代理请求尚未退出")
	}
}

func TestRevokeExpiredBudgetRetainsCleanupForRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store := secrets.NewStore()
	model, _ := store.Put(ctx, contracts.AccountInfo{Kind: "model", Name: "m", ProfileID: "m", Host: "127.0.0.1"}, []byte("fixture-model"))
	mcp, _ := store.Put(ctx, contracts.AccountInfo{Kind: "mcp", Name: "t", ProfileID: "t", Host: "127.0.0.1"}, []byte("fixture-mcp"))
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }))
	defer up.Close()
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var once sync.Once
	releaseNow := func() { once.Do(func() { close(release) }) }
	defer releaseNow()
	h := New(store, policy.NewBuilder(nil))
	bind, err := h.Start(ctx, GateSpec{SessionID: "retry", ModelAccountID: model, MCPAccountID: mcp, Policy: policy.PolicySnapshot{Model: policy.EndpointRule{Origin: up.URL, Paths: []string{"/v1/"}}}}, func(contracts.AuditData) { close(entered); <-release }, nil)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		defer close(done)
		req, _ := http.NewRequest("POST", bind.ModelBaseURL+"/v1/messages", nil)
		req.Header.Set("Authorization", "Bearer "+bind.Sentinel)
		if resp, e := http.DefaultClient.Do(req); e == nil {
			resp.Body.Close()
		}
	}()
	<-entered
	expired, stop := context.WithCancel(ctx)
	stop()
	if err = h.Revoke(expired, "retry"); err == nil {
		t.Fatal("在途未结束却报告清理完成")
	}
	if _, ok := h.getSession("retry"); ok {
		t.Fatal("预算过期后令牌仍可用")
	}
	releaseNow()
	budget, finish := context.WithTimeout(ctx, time.Second)
	defer finish()
	if err = h.Revoke(budget, "retry"); err != nil {
		t.Fatal(err)
	}
	<-done
	if err = h.Revoke(budget, "retry"); err != nil {
		t.Fatal("重复清理不幂等")
	}
}
