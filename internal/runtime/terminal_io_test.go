//go:build darwin

package runtime

import (
	"context"
	"errors"
	"github.com/liyufei916/agent-harbor/internal/contracts"
	"os/exec"
	"testing"
	"time"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
)

// PTY 的文件描述符也必须接入 Go poller，否则取消无法中断满缓冲区写入。
func TestTerminalInputCancellation(t *testing.T) {
	r := NewMacOS()
	e := &entry{}
	// 只测试宿主终端传输：固定 sleep 不读取输入，不接触模型或工作区。
	process, err := r.execTerminal(context.Background(), e, exec.Command("/bin/sleep", "10"), LaunchSpec{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	p := process.(*macProcess)
	defer func() { p.kill(true); p.Wait() }()
	termios, err := pty.GetsizeFull(p.terminal)
	if err != nil || termios.Cols != 80 {
		t.Fatalf("%+v %v", termios, err)
	}
	// 先通过终端从属端关闭回显和规范行模式，确保测试的是流控阻塞。
	attrs, err := unix.IoctlGetTermios(int(p.terminal.Fd()), unix.TIOCGETA)
	if err != nil {
		t.Fatal(err)
	}
	attrs.Lflag &^= unix.ECHO | unix.ICANON
	if err := unix.IoctlSetTermios(int(p.terminal.Fd()), unix.TIOCSETA, attrs); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- p.Write(ctx, make([]byte, 4<<20)) }()
	select {
	case err := <-done:
		var public *contracts.AppError
		if !errors.As(err, &public) || public.Code != contracts.ErrCancelled {
			t.Fatalf("写入未被 context 取消: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("终端写入取消未生效")
	}
}
