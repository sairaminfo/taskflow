package httpapi

import (
	"log/slog"
	"net/http"
)

// NewRouter wires routes using the Go 1.22 http.ServeMux method+pattern syntax
// and wraps everything in logging and panic-recovery middleware.
func NewRouter(h *Handler, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("GET /api/v1/tasks", h.listTasks)
	mux.HandleFunc("POST /api/v1/tasks", h.createTask)
	mux.HandleFunc("GET /api/v1/tasks/{id}", h.getTask)
	mux.HandleFunc("PATCH /api/v1/tasks/{id}", h.updateTask)
	mux.HandleFunc("DELETE /api/v1/tasks/{id}", h.deleteTask)

	return chain(mux, Recover(logger), Logging(logger))
}
