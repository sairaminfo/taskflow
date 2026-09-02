// Package worker provides an asynchronous event processor. It implements
// service.Publisher and demonstrates idiomatic Go concurrency: a buffered
// channel as a queue, a pool of worker goroutines, sync.WaitGroup for
// lifecycle, and context-based cancellation with a graceful drain on shutdown.
package worker

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/proofpoint/taskflow/internal/service"
)

// EventProcessor consumes task events off a buffered channel using a worker
// pool. In a real system each worker might send a notification, update a search
// index, or emit metrics; here it logs and counts.
type EventProcessor struct {
	events    chan service.Event
	workers   int
	logger    *slog.Logger
	wg        sync.WaitGroup
	processed atomic.Int64
	dropped   atomic.Int64
	closeOnce sync.Once
}

// New builds an EventProcessor. bufferSize bounds the queue so a burst of
// events cannot grow memory without limit; workers sets pool concurrency.
func New(logger *slog.Logger, bufferSize, workers int) *EventProcessor {
	if bufferSize < 1 {
		bufferSize = 1
	}
	if workers < 1 {
		workers = 1
	}
	return &EventProcessor{
		events:  make(chan service.Event, bufferSize),
		workers: workers,
		logger:  logger,
	}
}

// Publish implements service.Publisher. It is non-blocking: if the queue is
// full it drops the event and records it rather than stalling the request path.
// This keeps API latency bounded even under event backpressure.
func (p *EventProcessor) Publish(ev service.Event) {
	select {
	case p.events <- ev:
	default:
		p.dropped.Add(1)
		p.logger.Warn("event queue full, dropping event", slog.String("type", ev.Type))
	}
}

// Start launches the worker pool. Workers run until the events channel is
// closed (via Stop) and the context guards against a blocked shutdown.
func (p *EventProcessor) Start(ctx context.Context) {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.run(ctx, i)
	}
	p.logger.Info("event processor started", slog.Int("workers", p.workers))
}

func (p *EventProcessor) run(ctx context.Context, id int) {
	defer p.wg.Done()
	for {
		select {
		case ev, ok := <-p.events:
			if !ok {
				return // channel closed and drained
			}
			p.handle(ev)
		case <-ctx.Done():
			// Drain whatever is already buffered, then exit.
			for {
				select {
				case ev, ok := <-p.events:
					if !ok {
						return
					}
					p.handle(ev)
				default:
					return
				}
			}
		}
	}
}

func (p *EventProcessor) handle(ev service.Event) {
	p.processed.Add(1)
	p.logger.Info("processed task event",
		slog.String("type", ev.Type),
		slog.String("task_id", ev.Task.ID),
		slog.String("title", ev.Task.Title),
	)
}

// Stop closes the queue and waits for workers to finish draining. Safe to call
// once; guarded by sync.Once.
func (p *EventProcessor) Stop() {
	p.closeOnce.Do(func() { close(p.events) })
	p.wg.Wait()
	p.logger.Info("event processor stopped",
		slog.Int64("processed", p.processed.Load()),
		slog.Int64("dropped", p.dropped.Load()),
	)
}

// Stats returns counters, useful for tests and metrics endpoints.
func (p *EventProcessor) Stats() (processed, dropped int64) {
	return p.processed.Load(), p.dropped.Load()
}
