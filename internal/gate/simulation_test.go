package gate

import (
	"context"
	"os"
	"testing"

	"github.com/liyufei916/agent-harbor/internal/contracts"
	"github.com/liyufei916/agent-harbor/internal/policy"
	"github.com/liyufei916/agent-harbor/internal/secrets"
)

// 独立 Postman 进程作为上游；运行真实 Connections HTTP 客户端，不替换 Transport。
func TestPostmanProtocolSimulation(t *testing.T) {
	origin := os.Getenv("HARBOR_TEST_SIMULATION_ORIGIN")
	if origin == "" {
		t.Skip("需要本机 Postman simulation")
	}
	p := policy.NewBuilder(map[string]policy.FixedProfile{
		"model": {ID: "model", Kind: "model", Protocol: "anthropic", Model: "fixture-model-name", Origin: origin, Host: "127.0.0.1", TestPath: "/upstream/v1/messages", AllowInsecureLoopback: true},
		"mcp":   {ID: "mcp", Kind: "mcp", Protocol: "streamable-http", Origin: origin, Host: "127.0.0.1", TestPath: "/upstream/mcp", Tools: []string{"approved_read"}, AllowInsecureLoopback: true},
	})
	store := secrets.NewStore()
	conn := NewConnections(store, p)
	if _, err := conn.ConnectModel(context.Background(), contracts.ConnectInput{ProfileID: "model", Name: "fixture", APIKey: "fixture-model"}); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ConnectMcp(context.Background(), contracts.ConnectInput{ProfileID: "mcp", Name: "fixture", APIKey: "fixture-mcp"}); err != nil {
		t.Fatal(err)
	}
	_, err := conn.ConnectMcp(context.Background(), contracts.ConnectInput{ProfileID: "mcp", Name: "invalid", APIKey: "fixture-invalid"})
	if err != ErrUpstreamAuth || store.Count() != 2 {
		t.Fatalf("错误认证仍保存账号：%v count=%d", err, store.Count())
	}
}
