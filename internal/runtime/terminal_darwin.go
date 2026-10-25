//go:build darwin

package runtime

import (
	"context"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/creack/pty"
	"github.com/liyufei916/agent-harbor/internal/contracts"
	"golang.org/x/sys/unix"
)

// 终端由可信宿主创建，子进程进入相同 Seatbelt 规则；不开放 /dev 或 fork。
func (r *MacOS) execTerminal(ctx context.Context, e *entry, cmd *exec.Cmd, launch LaunchSpec, output func(contracts.OutputData)) (Process, error) {
	master, slave, err := pty.Open()
	if err != nil {
		return nil, runtimeError("ISOLATION_UNAVAILABLE")
	}
	defer slave.Close()
	// 在 NewFile 之前设置 nonblock，让 Go poller 支持可取消读写期限。
	fd, err := unix.Dup(int(master.Fd()))
	if err != nil {
		master.Close()
		return nil, runtimeError("ISOLATION_UNAVAILABLE")
	}
	master.Close()
	unix.CloseOnExec(fd)
	if err = unix.SetNonblock(fd, true); err != nil {
		unix.Close(fd)
		return nil, runtimeError("ISOLATION_UNAVAILABLE")
	}
	master = os.NewFile(uintptr(fd), "harbor-pty")
	if err = pty.Setsize(master, &pty.Winsize{Cols: 80, Rows: 24}); err != nil {
		master.Close()
		return nil, runtimeError("ISOLATION_UNAVAILABLE")
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = slave, slave, slave
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	cmd.Env = append(cmd.Env, "TERM=xterm-256color")
	if err = cmd.Start(); err != nil {
		master.Close()
		return nil, runtimeError("ISOLATION_UNAVAILABLE")
	}
	slave.Close()
	p := &macProcess{cmd: cmd, done: make(chan struct{}), rootDone: make(chan struct{}), terminal: master, input: newProcessInput(master, nil)}
	e.process = p
	var readers sync.WaitGroup
	readers.Add(1)
	go func() { defer readers.Done(); readOutput(master, "stdout", output) }()
	go func() {
		err := cmd.Wait()
		close(p.rootDone)
		readers.Wait()
		p.mu.Lock()
		master.Close()
		p.result = Exit{Code: -1, Cancelled: p.cancelled}
		if cmd.ProcessState != nil {
			p.confirmed = true
			p.result.Code = cmd.ProcessState.ExitCode()
		}
		if err != nil {
			p.result.Err = runtimeError("AGENT_EXIT_FAILED")
		}
		p.mu.Unlock()
		close(p.done)
	}()
	go func() {
		select {
		case <-ctx.Done():
			p.kill(true)
		case <-p.rootDone:
		}
	}()
	if len(launch.Stdin) > 0 {
		if err := p.Write(ctx, launch.Stdin); err != nil {
			p.kill(true)
			return p, runtimeError("ISOLATION_UNAVAILABLE")
		}
	}
	return p, nil
}
