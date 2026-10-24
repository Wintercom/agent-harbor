// 本文件按 docs/CONTRACT.md v1 定义，不包含业务实现。
package contracts

type AppError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

type AgentInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	Available bool   `json:"available"`
	Reason    string `json:"reason"`
}

type ConnectInput struct {
	ProfileID string `json:"profileId"`
	Name      string `json:"name"`
	APIKey    string `json:"apiKey"`
}

type AccountInfo struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"` // model 或 mcp
	Name      string `json:"name"`
	ProfileID string `json:"profileId"`
	Host      string `json:"host"`
}

type SessionSpec struct {
	Workspace      string   `json:"workspace"`
	AgentID        string   `json:"agentId"`
	ModelAccountID string   `json:"modelAccountId"`
	MCPIDs         []string `json:"mcpIds"`
	Task           string   `json:"task"`
}

type SessionSnapshot struct {
	SessionID       string    `json:"sessionId"`
	State           string    `json:"state"`
	Seq             uint64    `json:"seq"`
	Workspace       string    `json:"workspace"`
	AgentID         string    `json:"agentId"`
	CleanupComplete bool      `json:"cleanupComplete"`
	Error           *AppError `json:"error"`
}

type AgentsResponse struct {
	Agents []AgentInfo `json:"agents"`
	Error  *AppError   `json:"error"`
}

type WorkspaceResponse struct {
	Path      string    `json:"path"`
	Cancelled bool      `json:"cancelled"`
	Error     *AppError `json:"error"`
}

type AccountResponse struct {
	Account *AccountInfo `json:"account"`
	Error   *AppError    `json:"error"`
}

type SessionResponse struct {
	Session *SessionSnapshot `json:"session"`
	Error   *AppError        `json:"error"`
}

type DesktopAPI interface {
	ListAgents() AgentsResponse
	SelectWorkspace() WorkspaceResponse
	ConnectModel(ConnectInput) AccountResponse
	ConnectMcp(ConnectInput) AccountResponse
	CreateSession(SessionSpec) SessionResponse
	GetSession(string) SessionResponse
	StopSession(string) SessionResponse
}
