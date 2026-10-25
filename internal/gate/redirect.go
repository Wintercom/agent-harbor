package gate

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/liyufei916/agent-harbor/internal/contracts"
)

// followCrossHostRedirect 在用户扩权且 DNS/IP 通过后跟随一跳；不注入原账号凭据。
func (h *Harbor) followCrossHostRedirect(ctx context.Context, sid string, rt *sessionRuntime, resp *http.Response, originAuth string) (*http.Response, string, error) {
	locURL, err := redirectAbsoluteURL(resp)
	defer resp.Body.Close()
	if err != nil || locURL == nil || locURL.Host == "" {
		return nil, "", fmt.Errorf("gate: bad redirect")
	}
	locAuth := strings.ToLower(locURL.Host)
	locHost := strings.ToLower(locURL.Hostname())
	if strings.EqualFold(locAuth, originAuth) {
		return nil, locHost, fmt.Errorf("gate: same-origin redirect not blocked")
	}

	allowed := h.HasExtraHost(sid, locHost)
	if !allowed && rt.spec.RequestPermission != nil {
		decision, perr := rt.spec.RequestPermission(ctx, contracts.PermissionNeeded{
			Target: locHost,
			Action: "请求跟随跨主机重定向",
		})
		if perr == nil && (decision == "once" || decision == "project") {
			_ = h.AllowExtraHost(sid, locHost, decision)
			allowed = true
			if decision == "once" {
				_ = h.HasExtraHost(sid, locHost)
			}
		}
	}
	if !allowed {
		return nil, locHost, fmt.Errorf("gate: redirect not permitted")
	}
	if err := ValidatePublicHostname(locHost); err != nil {
		return nil, locHost, err
	}
	if locURL.Scheme != "https" && !(locURL.Scheme == "http" && egressAllowLoopback) {
		return nil, locHost, fmt.Errorf("gate: redirect scheme denied")
	}

	method := http.MethodGet
	if resp.Request != nil {
		method = redirectFollowMethod(resp.StatusCode, resp.Request.Method)
	}
	followReq, err := http.NewRequestWithContext(ctx, method, locURL.String(), nil)
	if err != nil {
		return nil, locHost, err
	}
	followReq.Header.Set("Accept-Encoding", "identity")
	if resp.Request != nil {
		if ua := resp.Request.Header.Get("User-Agent"); ua != "" {
			followReq.Header.Set("User-Agent", ua)
		}
	}

	next, err := h.client.Do(followReq)
	if err != nil {
		return nil, locHost, err
	}
	if blockedRedirect(next, locAuth) {
		next.Body.Close()
		return nil, locHost, fmt.Errorf("gate: multi-hop redirect denied")
	}
	return next, locHost, nil
}

func blockedRedirect(resp *http.Response, allowedAuthority string) bool {
	if resp.StatusCode < 300 || resp.StatusCode >= 400 {
		return false
	}
	u, err := redirectAbsoluteURL(resp)
	if err != nil || u == nil || u.Host == "" {
		return false
	}
	return !strings.EqualFold(u.Host, allowedAuthority)
}

func redirectAbsoluteURL(resp *http.Response) (*url.URL, error) {
	loc := resp.Header.Get("Location")
	if loc == "" || resp.Request == nil || resp.Request.URL == nil {
		return nil, fmt.Errorf("missing location")
	}
	return resp.Request.URL.Parse(loc)
}

func redirectFollowMethod(status int, original string) string {
	switch status {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther:
		return http.MethodGet
	default:
		if original == "" {
			return http.MethodGet
		}
		return original
	}
}

func hostOfOrigin(origin string) string {
	u, err := url.Parse(origin)
	if err != nil {
		return origin
	}
	return u.Hostname()
}

func authorityOfOrigin(origin string) string {
	u, err := url.Parse(origin)
	if err != nil {
		return strings.ToLower(strings.TrimSpace(origin))
	}
	return strings.ToLower(u.Host)
}
