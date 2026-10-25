package claude

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/liyufei916/agent-harbor/internal/runtime"
)

func TestProbeNodeWrappedClaude(t *testing.T) {
	if os.Getenv("HARBOR_TEST_CLAUDE") != "1" {
		t.Skip("HARBOR_TEST_CLAUDE=1")
	}
	node, err := filepath.EvalSymlinks("/opt/homebrew/bin/node")
	if err != nil {
		t.Skip(err)
	}
	script, err := filepath.EvalSymlinks("/opt/homebrew/bin/claude")
	if err != nil {
		t.Skip(err)
	}
	root := filepath.Dir(script)
	out, err := runtime.ProbeInstallation(context.Background(), node, []string{script, "--version"}, []string{root})
	if err != nil {
		t.Fatalf("version probe: %v", err)
	}
	t.Logf("version=%q", out)
	help, err := runtime.ProbeInstallation(context.Background(), node, []string{script, "--bare", "--help"}, []string{root})
	if err != nil {
		t.Fatalf("help probe: %v", err)
	}
	t.Logf("help bytes=%d", len(help))
}
