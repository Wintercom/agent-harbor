package orchestrator

import (
	"context"
	"testing"
	"time"
)

func TestShutdownCancelsAndWaitsForConnection(t *testing.T) {
	o := setup(t, newFixture(), Timeouts{}, nil)
	entered, canceled, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- o.WithConnection(context.Background(), func(ctx context.Context) error {
			close(entered)
			<-ctx.Done()
			close(canceled)
			<-release
			return ctx.Err()
		})
	}()
	<-entered
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	err := o.Shutdown(ctx)
	assertCode(t, err, "CLEANUP_FAILED")
	<-canceled
	close(release)
	<-done
	if err = o.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, err = o.Create(context.Background(), spec())
	assertCode(t, err, "CANCELLED")
}
