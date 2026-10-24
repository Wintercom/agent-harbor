package kits

import (
	"context"
	"errors"
	"testing"

	"github.com/liyufei916/agent-harbor/internal/contracts"
	"github.com/liyufei916/agent-harbor/internal/runtime"
)

type registryKit struct {
	info  KitInfo
	err   error
	calls *int
}

func (k registryKit) Detect(ctx context.Context) (KitInfo, error) {
	if k.calls != nil {
		(*k.calls)++
	}
	if err := ctx.Err(); err != nil {
		return KitInfo{}, err
	}
	return k.info, k.err
}
func (k registryKit) Build(context.Context, KitInput) (runtime.LaunchSpec, error) {
	return runtime.LaunchSpec{}, nil
}

func TestRegistryDetectAllIsSortedAndUnavailableHasNoPaths(t *testing.T) {
	calls := 0
	r, err := NewRegistry(map[string]Kit{
		"zeta":  registryKit{info: KitInfo{Agent: contracts.AgentInfo{ID: "zeta", Available: true}, ReadOnlyPaths: []string{"/zeta"}}, calls: &calls},
		"alpha": registryKit{info: KitInfo{Agent: contracts.AgentInfo{Available: false}, ReadOnlyPaths: []string{"/should-clear"}}, calls: &calls},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.DetectAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || len(got) != 2 || got[0].Agent.ID != "alpha" || got[1].Agent.ID != "zeta" {
		t.Fatalf("检测结果未排序：%+v", got)
	}
	if got[0].ReadOnlyPaths != nil {
		t.Fatal("不可用 Agent 泄漏了只读路径")
	}
}

func TestRegistryRejectsInconsistentKitID(t *testing.T) {
	r, _ := NewRegistry(map[string]Kit{"claude": registryKit{info: KitInfo{Agent: contracts.AgentInfo{ID: "codex", Available: true}, ReadOnlyPaths: []string{"/wrong"}}}})
	got, err := r.DetectAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Agent.Available || got[0].Agent.ID != "codex" || got[0].ReadOnlyPaths != nil {
		t.Fatalf("不一致标识未关闭：%+v", got)
	}
}

func TestRegistryStopsOnContextAndPreservesInternalErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r, _ := NewRegistry(map[string]Kit{"claude": registryKit{}})
	if _, err := r.DetectAll(ctx); err == nil || err.Error() != "CANCELLED" {
		t.Fatalf("取消错误=%v", err)
	}
	want := errors.New("检测内部错误")
	r, _ = NewRegistry(map[string]Kit{"claude": registryKit{err: want}})
	if _, err := r.DetectAll(context.Background()); !errors.Is(err, want) {
		t.Fatalf("内部错误未保留：%v", err)
	}
	if _, err := NewRegistry(map[string]Kit{"": registryKit{}}); err == nil {
		t.Fatal("接受空 Agent ID")
	}
	if _, err := NewRegistry(map[string]Kit{"claude": nil}); err == nil {
		t.Fatal("接受 nil Kit")
	}
}
