package orchestrator

import (
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/liyufei916/agent-harbor/internal/contracts"
)

const outputLimit = 1 << 20
const chunkLimit = 16 << 10
const redacted = "[已过滤]"

// 每次 New 只创建一个分发器；回调在业务锁外调用，慢 UI 不阻塞进程读取。
// 输出排队最多 1 MiB，状态和错误不丢弃；回调须最终返回。
type eventQueue struct {
	mu     sync.Mutex
	items  []contracts.Event
	bytes  int
	marker string
	wake   chan struct{}
	closed bool
	done   chan struct{}
	sink   func(contracts.Event)
}

func newEventQueue(sink func(contracts.Event)) *eventQueue {
	q := &eventQueue{wake: make(chan struct{}, 1), done: make(chan struct{}), sink: sink}
	go q.run()
	return q
}

func (q *eventQueue) push(e contracts.Event) {
	if e.Validate() != nil {
		return
	}
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return
	}
	q.items = append(q.items, e)
	if e.Output != nil {
		q.bytes += len(e.Output.Text)
		for q.bytes > outputLimit {
			for i, old := range q.items {
				key := old.SessionID + "#" + fmt.Sprint(old.Seq)
				if old.Output != nil && key != q.marker {
					q.bytes -= len(old.Output.Text)
					// 提示沿用被截断事件的水位，保持严格递增。
					if q.marker == "" {
						old.Output = &contracts.OutputData{Stream: "stderr", Text: "[输出已截断]\n"}
						q.items[i] = old
						q.bytes += len(old.Output.Text)
						q.marker = key
					} else {
						copy(q.items[i:], q.items[i+1:])
						q.items[len(q.items)-1] = contracts.Event{}
						q.items = q.items[:len(q.items)-1]
					}
					break
				}
			}
		}
	}
	q.mu.Unlock()
	select {
	case q.wake <- struct{}{}:
	default:
	}
}

func (q *eventQueue) close() {
	q.mu.Lock()
	q.closed = true
	q.mu.Unlock()
	select {
	case q.wake <- struct{}{}:
	default:
	}
}

func (q *eventQueue) run() {
	defer close(q.done)
	for range q.wake {
		for {
			q.mu.Lock()
			if len(q.items) == 0 {
				closed := q.closed
				q.mu.Unlock()
				if closed {
					return
				}
				break
			}
			e := q.items[0]
			q.items[0] = contracts.Event{}
			q.items = q.items[1:]
			if e.Output != nil {
				q.bytes -= len(e.Output.Text)
				if e.SessionID+"#"+fmt.Sprint(e.Seq) == q.marker {
					q.marker = ""
				}
			}
			q.mu.Unlock()
			if q.sink != nil {
				q.sink(e)
			}
		}
	}
}

// 保留可能属于 Sentinel 前缀的尾部，覆盖跨任意输出块的拆分。
type streamFilter struct{ token, pending string }

func (f *streamFilter) write(text string) string {
	text = strings.ReplaceAll(f.pending+text, f.token, redacted)
	f.pending = ""
	for n := min(len(text), len(f.token)-1); n > 0; n-- {
		if strings.HasSuffix(text, f.token[:n]) {
			f.pending = text[len(text)-n:]
			return text[:len(text)-n]
		}
	}
	return text
}

func chunks(text string, emit func(string)) {
	text = strings.ToValidUTF8(text, "�")
	for len(text) > 0 {
		n := min(len(text), chunkLimit)
		for n < len(text) && !utf8.RuneStart(text[n]) {
			n--
		}
		emit(text[:n])
		text = text[n:]
	}
}
