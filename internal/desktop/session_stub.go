package desktop

import (
	"context"
	"errors"

	"github.com/liyufei916/agent-harbor/internal/contracts"
)

var errOrchestratorDown = errors.New(contracts.ErrOrchestratorUnavailable)

// unavailableSession 仅在 AttachOrchestrator 之前使用。
type unavailableSession struct{}

func (unavailableSession) Create(context.Context, contracts.SessionSpec) (contracts.SessionSnapshot, error) {
	return contracts.SessionSnapshot{}, errOrchestratorDown
}

func (unavailableSession) Get(context.Context, string) (contracts.SessionSnapshot, error) {
	return contracts.SessionSnapshot{}, errOrchestratorDown
}

func (unavailableSession) Stop(context.Context, string) (contracts.SessionSnapshot, error) {
	return contracts.SessionSnapshot{}, errOrchestratorDown
}

func (unavailableSession) Shutdown(context.Context) error { return nil }
