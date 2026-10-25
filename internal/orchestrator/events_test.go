package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/liyufei916/agent-harbor/internal/contracts"
	"github.com/liyufei916/agent-harbor/internal/gate"
	"github.com/liyufei916/agent-harbor/internal/runtime"
)

func TestSentinelSplitAtEveryBoundary(t *testing.T) {
	for split := 1; split < len(sentinel); split++ {
		filter := streamFilter{token: sentinel}
		text := filter.write("中文前缀"+sentinel[:split]) + filter.write(sentinel[split:]+"中文后缀")
		if text != "中文前缀"+redacted+"中文后缀" || filter.pending != "" {
			t.Fatalf("拆分=%d 输出=%q", split, text)
		}
	}
}

func TestOutputRedactionAndUTF8(t *testing.T) {
	f := newFixture()
	f.beforeExec = func(output func(contracts.OutputData)) {
		for _, stream := range []string{"stdout", "stderr"} {
			output(contracts.OutputData{Stream: stream, Text: "开始" + sentinel[:17]})
			output(contracts.OutputData{Stream: stream, Text: sentinel[17:] + strings.Repeat("中", 16000)})
		}
	}
	o := setup(t, f, Timeouts{}, nil)
	s := create(t, o)
	awaitState(t, o, s.SessionID, contracts.StateRunning)
	f.exit <- runtime.Exit{Code: 0}
	awaitState(t, o, s.SessionID, contracts.StateStopped)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := o.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	streams := map[string]string{}
	for _, e := range f.events {
		if e.Output == nil {
			continue
		}
		text := e.Output.Text
		if len(text) > chunkLimit || !utf8.ValidString(text) || strings.Contains(text, sentinel) {
			t.Fatal("输出块不满足边界约束")
		}
		streams[e.Output.Stream] += text
	}
	for _, stream := range []string{"stdout", "stderr"} {
		if streams[stream] != "开始"+redacted+strings.Repeat("中", 16000) {
			t.Fatalf("%s 输出不完整", stream)
		}
	}
}

func TestSlowSinkBoundsOutputAndKeepsTerminal(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	var mu sync.Mutex
	var received []contracts.Event
	q := newEventQueue(func(e contracts.Event) {
		once.Do(func() { close(entered); <-release })
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	})
	q.push(contracts.Event{SessionID: "session", Seq: 1, Type: "state_changed", State: &contracts.SessionSnapshot{SessionID: "session", Seq: 1, State: contracts.StateRunning}})
	<-entered
	for seq := uint64(2); seq < 200; seq++ {
		q.push(contracts.Event{SessionID: "session", Seq: seq, Type: "output", Output: &contracts.OutputData{Stream: "stdout", Text: strings.Repeat("x", chunkLimit)}})
	}
	q.push(contracts.Event{SessionID: "session", Seq: 200, Type: "error", Error: publicError("INTERNAL_ERROR")})
	q.push(contracts.Event{SessionID: "session", Seq: 201, Type: "state_changed", State: &contracts.SessionSnapshot{SessionID: "session", Seq: 201, State: contracts.StateFailed}})
	q.mu.Lock()
	queued := q.bytes
	q.mu.Unlock()
	if queued > outputLimit {
		t.Fatalf("输出缓存越界=%d", queued)
	}
	q.close()
	close(release)
	select {
	case <-q.done:
	case <-time.After(time.Second):
		t.Fatal("分发未结束")
	}
	mu.Lock()
	defer mu.Unlock()
	previous := uint64(0)
	truncated, notified, terminal := false, false, false
	for _, e := range received {
		if e.Seq <= previous {
			t.Fatal("截断导致事件顺序错误")
		}
		previous = e.Seq
		if e.Output != nil && strings.Contains(e.Output.Text, "输出已截断") {
			truncated = true
		}
		if e.Error != nil {
			notified = true
		}
		if e.Seq == 201 {
			terminal = true
		}
	}
	if !truncated || !notified || !terminal {
		t.Fatal("截断提示、错误或终态丢失")
	}
}

func TestPublicErrorsNeverReuseRawMessage(t *testing.T) {
	for code := range messages {
		app := safeError(&contracts.AppError{Code: code, Message: sentinel, Retryable: true}, "INTERNAL_ERROR")
		if strings.Contains(app.Message, sentinel) {
			t.Fatal("错误泄漏原文")
		}
		if app.Retryable != (code == "UPSTREAM_TIMEOUT" || code == "UPSTREAM_UNAVAILABLE") {
			t.Fatal("重试语义错误")
		}
	}
}

func TestEarlyOutputTruncationIsVisible(t *testing.T) {
	f := newFixture()
	f.beforeExec = func(output func(contracts.OutputData)) {
		for range 80 {
			output(contracts.OutputData{Stream: "stdout", Text: strings.Repeat("x", chunkLimit)})
		}
	}
	o := setup(t, f, Timeouts{}, nil)
	s := create(t, o)
	awaitState(t, o, s.SessionID, contracts.StateRunning)
	_, _ = o.Stop(context.Background(), s.SessionID)
	awaitState(t, o, s.SessionID, contracts.StateStopped)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := o.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, e := range f.events {
		if e.Output != nil && strings.Contains(e.Output.Text, "输出已截断") {
			return
		}
	}
	t.Fatal("早到输出被截断却没有提示")
}

func TestBindingRejectsInvalidPorts(t *testing.T) {
	for _, address := range []string{"127.0.0.1:banana", "127.0.0.1:65536", "127.0.0.1:-1", "0.0.0.0:4000", "127.0.0.1:0"} {
		binding := gate.GateBinding{Sentinel: sentinel, AllowedTCP: []string{address}, ModelBaseURL: "http://" + address + "/model", MCPURL: "http://" + address + "/mcp"}
		if validBinding(binding) {
			t.Fatalf("接受非法 Gate 端口：%s", address)
		}
	}
}

func TestOutputQueueDoesNotConfuseSessionSequenceNumbers(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	q := newEventQueue(func(contracts.Event) { once.Do(func() { close(entered); <-release }) })
	q.push(contracts.Event{SessionID: "first", Seq: 1, Type: "state_changed", State: &contracts.SessionSnapshot{SessionID: "first", Seq: 1, State: contracts.StateRunning}})
	<-entered
	defer close(release)
	done := make(chan struct{})
	go func() {
		for n := 0; n < 100; n++ {
			q.push(contracts.Event{SessionID: fmt.Sprintf("session-%d", n), Seq: 3, Type: "output", Output: &contracts.OutputData{Stream: "stdout", Text: strings.Repeat("x", chunkLimit)}})
		}
		q.close()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("不同会话重复 seq 导致输出截断死循环")
	}
}

// UI 卡住时，结构化交互输出也必须计入共同预算。
func TestSlowSinkBoundsMixedPtyOutput(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	q := newEventQueue(func(contracts.Event) { once.Do(func() { close(entered); <-release }) })
	q.push(contracts.Event{SessionID: "s", Seq: 1, Type: "error", Error: publicError("INTERNAL_ERROR")})
	<-entered
	defer func() { q.close(); close(release); <-q.done }()
	for seq := uint64(2); seq < 160; seq++ {
		e := contracts.Event{SessionID: "s", Seq: seq, Type: "pty", Pty: &contracts.OutputData{Stream: "stdout", Text: strings.Repeat("x", chunkLimit)}}
		if seq%2 == 0 {
			e.Type, e.Output, e.Pty = "output", e.Pty, nil
		}
		q.push(e)
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	actual := 0
	for _, e := range q.items {
		if e.Output != nil {
			actual += len(e.Output.Text)
		}
		if e.Pty != nil {
			actual += len(e.Pty.Text)
		}
	}
	if actual > outputLimit || q.bytes != actual {
		t.Fatalf("实际排队=%d，计数=%d，上限=%d", actual, q.bytes, outputLimit)
	}
}
