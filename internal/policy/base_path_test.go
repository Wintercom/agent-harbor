package policy

import "testing"

func TestJoinBasePathAvoidsDuplicateProviderPrefix(t *testing.T) {
	for _, test := range []struct {
		base, path, want string
	}{
		{"", "/v1/responses", "/v1/responses"},
		{"/v1", "/v1/responses", "/v1/responses"},
		{"/anthropic", "/v1/messages", "/anthropic/v1/messages"},
		{"/api/", "/responses", "/api/responses"},
	} {
		if got := JoinBasePath(test.base, test.path); got != test.want {
			t.Errorf("JoinBasePath(%q, %q) = %q, want %q", test.base, test.path, got, test.want)
		}
	}
}
