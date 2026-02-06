package main

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse is the standard error body.
type ErrorResponse struct {
	Error string `json:"error"`
}

// TaskListResponse wraps the array of tasks.
type TaskListResponse struct {
	Tasks []Task `json:"tasks"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}

// HandleHealth handles GET /health.
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleListTasks returns a handler for GET /tasks.
func HandleListTasks(store *TaskStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filterStatus := r.URL.Query().Get("status")
		if filterStatus != "" && !validStatuses[filterStatus] {
			writeError(w, http.StatusBadRequest, "invalid status filter")
			return
		}
		tasks := store.All(filterStatus)
		writeJSON(w, http.StatusOK, TaskListResponse{Tasks: tasks})
	}
}

// HandleCreateTask returns a handler for POST /tasks.
func HandleCreateTask(store *TaskStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

		var req struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.Title == "" {
			writeError(w, http.StatusBadRequest, "title is required")
			return
		}
		if len(req.Title) > 255 {
			writeError(w, http.StatusBadRequest, "title must be 255 characters or fewer")
			return
		}
		if len(req.Description) > 4096 {
			writeError(w, http.StatusBadRequest, "description must be 4096 characters or fewer")
			return
		}
		task, err := store.Create(Task{Title: req.Title, Description: req.Description})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusCreated, task)
	}
}

// HandleGetTask returns a handler for GET /tasks/{id}.
func HandleGetTask(store *TaskStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		task, ok := store.GetByID(id)
		if !ok {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		writeJSON(w, http.StatusOK, task)
	}
}

// HandleUpdateTask returns a handler for PUT /tasks/{id}.
func HandleUpdateTask(store *TaskStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		id := r.PathValue("id")

		var patch UpdateFields
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if patch.Title != nil && *patch.Title == "" {
			writeError(w, http.StatusBadRequest, "title must not be empty")
			return
		}
		if patch.Title != nil && len(*patch.Title) > 255 {
			writeError(w, http.StatusBadRequest, "title must be 255 characters or fewer")
			return
		}
		if patch.Description != nil && len(*patch.Description) > 4096 {
			writeError(w, http.StatusBadRequest, "description must be 4096 characters or fewer")
			return
		}
		if patch.Status != nil && !validStatuses[*patch.Status] {
			writeError(w, http.StatusBadRequest, "invalid status")
			return
		}
		task, ok := store.Update(id, patch)
		if !ok {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		writeJSON(w, http.StatusOK, task)
	}
}

// HandleDeleteTask returns a handler for DELETE /tasks/{id}.
func HandleDeleteTask(store *TaskStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !store.Delete(id) {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
