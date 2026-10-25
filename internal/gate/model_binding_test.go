package gate

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/liyufei916/agent-harbor/internal/contracts"
	"github.com/liyufei916/agent-harbor/internal/policy"
	"github.com/liyufei916/agent-harbor/internal/secrets"
)

type bindingTransport func(*http.Request) (*http.Response, error)

func (f bindingTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestModelSwitchOnlyAffectsSubsequentRequests(t *testing.T) {
	store := secrets.NewStore()
	put := func(name, profile string) string {
		t.Helper()
		id, err := store.Put(context.Background(), contracts.AccountInfo{Kind: "model", Name: name, ProfileID: profile, Host: "models.example.test"}, []byte(name+"-secret"))
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	oldID, newID, wrongID := put("old", "compatible"), put("new", "compatible"), put("wrong", "other-protocol")
	h := New(store, policy.NewBuilder(nil))
	rt := &sessionRuntime{ctx: context.Background(), sentinel: sentinelPrefix + strings.Repeat("a", 64), spec: GateSpec{ModelAccountID: oldID, Policy: policy.PolicySnapshot{Model: policy.EndpointRule{Origin: "https://models.example.test", Paths: []string{"/v1/messages"}}}}}
	h.sessions["session"] = rt
	entered, release := make(chan string, 2), make(chan struct{})
	defer close(release)
	h.client.Transport = bindingTransport(func(r *http.Request) (*http.Response, error) {
		auth := r.Header.Get("Authorization")
		entered <- auth
		if auth == "Bearer old-secret" {
			select {
			case <-release:
			case <-r.Context().Done():
				return nil, r.Context().Err()
			}
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("response"))}, nil
	})
	proxy := &modelProxy{h: h, sid: "session"}
	call := func() int {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		request := httptest.NewRequest("POST", "/v1/messages", strings.NewReader("{}")).WithContext(ctx)
		request.Header.Set("Authorization", "Bearer "+rt.sentinel)
		response := httptest.NewRecorder()
		proxy.handle(response, request)
		return response.Code
	}
	first := make(chan int, 1)
	go func() { first <- call() }()
	select {
	case auth := <-entered:
		if auth != "Bearer old-secret" {
			t.Fatalf("首个账号错误：%s", auth)
		}
	case <-time.After(time.Second):
		t.Fatal("首个请求未发出")
	}
	if err := h.SwitchModel(context.Background(), "session", newID); err != nil {
		t.Fatal(err)
	}
	if err := h.SwitchModel(context.Background(), "session", wrongID); err == nil {
		t.Fatal("接受不兼容 profile")
	}
	if err := h.SetModelAccount("session", wrongID); err == nil {
		t.Fatal("旧账号切换入口绕过兼容性检查")
	}
	if code := call(); code != 200 {
		t.Fatalf("切换后请求失败：%d", code)
	}
	select {
	case auth := <-entered:
		if auth != "Bearer new-secret" {
			t.Fatalf("新账号未生效：%s", auth)
		}
	case <-time.After(time.Second):
		t.Fatal("新请求未发出")
	}
	select {
	case <-first:
		t.Fatal("切换取消了原请求")
	default:
	}
	// 原请求继续使用捕获的旧账号；结束由独立信号控制。
	release <- struct{}{}
	select {
	case code := <-first:
		if code != 200 {
			t.Fatalf("原请求失败：%d", code)
		}
	case <-time.After(time.Second):
		t.Fatal("原请求未结束")
	}
	h.mu.Lock()
	rt.revoked = true
	h.mu.Unlock()
	if err := h.SwitchModel(context.Background(), "session", oldID); err == nil {
		t.Fatal("撤销后仍能切换")
	}
}
