//go:build darwin

package runtime

import (
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// killProcessTree 以 root 为锚扩展同 UID 下 PPID 闭包，反复 SIGKILL 直至无存活成员或用尽重试。
// 进程脱离进程组后仍可能漏网；调用方须把超时视为 CLEANUP_FAILED。
func killProcessTree(root int) {
	if root <= 1 {
		return
	}
	seen := map[int]struct{}{root: {}}
	for attempt := 0; attempt < 40; attempt++ {
		expandDescendants(seen)
		alive := 0
		for pid := range seen {
			if pid <= 1 {
				continue
			}
			if err := unix.Kill(pid, 0); err != nil {
				continue
			}
			alive++
			_ = unix.Kill(pid, unix.SIGKILL)
		}
		if alive == 0 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func expandDescendants(seen map[int]struct{}) {
	uid := os.Getuid()
	for {
		added := false
		procs, err := unix.SysctlKinfoProcSlice("kern.proc.uid", uid)
		if err != nil {
			return
		}
		for i := range procs {
			pid := int(procs[i].Proc.P_pid)
			ppid := int(procs[i].Eproc.Ppid)
			if _, ok := seen[ppid]; !ok {
				continue
			}
			if _, exists := seen[pid]; exists {
				continue
			}
			seen[pid] = struct{}{}
			added = true
		}
		if !added {
			return
		}
	}
}
