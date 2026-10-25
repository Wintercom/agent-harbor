package policy

import "testing"

func TestCustomMCPProfileHostOnly(t *testing.T) {
	p, err := CustomMCPProfile("https://mcp.context7.com")
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != ProfileCustom || p.Kind != "mcp" || p.Host != "mcp.context7.com" || p.TestPath != "/mcp" {
		t.Fatalf("%+v", p)
	}
	if !p.ValidConnection() {
		t.Fatal("custom mcp 应通过 ValidConnection")
	}
}

func TestCustomMCPProfileWithPath(t *testing.T) {
	p, err := CustomMCPProfile("https://api.example.com/v1/mcp")
	if err != nil {
		t.Fatal(err)
	}
	if p.TestPath != "/v1/mcp" || p.Paths[0] != "/v1/mcp" {
		t.Fatalf("%+v", p)
	}
}

func TestCustomMCPProbeCandidatesAddsMcpSuffix(t *testing.T) {
	cands, err := CustomMCPProbeCandidates("https://llmapi.example.com/v1")
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, c := range cands {
		paths = append(paths, c.TestPath)
	}
	want := []string{"/v1", "/v1/mcp", "/mcp"}
	if len(paths) < 3 || paths[0] != want[0] || paths[1] != want[1] || paths[2] != want[2] {
		t.Fatalf("paths=%v want prefix %v", paths, want)
	}
}
