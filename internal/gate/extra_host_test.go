package gate

import (
	"context"
	"testing"

	"github.com/liyufei916/agent-harbor/internal/policy"
	"github.com/liyufei916/agent-harbor/internal/secrets"
)

func TestHasExtraHostOnceConsumes(t *testing.T) {
	h := New(secrets.NewStore(), policy.NewBuilder(nil))
	h.sessions["s-extra"] = &sessionRuntime{
		ctx:        context.Background(),
		extraHosts: make(map[string]string),
	}

	if err := h.AllowExtraHost("missing", "cdn.example.test", "once"); err == nil {
		t.Fatal("expected error for missing session")
	}
	if err := h.AllowExtraHost("s-extra", "CDN.Example.TEST", "once"); err != nil {
		t.Fatal(err)
	}
	if !h.HasExtraHost("s-extra", "cdn.example.test") {
		t.Fatal("expected once allow")
	}
	if h.HasExtraHost("s-extra", "cdn.example.test") {
		t.Fatal("once should be consumed")
	}

	if err := h.AllowExtraHost("s-extra", "cdn.example.test", "project"); err != nil {
		t.Fatal(err)
	}
	if !h.HasExtraHost("s-extra", "cdn.example.test") {
		t.Fatal("project allow")
	}
	if !h.HasExtraHost("s-extra", "cdn.example.test") {
		t.Fatal("project should remain")
	}
}
