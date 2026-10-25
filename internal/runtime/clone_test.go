package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCloneWorkspaceIndependentFiles(t *testing.T) {
	source, destination := t.TempDir(), filepath.Join(t.TempDir(), "copy")
	if err := os.Mkdir(filepath.Join(source, "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	original := filepath.Join(source, "nested", "file.txt")
	if err := os.WriteFile(original, []byte("原始内容"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := cloneWorkspace(context.Background(), source, destination); err != nil {
		t.Fatal(err)
	}
	copy := filepath.Join(destination, "nested", "file.txt")
	if data, err := os.ReadFile(copy); err != nil || string(data) != "原始内容" {
		t.Fatalf("副本内容错误：%q %v", data, err)
	}
	if err := os.WriteFile(copy, []byte("修改副本"), 0600); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(original); err != nil || string(data) != "原始内容" {
		t.Fatalf("原文件受到影响：%q %v", data, err)
	}
}

func TestCloneWorkspaceRejectsSymlinksAndCancellation(t *testing.T) {
	t.Run("符号链接", func(t *testing.T) {
		source := t.TempDir()
		if err := os.Symlink(t.TempDir(), filepath.Join(source, "outside")); err != nil {
			t.Fatal(err)
		}
		if err := cloneWorkspace(context.Background(), source, filepath.Join(t.TempDir(), "copy")); err == nil {
			t.Fatal("不应复制符号链接")
		}
	})
	t.Run("取消", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := cloneWorkspace(ctx, t.TempDir(), filepath.Join(t.TempDir(), "copy")); err == nil {
			t.Fatal("取消后仍复制")
		}
	})
	t.Run("超大文件", func(t *testing.T) {
		source := t.TempDir()
		file, err := os.Create(filepath.Join(source, "large.bin"))
		if err != nil {
			t.Fatal(err)
		}
		if err := file.Truncate(64<<20 + 1); err != nil {
			file.Close()
			t.Fatal(err)
		}
		file.Close()
		if err := cloneWorkspace(context.Background(), source, filepath.Join(t.TempDir(), "copy")); err == nil {
			t.Fatal("不应复制超限文件")
		}
	})
}
