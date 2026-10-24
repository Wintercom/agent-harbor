//go:build darwin && cgo

package runtime

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/liyufei916/agent-harbor/internal/contracts"
	posixprobe "github.com/liyufei916/agent-harbor/internal/runtime/testdata/posix"
)

var probePath string

func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "--harbor-runtime-probe" {
		os.Exit(helper(os.Args[2:]))
	}
	var err error
	probePath, err = os.Executable()
	if err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func helper(args []string) int {
	if len(args) == 0 {
		return 2
	}
	denied := func(err error) int {
		if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
			return 13
		}
		return 14
	}
	switch args[0] {
	case "read":
		data, err := os.ReadFile(args[1])
		if err != nil {
			return denied(err)
		}
		_, _ = os.Stdout.Write(data)
	case "write":
		if err := os.WriteFile(args[1], []byte("written"), 0600); err != nil {
			return denied(err)
		}
	case "fork", "spawn", "spawn-group":
		command := exec.Command("/usr/bin/true")
		if args[0] == "spawn-group" {
			command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		}
		if err := command.Run(); err != nil {
			return denied(err)
		}
	case "posix-spawn", "posix-spawn-group":
		if code := posixprobe.Spawn(args[0] == "posix-spawn-group"); code != 0 {
			return denied(syscall.Errno(code))
		}
	case "setsid":
		if _, err := syscall.Setsid(); err != nil {
			return denied(err)
		}
	case "setpgid":
		if err := syscall.Setpgid(0, 0); err != nil {
			return denied(err)
		}
	case "connect", "udp", "connect6", "unix":
		network, address := "tcp4", ""
		switch args[0] {
		case "connect":
			address = net.JoinHostPort(args[1], args[2])
		case "udp":
			network, address = "udp4", net.JoinHostPort(args[1], args[2])
		case "connect6":
			network, address = "tcp6", net.JoinHostPort("::1", args[1])
		case "unix":
			network, address = "unix", args[1]
		}
		connection, err := net.DialTimeout(network, address, time.Second)
		if err != nil {
			return denied(err)
		}
		connection.Close()
	case "env":
		for _, item := range os.Environ() {
			fmt.Println(item)
		}
	case "output":
		_, _ = os.Stdout.Write([]byte{0xe4})
		time.Sleep(10 * time.Millisecond)
		_, _ = os.Stdout.Write([]byte{0xb8, 0xad})
		_, _ = os.Stderr.WriteString("final")
	case "sleep":
		for {
			time.Sleep(time.Hour)
		}
	default:
		return 2
	}
	return 0
}

// 证明仅拦截 setsid/setpgid 不足以约束进程组，禁止据此开放生产 fork。
func TestNativeSpawnBypassesSyscallGroupRestrictions(t *testing.T) {
	r := NewMacOS()
	spec := probeSpec(t)
	for _, mode := range []string{"fork", "posix-spawn", "setsid", "setpgid", "posix-spawn-group"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			handle, err := r.Create(ctx, spec)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				cleanup, stop := context.WithTimeout(context.Background(), time.Second)
				defer stop()
				if err := r.Destroy(cleanup, handle); err != nil {
					t.Error(err)
				}
			}()
			e, _ := r.lookup(handle)
			original, err := os.ReadFile(e.profile)
			if err != nil {
				t.Fatal(err)
			}
			profile := strings.Replace(string(original), "(deny process-fork)", "(allow process-fork)\n(deny syscall-unix (syscall-number SYS_setsid SYS_setpgid))", 1)
			if err := os.WriteFile(e.profile, []byte(profile), 0600); err != nil {
				t.Fatal(err)
			}
			process, err := r.Exec(ctx, handle, LaunchSpec{Command: probePath, Args: []string{"--harbor-runtime-probe", mode}}, nil)
			if err != nil {
				t.Fatal(err)
			}
			exit := process.Wait()
			want := 13
			if mode == "fork" || mode == "posix-spawn" || mode == "posix-spawn-group" {
				want = 0
			}
			if exit.Code != want {
				t.Fatalf("候选规则行为变化，须重新评估进程组约束：mode=%s exit=%+v", mode, exit)
			}
		})
	}
}

func workspace(t *testing.T) string {
	t.Helper()
	path, err := os.MkdirTemp("/private/tmp", `harbor workspace "中文"-`)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(path) })
	return path
}

func listener(t *testing.T, network, address string) net.Listener {
	t.Helper()
	ln, err := net.Listen(network, address)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			connection, err := ln.Accept()
			if err != nil {
				return
			}
			connection.Close()
		}
	}()
	return ln
}

func runProbe(t *testing.T, r *MacOS, spec RuntimeSpec, args ...string) (Exit, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	handle, err := r.Create(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := r.Destroy(ctx, handle); err != nil {
			t.Error(err)
		}
	})
	var mu sync.Mutex
	var output strings.Builder
	process, err := r.Exec(ctx, handle, LaunchSpec{Command: probePath, Args: append([]string{"--harbor-runtime-probe"}, args...)}, func(data contracts.OutputData) { mu.Lock(); defer mu.Unlock(); output.WriteString(data.Text) })
	if err != nil {
		t.Fatal(err)
	}
	exit := process.Wait()
	if exit.Code < 0 {
		t.Logf("探针进程状态：%s", process.(*macProcess).cmd.ProcessState.String())
	}
	mu.Lock()
	text := output.String()
	mu.Unlock()
	return exit, text
}

func probeSpec(t *testing.T) RuntimeSpec {
	t.Helper()
	ln := listener(t, "tcp4", "127.0.0.1:0")
	return RuntimeSpec{SessionID: "test", Workspace: workspace(t), MountMode: "direct", AllowedTCP: []string{ln.Addr().String()}, ReadOnlyPaths: []string{probePath}}
}

func TestMacOSFileBoundaryAndNoFork(t *testing.T) {
	r := NewMacOS()
	spec := probeSpec(t)
	allowed := filepath.Join(spec.Workspace, "allowed.txt")
	if err := os.WriteFile(allowed, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(workspace(t), "outside.txt")
	if err := os.WriteFile(outside, []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(spec.Workspace, "escape")); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		args []string
		code int
		text string
	}{
		{"read", []string{"read", allowed}, 0, "fixture"},
		{"write", []string{"write", filepath.Join(spec.Workspace, "created")}, 0, ""},
		{"outside", []string{"read", outside}, 13, ""},
		{"outside-write", []string{"write", outside}, 13, ""},
		{"symlink", []string{"read", filepath.Join(spec.Workspace, "escape")}, 13, ""},
		{"fork", []string{"fork"}, 13, ""},
		{"posix-spawn", []string{"posix-spawn"}, 13, ""},
		{"posix-spawn-new-group", []string{"posix-spawn-group"}, 13, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			exit, text := runProbe(t, r, spec, tc.args...)
			if exit.Code != tc.code || text != tc.text {
				t.Fatalf("退出=%+v 输出=%q", exit, text)
			}
		})
	}
}

func TestMacOSExactGateNetwork(t *testing.T) {
	r := NewMacOS()
	spec := probeSpec(t)
	_, port, _ := net.SplitHostPort(spec.AllowedTCP[0])
	other := listener(t, "tcp4", "127.0.0.1:0")
	_, otherPort, _ := net.SplitHostPort(other.Addr().String())
	listener(t, "tcp6", net.JoinHostPort("::1", port))
	socket := filepath.Join(workspace(t), "control.sock")
	listener(t, "unix", socket)
	for _, tc := range []struct {
		name string
		args []string
		code int
	}{
		{"gate", []string{"connect", "127.0.0.1", port}, 0},
		{"other-port", []string{"connect", "127.0.0.1", otherPort}, 13},
		{"other-loopback-ip", []string{"connect", "127.0.0.2", port}, 13},
		{"ipv6-same-port", []string{"connect6", port}, 13},
		{"udp-same-port", []string{"udp", "127.0.0.1", port}, 13},
		{"unix-control", []string{"unix", socket}, 13},
		// TEST-NET-1 无真实上游；内核应在连接发起时立即拒绝。
		{"direct-ip", []string{"connect", "192.0.2.1", "443"}, 13},
	} {
		t.Run(tc.name, func(t *testing.T) {
			exit, text := runProbe(t, r, spec, tc.args...)
			if exit.Code != tc.code {
				t.Fatalf("退出=%+v 输出=%q", exit, text)
			}
		})
	}
}

func TestMacOSPrivateHomeEnvironmentAndFinalOutput(t *testing.T) {
	t.Setenv("HARBOR_TEST_SECRET", "host-fixture-do-not-inherit")
	r := NewMacOS()
	spec := probeSpec(t)
	exit, text := runProbe(t, r, spec, "env")
	if exit.Code != 0 || strings.Contains(text, "host-fixture") || strings.Contains(text, "HARBOR_TEST_SECRET") || !strings.Contains(text, "HOME=/private/tmp/agent-harbor-session-") {
		t.Fatalf("环境未净化：退出=%+v 输出=%q", exit, text)
	}
	exit, text = runProbe(t, r, spec, "output")
	if exit.Code != 0 || !strings.Contains(text, "中") || !strings.Contains(text, "final") {
		t.Fatalf("Wait 未排空输出：%+v %q", exit, text)
	}
}

func TestMacOSCancelDestroyAndRepeatedWait(t *testing.T) {
	r := NewMacOS()
	spec := probeSpec(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handle, err := r.Create(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	process, err := r.Exec(ctx, handle, LaunchSpec{Command: probePath, Args: []string{"--harbor-runtime-probe", "sleep"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	cleanup, stop := context.WithTimeout(context.Background(), time.Second)
	defer stop()
	if err := r.Destroy(cleanup, handle); err != nil {
		t.Fatal(err)
	}
	first, second := process.Wait(), process.Wait()
	if !first.Cancelled || first.Code != second.Code || first.Cancelled != second.Cancelled {
		t.Fatalf("等待不幂等：%+v %+v", first, second)
	}
	if _, err := os.Stat(handle.PrivateHome); !os.IsNotExist(err) {
		t.Fatal("私有 HOME 未清理")
	}
	if err := r.Destroy(cleanup, handle); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Exec(context.Background(), handle, LaunchSpec{Command: probePath}, nil); err == nil {
		t.Fatal("已销毁句柄仍可执行")
	}
}

func TestMacOSDestroyReleasesSensitiveMetadataButKeepsIdempotentTombstone(t *testing.T) {
	r := NewMacOS()
	spec := probeSpec(t)
	handle, err := r.Create(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Destroy(context.Background(), handle); err != nil {
		t.Fatal(err)
	}
	r.mu.Lock()
	entry := r.entries[handle.ID]
	r.mu.Unlock()
	if entry == nil {
		t.Fatal("缺少幂等墓碑")
	}
	entry.mu.Lock()
	root, workspace, profile, process, readOnly, destroyed := entry.root, entry.workspace, entry.profile, entry.process, entry.readOnly, entry.destroyed
	entry.mu.Unlock()
	if root != "" || workspace != "" || profile != "" || process != nil || readOnly != nil || !destroyed {
		t.Fatalf("清理后仍保留敏感资源元数据：root=%q workspace=%q profile=%q process=%v readOnly=%v destroyed=%v", root, workspace, profile, process, readOnly, destroyed)
	}
	if err := r.Destroy(context.Background(), handle); err != nil {
		t.Fatal(err)
	}
}

func TestMacOSCleanupWaitsForOutputAndCanRetry(t *testing.T) {
	r := NewMacOS()
	spec := probeSpec(t)
	handle, err := r.Create(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	process, err := r.Exec(context.Background(), handle, LaunchSpec{Command: probePath, Args: []string{"--harbor-runtime-probe", "output"}}, func(contracts.OutputData) { once.Do(func() { close(entered); <-release }) })
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("未收到输出回调")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	err = r.Destroy(ctx, handle)
	cancel()
	if err == nil {
		t.Fatal("输出回调未排空却报告清理完成")
	}
	if _, err := os.Stat(handle.PrivateHome); err != nil {
		t.Fatal("未确认清理前删除了 HOME")
	}
	close(release)
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := r.Destroy(ctx, handle); err != nil {
		t.Fatal(err)
	}
	_ = process.Wait()
}

func TestMacOSRejectsUnsafeSpecs(t *testing.T) {
	r := NewMacOS()
	base := probeSpec(t)
	for _, modify := range []func(*RuntimeSpec){
		func(s *RuntimeSpec) { s.MountMode = "clone" }, func(s *RuntimeSpec) { s.Workspace = "/" },
		func(s *RuntimeSpec) { s.AllowedTCP = []string{"0.0.0.0:1234"} },
		func(s *RuntimeSpec) { s.AllowedTCP = []string{"127.0.0.1:" + strconv.Itoa(65536)} },
		func(s *RuntimeSpec) { s.ReadOnlyPaths = []string{"/usr/bin/sudo"} },
	} {
		spec := base
		modify(&spec)
		if handle, err := r.Create(context.Background(), spec); err == nil {
			_ = r.Destroy(context.Background(), handle)
			t.Fatal("接受不安全运行规格")
		}
	}
}

func TestHealthRejectsProfileThatFailsOpen(t *testing.T) {
	original := profileTemplate
	t.Cleanup(func() { profileTemplate = original })
	profileTemplate = strings.Replace(original, "(deny default)", "(allow default)", 1)
	if err := NewMacOS().Health(context.Background()); err == nil {
		t.Fatal("profile 已允许任意文件读取，Health 却报告可用")
	}
}

func TestHealthRejectsProfileThatAllowsDescendants(t *testing.T) {
	original := profileTemplate
	t.Cleanup(func() { profileTemplate = original })
	profileTemplate = strings.Replace(original, "(deny process-fork)", "(allow process-fork)", 1)
	if err := NewMacOS().Health(context.Background()); err == nil {
		t.Fatal("profile 允许创建后代，Health 却报告可用")
	}
}
