// Package contracts 存放前后端共享的可序列化 DTO。
// 负责人：A 主改，B 复核绑定兼容，C 核对页面字段。
// 字段与 docs/CONTRACT.md v1 一致，不包含业务实现。
package contracts

// AppError 是前端可见的业务错误。message 使用固定可读模板，不附加原始上游细节。
type AppError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

// AgentInfo 描述本机可检测的 Agent 能力。
type AgentInfo struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Version        string `json:"version"`
	Available      bool   `json:"available"`
	Reason         string `json:"reason"`
	ModelProfileID string `json:"modelProfileId,omitempty"`
}

// ConnectInput 为模型/MCP 连接的单向输入；APIKey 不得回显或写入事件。
type ConnectInput struct {
	ProfileID string `json:"profileId"`
	Name      string `json:"name"`
	APIKey    string `json:"apiKey"`
}

// AccountInfo 是脱敏后的账号元信息。
type AccountInfo struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"` // model 或 mcp
	Name      string `json:"name"`
	ProfileID string `json:"profileId"`
	Host      string `json:"host"`
}

// SessionSpec 是创建会话的前端请求。
type SessionSpec struct {
	Workspace      string   `json:"workspace"`
	AgentID        string   `json:"agentId"`
	ModelAccountID string   `json:"modelAccountId"`
	MCPIDs         []string `json:"mcpIds"`
	Task           string   `json:"task"`
}

// SessionSnapshot 是会话对外快照。
type SessionSnapshot struct {
	SessionID       string    `json:"sessionId"`
	State           string    `json:"state"`
	Seq             uint64    `json:"seq"`
	Workspace       string    `json:"workspace"`
	AgentID         string    `json:"agentId"`
	CleanupComplete bool      `json:"cleanupComplete"`
	Error           *AppError `json:"error"`
}

// AgentsResponse 是 ListAgents 的统一返回。
type AgentsResponse struct {
	Agents []AgentInfo `json:"agents"`
	Error  *AppError   `json:"error"`
}

// WorkspaceResponse 是 SelectWorkspace 的统一返回。
type WorkspaceResponse struct {
	Path      string    `json:"path"`
	Cancelled bool      `json:"cancelled"`
	Error     *AppError `json:"error"`
}

// AccountResponse 是 ConnectModel / ConnectMcp 的统一返回。
type AccountResponse struct {
	Account *AccountInfo `json:"account"`
	Error   *AppError    `json:"error"`
}

// SessionResponse 是 Create/Get/Stop Session 的统一返回。
type SessionResponse struct {
	Session *SessionSnapshot `json:"session"`
	Error   *AppError        `json:"error"`
}

// DesktopAPI 是绑定到桌面的业务方法集合。实现在根包 DesktopAPI，不在本包执行 IO。
type DesktopAPI interface {
	ListAgents() AgentsResponse
	SelectWorkspace() WorkspaceResponse
	ConnectModel(ConnectInput) AccountResponse
	ConnectMcp(ConnectInput) AccountResponse
	CreateSession(SessionSpec) SessionResponse
	GetSession(string) SessionResponse
	StopSession(string) SessionResponse
}
