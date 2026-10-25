package dsh

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/liyufei916/agent-harbor/internal/contracts"
	"github.com/liyufei916/agent-harbor/internal/gate"
	"github.com/liyufei916/agent-harbor/internal/kits"
	"github.com/liyufei916/agent-harbor/internal/orchestrator"
	"github.com/liyufei916/agent-harbor/internal/policy"
	"github.com/liyufei916/agent-harbor/internal/runtime"
	"github.com/liyufei916/agent-harbor/internal/secrets"
)

// 实际已安装的 dsh 进程、OS 沙箱及 Gate；上游为受控 HTTP fixture，不代表真实服务验收。
func TestInstalledDSHThroughGate(t *testing.T) {
	if os.Getenv("HARBOR_TEST_DSH") != "1" {
		t.Skip("需要本机独立 dsh 安装")
	}
	var modelCalls, mcpCalls atomic.Int32
	var mode atomic.Int32
	entered := make(chan struct{}, 1)
	upstreamCanceled := make(chan struct{}, 1)
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if r.Method == "POST" {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid", 400)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/anthropic/v1/messages" {
			if r.Header.Get("X-Api-Key") != "fixture-upstream-model" {
				http.Error(w, "auth", 401)
				return
			}
			if mode.Load() == 1 {
				entered <- struct{}{}
				<-r.Context().Done()
				upstreamCanceled <- struct{}{}
				return
			}
			if mode.Load() == 2 {
				http.Error(w, "test authentication revoked", http.StatusUnauthorized)
				return
			}
			if body["stream"] != true {
				json.NewEncoder(w).Encode(map[string]any{"type": "message", "content": []any{map[string]any{"type": "text", "text": "OK"}}})
				return
			}
			n := modelCalls.Add(1)
			t.Logf("模型请求 %d", n)
			content := map[string]any{"type": "text", "text": "HARBOR-DSH-COMPLETE"}
			stop := "end_turn"
			switch (n - 1) % 5 {
			case 0:
				content = map[string]any{"type": "tool_use", "id": fmt.Sprintf("read-%d", n), "name": "read", "input": map[string]any{"file_path": "input.txt"}}
				stop = "tool_use"
			case 1:
				if !strings.Contains(fmt.Sprint(body["messages"]), "HARBOR-DSH-INPUT") {
					t.Error("dsh 没有返回真实文件读取结果")
				}
				content = map[string]any{"type": "tool_use", "id": fmt.Sprintf("write-%d", n), "name": "write", "input": map[string]any{"file_path": "result.txt", "content": "HARBOR-DSH-FILE-OK\n"}}
				stop = "tool_use"
			case 2:
				content = map[string]any{"type": "tool_use", "id": fmt.Sprintf("edit-%d", n), "name": "edit", "input": map[string]any{"file_path": "result.txt", "old_string": "FILE", "new_string": "EDIT"}}
				stop = "tool_use"
			case 3:
				content = map[string]any{"type": "tool_use", "id": fmt.Sprintf("mcp-%d", n), "name": "mcp__harbor__approved_read", "input": map[string]any{}}
				stop = "tool_use"
			}
			w.Header().Set("Content-Type", "text/event-stream")
			send := func(event string, v any) {
				raw, _ := json.Marshal(v)
				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, raw)
			}
			send("message_start", map[string]any{"type": "message_start", "message": map[string]any{"id": fmt.Sprintf("msg-%d", n), "type": "message", "role": "assistant", "model": "fixture-model", "content": []any{}, "usage": map[string]any{"input_tokens": 10, "output_tokens": 0}}})
			send("content_block_start", map[string]any{"type": "content_block_start", "index": 0, "content_block": content})
			send("content_block_stop", map[string]any{"type": "content_block_stop", "index": 0})
			send("message_delta", map[string]any{"type": "message_delta", "delta": map[string]any{"stop_reason": stop}, "usage": map[string]any{"output_tokens": 10}})
			send("message_stop", map[string]any{"type": "message_stop"})
			return
		}
		if r.URL.Path != "/mcp" || r.Header.Get("Authorization") != "Bearer fixture-upstream-mcp" {
			http.Error(w, "denied", 401)
			return
		}
		if r.Method == "DELETE" {
			w.WriteHeader(204)
			return
		}
		if r.Method != "POST" {
			w.WriteHeader(405)
			return
		}
		method, _ := body["method"].(string)
		var result any
		switch method {
		case "initialize":
			result = map[string]any{"protocolVersion": "2025-03-26", "capabilities": map[string]any{"tools": map[string]any{}}, "serverInfo": map[string]any{"name": "fixture", "version": "1"}}
		case "notifications/initialized":
			w.WriteHeader(202)
			return
		case "tools/list":
			result = map[string]any{"tools": []any{map[string]any{"name": "approved_read", "description": "只读 fixture", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{}}}}}
		case "tools/call":
			mcpCalls.Add(1)
			result = map[string]any{"content": []any{map[string]any{"type": "text", "text": "HARBOR-DSH-MCP-OK"}}}
		default:
			json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": body["id"], "error": map[string]any{"code": -32601, "message": "Method not found"}})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": body["id"], "result": result})
	}))
	defer up.Close()
	store := secrets.NewStore()
	pol := policy.NewBuilder(map[string]policy.FixedProfile{
		"dsh-model": {ID: "dsh-model", Kind: "model", Origin: up.URL, Host: "127.0.0.1", BasePath: "/anthropic", Paths: []string{"/v1/messages"}, TestPath: "/v1/messages", Model: "fixture-model", Protocol: "anthropic", AllowInsecureLoopback: true},
		"demo-mcp":  {ID: "demo-mcp", Kind: "mcp", Origin: up.URL, Host: "127.0.0.1", Paths: []string{"/mcp"}, TestPath: "/mcp", Protocol: "streamable-http", Tools: []string{"approved_read"}, AllowInsecureLoopback: true},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	conn := gate.NewConnections(store, pol)
	model, err := conn.ConnectModel(ctx, contracts.ConnectInput{ProfileID: "dsh-model", Name: "fixture", APIKey: "fixture-upstream-model"})
	if err != nil {
		t.Fatal(err)
	}
	mcp, err := conn.ConnectMcp(ctx, contracts.ConnectInput{ProfileID: "demo-mcp", Name: "fixture", APIKey: "fixture-upstream-mcp"})
	if err != nil {
		t.Fatal(err)
	}
	k, _ := New(os.Getenv("HARBOR_DSH_NODE"), os.Getenv("HARBOR_DSH_INSTALLATION"), "fixture-model")
	var mu sync.Mutex
	var output strings.Builder
	var audit []contracts.AuditData
	svc, err := orchestrator.New(ctx, orchestrator.Dependencies{Runtime: runtime.NewMacOS(), Kits: map[string]kits.Kit{"dsh": k}, Catalog: store, Policy: pol, Gate: gate.New(store, pol)}, func(e contracts.Event) {
		mu.Lock()
		defer mu.Unlock()
		if e.Output != nil {
			output.WriteString(e.Output.Text)
		}
		if e.Audit != nil {
			audit = append(audit, *e.Audit)
		}
	}, orchestrator.Timeouts{})
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Shutdown(context.Background())
	ws, err := os.MkdirTemp("/private/tmp", "harbor-dsh-integration-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(ws)
	os.WriteFile(filepath.Join(ws, "input.txt"), []byte("HARBOR-DSH-INPUT"), 0600)
	for round := 0; round < 2; round++ {
		snapshot, err := svc.Create(ctx, contracts.SessionSpec{AgentID: "dsh", Workspace: ws, ModelAccountID: model.ID, MCPIDs: []string{mcp.ID}, Task: "Read input.txt, write result.txt, call approved_read, and return result."})
		if err != nil {
			t.Fatal(err)
		}
		for !snapshot.CleanupComplete {
			select {
			case <-ctx.Done():
				t.Fatal("任务未结束")
			case <-time.After(25 * time.Millisecond):
			}
			snapshot, err = svc.Get(ctx, snapshot.SessionID)
			if err != nil {
				t.Fatal(err)
			}
		}
		mu.Lock()
		text := output.String()
		mu.Unlock()
		if snapshot.State != "stopped" {
			t.Fatalf("会话失败 %+v\n%s", snapshot, text)
		}
		raw, _ := os.ReadFile(filepath.Join(ws, "result.txt"))
		if string(raw) != "HARBOR-DSH-EDIT-OK\n" {
			t.Fatalf("文件结果不符: %q\n%s", raw, text)
		}
		if !strings.Contains(text, "HARBOR-DSH-MCP-OK") || !strings.Contains(text, "HARBOR-DSH-COMPLETE") {
			t.Fatalf("缺少真实工具或最终输出\n%s", text)
		}
	}
	if modelCalls.Load() != 10 || mcpCalls.Load() != 2 {
		t.Fatalf("调用数不符: model=%d mcp=%d", modelCalls.Load(), mcpCalls.Load())
	}
	// 真实 dsh 请求仍在途时停止，必须取消上游并在清理完成后才能再次创建。
	mode.Store(1)
	spec := contracts.SessionSpec{AgentID: "dsh", Workspace: ws, ModelAccountID: model.ID, MCPIDs: []string{mcp.ID}, Task: "Return OK"}
	active, err := svc.Create(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("取消测试未到达模型上游")
	}
	if _, err = svc.Stop(ctx, active.SessionID); err != nil {
		t.Fatal(err)
	}
	for !active.CleanupComplete {
		select {
		case <-ctx.Done():
			t.Fatal("取消清理超时")
		case <-time.After(25 * time.Millisecond):
		}
		active, err = svc.Get(ctx, active.SessionID)
		if err != nil {
			t.Fatal(err)
		}
	}
	select {
	case <-upstreamCanceled:
	case <-time.After(time.Second):
		t.Fatal("上游请求未取消")
	}
	if _, err = svc.Stop(ctx, active.SessionID); err != nil {
		t.Fatal("重复停止失败", err)
	}
	// 真实进程遇认证失败不能报告成功，清理后再次运行应恢复。
	mode.Store(2)
	failed, err := svc.Create(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	for !failed.CleanupComplete {
		select {
		case <-ctx.Done():
			t.Fatal("认证故障清理超时")
		case <-time.After(25 * time.Millisecond):
		}
		failed, err = svc.Get(ctx, failed.SessionID)
		if err != nil {
			t.Fatal(err)
		}
	}
	if failed.State != "failed" {
		t.Fatal("上游认证失败未反映到真实 dsh 会话")
	}
	mode.Store(0)
	modelCalls.Store(0)
	recovered, err := svc.Create(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	for !recovered.CleanupComplete {
		select {
		case <-ctx.Done():
			t.Fatal("恢复运行超时")
		case <-time.After(25 * time.Millisecond):
		}
		recovered, err = svc.Get(ctx, recovered.SessionID)
		if err != nil {
			t.Fatal(err)
		}
	}
	if recovered.State != "stopped" || modelCalls.Load() != 5 || mcpCalls.Load() != 3 {
		t.Fatalf("故障后恢复失败 %+v", recovered)
	}
	mu.Lock()
	defer mu.Unlock()
	if strings.Contains(output.String(), "fixture-upstream-") || strings.Contains(output.String(), "harbor_sentinel_") {
		t.Fatal("公开输出泄露凭据")
	}
	found := false
	for _, a := range audit {
		if a.Tool == "approved_read" && a.Result == "allowed" {
			found = true
		}
	}
	if !found {
		t.Fatal("缺少 MCP 审计")
	}
}
