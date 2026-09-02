// Package repository defines the persistence contract for tasks and provides
// implementations. The interface lives here so the service layer depends on an
// abstraction, not a concrete store (dependency inversion). Swapping the
// in-memory store for Postgres later means adding a new implementation only.
package repository

import (
	"context"

	"github.com/proofpoint/taskflow/internal/domain"
)

// TaskRepository is the storage contract used by the service layer.
type TaskRepository interface {
	Create(ctx context.Context, t domain.Task) error
	Get(ctx context.Context, id string) (domain.Task, error)
	List(ctx context.Context) ([]domain.Task, error)
	Update(ctx context.Context, t domain.Task) error
	Delete(ctx context.Context, id string) error
}
