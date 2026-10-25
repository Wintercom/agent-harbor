package orchestrator

import (
	"context"
	"testing"
)

func TestHealthFailsClosedAndRedactsErrors(t *testing.T) {
	f := newFixture()
	o := setup(t, f, Timeouts{}, nil)
	if err := o.Health(context.Background()); err != nil {
		t.Fatal(err)
	}
	f.failAt = "health"
	assertCode(t, o.Health(context.Background()), "ISOLATION_UNAVAILABLE")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assertCode(t, o.Health(ctx), "CANCELLED")
	if err := o.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	assertCode(t, o.Health(context.Background()), "CANCELLED")
}
