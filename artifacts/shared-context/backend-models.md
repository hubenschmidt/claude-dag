# Backend Models — Decisions

## Task struct (store.go)
- `Task` struct with json tags: `id`, `title`, `description`, `status`, `created_at`, `updated_at`
- `UpdateFields` struct uses `*string` pointer fields for partial update semantics
- Status validation via `validStatuses` map (O(1) lookup): `todo`, `in_progress`, `done`

## TaskStore API
- `NewTaskStore() *TaskStore`
- `All(filterStatus string) []Task` — empty string means no filter; results sorted by `CreatedAt` ascending
- `GetByID(id string) (Task, bool)`
- `Create(task Task) (Task, error)` — sets ID, timestamps, default status server-side; returns error on UUID failure
- `Update(id string, patch UpdateFields) (Task, bool)`
- `Delete(id string) bool`

## UUID generation
- `newUUID() (string, error)` — UUID v4 via `crypto/rand`, propagates rand.Read errors

## Package
- All backend code in `package main`
