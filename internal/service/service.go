// Package service holds business logic: validation, ID and timestamp
// generation, and orchestration between the transport and repository layers.
// It depends only on domain types and the repository interface.
package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/proofpoint/taskflow/internal/domain"
	"github.com/proofpoint/taskflow/internal/repository"
)

// Clock abstracts time so tests can inject deterministic timestamps.
type Clock func() time.Time

// Publisher receives domain events asynchronously. It is intentionally
// non-blocking from the service's perspective; a nil Publisher is a no-op.
type Publisher interface {
	Publish(event Event)
}

// Event describes something that happened to a task.
type Event struct {
	Type string      // "created" | "updated" | "deleted"
	Task domain.Task // task state at the time of the event
	At   time.Time
}

// CreateInput and UpdateInput are the write DTOs the service accepts. Keeping
// them separate from domain.Task means callers cannot set server-owned fields
// like ID or timestamps.
type CreateInput struct {
	Title       string
	Description string
}

type UpdateInput struct {
	Title       *string
	Description *string
	Status      *domain.Status
}

// TaskService implements the core use cases.
type TaskService struct {
	repo repository.TaskRepository
	now  Clock
	pub  Publisher
}

// Option configures a TaskService.
type Option func(*TaskService)

// WithClock overrides the time source (useful in tests).
func WithClock(c Clock) Option { return func(s *TaskService) { s.now = c } }

// WithPublisher attaches an event publisher.
func WithPublisher(p Publisher) Option { return func(s *TaskService) { s.pub = p } }

// New builds a TaskService with sensible defaults.
func New(repo repository.TaskRepository, opts ...Option) *TaskService {
	s := &TaskService{repo: repo, now: time.Now}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *TaskService) publish(evType string, t domain.Task) {
	if s.pub == nil {
		return
	}
	s.pub.Publish(Event{Type: evType, Task: t, At: s.now()})
}

// Create validates input, assigns identity/timestamps, persists, and emits an event.
func (s *TaskService) Create(ctx context.Context, in CreateInput) (domain.Task, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return domain.Task{}, domain.ErrInvalidTitle
	}
	now := s.now().UTC()
	t := domain.Task{
		ID:          newID(),
		Title:       title,
		Description: strings.TrimSpace(in.Description),
		Status:      domain.StatusTodo,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(ctx, t); err != nil {
		return domain.Task{}, err
	}
	s.publish("created", t)
	return t, nil
}

// Get returns a single task or domain.ErrNotFound.
func (s *TaskService) Get(ctx context.Context, id string) (domain.Task, error) {
	return s.repo.Get(ctx, id)
}

// List returns all tasks.
func (s *TaskService) List(ctx context.Context) ([]domain.Task, error) {
	return s.repo.List(ctx)
}

// Update applies a partial update. Only non-nil fields are changed.
func (s *TaskService) Update(ctx context.Context, id string, in UpdateInput) (domain.Task, error) {
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		return domain.Task{}, err
	}
	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		if title == "" {
			return domain.Task{}, domain.ErrInvalidTitle
		}
		t.Title = title
	}
	if in.Description != nil {
		t.Description = strings.TrimSpace(*in.Description)
	}
	if in.Status != nil {
		if !in.Status.Valid() {
			return domain.Task{}, domain.ErrInvalidStatus
		}
		t.Status = *in.Status
	}
	t.UpdatedAt = s.now().UTC()
	if err := s.repo.Update(ctx, t); err != nil {
		return domain.Task{}, err
	}
	s.publish("updated", t)
	return t, nil
}

// Delete removes a task and emits an event.
func (s *TaskService) Delete(ctx context.Context, id string) error {
	t, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.publish("deleted", t)
	return nil
}

// newID returns a random 128-bit hex identifier. crypto/rand.Read on a 16-byte
// buffer does not fail in practice; if it ever did we fall back to a
// timestamp-derived value rather than panicking.
func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(b)
}
