// 本文件按 docs/CONTRACT.md v1 定义，不包含业务实现。
package runtime

import (
	"context"

	"github.com/liyufei916/agent-harbor/internal/contracts"
)

type RuntimeSpec struct {
	SessionID     string
	Workspace     string
	MountMode     string
	AllowedTCP    []string
	ReadOnlyPaths []string
}

type Handle struct {
	ID          string
	PrivateHome string
}

type Exit struct {
	Code      int
	Cancelled bool
	Err       error
}

type Process interface {
	Wait() Exit // 返回前完成 stdout/stderr 读取与最终输出回调
}

type Runtime interface {
	Health(context.Context) error
	Create(context.Context, RuntimeSpec) (Handle, error)
	Exec(context.Context, Handle, LaunchSpec, func(contracts.OutputData)) (Process, error)
	Destroy(context.Context, Handle) error
}

type ConfigFile struct {
	RelativePath string // 只能位于 Runtime 私有 HOME 下
	Content      []byte // 只允许 Gate URL、Sentinel 等无上游凭据内容
}

type LaunchSpec struct {
	Command     string
	Args        []string
	Env         map[string]string
	Stdin       []byte
	ConfigFiles []ConfigFile
}
