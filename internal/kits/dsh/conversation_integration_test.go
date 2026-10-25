package dsh

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
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

// 真实固定版 dsh + 实际 OS 沙箱 + Gate；模型是受控 HTTP fixture。
func TestInstalledDSHConversationKeepsContext(t *testing.T) {
	if os.Getenv("HARBOR_TEST_DSH") != "1" {
		t.Skip("需要本机固定版 dsh 安装")
	}
	var mu sync.Mutex
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" || r.Header.Get("X-Api-Key") != "conversation-model-fixture" {
			http.Error(w, "denied", 401)
			return
		}
		var body map[string]any
		if json.NewDecoder(r.Body).Decode(&body) != nil {
			http.Error(w, "invalid", 400)
			return
		}
		mu.Lock()
		calls++
		round := calls
		mu.Unlock()
		messages := fmt.Sprint(body["messages"])
		if round == 2 && (!strings.Contains(messages, "ROUND-ONE") || !strings.Contains(messages, "answer-one") || !strings.Contains(messages, "ROUND-TWO")) {
			t.Error("第二轮模型请求未保留完整历史")
		}
		text := "answer-one"
		if round == 2 {
			text = "answer-two"
		}
		w.Header().Set("Content-Type", "text/event-stream")
		send := func(kind string, payload any) {
			raw, _ := json.Marshal(payload)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", kind, raw)
		}
		send("message_start", map[string]any{"type": "message_start", "message": map[string]any{"id": fmt.Sprint(round), "type": "message", "role": "assistant", "model": "fixture-model", "content": []any{}, "usage": map[string]any{"input_tokens": 1, "output_tokens": 0}}})
		send("content_block_start", map[string]any{"type": "content_block_start", "index": 0, "content_block": map[string]any{"type": "text", "text": text}})
		send("content_block_stop", map[string]any{"type": "content_block_stop", "index": 0})
		send("message_delta", map[string]any{"type": "message_delta", "delta": map[string]any{"stop_reason": "end_turn"}, "usage": map[string]any{"output_tokens": 1}})
		send("message_stop", map[string]any{"type": "message_stop"})
	}))
	defer upstream.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	store := secrets.NewStore()
	model, err := store.Put(ctx, contracts.AccountInfo{Kind: "model", Name: "fixture", ProfileID: policy.ProfileDSHModel, Host: "127.0.0.1"}, []byte("conversation-model-fixture"))
	if err != nil {
		t.Fatal(err)
	}
	pol := policy.NewBuilder(map[string]policy.FixedProfile{policy.ProfileDSHModel: {ID: policy.ProfileDSHModel, Kind: "model", Origin: upstream.URL, Host: "127.0.0.1", Paths: []string{"/v1/messages"}, Protocol: "anthropic", Model: "fixture-model", AllowInsecureLoopback: true}})
	kit, err := NewConversation(os.Getenv("HARBOR_DSH_NODE"), os.Getenv("HARBOR_DSH_INSTALLATION"), "fixture-model")
	if err != nil {
		t.Fatal(err)
	}
	var output strings.Builder
	events := make(chan struct{}, 1)
	svc, err := orchestrator.New(ctx, orchestrator.Dependencies{Runtime: runtime.NewMacOS(), Kits: map[string]kits.Kit{"dsh": kit}, Catalog: store, Policy: pol, Gate: gate.New(store, pol)}, func(e contracts.Event) {
		mu.Lock()
		if e.Pty != nil {
			output.WriteString(e.Pty.Text)
		}
		if e.Output != nil {
			output.WriteString(e.Output.Text)
		}
		mu.Unlock()
		select {
		case events <- struct{}{}:
		default:
		}
	}, orchestrator.Timeouts{})
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Shutdown(context.Background())
	ws, err := os.MkdirTemp("/private/tmp", "harbor-dsh-conversation-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(ws)
	snapshot, err := svc.Create(ctx, contracts.SessionSpec{Workspace: ws, AgentID: "dsh", ModelAccountID: model, Task: "Remember ROUND-ONE"})
	if err != nil {
		t.Fatal(err)
	}
	await := func(marker string) {
		t.Helper()
		for {
			mu.Lock()
			text := output.String()
			mu.Unlock()
			if strings.Contains(text, marker) {
				return
			}
			current, err := svc.Get(ctx, snapshot.SessionID)
			if err != nil || current.CleanupComplete {
				t.Fatalf("会话提前结束: %+v %v\n%s", current, err, text)
			}
			select {
			case <-events:
			case <-ctx.Done():
				t.Fatalf("等待 %s\n%s", marker, text)
			}
		}
	}
	await(`"round":1`)
	if err := svc.WritePty(ctx, contracts.PtyInput{SessionID: snapshot.SessionID, Data: "Now ROUND-TWO"}); err != nil {
		t.Fatal(err)
	}
	await(`"round":2`)
	mu.Lock()
	count, text := calls, output.String()
	mu.Unlock()
	if count != 2 || !strings.Contains(text, "answer-two") || strings.Contains(text, "conversation-model-fixture") || strings.Contains(text, "harbor_sentinel_") {
		t.Fatalf("count=%d\n%s", count, text)
	}
	if _, err := svc.Stop(ctx, snapshot.SessionID); err != nil {
		t.Fatal(err)
	}
}
