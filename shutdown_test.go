package main

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestExitWaitsForCleanupAndAllowsRetry(t *testing.T) {
	var attempts atomic.Int32
	failed, ready := make(chan struct{}, 2), make(chan struct{}, 2)
	gate := newExitGate(func(context.Context) error {
		if attempts.Add(1) == 1 {
			return errors.New("fixture")
		}
		return nil
	}, func() { ready <- struct{}{} }, func() { failed <- struct{}{} })
	if gate.request() {
		t.Fatal("清理前退出")
	}
	select {
	case <-failed:
	case <-time.After(time.Second):
		t.Fatal("未报告清理失败")
	}
	select {
	case <-ready:
		t.Fatal("清理失败后仍退出")
	default:
	}
	if gate.request() {
		t.Fatal("重试尚未确认就退出")
	}
	select {
	case <-ready:
	case <-time.After(time.Second):
		t.Fatal("重试未完成")
	}
	if !gate.request() {
		t.Fatal("清理完成后拒绝退出")
	}
}
