package orchestrator

import (
	"context"

	"github.com/liyufei916/agent-harbor/internal/contracts"
)

// ModelSwitcher 只更新 Gate 内存账号绑定，不执行上游请求、不回调编排。
type ModelSwitcher interface {
	SwitchModel(context.Context, string, string) error
}

func (o *Service) SwitchModel(ctx context.Context, input contracts.SwitchModelInput) (contracts.SessionSnapshot, error) {
	if ctx.Err() != nil {
		return contracts.SessionSnapshot{}, publicError("CANCELLED")
	}
	if input.SessionID == "" || input.ModelAccountID == "" {
		return contracts.SessionSnapshot{}, publicError("INVALID_ARGUMENT")
	}
	o.mu.Lock()
	s, ok := o.sessions[input.SessionID]
	if !ok {
		o.mu.Unlock()
		return contracts.SessionSnapshot{}, publicError("SESSION_NOT_FOUND")
	}
	if s.snapshot.State != contracts.StateRunning || s.ctx.Err() != nil {
		o.mu.Unlock()
		return contracts.SessionSnapshot{}, publicError("INVALID_ARGUMENT")
	}
	oldID := s.spec.ModelAccountID
	o.mu.Unlock()
	current, err := o.deps.Catalog.Lookup(ctx, oldID)
	if err != nil {
		return contracts.SessionSnapshot{}, safeError(err, "ACCOUNT_NOT_FOUND")
	}
	next, err := o.deps.Catalog.Lookup(ctx, input.ModelAccountID)
	if err != nil {
		return contracts.SessionSnapshot{}, safeError(err, "ACCOUNT_NOT_FOUND")
	}
	if next.ID != input.ModelAccountID || next.Kind != "model" || current.Kind != "model" || next.ProfileID != current.ProfileID || next.Host != current.Host {
		return contracts.SessionSnapshot{}, publicError("INVALID_ARGUMENT")
	}
	switcher, ok := o.deps.Gate.(ModelSwitcher)
	if !ok {
		return contracts.SessionSnapshot{}, publicError("UNIMPLEMENTED")
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if ctx.Err() != nil {
		return contracts.SessionSnapshot{}, publicError("CANCELLED")
	}
	if o.sessions[input.SessionID] != s || s.snapshot.State != contracts.StateRunning || s.ctx.Err() != nil || s.spec.ModelAccountID != oldID {
		return contracts.SessionSnapshot{}, publicError("INVALID_ARGUMENT")
	}
	if err := switcher.SwitchModel(ctx, input.SessionID, input.ModelAccountID); err != nil {
		return contracts.SessionSnapshot{}, safeError(err, "INTERNAL_ERROR")
	}
	s.spec.ModelAccountID = input.ModelAccountID
	return snapshot(s), nil
}
