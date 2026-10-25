//go:build darwin && harbor_platform

package runtime

import (
	"context"
	"github.com/liyufei916/agent-harbor/internal/contracts"
	"strings"
	"testing"
	"time"
)

func TestRealTerminalEchoResizeAndClose(t *testing.T) {
	r := NewMacOS()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	h, err := r.Create(ctx, probeSpec(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := r.Destroy(context.Background(), h); err != nil {
			t.Error(err)
		}
	}()
	output := make(chan string, 64)
	process, err := r.Exec(ctx, h, LaunchSpec{Command: probePath, Args: []string{"--harbor-runtime-probe", "terminal"}, Terminal: true}, func(data contracts.OutputData) { output <- data.Text })
	if err != nil {
		t.Fatal(err)
	}
	p := process.(InteractiveProcess)
	text := ""
	await := func(want string) {
		t.Helper()
		for !strings.Contains(text, want) {
			select {
			case data := <-output:
				text += data
			case <-ctx.Done():
				t.Fatalf("等待 %q，实际 %q", want, text)
			}
		}
	}
	await("tty:80x24")
	if err := p.Resize(120, 40); err != nil {
		t.Fatal(err)
	}
	if err := p.Write(ctx, []byte("size\n")); err != nil {
		t.Fatal(err)
	}
	await("tty:120x40")
	if err := p.Write(ctx, []byte("中文-round-two\n")); err != nil {
		t.Fatal(err)
	}
	await("received:中文-round-two")
	if err := p.Resize(0, 40); err == nil {
		t.Fatal("接受非法窗口")
	}
	if err := r.Destroy(ctx, h); err != nil {
		t.Fatal(err)
	}
	process.Wait()
	if p.Write(ctx, []byte("late")) == nil || p.Resize(80, 24) == nil {
		t.Fatal("终态终端仍可操作")
	}
}
