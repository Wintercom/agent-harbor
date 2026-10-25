package desktop

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/liyufei916/agent-harbor/internal/contracts"
)

func TestExchangeAuthCodeWithSecret(t *testing.T) {
	var got url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			http.NotFound(w, r)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		got, _ = url.ParseQuery(string(raw))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"tok-from-exchange","token_type":"bearer"}`))
	}))
	defer srv.Close()

	tok, err := exchangeAuthCode(context.Background(), srv.URL, "auth-code", "http://127.0.0.1:9/callback", "agent-harbor", "super-secret")
	if err != nil || tok != "tok-from-exchange" {
		t.Fatalf("tok=%q err=%v", tok, err)
	}
	if got.Get("grant_type") != "authorization_code" || got.Get("code") != "auth-code" || got.Get("client_secret") != "super-secret" {
		t.Fatalf("%v", got)
	}
}

func TestOAuthCallbackExchangesCode(t *testing.T) {
	idp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"access_token":"mcp-live-token"}`))
	}))
	defer idp.Close()

	app := NewApp()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	p := &oauthPending{
		id: "oauth_test", name: "Wiki", origin: idp.URL, profileID: "demo-mcp",
		redirectURI: "http://127.0.0.1/callback", ln: ln, done: make(chan struct{}),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		app.completeMcpOAuth(p, w, r)
	})
	p.server = &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	app.mu.Lock()
	app.oauth = map[string]*oauthPending{p.id: p}
	app.mu.Unlock()
	go func() { _ = p.server.Serve(ln); close(p.done) }()
	defer func() { _ = app.cancelMcpOAuth(p.id) }()

	resp, err := http.Get("http://" + ln.Addr().String() + "/callback?code=abc&state=" + p.id)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != 200 || !strings.Contains(string(body), "登录完成") {
		t.Fatalf("status=%d body=%s", resp.StatusCode, body)
	}
	accounts := app.secrets.List(context.Background(), "mcp")
	if len(accounts) != 1 {
		t.Fatalf("%+v", accounts)
	}
	key, err := app.secrets.Resolve(context.Background(), accounts[0].ID)
	if err != nil || string(key) != "mcp-live-token" {
		t.Fatalf("key=%q err=%v", key, err)
	}
}

func TestStartMcpOAuthUsesCodeFlow(t *testing.T) {
	app := NewApp()
	api := NewDesktopAPI(app)
	started := api.StartMcpOAuth(contracts.StartMcpOAuthInput{
		Name: "Notion", Origin: "https://auth.example.test",
	})
	if started.Error != nil {
		t.Fatal(started.Error)
	}
	if !strings.Contains(started.AuthURL, "response_type=code") || !strings.Contains(started.AuthURL, "client_id=agent-harbor") {
		t.Fatalf("%s", started.AuthURL)
	}
	_ = api.CancelMcpOAuth(started.PendingID)
}
