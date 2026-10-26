//go:build darwin

package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/liyufei916/agent-harbor/internal/contracts"
)

func TestMacOSPathGuards(t *testing.T) {
	for _, path := range []string{"/", "/usr", "/private/tmp", "/etc", "/System", "/Library"} {
		if !forbiddenWorkspace(path) {
			t.Errorf("forbiddenWorkspace(%q) = false", path)
		}
		if safeInstallationRoot(path) {
			t.Errorf("safeInstallationRoot(%q) = true", path)
		}
	}
	if !within("/opt/homebrew", "/opt/homebrew/bin/node") || !safeInstallationRoot("/opt/homebrew/bin") {
		t.Fatal("installation path guard mismatch")
	}
}

func TestMacOSHealthCancellationAndLookup(t *testing.T) {
	r := NewMacOS()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := r.Health(ctx); err == nil || err.(*contracts.AppError).Code != "CANCELLED" {
		t.Fatalf("Health cancelled = %v", err)
	}
	if _, err := r.lookup(Handle{ID: "missing"}); err == nil {
		t.Fatal("missing runtime handle accepted")
	}
	_ = r.HealthFor(context.Background(), true)
	if _, err := ProbeExecutable(context.Background(), "/definitely/missing", nil); err == nil {
		t.Fatal("missing executable probe succeeded")
	}
	if output, err := ProbeExecutable(context.Background(), "/usr/bin/true", nil); err != nil || output != "" {
		t.Fatalf("true probe: %q %v", output, err)
	}
	workspace := t.TempDir()
	for _, spec := range []RuntimeSpec{
		{Workspace: workspace, MountMode: "direct", ReadOnlyPaths: []string{"/usr/bin/true"}},
		{SessionID: "s", Workspace: workspace, MountMode: "invalid", ReadOnlyPaths: []string{"/usr/bin/true"}},
		{SessionID: "s", Workspace: workspace, MountMode: "direct"},
	} {
		if _, err := r.create(context.Background(), spec, true); err == nil {
			t.Fatalf("invalid create spec accepted: %#v", spec)
		}
	}
	for _, spec := range []RuntimeSpec{
		{SessionID: "s", Workspace: filepath.Join(workspace, "missing"), MountMode: "direct", ReadOnlyPaths: []string{"/usr/bin/true"}},
		{SessionID: "s", Workspace: "/usr", MountMode: "direct", ReadOnlyPaths: []string{"/usr/bin/true"}},
		{SessionID: "s", Workspace: workspace, MountMode: "direct", ReadOnlyPaths: []string{filepath.Join(workspace, "missing")}},
		{SessionID: "s", Workspace: workspace, MountMode: "direct", ReadOnlyPaths: []string{workspace}},
	} {
		if _, err := r.create(context.Background(), spec, true); err == nil {
			t.Fatalf("unsafe create spec accepted: %#v", spec)
		}
	}
}

func TestMacProcessGuards(t *testing.T) {
	p := &macProcess{rootDone: make(chan struct{})}
	if err := p.Write(context.Background(), []byte("x")); err == nil {
		t.Fatal("Write without input succeeded")
	}
	for _, size := range [][2]int{{0, 1}, {1, 0}, {1001, 1}} {
		if err := p.Resize(size[0], size[1]); err == nil {
			t.Fatalf("invalid resize accepted: %v", size)
		}
	}
	if err := p.Resize(80, 24); err == nil {
		t.Fatal("resize without terminal succeeded")
	}
	close(p.rootDone)
	if err := p.Resize(80, 24); err == nil {
		t.Fatal("resize after exit succeeded")
	}
	p.kill(true)
}

func TestMacOSExecValidationAndDestroy(t *testing.T) {
	newRuntime := func() (*MacOS, Handle) {
		r := NewMacOS()
		root := t.TempDir()
		home := filepath.Join(root, "home")
		if err := os.Mkdir(home, 0700); err != nil {
			t.Fatal(err)
		}
		h := Handle{ID: "fixture", PrivateHome: home, Workspace: root}
		r.entries[h.ID] = &entry{handle: h, root: root, workspace: root, profile: filepath.Join(root, "profile"), readOnly: map[string]bool{"/usr/bin/true": true}}
		return r, h
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r, h := newRuntime()
	if _, err := r.Exec(ctx, h, LaunchSpec{Command: "/usr/bin/true"}, nil); err == nil {
		t.Fatal("cancelled Exec succeeded")
	}
	r, h = newRuntime()
	r.entries[h.ID].attempted = true
	if _, err := r.Exec(context.Background(), h, LaunchSpec{Command: "/usr/bin/true"}, nil); err == nil {
		t.Fatal("reused Exec succeeded")
	}
	for _, launch := range []LaunchSpec{
		{Command: "/usr/bin/false", Terminal: true, Interactive: true},
		{Command: "/usr/bin/true", Args: []string{"bad\x00arg"}},
		{Command: "/usr/bin/true", Env: map[string]string{"PATH": "/tmp"}},
	} {
		r, h = newRuntime()
		if _, err := r.Exec(context.Background(), h, launch, nil); err == nil {
			t.Fatalf("invalid launch succeeded: %#v", launch)
		}
	}
	r, h = newRuntime()
	if err := r.Destroy(context.Background(), h); err != nil {
		t.Fatal(err)
	}
	if err := r.Destroy(context.Background(), h); err != nil {
		t.Fatal("Destroy should be idempotent: ", err)
	}
}

func TestMacOSCreateExecDestroyClosedLoop(t *testing.T) {
	workspace, err := os.MkdirTemp("/private/tmp", "harbor-closed-loop-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(workspace) })
	r := NewMacOS()
	h, err := r.create(context.Background(), RuntimeSpec{
		SessionID: "closed-loop", Workspace: workspace, MountMode: "direct",
		ReadOnlyPaths: []string{"/usr/bin/true"}, AllowDescendants: true,
	}, true)
	if err != nil {
		t.Fatalf("Create closed loop: %v", err)
	}
	process, err := r.Exec(context.Background(), h, LaunchSpec{Command: "/usr/bin/true"}, nil)
	if err != nil {
		_ = r.Destroy(context.Background(), h)
		t.Fatalf("Exec closed loop: %v", err)
	}
	exit := process.Wait()
	if exit.Code != 0 || exit.Cancelled {
		_ = r.Destroy(context.Background(), h)
		t.Fatalf("unexpected exit: %+v", exit)
	}
	if err := r.Destroy(context.Background(), h); err != nil {
		t.Fatal("Destroy closed loop: ", err)
	}
}

func TestMacOSCloneClosedLoop(t *testing.T) {
	workspace, err := os.MkdirTemp("/private/tmp", "harbor-clone-loop-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(workspace) })
	if err := os.WriteFile(filepath.Join(workspace, "fixture.txt"), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	r := NewMacOS()
	h, err := r.create(context.Background(), RuntimeSpec{SessionID: "clone-loop", Workspace: workspace, MountMode: "clone", ReadOnlyPaths: []string{"/usr/bin/true"}, AllowDescendants: true}, true)
	if err != nil {
		t.Fatalf("clone Create: %v", err)
	}
	if h.Workspace == workspace {
		_ = r.Destroy(context.Background(), h)
		t.Fatal("clone workspace points to source")
	}
	if data, err := os.ReadFile(filepath.Join(h.Workspace, "fixture.txt")); err != nil || string(data) != "fixture" {
		_ = r.Destroy(context.Background(), h)
		t.Fatalf("clone content: %q %v", data, err)
	}
	p, err := r.Exec(context.Background(), h, LaunchSpec{Command: "/usr/bin/true"}, nil)
	if err != nil {
		_ = r.Destroy(context.Background(), h)
		t.Fatal(err)
	}
	if got := p.Wait(); got.Code != 0 {
		_ = r.Destroy(context.Background(), h)
		t.Fatalf("clone exit: %+v", got)
	}
	if err := r.Destroy(context.Background(), h); err != nil {
		t.Fatal(err)
	}
}

func TestMacOSInteractiveClosedLoop(t *testing.T) {
	workspace, err := os.MkdirTemp("/private/tmp", "harbor-interactive-loop-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(workspace) })
	r := NewMacOS()
	h, err := r.create(context.Background(), RuntimeSpec{SessionID: "interactive-loop", Workspace: workspace, MountMode: "direct", ReadOnlyPaths: []string{"/bin/cat"}, AllowDescendants: true}, true)
	if err != nil {
		t.Fatalf("interactive Create: %v", err)
	}
	var output string
	ctx, cancel := context.WithCancel(context.Background())
	p, err := r.Exec(ctx, h, LaunchSpec{Command: "/bin/cat", Interactive: true, Stdin: []byte("hello\n")}, func(data contracts.OutputData) {
		output += data.Text
	})
	if err != nil {
		_ = r.Destroy(context.Background(), h)
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	cancel()
	if got := p.Wait(); !got.Cancelled {
		_ = r.Destroy(context.Background(), h)
		t.Fatalf("interactive exit: %+v", got)
	}
	if !strings.Contains(output, "hello") {
		t.Fatalf("interactive output: %q", output)
	}
	if err := r.Destroy(context.Background(), h); err != nil {
		t.Fatal(err)
	}
}

func TestMacOSTerminalClosedLoop(t *testing.T) {
	workspace, err := os.MkdirTemp("/private/tmp", "harbor-terminal-loop-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(workspace) })
	r := NewMacOS()
	h, err := r.create(context.Background(), RuntimeSpec{SessionID: "terminal-loop", Workspace: workspace, MountMode: "direct", ReadOnlyPaths: []string{"/bin/echo"}, AllowDescendants: true}, true)
	if err != nil {
		t.Fatalf("terminal Create: %v", err)
	}
	var output string
	p, err := r.Exec(context.Background(), h, LaunchSpec{Command: "/bin/echo", Args: []string{"terminal-ok"}, Terminal: true}, func(data contracts.OutputData) {
		output += data.Text
	})
	if err != nil {
		_ = r.Destroy(context.Background(), h)
		t.Fatal(err)
	}
	if got := p.Wait(); got.Code != 0 {
		_ = r.Destroy(context.Background(), h)
		t.Fatalf("terminal exit: %+v", got)
	}
	if !strings.Contains(output, "terminal-ok") {
		t.Fatalf("terminal output: %q", output)
	}
	if err := r.Destroy(context.Background(), h); err != nil {
		t.Fatal(err)
	}
}
