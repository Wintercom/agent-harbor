package orchestrator

import (
	"errors"

	"github.com/liyufei916/agent-harbor/internal/contracts"
)

// 即使提供方错误含敏感文本，也只向页面发送固定模板。
var messages = map[string]string{
	"INVALID_ARGUMENT":      "会话参数不合法。",
	"ACCOUNT_NOT_FOUND":     "账号尚未连接或不存在。",
	"SESSION_BUSY":          "已有会话正在准备、运行或等待清理。",
	"SESSION_LIMIT":         "已达到同时运行的会话上限。",
	"SESSION_NOT_FOUND":     "会话不存在。",
	"AGENT_UNSUPPORTED":     "当前 Agent 尚未安装或未经兼容性验证。",
	"ISOLATION_UNAVAILABLE": "操作系统隔离不可用，已拒绝启动。",
	"WORKSPACE_DENIED":      "工作区未通过授权检查。",
	"WORKSPACE_BUSY":        "该工作区已有会话正在使用。",
	"UPSTREAM_AUTH_FAILED":  "上游认证或权限检查失败。",
	"UPSTREAM_TIMEOUT":      "操作超时，请检查连接后重新操作。",
	"UPSTREAM_UNAVAILABLE":  "上游服务不可用。",
	"TARGET_DENIED":         "目标或工具未经授权。",
	"INVALID_SENTINEL":      "会话授权已失效。",
	"AGENT_EXIT_FAILED":     "Agent 异常退出。",
	"CANCELLED":             "操作已取消。",
	"CLEANUP_FAILED":        "资源清理未确认完成，请重试停止；暂不能创建新会话。",
	"INTERNAL_ERROR":        "内部操作失败。",
	"UNIMPLEMENTED":         "该能力尚未实现。",
}

func publicError(code string) *contracts.AppError {
	message, ok := messages[code]
	if !ok {
		code, message = "INTERNAL_ERROR", messages["INTERNAL_ERROR"]
	}
	return &contracts.AppError{Code: code, Message: message, Retryable: code == "UPSTREAM_TIMEOUT" || code == "UPSTREAM_UNAVAILABLE"}
}

func safeError(err error, fallback string) *contracts.AppError {
	var app *contracts.AppError
	if errors.As(err, &app) && app != nil {
		if _, ok := messages[app.Code]; ok {
			return publicError(app.Code)
		}
	}
	return publicError(fallback)
}
