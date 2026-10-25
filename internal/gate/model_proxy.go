package gate

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/liyufei916/agent-harbor/internal/contracts"
	"github.com/liyufei916/agent-harbor/internal/policy"
)

// modelProxy 是固定模型 HTTPS 上游的显式 Base URL 反向代理。
// 不实现通用 CONNECT / HTTPS MITM。
type modelProxy struct {
	h      *Harbor
	sid    string
	ln     net.Listener
	server *http.Server
}

func newModelProxy(h *Harbor, sessionID string, ln net.Listener) *modelProxy {
	p := &modelProxy{h: h, sid: sessionID, ln: ln}
	mux := http.NewServeMux()
	mux.HandleFunc("/", p.handle)
	p.server = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	return p
}

func (p *modelProxy) close() {
	// 取消时强制关闭活动连接；Revoke 另行等待全部 handler 与 Serve 退出。
	_ = p.server.Close()
	_ = p.ln.Close()
}

func (p *modelProxy) serve(ctx context.Context) {

	err := p.server.Serve(p.ln)
	if err != nil && err != http.ErrServerClosed {
		rt, _ := p.h.getSession(p.sid)
		p.h.emitFatal(rt, err)
	}
}

func (p *modelProxy) handle(w http.ResponseWriter, r *http.Request) {
	rt, ok := p.h.acquireSession(p.sid)
	if !ok {
		http.Error(w, "session gone", http.StatusServiceUnavailable)
		return
	}
	defer rt.active.Done()
	originAuth := authorityOfOrigin(rt.spec.Policy.Model.Origin)
	host := hostOfOrigin(rt.spec.Policy.Model.Origin)
	if !policy.PathAllowed(rt.spec.Policy.Model, r.URL.Path) {
		p.h.emitAudit(rt, contracts.AuditData{Target: host, Tool: auditToolModel, Result: "denied"})
		http.Error(w, "path not allowed", http.StatusForbidden)
		return
	}
	if !p.authorize(r, rt) {
		p.h.emitAudit(rt, contracts.AuditData{Target: host, Tool: auditToolModel, Result: "denied"})
		http.Error(w, "invalid sentinel", http.StatusUnauthorized)
		return
	}

	if !policy.ValidBasePath(rt.spec.Policy.Model.BasePath) {
		http.Error(w, "invalid model route", http.StatusForbidden)
		return
	}
	modelURL := *r.URL
	modelURL.Path = rt.spec.Policy.Model.BasePath + r.URL.Path
	upstreamURL, err := joinOrigin(rt.spec.Policy.Model.Origin, &modelURL)
	if err != nil {
		http.Error(w, "bad upstream", http.StatusBadGateway)
		return
	}
	ctx, cancel := p.h.upstreamContext(r.Context())
	defer cancel()
	outReq, err := http.NewRequestWithContext(ctx, r.Method, upstreamURL, r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadGateway)
		return
	}
	copyHeaders(outReq.Header, r.Header)
	outReq.Header.Set("Accept-Encoding", "identity")
	key, err := p.h.secrets.Resolve(r.Context(), rt.spec.ModelAccountID)
	if err != nil {
		http.Error(w, "credential missing", http.StatusBadGateway)
		return
	}
	p.applyModelKey(outReq, string(key))

	resp, err := p.h.client.Do(outReq)
	if err != nil {
		p.h.emitAudit(rt, contracts.AuditData{Target: host, Tool: auditToolModel, Result: "failed"})
		http.Error(w, "upstream unavailable", streamTimeoutStatus(err))
		return
	}
	if blockedRedirect(resp, originAuth) {
		followed, locHost, ferr := p.followCrossHostRedirect(r.Context(), rt, resp, originAuth)
		if ferr != nil || followed == nil {
			if locHost == "" {
				locHost = host
			}
			p.h.emitAudit(rt, contracts.AuditData{Target: locHost, Tool: auditToolModel, Result: "denied"})
			http.Error(w, "redirect blocked", http.StatusBadGateway)
			return
		}
		resp = followed
		host = locHost
		originAuth = locHost
		// 跨主机跟随不得携带原账号凭据；脱敏仍抹 Sentinel。
		key = nil
	}
	p.h.emitAudit(rt, contracts.AuditData{Target: host, Tool: auditToolModel, Result: "allowed"})
	idle, total := p.h.streamTimeouts()
	secretsToRedact := []string{rt.sentinel}
	if len(key) > 0 {
		secretsToRedact = append(secretsToRedact, string(key))
	}
	if err := writeRelay(w, resp, secretsToRedact, idle, total); err != nil {
		if streamTimeoutStatus(err) == http.StatusGatewayTimeout {
			p.h.emitAudit(rt, contracts.AuditData{Target: host, Tool: auditToolModel, Result: "failed"})
		}
	}
}

// followCrossHostRedirect 在用户扩权且 DNS/IP 通过后跟随一跳；不注入原账号凭据。
func (p *modelProxy) followCrossHostRedirect(ctx context.Context, rt *sessionRuntime, resp *http.Response, originAuth string) (*http.Response, string, error) {
	locURL, err := redirectAbsoluteURL(resp)
	defer resp.Body.Close()
	if err != nil || locURL == nil || locURL.Host == "" {
		return nil, "", fmt.Errorf("gate: bad redirect")
	}
	locAuth := strings.ToLower(locURL.Host)
	locHost := strings.ToLower(locURL.Hostname())
	if strings.EqualFold(locAuth, originAuth) {
		return nil, locHost, fmt.Errorf("gate: same-origin redirect not blocked")
	}

	allowed := p.h.HasExtraHost(p.sid, locHost)
	if !allowed && rt.spec.RequestPermission != nil {
		decision, perr := rt.spec.RequestPermission(ctx, contracts.PermissionNeeded{
			Target: locHost,
			Action: "请求跟随跨主机重定向",
		})
		if perr == nil && (decision == "once" || decision == "project") {
			_ = p.h.AllowExtraHost(p.sid, locHost, decision)
			allowed = true
			if decision == "once" {
				_ = p.h.HasExtraHost(p.sid, locHost) // 本请求消费
			}
		}
	}
	if !allowed {
		return nil, locHost, fmt.Errorf("gate: redirect not permitted")
	}
	if err := ValidatePublicHostname(locHost); err != nil {
		return nil, locHost, err
	}
	if locURL.Scheme != "https" && !(locURL.Scheme == "http" && egressAllowLoopback) {
		return nil, locHost, fmt.Errorf("gate: redirect scheme denied")
	}

	followReq, err := http.NewRequestWithContext(ctx, redirectFollowMethod(resp.StatusCode, resp.Request.Method), locURL.String(), nil)
	if err != nil {
		return nil, locHost, err
	}
	// 明确不复制 Authorization / API Key / Cookie，避免凭据跨主机。
	followReq.Header.Set("Accept-Encoding", "identity")
	if ua := resp.Request.Header.Get("User-Agent"); ua != "" {
		followReq.Header.Set("User-Agent", ua)
	}

	next, err := p.h.client.Do(followReq)
	if err != nil {
		return nil, locHost, err
	}
	if blockedRedirect(next, locAuth) {
		next.Body.Close()
		return nil, locHost, fmt.Errorf("gate: multi-hop redirect denied")
	}
	return next, locHost, nil
}

func (p *modelProxy) authorize(r *http.Request, rt *sessionRuntime) bool {
	if tok, ok := parseBearerSentinel(r.Header.Get("Authorization")); ok && sentinelMatch(tok, rt.sentinel) {
		return true
	}
	for _, name := range []string{"X-Api-Key", "Api-Key", "x-api-key"} {
		if looksLikeSentinel(r.Header.Get(name)) && sentinelMatch(strings.TrimSpace(r.Header.Get(name)), rt.sentinel) {
			return true
		}
	}
	return false
}

func (p *modelProxy) applyModelKey(req *http.Request, keyStr string) {
	if auth := req.Header.Get("Authorization"); looksLikeSentinel(auth) {
		req.Header.Set("Authorization", "Bearer "+keyStr)
	}
	for _, name := range []string{"X-Api-Key", "Api-Key", "x-api-key"} {
		if looksLikeSentinel(req.Header.Get(name)) {
			req.Header.Set(name, keyStr)
		}
	}
	if req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Bearer "+keyStr)
	}
}

func blockedRedirect(resp *http.Response, allowedAuthority string) bool {
	if resp.StatusCode < 300 || resp.StatusCode >= 400 {
		return false
	}
	u, err := redirectAbsoluteURL(resp)
	if err != nil || u == nil || u.Host == "" {
		return false
	}
	return !strings.EqualFold(u.Host, allowedAuthority)
}

func redirectAbsoluteURL(resp *http.Response) (*url.URL, error) {
	loc := resp.Header.Get("Location")
	if loc == "" || resp.Request == nil || resp.Request.URL == nil {
		return nil, fmt.Errorf("missing location")
	}
	return resp.Request.URL.Parse(loc)
}

func redirectFollowMethod(status int, original string) string {
	switch status {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther:
		return http.MethodGet
	default:
		if original == "" {
			return http.MethodGet
		}
		return original
	}
}

func hostOfOrigin(origin string) string {
	u, err := url.Parse(origin)
	if err != nil {
		return origin
	}
	return u.Hostname()
}

func authorityOfOrigin(origin string) string {
	u, err := url.Parse(origin)
	if err != nil {
		return strings.ToLower(strings.TrimSpace(origin))
	}
	return strings.ToLower(u.Host)
}

func joinOrigin(origin string, u *url.URL) (string, error) {
	base, err := url.Parse(origin)
	if err != nil {
		return "", err
	}
	rel := &url.URL{Path: u.Path, RawQuery: u.RawQuery}
	return base.ResolveReference(rel).String(), nil
}

func copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		if strings.EqualFold(k, "Host") || strings.EqualFold(k, "Content-Length") {
			continue
		}
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
