package gate

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Wintercom/agent-harbor/internal/contracts"
	"github.com/Wintercom/agent-harbor/internal/policy"
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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = p.server.Shutdown(ctx)
	_ = p.ln.Close()
}

func (p *modelProxy) serve(ctx context.Context) {
	go func() {
		<-ctx.Done()
		p.close()
	}()
	err := p.server.Serve(p.ln)
	if err != nil && err != http.ErrServerClosed {
		rt, _ := p.h.getSession(p.sid)
		p.h.emitFatal(rt, err)
	}
}

func (p *modelProxy) handle(w http.ResponseWriter, r *http.Request) {
	rt, ok := p.h.getSession(p.sid)
	if !ok {
		http.Error(w, "session gone", http.StatusServiceUnavailable)
		return
	}
	host := hostOfOrigin(rt.spec.Policy.Model.Origin)
	if !policy.PathAllowed(rt.spec.Policy.Model, r.URL.Path) {
		p.h.emitAudit(rt, contracts.AuditData{Target: host, Tool: "", Result: "denied"})
		http.Error(w, "path not allowed", http.StatusForbidden)
		return
	}
	if !p.authorize(r, rt) {
		p.h.emitAudit(rt, contracts.AuditData{Target: host, Tool: "", Result: "denied"})
		http.Error(w, "invalid sentinel", http.StatusUnauthorized)
		return
	}

	upstreamURL, err := joinOrigin(rt.spec.Policy.Model.Origin, r.URL)
	if err != nil {
		http.Error(w, "bad upstream", http.StatusBadGateway)
		return
	}
	outReq, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL, r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadGateway)
		return
	}
	copyHeaders(outReq.Header, r.Header)
	if err := p.injectModelKey(outReq, rt); err != nil {
		http.Error(w, "credential missing", http.StatusBadGateway)
		return
	}

	resp, err := p.h.client.Do(outReq)
	if err != nil {
		p.h.emitAudit(rt, contracts.AuditData{Target: host, Tool: "", Result: "failed"})
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	p.h.emitAudit(rt, contracts.AuditData{Target: host, Tool: "", Result: "allowed"})
	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (p *modelProxy) authorize(r *http.Request, rt *sessionRuntime) bool {
	if tok, ok := parseBearerSentinel(r.Header.Get("Authorization")); ok && tok == rt.sentinel {
		return true
	}
	for _, name := range []string{"X-Api-Key", "Api-Key", "x-api-key"} {
		if looksLikeSentinel(r.Header.Get(name)) && strings.TrimSpace(r.Header.Get(name)) == rt.sentinel {
			return true
		}
	}
	return false
}

func (p *modelProxy) injectModelKey(req *http.Request, rt *sessionRuntime) error {
	key, err := p.h.secrets.Resolve(req.Context(), rt.spec.ModelAccountID)
	if err != nil {
		return err
	}
	keyStr := string(key)
	if auth := req.Header.Get("Authorization"); looksLikeSentinel(auth) {
		req.Header.Set("Authorization", "Bearer "+keyStr)
	}
	for _, name := range []string{"X-Api-Key", "Api-Key", "x-api-key"} {
		if looksLikeSentinel(req.Header.Get(name)) {
			req.Header.Set(name, keyStr)
		}
	}
	// 若客户端未带 sentinel 头但已通过 authorize，仍确保上游有 Bearer
	if req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Bearer "+keyStr)
	}
	return nil
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
