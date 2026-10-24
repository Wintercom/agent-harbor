// Package gate 实现 Harbor Gate：模型反代、MCP Gateway 与连接测试。
// 负责人：研发 B。不导入 orchestrator / runtime / kits。
package gate

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Wintercom/agent-harbor/internal/contracts"
	"github.com/Wintercom/agent-harbor/internal/policy"
	"github.com/Wintercom/agent-harbor/internal/secrets"
)

var (
	ErrSessionExists   = errors.New("gate: session already started")
	ErrSessionNotFound = errors.New("gate: session not found")
	ErrRevoked         = errors.New("gate: sentinel revoked")
)

// GateSpec 是启动闸口的编排输入。
type GateSpec struct {
	SessionID      string
	Policy         policy.PolicySnapshot
	ModelAccountID string
	MCPAccountID   string
}

// GateBinding 返回给编排层的注入信息（不得进入前端 DTO）。
type GateBinding struct {
	ModelBaseURL string
	MCPURL       string
	Sentinel     string
	AllowedTCP   []string
}

type sessionRuntime struct {
	spec      GateSpec
	sentinel  string
	model     *modelProxy
	mcp       *mcpGateway
	modelAddr string
	mcpAddr   string
	cancel    context.CancelFunc
	audit     func(contracts.AuditData)
	fatal     func(error)
}

// Harbor 是控制面出站闸口。
type Harbor struct {
	mu       sync.Mutex
	secrets  *secrets.Store
	policy   *policy.Builder
	client   *http.Client
	sessions map[string]*sessionRuntime
}

// New 创建 Gate；policy 用于查找固定 profile（连接测试）。
func New(store *secrets.Store, pol *policy.Builder) *Harbor {
	return &Harbor{
		secrets: store,
		policy:  pol,
		client: &http.Client{
			Timeout: 120 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		sessions: make(map[string]*sessionRuntime),
	}
}

// Start 创建随机 Sentinel，启动精确绑定 127.0.0.1 的模型与 MCP 端口。
// 仅在监听器 ready 后返回；部分失败会关闭已创建资源。
func (h *Harbor) Start(ctx context.Context, spec GateSpec, audit func(contracts.AuditData), fatal func(error)) (GateBinding, error) {
	if spec.SessionID == "" || spec.ModelAccountID == "" || spec.MCPAccountID == "" {
		return GateBinding{}, fmt.Errorf("gate: incomplete spec")
	}
	if _, err := h.secrets.Lookup(ctx, spec.ModelAccountID); err != nil {
		return GateBinding{}, err
	}
	if _, err := h.secrets.Lookup(ctx, spec.MCPAccountID); err != nil {
		return GateBinding{}, err
	}
	sentinel, err := newSentinel()
	if err != nil {
		return GateBinding{}, err
	}

	h.mu.Lock()
	if _, exists := h.sessions[spec.SessionID]; exists {
		h.mu.Unlock()
		return GateBinding{}, ErrSessionExists
	}
	h.mu.Unlock()

	modelLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return GateBinding{}, err
	}
	mcpLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		_ = modelLn.Close()
		return GateBinding{}, err
	}

	runCtx, cancel := context.WithCancel(context.Background())
	mp := newModelProxy(h, spec.SessionID, modelLn)
	gw := newMCPGateway(h, spec.SessionID, mcpLn)

	rt := &sessionRuntime{
		spec:      spec,
		sentinel:  sentinel,
		model:     mp,
		mcp:       gw,
		modelAddr: modelLn.Addr().String(),
		mcpAddr:   mcpLn.Addr().String(),
		cancel:    cancel,
		audit:     audit,
		fatal:     fatal,
	}

	h.mu.Lock()
	if _, exists := h.sessions[spec.SessionID]; exists {
		h.mu.Unlock()
		cancel()
		_ = modelLn.Close()
		_ = mcpLn.Close()
		return GateBinding{}, ErrSessionExists
	}
	h.sessions[spec.SessionID] = rt
	h.mu.Unlock()

	go mp.serve(runCtx)
	go gw.serve(runCtx)

	// 确认端口可接受连接
	if err := waitReady(ctx, rt.modelAddr, rt.mcpAddr); err != nil {
		_ = h.Revoke(context.Background(), spec.SessionID)
		return GateBinding{}, err
	}

	return GateBinding{
		ModelBaseURL: "http://" + rt.modelAddr,
		MCPURL:       "http://" + rt.mcpAddr + "/mcp",
		Sentinel:     sentinel,
		AllowedTCP:   []string{rt.modelAddr, rt.mcpAddr},
	}, nil
}

func waitReady(ctx context.Context, addrs ...string) error {
	deadline := time.Now().Add(2 * time.Second)
	if dl, ok := ctx.Deadline(); ok && dl.Before(deadline) {
		deadline = dl
	}
	for _, addr := range addrs {
		for {
			d := net.Dialer{Timeout: 100 * time.Millisecond}
			conn, err := d.DialContext(ctx, "tcp", addr)
			if err == nil {
				_ = conn.Close()
				break
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("gate: listener not ready: %s", addr)
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	return nil
}

// Revoke 先本地失效令牌，再关闭监听器；幂等。
func (h *Harbor) Revoke(ctx context.Context, sessionID string) error {
	h.mu.Lock()
	rt, ok := h.sessions[sessionID]
	if ok {
		delete(h.sessions, sessionID)
	}
	h.mu.Unlock()
	if !ok {
		return nil
	}
	rt.sentinel = "" // 本地失效
	rt.cancel()
	if rt.model != nil {
		rt.model.close()
	}
	if rt.mcp != nil {
		rt.mcp.close()
	}
	_ = ctx
	return nil
}

func (h *Harbor) getSession(sessionID string) (*sessionRuntime, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	rt, ok := h.sessions[sessionID]
	return rt, ok
}

func (h *Harbor) emitAudit(rt *sessionRuntime, data contracts.AuditData) {
	if rt != nil && rt.audit != nil {
		rt.audit(data)
	}
}

func (h *Harbor) emitFatal(rt *sessionRuntime, err error) {
	if rt != nil && rt.fatal != nil && err != nil {
		rt.fatal(err)
	}
}

// Health 返回闸口进程自身存活状态。
func (h *Harbor) Health() string {
	return "ok"
}
