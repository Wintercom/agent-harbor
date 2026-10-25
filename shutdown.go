package main

import (
	"context"
	"sync"
	"time"
)

// 退出检查不阻塞 GUI；失败保留窗口和凭据以便重试清理。
type exitGate struct {
	mu                sync.Mutex
	running, complete bool
	shutdown          func(context.Context) error
	ready, failed     func()
}

func newExitGate(shutdown func(context.Context) error, ready, failed func()) *exitGate {
	return &exitGate{shutdown: shutdown, ready: ready, failed: failed}
}
func (g *exitGate) request() bool {
	g.mu.Lock()
	if g.complete {
		g.mu.Unlock()
		return true
	}
	if g.running {
		g.mu.Unlock()
		return false
	}
	g.running = true
	g.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		err := g.shutdown(ctx)
		g.mu.Lock()
		g.running = false
		g.complete = err == nil
		g.mu.Unlock()
		if err != nil {
			g.failed()
		} else {
			g.ready()
		}
	}()
	return false
}
