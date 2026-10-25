package runtime

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestProcessInputSerializesWholeMessages(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()
	input := newProcessInput(writer, func(data []byte) ([]byte, error) { b, err := json.Marshal(string(data)); return append(b, '\n'), err })
	decoded := make(chan []string, 1)
	go func() {
		decoder := json.NewDecoder(reader)
		var all []string
		for range 12 {
			var s string
			if err := decoder.Decode(&s); err != nil {
				break
			}
			all = append(all, s)
		}
		decoded <- all
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var wg sync.WaitGroup
	for i := range 12 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := input.write(ctx, []byte(strings.Repeat(string(rune('a'+i)), 16000))); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	select {
	case messages := <-decoded:
		seen := map[string]bool{}
		for _, message := range messages {
			seen[message] = true
		}
		if len(seen) != 12 {
			t.Fatalf("并发输入交错或丢失：%d", len(seen))
		}
	case <-ctx.Done():
		t.Fatal("输入未排空")
	}
}

func TestProcessInputCancellationInterruptsBlockedWrite(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()
	input := newProcessInput(writer, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	finished := make(chan error, 1)
	go func() { finished <- input.write(ctx, []byte(strings.Repeat("x", 1<<20))) }()
	select {
	case err := <-finished:
		if err == nil {
			t.Fatal("阻塞写入不应成功")
		}
	case <-time.After(time.Second):
		t.Fatal("取消未中断 stdin 写入")
	}
}
