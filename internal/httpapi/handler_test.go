package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/proofpoint/taskflow/internal/domain"
	"github.com/proofpoint/taskflow/internal/repository"
	"github.com/proofpoint/taskflow/internal/service"
)

func newTestServer() http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := service.New(repository.NewMemoryStore())
	return NewRouter(NewHandler(svc), logger)
}

func TestHealth(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestCreateAndGetTask(t *testing.T) {
	srv := newTestServer()

	// Create
	body := `{"title":"ship it","description":"today"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", strings.NewReader(body))
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	var created domain.Task
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected an ID")
	}

	// Get
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+created.ID, nil)
	srv.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec2.Code)
	}
}

func TestCreateTask_InvalidTitle(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", strings.NewReader(`{"title":""}`))
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGetTask_NotFound(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/missing", nil)
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestUpdateTask(t *testing.T) {
	srv := newTestServer()

	// seed one
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/tasks", strings.NewReader(`{"title":"a"}`)))
	var created domain.Task
	_ = json.Unmarshal(rec.Body.Bytes(), &created)

	// patch status
	rec2 := httptest.NewRecorder()
	patch := httptest.NewRequest(http.MethodPatch, "/api/v1/tasks/"+created.ID, bytes.NewBufferString(`{"status":"in_progress"}`))
	srv.ServeHTTP(rec2, patch)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec2.Code, rec2.Body.String())
	}
	var updated domain.Task
	_ = json.Unmarshal(rec2.Body.Bytes(), &updated)
	if updated.Status != domain.StatusInProgress {
		t.Errorf("expected in_progress, got %q", updated.Status)
	}
}

func TestDeleteTask(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/tasks", strings.NewReader(`{"title":"a"}`)))
	var created domain.Task
	_ = json.Unmarshal(rec.Body.Bytes(), &created)

	rec2 := httptest.NewRecorder()
	srv.ServeHTTP(rec2, httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/"+created.ID, nil))
	if rec2.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec2.Code)
	}
}
