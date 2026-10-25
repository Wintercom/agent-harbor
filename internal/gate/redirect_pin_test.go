package gate

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/liyufei916/agent-harbor/internal/policy"
	"github.com/liyufei916/agent-harbor/internal/secrets"
)

func TestRedirectPinsValidatedAddressWhenDialing(t *testing.T) {
	oldLookup := lookupIP
	lookupIP = func(string) ([]net.IP, error) { return []net.IP{net.ParseIP("203.0.113.10")}, nil }
	defer func() { lookupIP = oldLookup }()
	h := New(secrets.NewStore(), policy.NewBuilder(nil))
	rt := &sessionRuntime{extraHosts: map[string]string{"public.example.test": "project"}}
	h.sessions["session"] = rt
	dialed := ""
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = func(_ context.Context, _, address string) (net.Conn, error) {
		dialed = address
		return nil, errors.New("fixture dial stopped")
	}
	h.client.Transport = transport
	u, _ := url.Parse("https://original.example.test/path")
	response := &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{"https://public.example.test/data"}}, Request: &http.Request{Method: "GET", URL: u}, Body: io.NopCloser(strings.NewReader(""))}
	h.followCrossHostRedirect(context.Background(), "session", rt, response, u.Host)
	if dialed != "203.0.113.10:443" {
		t.Fatalf("实际拨号仍使用可重绑定的主机名: %q", dialed)
	}
}
