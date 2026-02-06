# Backend Server — Decisions

## Server
- Listens on `:8080`
- Uses `http.ServeMux` with Go 1.22+ method+path patterns
- Graceful shutdown via `os/signal` (SIGINT, SIGTERM) with 5s timeout

## CORS Middleware
- `corsMiddleware(next http.Handler) http.Handler`
- Sets `Access-Control-Allow-Origin: *`
- Sets `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS`
- Sets `Access-Control-Allow-Headers: Content-Type`
- Returns 204 for OPTIONS preflight requests
- Wraps the mux: `srv.Handler = corsMiddleware(mux)`

## Route registration
- `mux.HandleFunc("GET /health", HandleHealth)`
- `mux.HandleFunc("GET /tasks", HandleListTasks(store))`
- `mux.HandleFunc("POST /tasks", HandleCreateTask(store))`
- `mux.HandleFunc("GET /tasks/{id}", HandleGetTask(store))`
- `mux.HandleFunc("PUT /tasks/{id}", HandleUpdateTask(store))`
- `mux.HandleFunc("DELETE /tasks/{id}", HandleDeleteTask(store))`
