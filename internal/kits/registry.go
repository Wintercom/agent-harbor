package kits

import (
	"context"
	"sort"
)

// Registry 只保存宿主机已明确装配的 Kit；不会根据 PATH 自动发现未知 Agent。
type Registry struct {
	items map[string]Kit
}

func NewRegistry(items map[string]Kit) (*Registry, error) {
	if len(items) == 0 {
		return nil, kitRegistryError("INVALID_ARGUMENT")
	}
	copyItems := make(map[string]Kit, len(items))
	for id, kit := range items {
		if id == "" || kit == nil {
			return nil, kitRegistryError("INVALID_ARGUMENT")
		}
		copyItems[id] = kit
	}
	return &Registry{items: copyItems}, nil
}

// DetectAll 保持注册顺序稳定；单个 Agent 不可用时返回 Available=false，
// 只有检测实现本身的内部错误才终止整批检测。
func (r *Registry) DetectAll(ctx context.Context) ([]KitInfo, error) {
	ids := make([]string, 0, len(r.items))
	for id := range r.items {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]KitInfo, 0, len(ids))
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return nil, kitRegistryError("CANCELLED")
		}
		info, err := r.items[id].Detect(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil, kitRegistryError("CANCELLED")
			}
			return nil, err
		}
		if info.Agent.ID == "" {
			info.Agent.ID = id
		}
		if info.Agent.ID != id {
			info.Agent.Available = false
			info.Agent.Reason = "Agent 检测返回了不一致的标识。"
			info.ReadOnlyPaths = nil
		}
		if !info.Agent.Available {
			info.ReadOnlyPaths = nil
		}
		result = append(result, info)
	}
	return result, nil
}

type registryError struct{ code string }

func (e *registryError) Error() string   { return e.code }
func kitRegistryError(code string) error { return &registryError{code: code} }
