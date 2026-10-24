package gate

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Wintercom/agent-harbor/internal/contracts"
	"github.com/Wintercom/agent-harbor/internal/policy"
)

// mcpGateway 是沙箱唯一 MCP 入口；在宿主机持证访问固定上游。
type mcpGateway struct {
	h      *Harbor
	sid    string
	ln     net.Listener
	server *http.Server
}

func newMCPGateway(h *Harbor, sessionID string, ln net.Listener) *mcpGateway {
	gw := &mcpGateway{h: h, sid: sessionID, ln: ln}
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", gw.handleMCP)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	gw.server = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	return gw
}

func (g *mcpGateway) close() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = g.server.Shutdown(ctx)
	_ = g.ln.Close()
}

func (g *mcpGateway) serve(ctx context.Context) {
	go func() {
		<-ctx.Done()
		g.close()
	}()
	err := g.server.Serve(g.ln)
	if err != nil && err != http.ErrServerClosed {
		rt, _ := g.h.getSession(g.sid)
		g.h.emitFatal(rt, err)
	}
}

func (g *mcpGateway) handleMCP(w http.ResponseWriter, r *http.Request) {
	rt, ok := g.h.getSession(g.sid)
	if !ok {
		http.Error(w, "session gone", http.StatusServiceUnavailable)
		return
	}
	host := hostOfOrigin(rt.spec.Policy.MCP.Origin)
	tok, ok := parseBearerSentinel(r.Header.Get("Authorization"))
	if !ok || tok != rt.sentinel {
		g.h.emitAudit(rt, contracts.AuditData{Target: host, Tool: "", Result: "denied"})
		http.Error(w, "invalid sentinel", http.StatusUnauthorized)
		return
	}
	if !policy.PathAllowed(rt.spec.Policy.MCP, "/mcp") {
		g.h.emitAudit(rt, contracts.AuditData{Target: host, Tool: "", Result: "denied"})
		http.Error(w, "path not allowed", http.StatusForbidden)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	tool := extractToolName(body)
	if tool != "" && len(rt.spec.Policy.Tools) > 0 && !toolAllowed(rt.spec.Policy.Tools, tool) {
		g.h.emitAudit(rt, contracts.AuditData{Target: host, Tool: tool, Result: "denied"})
		http.Error(w, "tool not allowed", http.StatusForbidden)
		return
	}

	key, err := g.h.secrets.Resolve(r.Context(), rt.spec.MCPAccountID)
	if err != nil {
		http.Error(w, "credential missing", http.StatusBadGateway)
		return
	}
	upstreamURL := strings.TrimRight(rt.spec.Policy.MCP.Origin, "/") + "/mcp"
	upstream, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL, strings.NewReader(string(body)))
	if err != nil {
		http.Error(w, "bad upstream", http.StatusBadGateway)
		return
	}
	ct := r.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/json"
	}
	upstream.Header.Set("Content-Type", ct)
	upstream.Header.Set("Authorization", "Bearer "+string(key))

	resp, err := g.h.client.Do(upstream)
	if err != nil {
		g.h.emitAudit(rt, contracts.AuditData{Target: host, Tool: tool, Result: "failed"})
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	g.h.emitAudit(rt, contracts.AuditData{Target: host, Tool: tool, Result: "allowed"})
	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func toolAllowed(allow []string, tool string) bool {
	for _, a := range allow {
		if a == tool {
			return true
		}
	}
	return false
}

// extractToolName 从 JSON-RPC tools/call 粗提取工具名；解析失败返回空。
func extractToolName(body []byte) string {
	const marker = `"name"`
	s := string(body)
	if !strings.Contains(s, "tools/call") {
		return ""
	}
	i := strings.Index(s, marker)
	if i < 0 {
		return ""
	}
	rest := s[i+len(marker):]
	j := strings.Index(rest, `"`)
	if j < 0 {
		return ""
	}
	rest = rest[j+1:]
	k := strings.Index(rest, `"`)
	if k < 0 {
		return ""
	}
	return rest[:k]
}
