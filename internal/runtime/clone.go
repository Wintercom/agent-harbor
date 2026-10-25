package runtime

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func sessionRoot(mode, source string) (string, error) {
	if mode != "clone" {
		return os.MkdirTemp("/private/tmp", "agent-harbor-session-")
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	base = filepath.Join(base, "AgentHarbor", "sessions")
	if err := os.MkdirAll(base, 0700); err != nil {
		return "", err
	}
	base, err = filepath.EvalSymlinks(base)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(source, base)
	if err != nil || rel == "." || filepath.IsLocal(rel) {
		return "", runtimeError("WORKSPACE_DENIED")
	}
	return os.MkdirTemp(base, "session-")
}

// cloneWorkspace 使用目录句柄约束读取范围，避免源目录并发变化时越界。
// V1 拒绝所有符号链接与特殊文件；副本不与源文件共享 inode。
func cloneWorkspace(ctx context.Context, source, destination string) error {
	root, err := os.OpenRoot(source)
	if err != nil {
		return runtimeError("WORKSPACE_DENIED")
	}
	defer root.Close()
	if err := os.Mkdir(destination, 0700); err != nil {
		return runtimeError("ISOLATION_UNAVAILABLE")
	}
	remaining := int64(1 << 30)
	count := 0
	return fs.WalkDir(root.FS(), ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if ctx.Err() != nil {
			return runtimeError("CANCELLED")
		}
		if walkErr != nil {
			return runtimeError("WORKSPACE_DENIED")
		}
		if path == "." {
			return nil
		}
		count++
		if count > 100000 || entry.Type()&os.ModeSymlink != 0 {
			return runtimeError("WORKSPACE_DENIED")
		}
		info, err := entry.Info()
		if err != nil {
			return runtimeError("WORKSPACE_DENIED")
		}
		target := filepath.Join(destination, filepath.FromSlash(path))
		if info.IsDir() {
			return os.Mkdir(target, 0700)
		}
		if !info.Mode().IsRegular() || info.Size() > 64<<20 {
			return runtimeError("WORKSPACE_DENIED")
		}
		in, err := root.Open(path)
		if err != nil {
			return runtimeError("WORKSPACE_DENIED")
		}
		defer in.Close()
		opened, err := in.Stat()
		if err != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
			return runtimeError("WORKSPACE_DENIED")
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600|info.Mode().Perm()&0100)
		if err != nil {
			return runtimeError("ISOLATION_UNAVAILABLE")
		}
		defer out.Close()
		limit := min(int64(64<<20), remaining)
		buffer := make([]byte, 32<<10)
		var copied int64
		for {
			if ctx.Err() != nil {
				return runtimeError("CANCELLED")
			}
			n, readErr := in.Read(buffer)
			copied += int64(n)
			if copied > limit {
				return runtimeError("WORKSPACE_DENIED")
			}
			if _, err := out.Write(buffer[:n]); err != nil {
				return runtimeError("ISOLATION_UNAVAILABLE")
			}
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				return runtimeError("WORKSPACE_DENIED")
			}
		}
		remaining -= copied
		if err := out.Close(); err != nil {
			return runtimeError("ISOLATION_UNAVAILABLE")
		}
		return nil
	})
}
