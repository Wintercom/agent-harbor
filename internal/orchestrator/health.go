package orchestrator

import "context"

// Health 每次执行 Runtime 的实际探针；不缓存成功或自动退回非隔离执行。
func (o *Service) Health(ctx context.Context) error {
	if ctx.Err() != nil {
		return publicError("CANCELLED")
	}
	o.mu.Lock()
	closing := o.closing || o.app.Err() != nil
	o.mu.Unlock()
	if closing {
		return publicError("CANCELLED")
	}
	call, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(o.app, cancel)
	defer func() { stop(); cancel() }()
	err := o.deps.Runtime.Health(call)
	if call.Err() != nil {
		return publicError("CANCELLED")
	}
	if err != nil {
		return publicError("ISOLATION_UNAVAILABLE")
	}
	return nil
}
