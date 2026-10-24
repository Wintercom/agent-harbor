package gate

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wintercom/agent-harbor/internal/contracts"
	"github.com/Wintercom/agent-harbor/internal/policy"
	"github.com/Wintercom/agent-harbor/internal/secrets"
)

func TestSentinelLength(t *testing.T) {
	s, err := newSentinel()
	if err != nil {
		t.Fatal(err)
	}
	tok, ok := parseBearerSentinel("Bearer " + s)
	if !ok || tok != s {
		t.Fatalf("%v %q", ok, tok)
	}
	if len(s) < len(sentinelPrefix)+64 {
		t.Fatalf("too short: %s", s)
	}
}

func TestModelProxyRewriteAndDeny(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, r.Header.Get("Authorization"))
	}))
	defer origin.Close()

	store := secrets.NewStore()
	ctx := context.Background()
	modelID, err := store.Put(ctx, contracts.AccountInfo{
		Kind: "model", Name: "demo", ProfileID: policy.ProfileDemoModel, Host: "127.0.0.1",
	}, []byte("sk-real-key"))
	if err != nil {
		t.Fatal(err)
	}
	mcpID, err := store.Put(ctx, contracts.AccountInfo{
		Kind: "mcp", Name: "demo-mcp", ProfileID: policy.ProfileDemoMCP, Host: "127.0.0.1",
	}, []byte("mcp-key"))
	if err != nil {
		t.Fatal(err)
	}

	pol := policy.NewBuilder(map[string]policy.FixedProfile{
		policy.ProfileDemoModel: {
			ID: policy.ProfileDemoModel, Kind: "model", Origin: origin.URL, Paths: []string{"/"}, Host: "127.0.0.1",
		},
		policy.ProfileDemoMCP: {
			ID: policy.ProfileDemoMCP, Kind: "mcp", Origin: origin.URL, Paths: []string{"/mcp"}, Host: "127.0.0.1",
		},
	})
	h := New(store, pol)

	var audits []contracts.AuditData
	bind, err := h.Start(ctx, GateSpec{
		SessionID:      "sess1",
		Policy:         policy.PolicySnapshot{Model: policy.EndpointRule{Origin: origin.URL, Paths: []string{"/"}}, MCP: policy.EndpointRule{Origin: origin.URL, Paths: []string{"/mcp"}}, MCPID: mcpID},
		ModelAccountID: modelID,
		MCPAccountID:   mcpID,
	}, func(a contracts.AuditData) { audits = append(audits, a) }, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Revoke(ctx, "sess1")

	req, _ := http.NewRequest(http.MethodGet, bind.ModelBaseURL+"/v1/chat", nil)
	req.Header.Set("Authorization", "Bearer "+bind.Sentinel)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if string(body) != "Bearer sk-real-key" {
		t.Fatalf("rewrite failed: %q", body)
	}

	req2, _ := http.NewRequest(http.MethodGet, bind.ModelBaseURL+"/v1/chat", nil)
	req2.Header.Set("Authorization", "Bearer wrong")
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp2.Body.Close()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401 got %d", resp2.StatusCode)
	}
}

func TestRevokeInvalidatesSentinel(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer origin.Close()

	store := secrets.NewStore()
	ctx := context.Background()
	modelID, _ := store.Put(ctx, contracts.AccountInfo{
		Kind: "model", Name: "m", ProfileID: policy.ProfileDemoModel, Host: "h",
	}, []byte("k"))
	mcpID, _ := store.Put(ctx, contracts.AccountInfo{
		Kind: "mcp", Name: "t", ProfileID: policy.ProfileDemoMCP, Host: "h",
	}, []byte("k"))
	pol := policy.NewBuilder(map[string]policy.FixedProfile{
		policy.ProfileDemoModel: {ID: policy.ProfileDemoModel, Kind: "model", Origin: origin.URL, Paths: []string{"/"}, Host: "h"},
		policy.ProfileDemoMCP:   {ID: policy.ProfileDemoMCP, Kind: "mcp", Origin: origin.URL, Paths: []string{"/mcp"}, Host: "h"},
	})
	h := New(store, pol)
	bind, err := h.Start(ctx, GateSpec{
		SessionID: "s2",
		Policy: policy.PolicySnapshot{
			Model: policy.EndpointRule{Origin: origin.URL, Paths: []string{"/"}},
			MCP:   policy.EndpointRule{Origin: origin.URL, Paths: []string{"/mcp"}},
			MCPID: mcpID,
		},
		ModelAccountID: modelID,
		MCPAccountID:   mcpID,
	}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.Revoke(ctx, "s2"); err != nil {
		t.Fatal(err)
	}
	// 监听器关闭后请求应失败
	client := &http.Client{Timeout: 500 * time.Millisecond}
	_, err = client.Get(bind.ModelBaseURL + "/")
	if err == nil {
		t.Fatal("expected connection error after revoke")
	}
}

func TestConnectModelWithLocalUpstream(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ") != "good-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer origin.Close()

	store := secrets.NewStore()
	pol := policy.NewBuilder(map[string]policy.FixedProfile{
		policy.ProfileDemoModel: {
			ID: policy.ProfileDemoModel, Kind: "model", Origin: origin.URL,
			Paths: []string{"/"}, Host: "127.0.0.1", TestPath: "/",
		},
		policy.ProfileDemoMCP: {
			ID: policy.ProfileDemoMCP, Kind: "mcp", Origin: origin.URL,
			Paths: []string{"/mcp"}, Host: "127.0.0.1", TestPath: "/",
		},
	})
	conn := NewConnections(store, pol)
	meta, err := conn.ConnectModel(context.Background(), contracts.ConnectInput{
		ProfileID: policy.ProfileDemoModel, Name: "Demo", APIKey: "good-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	if meta.ID == "" || meta.Kind != "model" {
		t.Fatalf("%+v", meta)
	}
	_, err = conn.ConnectModel(context.Background(), contracts.ConnectInput{
		ProfileID: policy.ProfileDemoModel, Name: "Bad", APIKey: "bad-key",
	})
	if err != ErrUpstreamAuth {
		t.Fatalf("got %v", err)
	}
}
