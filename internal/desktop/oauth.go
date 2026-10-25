package desktop

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/liyufei916/agent-harbor/internal/contracts"
	"github.com/liyufei916/agent-harbor/internal/policy"
)

// OAuth 回环回调：浏览器完成登录后打到 127.0.0.1，用 access_token 或 code 写入 MCP 账号。
// 完整授权码换票（client_secret）仍待供应商登记；本路径供双轨 UI 与联调演示。

type oauthPending struct {
	id        string
	name      string
	origin    string
	profileID string
	ln        net.Listener
	server    *http.Server
	done      chan struct{}
}

func (a *App) startMcpOAuth(in contracts.StartMcpOAuthInput) (authURL, pendingID string, err error) {
	name := strings.TrimSpace(in.Name)
	origin := strings.TrimRight(strings.TrimSpace(in.Origin), "/")
	profileID := strings.TrimSpace(in.ProfileID)
	if name == "" {
		return "", "", contracts.NewAppError(contracts.ErrInvalidArgument, "显示名不能为空", false)
	}
	if origin == "" && profileID != "" {
		if prof, ok := a.policy.Profile(profileID); ok {
			origin = strings.TrimRight(prof.Origin, "/")
		}
	}
	if origin == "" {
		return "", "", contracts.NewAppError(contracts.ErrInvalidArgument, "OAuth 需要 HTTPS Origin", false)
	}
	u, perr := url.Parse(origin)
	if perr != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return "", "", contracts.NewAppError(contracts.ErrInvalidArgument, "OAuth Origin 必须是 https://host", false)
	}
	ln, lerr := net.Listen("tcp", "127.0.0.1:0")
	if lerr != nil {
		return "", "", contracts.NewAppError(contracts.ErrInternal, "无法监听本机回调", false)
	}
	var b [8]byte
	_, _ = rand.Read(b[:])
	id := "oauth_" + hex.EncodeToString(b[:])
	p := &oauthPending{
		id: id, name: name, origin: origin, profileID: profileID,
		ln: ln, done: make(chan struct{}),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		a.completeMcpOAuth(p, w, r)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = fmt.Fprintf(w, "Agent Harbor OAuth 回调就绪。请将 IdP 重定向到 /callback。")
	})
	p.server = &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	a.mu.Lock()
	if a.oauth == nil {
		a.oauth = make(map[string]*oauthPending)
	}
	a.oauth[id] = p
	a.mu.Unlock()
	go func() {
		_ = p.server.Serve(ln)
		close(p.done)
	}()
	redirect := url.QueryEscape(fmt.Sprintf("http://127.0.0.1:%d/callback", ln.Addr().(*net.TCPAddr).Port))
	authURL = origin + "/authorize?response_type=token&redirect_uri=" + redirect + "&state=" + url.QueryEscape(id)
	return authURL, id, nil
}

func (a *App) completeMcpOAuth(p *oauthPending, w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	token := strings.TrimSpace(q.Get("access_token"))
	if token == "" {
		token = strings.TrimSpace(q.Get("code"))
	}
	if token == "" {
		// 兼容隐式流 fragment 无法到服务端：提供说明页
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><meta charset=utf-8><title>Harbor OAuth</title>
<p>若地址栏含 access_token，请复制到工具页 Token 轨；或使用 ?access_token= / ?code= 查询参数完成回调。</p>
<script>
if (location.hash) {
  const p = new URLSearchParams(location.hash.slice(1));
  const t = p.get("access_token") || p.get("code");
  if (t) location.replace(location.pathname + "?access_token=" + encodeURIComponent(t) + "&state=" + encodeURIComponent(p.get("state")||""));
}
</script>`))
		return
	}
	host := ""
	if u, err := url.Parse(p.origin); err == nil {
		host = strings.ToLower(u.Hostname())
	}
	profileID := p.profileID
	if profileID == "" {
		profileID = policy.ProfileDemoMCP
	}
	meta := contracts.AccountInfo{
		Kind: "mcp", Name: p.name, ProfileID: profileID, Host: host,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	id, err := a.secrets.Put(ctx, meta, []byte(token))
	if err != nil {
		http.Error(w, "save failed", http.StatusBadGateway)
		return
	}
	_ = id
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("登录完成。可回到 Agent Harbor 工具页刷新列表。"))
	go a.cancelMcpOAuth(p.id)
}

func (a *App) cancelMcpOAuth(pendingID string) error {
	a.mu.Lock()
	p, ok := a.oauth[pendingID]
	if ok {
		delete(a.oauth, pendingID)
	}
	a.mu.Unlock()
	if !ok {
		return contracts.NewAppError(contracts.ErrInvalidArgument, "无此 OAuth 请求", false)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = p.server.Shutdown(ctx)
	_ = p.ln.Close()
	select {
	case <-p.done:
	case <-time.After(2 * time.Second):
	}
	return nil
}
