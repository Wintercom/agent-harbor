package claude

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/liyufei916/agent-harbor/internal/contracts"
	"github.com/liyufei916/agent-harbor/internal/runtime"
)

func TestProbeNodeWrappedClaude(t *testing.T) {
	if os.Getenv("HARBOR_TEST_CLAUDE") != "1" {
		t.Skip("HARBOR_TEST_CLAUDE=1")
	}
	k, err := New("claude", "local-detection-only")
	if err != nil {
		t.Fatal(err)
	}
	launch, err := k.resolveLaunch()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("launch=%+v", launch)

	ws, err := os.MkdirTemp("/private/tmp", "harbor-claude-diag-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(ws)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	r := runtime.NewMacOS()
	if err := r.Health(ctx); err != nil {
		t.Fatalf("health: %v", err)
	}
	handle, err := r.Create(ctx, runtime.RuntimeSpec{
		SessionID: "claude-diag", Workspace: ws, MountMode: "direct",
		AllowedTCP: []string{"127.0.0.1:9"}, ReadOnlyPaths: launch.ReadOnly,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer func() {
		_ = r.Destroy(context.Background(), handle)
	}()
	var out strings.Builder
	proc, err := r.Exec(ctx, handle, runtime.LaunchSpec{
		Command: launch.Command,
		Args:    append(append([]string{}, launch.Args...), "--version"),
	}, func(d contracts.OutputData) {
		out.WriteString(d.Stream + ":")
		out.WriteString(d.Text)
		out.WriteByte('\n')
	})
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	exit := proc.Wait()
	t.Logf("exit=%+v output:\n%s", exit, out.String())
	if exit.Code != 0 {
		t.Fatalf("non-zero exit")
	}
}
