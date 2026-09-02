package repository

import (
	"context"
	"sort"
	"sync"

	"github.com/proofpoint/taskflow/internal/domain"
)

// MemoryStore is a goroutine-safe, in-memory TaskRepository. The RWMutex allows
// many concurrent readers while serializing writers, which suits the
// read-heavy access pattern of a task API.
type MemoryStore struct {
	mu    sync.RWMutex
	tasks map[string]domain.Task
}

// NewMemoryStore returns an initialized in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{tasks: make(map[string]domain.Task)}
}

// compile-time assertion that MemoryStore satisfies the interface.
var _ TaskRepository = (*MemoryStore)(nil)

func (s *MemoryStore) Create(ctx context.Context, t domain.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[t.ID] = t
	return nil
}

func (s *MemoryStore) Get(ctx context.Context, id string) (domain.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[id]
	if !ok {
		return domain.Task{}, domain.ErrNotFound
	}
	return t, nil
}

func (s *MemoryStore) List(ctx context.Context) ([]domain.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		out = append(out, t)
	}
	// Deterministic ordering: newest first, then by ID for stable ties.
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func (s *MemoryStore) Update(ctx context.Context, t domain.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[t.ID]; !ok {
		return domain.ErrNotFound
	}
	s.tasks[t.ID] = t
	return nil
}

func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[id]; !ok {
		return domain.ErrNotFound
	}
	delete(s.tasks, id)
	return nil
}
