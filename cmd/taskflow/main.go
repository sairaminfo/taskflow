// Command taskflow is the entrypoint for the TaskFlow microservice. It wires
// the layers together (repository -> service -> HTTP), starts the async event
// worker, and manages a graceful shutdown on SIGINT/SIGTERM.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/proofpoint/taskflow/internal/config"
	"github.com/proofpoint/taskflow/internal/httpapi"
	"github.com/proofpoint/taskflow/internal/repository"
	"github.com/proofpoint/taskflow/internal/service"
	"github.com/proofpoint/taskflow/internal/worker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server exited with error", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg := config.Load()

	// Compose the dependency graph.
	repo := repository.NewMemoryStore()
	events := worker.New(logger, cfg.EventBuffer, cfg.EventWorkers)
	svc := service.New(repo, service.WithPublisher(events))
	handler := httpapi.NewHandler(svc)
	router := httpapi.NewRouter(handler, logger)

	// Start the async event processor. Its context is cancelled during shutdown.
	workerCtx, cancelWorker := context.WithCancel(context.Background())
	defer cancelWorker()
	events.Start(workerCtx)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Run the server in its own goroutine so main can wait for signals.
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("http server listening", slog.String("addr", cfg.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	// Wait for either a fatal server error or an OS termination signal.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	// Graceful shutdown ordering matters:
	// 1. Stop accepting new HTTP requests and let in-flight ones finish. Only
	//    after this returns is it safe to stop the worker, because handlers can
	//    no longer call events.Publish (which would panic on a closed channel).
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", slog.Any("error", err))
		_ = srv.Close()
	}

	// 2. Now drain and stop the event worker.
	cancelWorker()
	events.Stop()

	logger.Info("shutdown complete")
	return nil
}
