package contracts

// HarborEventName 是会话事件的唯一前端通道名。
const HarborEventName = "harbor:event"

// 事件 type 取值。
const (
	EventTypeStateChanged = "state_changed"
	EventTypeOutput       = "output"
	EventTypeAudit        = "audit"
	EventTypeError        = "error"
)

// 公开错误码（与 docs/CONTRACT.md 一致）。
const (
	ErrInvalidArgument      = "INVALID_ARGUMENT"
	ErrAccountNotFound      = "ACCOUNT_NOT_FOUND"
	ErrSessionBusy          = "SESSION_BUSY"
	ErrSessionNotFound      = "SESSION_NOT_FOUND"
	ErrAgentUnsupported     = "AGENT_UNSUPPORTED"
	ErrIsolationUnavailable = "ISOLATION_UNAVAILABLE"
	ErrWorkspaceDenied      = "WORKSPACE_DENIED"
	ErrUpstreamAuthFailed   = "UPSTREAM_AUTH_FAILED"
	ErrUpstreamTimeout      = "UPSTREAM_TIMEOUT"
	ErrUpstreamUnavailable  = "UPSTREAM_UNAVAILABLE"
	ErrTargetDenied         = "TARGET_DENIED"
	ErrInvalidSentinel      = "INVALID_SENTINEL"
	ErrAgentExitFailed      = "AGENT_EXIT_FAILED"
	ErrCancelled            = "CANCELLED"
	ErrCleanupFailed        = "CLEANUP_FAILED"
	ErrInternal             = "INTERNAL_ERROR"
	ErrOrchestratorUnavailable = "ORCHESTRATOR_UNAVAILABLE"
)

// NewAppError 构造公开错误；message 应为固定模板。
func NewAppError(code, message string, retryable bool) *AppError {
	return &AppError{Code: code, Message: message, Retryable: retryable}
}
