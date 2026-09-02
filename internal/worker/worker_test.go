package worker

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/proofpoint/taskflow/internal/service"
)

func newTestProcessor(buf, workers int) *EventProcessor {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(logger, buf, workers)
}

func TestProcessor_ProcessesEvents(t *testing.T) {
	p := newTestProcessor(64, 2)
	p.Start(context.Background())

	const n = 20
	for i := 0; i < n; i++ {
		p.Publish(service.Event{Type: "created"})
	}
	// Stop drains the queue and waits for workers.
	p.Stop()

	processed, dropped := p.Stats()
	if processed != n {
		t.Fatalf("expected %d processed, got %d (dropped=%d)", n, processed, dropped)
	}
}

func TestProcessor_DropsWhenFull(t *testing.T) {
	// Buffer of 1, no workers consuming yet (don't call Start), so the queue
	// fills immediately and further publishes are dropped rather than blocking.
	p := newTestProcessor(1, 1)
	p.Publish(service.Event{Type: "created"}) // fills buffer
	p.Publish(service.Event{Type: "created"}) // dropped

	_, dropped := p.Stats()
	if dropped != 1 {
		t.Fatalf("expected 1 dropped, got %d", dropped)
	}
}
