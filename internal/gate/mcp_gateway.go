package gate

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/liyufei916/agent-harbor/internal/contracts"
	"github.com/liyufei916/agent-harbor/internal/policy"
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
	mux.HandleFunc("/mcp/", gw.handleMCP)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	gw.server = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	return gw
}

func (g *mcpGateway) close() {
	// 取消时强制关闭活动连接；Revoke 另行等待全部 handler 与 Serve 退出。
	_ = g.server.Close()
	_ = g.ln.Close()
}

func (g *mcpGateway) serve(ctx context.Context) {

	err := g.server.Serve(g.ln)
	if err != nil && err != http.ErrServerClosed {
		rt, _ := g.h.getSession(g.sid)
		g.h.emitFatal(rt, err)
	}
}

func (g *mcpGateway) handleMCP(w http.ResponseWriter, r *http.Request) {
	rt, ok := g.h.acquireSession(g.sid)
	if !ok {
		http.Error(w, "session gone", http.StatusServiceUnavailable)
		return
	}
	defer rt.active.Done()
	mount, found := g.mountForPath(rt, r.URL.Path)
	if !found {
		http.Error(w, "mount not allowed", http.StatusForbidden)
		return
	}
	host := hostOfOrigin(mount.Endpoint.Origin)
	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodDelete:
	default:
		g.h.emitAudit(rt, contracts.AuditData{Target: host, Tool: auditToolMCP, Result: "denied"})
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tok, ok := parseBearerSentinel(r.Header.Get("Authorization"))
	if !ok || !sentinelMatch(tok, rt.sentinel) {
		g.h.emitAudit(rt, contracts.AuditData{Target: host, Tool: auditToolMCP, Result: "denied"})
		http.Error(w, "invalid sentinel", http.StatusUnauthorized)
		return
	}
	if !policy.PathAllowed(mount.Endpoint, "/mcp") {
		g.h.emitAudit(rt, contracts.AuditData{Target: host, Tool: auditToolMCP, Result: "denied"})
		http.Error(w, "path not allowed", http.StatusForbidden)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	method, tool, parsed := parseMCPRequest(body)
	auditTool := auditToolMCP
	if tool != "" {
		auditTool = tool
	}
	if r.Method == http.MethodPost && len(bytes.TrimSpace(body)) > 0 && !parsed {
		g.h.emitAudit(rt, contracts.AuditData{Target: host, Tool: auditToolMCP, Result: "denied"})
		http.Error(w, "invalid json-rpc", http.StatusBadRequest)
		return
	}
	if method == "tools/call" && !toolAllowed(mount.Tools, tool) {
		g.h.emitAudit(rt, contracts.AuditData{Target: host, Tool: auditTool, Result: "denied"})
		http.Error(w, "tool not allowed", http.StatusForbidden)
		return
	}

	key, err := g.h.secrets.Resolve(r.Context(), mount.AccountID)
	if err != nil {
		http.Error(w, "credential missing", http.StatusBadGateway)
		return
	}
	ctx, cancel := g.h.upstreamContext(r.Context())
	defer cancel()
	upstreamURL := strings.TrimRight(mount.Endpoint.Origin, "/") + "/mcp"
	upstream, err := http.NewRequestWithContext(ctx, r.Method, upstreamURL, bytes.NewReader(body))
	if err != nil {
		http.Error(w, "bad upstream", http.StatusBadGateway)
		return
	}
	copyMCPHeaders(upstream.Header, r.Header)
	upstream.Header.Set("Authorization", "Bearer "+string(key))
	upstream.Header.Set("Accept-Encoding", "identity")

	resp, err := g.h.client.Do(upstream)
	if err != nil {
		g.h.emitAudit(rt, contracts.AuditData{Target: host, Tool: auditTool, Result: "failed"})
		http.Error(w, "upstream unavailable", streamTimeoutStatus(err))
		return
	}
	g.h.emitAudit(rt, contracts.AuditData{Target: host, Tool: auditTool, Result: "allowed"})
	secrets := []string{rt.sentinel, string(key)}
	if method == "tools/list" {
		if err := g.writeFilteredList(w, resp, mount.Tools, secrets); err != nil {
			if streamTimeoutStatus(err) == http.StatusGatewayTimeout {
				g.h.emitAudit(rt, contracts.AuditData{Target: host, Tool: auditTool, Result: "failed"})
			}
		}
		return
	}
	idle, total := g.h.streamTimeouts()
	if err := writeRelay(w, resp, secrets, idle, total); err != nil && streamTimeoutStatus(err) == http.StatusGatewayTimeout {
		g.h.emitAudit(rt, contracts.AuditData{Target: host, Tool: auditTool, Result: "failed"})
	}
}

func (g *mcpGateway) writeFilteredList(w http.ResponseWriter, resp *http.Response, allow, secrets []string) error {
	defer resp.Body.Close()
	idle, total := g.h.streamTimeouts()
	reader := &deadlineReader{r: resp.Body, idle: idle, deadline: time.Now().Add(total)}
	raw, err := readToolsList(reader, resp.Header.Get("Content-Type"), 2<<20)
	if err != nil && len(raw) == 0 {
		http.Error(w, "upstream timeout", streamTimeoutStatus(err))
		return err
	}
	filtered := filterToolsPayload(raw, allow)
	contentType := resp.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "text/event-stream") {
		if jsonBody, ok := toolsSSEToJSON(filtered); ok {
			filtered = jsonBody
			contentType = "application/json"
		}
	}
	red := newSecretRedactor(secrets)
	out := append(red.Push(filtered), red.Flush()...)
	copyHeaders(w.Header(), resp.Header)
	scrubHeaders(w.Header(), secrets)
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(resp.StatusCode)
	_, werr := w.Write(out)
	return werr
}

// toolsSSEToJSON converts the completed tools/list event to one JSON-RPC
// response. The gateway has already applied the allowlist; returning a finite
// response avoids making dsh consume an upstream notification stream.
func toolsSSEToJSON(body []byte) ([]byte, bool) {
	for _, line := range bytes.Split(body, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if !bytes.HasPrefix(line, []byte("data:")) {
			continue
		}
		payload := bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		var value map[string]any
		if json.Unmarshal(payload, &value) == nil && value["jsonrpc"] == "2.0" {
			return payload, true
		}
	}
	return nil, false
}

// readToolsList 只读取完整的 tools/list 事件。Streamable HTTP 的 SSE 响应可能
// 在发送事件后保持连接，等待 EOF 会让 dsh 启动同步一直超时。
func readToolsList(reader io.Reader, contentType string, limit int64) ([]byte, error) {
	if !strings.HasPrefix(contentType, "text/event-stream") {
		return io.ReadAll(io.LimitReader(reader, limit))
	}
	scanner := bufio.NewScanner(io.LimitReader(reader, limit))
	scanner.Buffer(make([]byte, 4096), int(limit))
	var event, out bytes.Buffer
	for scanner.Scan() {
		line := scanner.Bytes()
		event.Write(line)
		event.WriteByte('\n')
		if len(bytes.TrimSpace(line)) == 0 {
			if bytes.Contains(event.Bytes(), []byte(`"tools"`)) {
				out.Write(event.Bytes())
				return out.Bytes(), nil
			}
			out.Write(event.Bytes())
			event.Reset()
		}
	}
	if event.Len() > 0 {
		out.Write(event.Bytes())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func toolAllowed(allow []string, tool string) bool {
	if tool == "" {
		return false
	}
	for _, a := range allow {
		if a == tool {
			return true
		}
	}
	return false
}

func parseMCPRequest(body []byte) (method, tool string, ok bool) {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return "", "", true
	}
	var msg struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  struct {
			Name string `json:"name"`
		} `json:"params"`
	}
	if err := json.Unmarshal(body, &msg); err != nil || msg.JSONRPC != "2.0" || msg.Method == "" {
		return "", "", false
	}
	return msg.Method, msg.Params.Name, true
}

func filterToolsPayload(body []byte, allow []string) []byte {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return body
	}
	if bytes.Contains(body, []byte("data:")) && bytes.Contains(body, []byte("\n")) {
		return filterSSETools(body, allow)
	}
	filtered, ok := filterJSONTools(body, allow)
	if !ok {
		return body
	}
	return filtered
}

func filterJSONTools(body []byte, allow []string) ([]byte, bool) {
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return nil, false
	}
	if !filterToolsValue(v, allow) {
		return nil, false
	}
	out, err := json.Marshal(v)
	if err != nil {
		return nil, false
	}
	return out, true
}

func filterToolsValue(v any, allow []string) bool {
	msg, ok := v.(map[string]any)
	if !ok {
		return false
	}
	res, ok := msg["result"].(map[string]any)
	if !ok {
		return false
	}
	tools, ok := res["tools"].([]any)
	if !ok {
		return false
	}
	kept := make([]any, 0, len(tools))
	for _, item := range tools {
		tm, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, _ := tm["name"].(string)
		if toolAllowed(allow, name) {
			kept = append(kept, item)
		}
	}
	res["tools"] = kept
	return true
}

func filterSSETools(body []byte, allow []string) []byte {
	parts := bytes.Split(body, []byte("\n"))
	for i, line := range parts {
		trimmed := bytes.TrimSpace(line)
		if !bytes.HasPrefix(trimmed, []byte("data:")) {
			continue
		}
		payload := bytes.TrimSpace(bytes.TrimPrefix(trimmed, []byte("data:")))
		filtered, ok := filterJSONTools(payload, allow)
		if !ok {
			continue
		}
		parts[i] = append([]byte("data: "), filtered...)
	}
	return bytes.Join(parts, []byte("\n"))
}

func copyMCPHeaders(dst, src http.Header) {
	for _, name := range []string{
		"Content-Type",
		"Accept",
		"Mcp-Session-Id",
		"Mcp-Protocol-Version",
		"Last-Event-ID",
	} {
		if v := src.Get(name); v != "" {
			dst.Set(name, v)
		}
	}
	if dst.Get("Content-Type") == "" {
		dst.Set("Content-Type", "application/json")
	}
	if dst.Get("Accept") == "" {
		dst.Set("Accept", "application/json, text/event-stream")
	}
}

// 多挂载使用固定序号路径，不能用未知路径或账号 ID 越过授权表。
func (g *mcpGateway) mountForPath(rt *sessionRuntime, path string) (policy.MCPMount, bool) {
	mounts := rt.spec.Policy.MCPMounts
	if len(mounts) == 1 && path == "/mcp" {
		return mounts[0], true
	}
	if len(mounts) > 1 {
		for i, mount := range mounts {
			if path == "/mcp/"+strconv.Itoa(i) {
				return mount, true
			}
		}
	}
	return policy.MCPMount{}, false
}
