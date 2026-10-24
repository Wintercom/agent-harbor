package main

import (
	"context"
	"errors"

	"github.com/liyufei916/agent-harbor/internal/contracts"
)

// ErrOrchestratorDown 表示 A 尚未注入真实编排。
var ErrOrchestratorDown = errors.New(contracts.ErrOrchestratorUnavailable)

// UnavailableSession 是 B 自测桌面桥时使用的 SessionService 替身。
type UnavailableSession struct{}

func (UnavailableSession) Create(context.Context, contracts.SessionSpec) (contracts.SessionSnapshot, error) {
	return contracts.SessionSnapshot{}, ErrOrchestratorDown
}

func (UnavailableSession) Get(context.Context, string) (contracts.SessionSnapshot, error) {
	return contracts.SessionSnapshot{}, ErrOrchestratorDown
}

func (UnavailableSession) Stop(context.Context, string) (contracts.SessionSnapshot, error) {
	return contracts.SessionSnapshot{}, ErrOrchestratorDown
}

func (UnavailableSession) Shutdown(context.Context) error { return nil }

// RecordingSession 记录调用，供 B 的绑定单测使用。
type RecordingSession struct {
	LastCreate contracts.SessionSpec
	LastGet    string
	LastStop   string
	Snapshot   contracts.SessionSnapshot
	Err        error
}

func (r *RecordingSession) Create(_ context.Context, spec contracts.SessionSpec) (contracts.SessionSnapshot, error) {
	r.LastCreate = spec
	if r.Err != nil {
		return contracts.SessionSnapshot{}, r.Err
	}
	if r.Snapshot.SessionID == "" {
		r.Snapshot = contracts.SessionSnapshot{
			SessionID: "fake-session",
			State:     "provisioning",
			Seq:       1,
			Workspace: spec.Workspace,
			AgentID:   spec.AgentID,
		}
	}
	return r.Snapshot, nil
}

func (r *RecordingSession) Get(_ context.Context, id string) (contracts.SessionSnapshot, error) {
	r.LastGet = id
	if r.Err != nil {
		return contracts.SessionSnapshot{}, r.Err
	}
	return r.Snapshot, nil
}

func (r *RecordingSession) Stop(_ context.Context, id string) (contracts.SessionSnapshot, error) {
	r.LastStop = id
	if r.Err != nil {
		return contracts.SessionSnapshot{}, r.Err
	}
	r.Snapshot.State = "stopping"
	return r.Snapshot, nil
}

func (r *RecordingSession) Shutdown(context.Context) error { return nil }
