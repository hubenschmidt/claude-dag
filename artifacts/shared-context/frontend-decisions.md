# Frontend Decisions

## API Client
- All HTTP calls use `fetch()` to `http://localhost:8080`
- No wrapper library — calls are inline in components

## Component Structure
- `App` — root component, owns task state (`useState`), fetches via `useEffect`
- `TaskForm` — POST /tasks, calls `onCreated` callback with the new Task object
- `TaskList` — renders table of tasks + filter bar, delegates to TaskItem
- `TaskItem` — single table row, handles PUT (status cycle) and DELETE inline

## Response Shape Expectations
- `GET /tasks` returns `{ tasks: Task[] }`
- `POST /tasks` returns a `Task` object directly
- `PUT /tasks/{id}` returns a `Task` object directly
- `DELETE /tasks/{id}` returns 204 with no body

## Status Cycle
Click status badge cycles: `todo → in_progress → done → todo`

## Filtering
Query param `?status=<value>` sent to GET /tasks when filter is active.
