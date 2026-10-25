package gate

import (
	"context"
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

	upstreamURL, err := joinOrigin(rt.spec.Policy.Model.Origin, r.URL)
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
	if blockedRedirect(resp, host) {
		resp.Body.Close()
		p.h.emitAudit(rt, contracts.AuditData{Target: host, Tool: auditToolModel, Result: "denied"})
		http.Error(w, "redirect blocked", http.StatusBadGateway)
		return
	}
	p.h.emitAudit(rt, contracts.AuditData{Target: host, Tool: auditToolModel, Result: "allowed"})
	idle, total := p.h.streamTimeouts()
	if err := writeRelay(w, resp, []string{rt.sentinel, string(key)}, idle, total); err != nil {
		if streamTimeoutStatus(err) == http.StatusGatewayTimeout {
			p.h.emitAudit(rt, contracts.AuditData{Target: host, Tool: auditToolModel, Result: "failed"})
		}
	}
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

func blockedRedirect(resp *http.Response, allowedHost string) bool {
	if resp.StatusCode < 300 || resp.StatusCode >= 400 {
		return false
	}
	loc := resp.Header.Get("Location")
	if loc == "" {
		return false
	}
	u, err := url.Parse(loc)
	if err != nil {
		return true
	}
	if u.Host == "" {
		return false
	}
	return !strings.EqualFold(u.Hostname(), allowedHost)
}

func hostOfOrigin(origin string) string {
	u, err := url.Parse(origin)
	if err != nil {
		return origin
	}
	return u.Hostname()
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
