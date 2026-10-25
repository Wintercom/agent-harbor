package runtime

import (
	"context"
	"os"
	"time"
)

// processInput 串行写入完整消息，取消时中断阻塞的管道写入。
type processInput struct {
	pipe   *os.File
	encode func([]byte) ([]byte, error)
	gate   chan struct{}
}

func newProcessInput(pipe *os.File, encode func([]byte) ([]byte, error)) *processInput {
	return &processInput{pipe: pipe, encode: encode, gate: make(chan struct{}, 1)}
}

func (in *processInput) write(ctx context.Context, data []byte) error {
	if ctx.Err() != nil {
		return runtimeError("CANCELLED")
	}
	select {
	case in.gate <- struct{}{}:
		defer func() { <-in.gate }()
	case <-ctx.Done():
		return runtimeError("CANCELLED")
	}
	if ctx.Err() != nil {
		return runtimeError("CANCELLED")
	}
	if in.encode != nil {
		encoded, err := in.encode(data)
		if err != nil {
			return err
		}
		data = encoded
	}
	if err := in.pipe.SetWriteDeadline(time.Time{}); err != nil {
		return runtimeError("INTERNAL_ERROR")
	}
	cancelled := make(chan struct{})
	stop := context.AfterFunc(ctx, func() {
		_ = in.pipe.SetWriteDeadline(time.Now())
		close(cancelled)
	})
	defer func() {
		if !stop() {
			<-cancelled
		}
		_ = in.pipe.SetWriteDeadline(time.Time{})
	}()
	for len(data) > 0 {
		n, err := in.pipe.Write(data)
		if ctx.Err() != nil {
			return runtimeError("CANCELLED")
		}
		if err != nil || n == 0 {
			return runtimeError("AGENT_EXIT_FAILED")
		}
		data = data[n:]
	}
	return nil
}
