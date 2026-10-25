package gate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/liyufei916/agent-harbor/internal/contracts"
	"github.com/liyufei916/agent-harbor/internal/policy"
	"github.com/liyufei916/agent-harbor/internal/secrets"
)

func TestMCPConnectRequiresHandshakeAndSelectedTools(t *testing.T) {
	for _, scenario := range []string{"json", "sse", "missing-tool", "rpc-error", "wrong-id", "auth", "redirect"} {
		t.Run(scenario, func(t *testing.T) {
			var methods []string
			up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer fixture-token" {
					t.Error("认证头错误")
				}
				if scenario == "auth" {
					w.WriteHeader(401)
					return
				}
				if scenario == "redirect" {
					w.Header().Set("Location", "/other")
					w.WriteHeader(307)
					return
				}
				if r.Method == "DELETE" {
					methods = append(methods, "DELETE")
					w.WriteHeader(204)
					return
				}
				var req struct {
					Method string `json:"method"`
					ID     int    `json:"id"`
					Params struct {
						Cursor string `json:"cursor"`
					} `json:"params"`
				}
				if json.NewDecoder(r.Body).Decode(&req) != nil {
					t.Error("无效请求")
				}
				methods = append(methods, req.Method)
				if req.Method != "initialize" && r.Header.Get("Mcp-Session-Id") != "fixture-session" {
					t.Error("丢失上游会话标识")
				}
				if req.Method == "notifications/initialized" {
					w.WriteHeader(202)
					return
				}
				var result any
				switch req.Method {
				case "initialize":
					w.Header().Set("Mcp-Session-Id", "fixture-session")
					result = map[string]any{"protocolVersion": mcpProtocolVersion, "capabilities": map[string]any{"tools": map[string]any{}}, "serverInfo": map[string]string{"name": "fixture", "version": "1"}}
				case "tools/list":
					name := "approved"
					if scenario == "missing-tool" {
						name = "other"
					}
					if req.Params.Cursor == "" {
						result = map[string]any{"tools": []any{}, "nextCursor": "next"}
					} else {
						result = map[string]any{"tools": []any{map[string]string{"name": name}}, "nextCursor": ""}
					}
				default:
					t.Error("连接测试不应调用工具或未知方法")
				}
				id := req.ID
				if scenario == "wrong-id" {
					id += 100
				}
				msg := map[string]any{"jsonrpc": "2.0", "id": id, "result": result}
				if scenario == "rpc-error" {
					delete(msg, "result")
					msg["error"] = map[string]any{"code": -32603, "message": "fixture"}
				}
				raw, _ := json.Marshal(msg)
				if scenario == "sse" {
					w.Header().Set("Content-Type", "text/event-stream")
					fmt.Fprintf(w, "event: message\ndata: %s\n\n", raw)
					w.(http.Flusher).Flush()
					<-r.Context().Done()
				} else {
					w.Header().Set("Content-Type", "application/json")
					w.Write(raw)
				}
			}))
			defer up.Close()
			store := secrets.NewStore()
			p := policy.NewBuilder(map[string]policy.FixedProfile{"mcp": {ID: "mcp", Kind: "mcp", Protocol: "streamable-http", Origin: up.URL, Host: "127.0.0.1", TestPath: "/mcp", Tools: []string{"approved"}, AllowInsecureLoopback: true}})
			account, err := NewConnections(store, p).ConnectMcp(context.Background(), contracts.ConnectInput{ProfileID: "mcp", Name: "fixture", APIKey: "fixture-token"})
			success := scenario == "json" || scenario == "sse"
			if success {
				if err != nil || account.ID == "" || store.Count() != 1 {
					t.Fatalf("%+v %v", account, err)
				}
				if strings.Join(methods, ",") != "initialize,notifications/initialized,tools/list,tools/list,DELETE" {
					t.Fatalf("握手顺序 %v", methods)
				}
			} else if err == nil || store.Count() != 0 {
				t.Fatalf("无效上游仍保存账号 %v", err)
			}
		})
	}
}

func TestModelRejectsHTMLAndWrongProtocol(t *testing.T) {
	for _, body := range []string{"<html>ok</html>", `{"ok":true}`, `{"type":"message","content":[]}`} {
		up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
		store := secrets.NewStore()
		p := policy.NewBuilder(map[string]policy.FixedProfile{"m": {ID: "m", Kind: "model", Protocol: "anthropic", Model: "fixture", Origin: up.URL, Host: "127.0.0.1", TestPath: "/v1/messages", AllowInsecureLoopback: true}})
		_, err := NewConnections(store, p).ConnectModel(context.Background(), contracts.ConnectInput{ProfileID: "m", Name: "fixture", APIKey: "fixture"})
		up.Close()
		if err == nil || store.Count() != 0 {
			t.Fatal("错误协议被判成功")
		}
	}
}
