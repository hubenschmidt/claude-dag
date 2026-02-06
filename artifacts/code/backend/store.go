package main

import (
	"crypto/rand"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Task represents a unit of work to be tracked.
type Task struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UpdateFields holds optional fields for partial task updates.
type UpdateFields struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
}

// validStatuses provides O(1) lookup for status validation.
var validStatuses = map[string]bool{
	"todo":        true,
	"in_progress": true,
	"done":        true,
}

// TaskStore is a thread-safe in-memory store for tasks.
type TaskStore struct {
	mu    sync.RWMutex
	tasks map[string]Task
}

// NewTaskStore creates an empty TaskStore.
func NewTaskStore() *TaskStore {
	return &TaskStore{tasks: make(map[string]Task)}
}

// All returns all tasks sorted by CreatedAt ascending, optionally filtered by status.
func (s *TaskStore) All(filterStatus string) []Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		if filterStatus == "" || t.Status == filterStatus {
			result = append(result, t)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result
}

// GetByID returns a task by its ID and whether it was found.
func (s *TaskStore) GetByID(id string) (Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tasks[id]
	return t, ok
}

// Create adds a new task, setting ID, status default, and timestamps server-side.
func (s *TaskStore) Create(task Task) (Task, error) {
	id, err := newUUID()
	if err != nil {
		return Task{}, err
	}

	now := time.Now().UTC()
	task.ID = id
	task.CreatedAt = now
	task.UpdatedAt = now
	task.Status = "todo"

	s.mu.Lock()
	s.tasks[task.ID] = task
	s.mu.Unlock()

	return task, nil
}

// Update applies partial updates to an existing task. Returns the updated task and whether it was found.
func (s *TaskStore) Update(id string, patch UpdateFields) (Task, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[id]
	if !ok {
		return Task{}, false
	}

	if patch.Title != nil {
		t.Title = *patch.Title
	}
	if patch.Description != nil {
		t.Description = *patch.Description
	}
	if patch.Status != nil {
		t.Status = *patch.Status
	}

	t.UpdatedAt = time.Now().UTC()
	s.tasks[id] = t
	return t, true
}

// Delete removes a task by ID. Returns whether the task existed.
func (s *TaskStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.tasks[id]
	if !ok {
		return false
	}
	delete(s.tasks, id)
	return true
}

// newUUID generates a UUID v4 string using crypto/rand.
func newUUID() (string, error) {
	var b [16]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return "", fmt.Errorf("generating uuid: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
