# Backend Handlers — Decisions

## Routing (Go 1.22+ mux patterns)
- `GET /health` → `HandleHealth` (plain `http.HandlerFunc`)
- `GET /tasks` → `HandleListTasks(store)` — optional `?status=` filter
- `POST /tasks` → `HandleCreateTask(store)` — returns 201
- `GET /tasks/{id}` → `HandleGetTask(store)`
- `PUT /tasks/{id}` → `HandleUpdateTask(store)` — partial update via `UpdateFields`
- `DELETE /tasks/{id}` → `HandleDeleteTask(store)` — returns 204 on success

## Path parsing
- `{id}` extracted via `r.PathValue("id")`

## Body limits
- POST and PUT handlers wrap `r.Body` with `http.MaxBytesReader(w, r.Body, 1<<20)` (1 MiB cap)

## Response shapes
- List: `{"tasks": [...]}`
- Single: `{"id":..., "title":..., ...}`
- Error: `{"error": "message"}`
- Health: `{"status": "ok"}`

## Validation
- Create: title required, 1-255 chars; description optional, max 4096
- Update: title non-empty if provided, max 255; description max 4096; status must be valid enum
- Status filter on list must be valid enum if provided
