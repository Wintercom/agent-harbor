package orchestrator

import (
	"context"
	"testing"

	"github.com/liyufei916/agent-harbor/internal/contracts"
	"github.com/liyufei916/agent-harbor/internal/kits"
	"github.com/liyufei916/agent-harbor/internal/policy"
)

func TestBoundaryReportsOnlyMountedSessionResources(t *testing.T) {
	f := newFixture()
	p := redirectPolicy{policy.PolicySnapshot{Workspace: "/test/workspace", MountMode: "clone", Model: policy.EndpointRule{Origin: "https://models.example.test"}, MCPMounts: []policy.MCPMount{{AccountID: "mcp", Endpoint: policy.EndpointRule{Origin: "https://tools.example.test"}}}}}
	o, err := New(context.Background(), Dependencies{Runtime: f, Kits: map[string]kits.Kit{"claude": f}, Catalog: f, Policy: p, Gate: f}, f.record, Timeouts{})
	if err != nil {
		t.Fatal(err)
	}
	defer o.Shutdown(context.Background())
	input := spec()
	input.MountMode = "clone"
	s, err := o.Create(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	awaitState(t, o, s.SessionID, contracts.StateRunning)
	boundary, err := o.Boundary(context.Background(), s.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if boundary.Workspace != "/private/test-copy" || len(boundary.AllowedHosts) != 2 || boundary.AllowedHosts[0] != "models.example.test" || boundary.AllowedHosts[1] != "tools.example.test" || len(boundary.MCPSummary) != 1 {
		t.Fatalf("边界不对应实际挂载: %+v", boundary)
	}
	boundary.AllowedHosts[0] = "injected.example.test"
	again, err := o.Boundary(context.Background(), s.SessionID)
	if err != nil || again.AllowedHosts[0] != "models.example.test" {
		t.Fatal("调用方改写了边界快照")
	}
	project, err := o.ProjectWorkspace(context.Background(), s.SessionID)
	if err != nil || project != input.Workspace || project == boundary.Workspace {
		t.Fatalf("项目规则错误绑定到了副本: %q %v", project, err)
	}
	_, err = o.Boundary(context.Background(), "unknown")
	assertCode(t, err, "SESSION_NOT_FOUND")
}
