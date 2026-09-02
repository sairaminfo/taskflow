package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/proofpoint/taskflow/internal/domain"
	"github.com/proofpoint/taskflow/internal/repository"
)

func newTestService() *TaskService {
	fixed := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	return New(repository.NewMemoryStore(), WithClock(func() time.Time { return fixed }))
}

func TestCreate_Success(t *testing.T) {
	s := newTestService()
	task, err := s.Create(context.Background(), CreateInput{Title: "  write tests  ", Description: " important "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.ID == "" {
		t.Error("expected generated ID, got empty")
	}
	if task.Title != "write tests" {
		t.Errorf("title not trimmed: %q", task.Title)
	}
	if task.Description != "important" {
		t.Errorf("description not trimmed: %q", task.Description)
	}
	if task.Status != domain.StatusTodo {
		t.Errorf("expected default status todo, got %q", task.Status)
	}
}

func TestCreate_EmptyTitle(t *testing.T) {
	s := newTestService()
	_, err := s.Create(context.Background(), CreateInput{Title: "   "})
	if !errors.Is(err, domain.ErrInvalidTitle) {
		t.Fatalf("expected ErrInvalidTitle, got %v", err)
	}
}

func TestGet_NotFound(t *testing.T) {
	s := newTestService()
	_, err := s.Get(context.Background(), "does-not-exist")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdate_PartialAndStatusValidation(t *testing.T) {
	s := newTestService()
	created, err := s.Create(context.Background(), CreateInput{Title: "task"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Partial update: only status.
	inProgress := domain.StatusInProgress
	updated, err := s.Update(context.Background(), created.ID, UpdateInput{Status: &inProgress})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if updated.Status != domain.StatusInProgress {
		t.Errorf("status not updated: %q", updated.Status)
	}
	if updated.Title != "task" {
		t.Errorf("title should be unchanged, got %q", updated.Title)
	}

	// Invalid status rejected.
	bad := domain.Status("nonsense")
	if _, err := s.Update(context.Background(), created.ID, UpdateInput{Status: &bad}); !errors.Is(err, domain.ErrInvalidStatus) {
		t.Fatalf("expected ErrInvalidStatus, got %v", err)
	}
}

func TestDelete(t *testing.T) {
	s := newTestService()
	created, _ := s.Create(context.Background(), CreateInput{Title: "task"})
	if err := s.Delete(context.Background(), created.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, err := s.Get(context.Background(), created.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

// spyPublisher records events for assertions.
type spyPublisher struct{ events []Event }

func (p *spyPublisher) Publish(e Event) { p.events = append(p.events, e) }

func TestCreate_PublishesEvent(t *testing.T) {
	spy := &spyPublisher{}
	s := New(repository.NewMemoryStore(), WithPublisher(spy))
	if _, err := s.Create(context.Background(), CreateInput{Title: "task"}); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if len(spy.events) != 1 || spy.events[0].Type != "created" {
		t.Fatalf("expected one 'created' event, got %+v", spy.events)
	}
}
