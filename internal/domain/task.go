// Package domain holds the core business entities and errors for TaskFlow.
// It has no dependencies on transport, storage, or framework code, keeping the
// business model isolated and testable (clean architecture).
package domain

import (
	"errors"
	"time"
)

// Status represents the lifecycle state of a Task.
type Status string

const (
	StatusTodo       Status = "todo"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

// Valid reports whether s is a recognized status.
func (s Status) Valid() bool {
	switch s {
	case StatusTodo, StatusInProgress, StatusDone:
		return true
	default:
		return false
	}
}

// Task is the central domain entity.
type Task struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Domain-level errors. Transport layers map these to protocol codes
// (e.g. HTTP 404 / 400) without leaking storage details.
var (
	ErrNotFound      = errors.New("task not found")
	ErrInvalidTitle  = errors.New("title must not be empty")
	ErrInvalidStatus = errors.New("invalid status")
)
