package gate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/liyufei916/agent-harbor/internal/contracts"
	"github.com/liyufei916/agent-harbor/internal/policy"
	"github.com/liyufei916/agent-harbor/internal/secrets"
)

// 连接流程错误，由 app.go 映射为公开 AppError。
var (
	ErrInvalidConnectInput = errors.New(contracts.ErrInvalidArgument)
	ErrUpstreamAuth        = errors.New(contracts.ErrUpstreamAuthFailed)
	ErrUpstreamTimeout     = errors.New(contracts.ErrUpstreamTimeout)
	ErrUpstreamDown        = errors.New(contracts.ErrUpstreamUnavailable)
	ErrCancelled           = errors.New(contracts.ErrCancelled)
)

// Connections 实现模型/MCP 连接测试：成功后才写入 Secrets。
type Connections struct {
	secrets *secrets.Store
	policy  *policy.Builder
	client  *http.Client
}

// NewConnections 创建连接服务。
func NewConnections(store *secrets.Store, pol *policy.Builder) *Connections {
	return &Connections{
		secrets: store,
		policy:  pol,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// ConnectModel 在 10 秒内测试固定模型上游，成功才保存账号。
func (c *Connections) ConnectModel(ctx context.Context, in contracts.ConnectInput) (contracts.AccountInfo, error) {
	return c.connect(ctx, in, "model")
}

// ConnectMcp 在 10 秒内测试固定 MCP 上游，成功才保存账号。
func (c *Connections) ConnectMcp(ctx context.Context, in contracts.ConnectInput) (contracts.AccountInfo, error) {
	return c.connect(ctx, in, "mcp")
}

func (c *Connections) connect(ctx context.Context, in contracts.ConnectInput, kind string) (contracts.AccountInfo, error) {
	name := strings.TrimSpace(in.Name)
	profileID := strings.TrimSpace(in.ProfileID)
	key := in.APIKey
	if name == "" || profileID == "" || len(key) == 0 {
		return contracts.AccountInfo{}, ErrInvalidConnectInput
	}
	if utf8.RuneCountInString(name) > 100 || len(key) > 16384 {
		return contracts.AccountInfo{}, ErrInvalidConnectInput
	}
	prof, ok := c.policy.Profile(profileID)
	if !ok || prof.Kind != kind {
		return contracts.AccountInfo{}, ErrInvalidConnectInput
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := c.probe(ctx, prof, key); err != nil {
		return contracts.AccountInfo{}, err
	}

	meta := contracts.AccountInfo{
		Kind:      kind,
		Name:      name,
		ProfileID: profileID,
		Host:      prof.Host,
	}
	id, err := c.secrets.Put(ctx, meta, []byte(key))
	if err != nil {
		return contracts.AccountInfo{}, err
	}
	meta.ID = id
	return meta, nil
}

func (c *Connections) probe(ctx context.Context, prof policy.FixedProfile, apiKey string) error {
	path := prof.TestPath
	if path == "" {
		path = "/"
	}
	u := strings.TrimRight(prof.Origin, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return ErrUpstreamDown
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := c.client.Do(req)
	if err != nil {
		return mapProbeErr(ctx, err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return ErrUpstreamAuth
	case resp.StatusCode == http.StatusNotFound:
		// 部分上游对探测路径返回 404，仍视为认证可达。
		return nil
	case resp.StatusCode >= 500:
		return ErrUpstreamDown
	case resp.StatusCode >= 400:
		return ErrUpstreamDown
	default:
		return nil
	}
}

func mapProbeErr(ctx context.Context, err error) error {
	if errors.Is(err, context.Canceled) {
		return ErrCancelled
	}
	if errors.Is(err, context.DeadlineExceeded) || ctx.Err() == context.DeadlineExceeded {
		return ErrUpstreamTimeout
	}
	msg := err.Error()
	if strings.Contains(strings.ToLower(msg), "timeout") {
		return ErrUpstreamTimeout
	}
	return fmt.Errorf("%w: %v", ErrUpstreamDown, err)
}
