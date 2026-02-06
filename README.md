# Cathedral Swarm

A multi-agent orchestration system built on Claude Opus 4.6. Specialist agents collaborate through a dependency-aware task graph to scaffold full-stack applications from natural language descriptions.

## Prerequisites

- Go >= 1.22
- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) installed and authenticated

Verify both are working:

```bash
go version
claude --version
```

## Quick Start

```bash
# Build
go build -o swarm ./cmd/swarm

# Run with a goal
./swarm Build a todo app with user accounts and authentication
```

Or run directly with `go run`:

```bash
go run ./cmd/swarm Build a REST API in Go for a bookstore
```

## How It Works

1. **Orchestrator** receives your goal and creates an Architect task
2. **Architect agent** produces an API contract, data model, and task plan (written to `artifacts/contracts/`)
3. **Orchestrator** parses the task plan into a dependency-aware DAG
4. **Specialist agents** (Backend, Reviewer) execute in parallel waves via goroutines
5. Generated code is written to `artifacts/code/`
6. Review feedback is written to `artifacts/reviews/`

```
Goal
 └─> Architect ─── contracts + task plan
      └─> Backend ──┐
      └─> Database ─┤  (parallel)
      └─> Frontend ─┘
           └─> Reviewer ─── approve / reject
                └─> Integrator ─── final assembly
```

## Output

All artifacts are written to `./artifacts/`:

```
artifacts/
├── contracts/       # API contracts, data models, task plans
├── schemas/         # Database schemas and migrations
├── code/
│   ├── backend/     # Generated backend code
│   └── frontend/    # Generated frontend code
└── reviews/         # Code review feedback
```

## Project Structure

```
├── cmd/swarm/main.go                # CLI entry point
├── internal/
│   ├── model/model.go               # Shared types (Task, AgentRole, TaskStatus)
│   ├── orchestrator/
│   │   ├── orchestrator.go          # Core loop: architect → DAG → dispatch waves
│   │   ├── graph.go                 # Thread-safe DAG with dependency resolution
│   │   └── dispatcher.go            # Parallel agent dispatch via errgroup
│   ├── agent/
│   │   ├── agent.go                 # Agent interface
│   │   ├── architect.go             # System design + contract generation
│   │   ├── backend.go               # API code generation
│   │   ├── reviewer.go              # Code review + quality gates
│   │   └── parse.go                 # LLM response block parser
│   ├── artifact/artifact.go         # Filesystem artifact I/O
│   └── claude/client.go             # Claude Code CLI wrapper
└── spec/
    └── poc-cathedral-swarm.md       # Full POC specification
```

## Agent Roles

| Agent      | Input                      | Output                        | Status |
|------------|----------------------------|-------------------------------|--------|
| Architect  | Natural language goal      | API contract, data model, DAG | Done   |
| Backend    | Contracts + data model     | Go API server code            | Done   |
| Reviewer   | Generated code + contracts | APPROVED / REJECTED verdict   | Done   |
| Frontend   | API contract               | React components              | Phase 2|
| Database   | Data model                 | Schema migrations + seeds     | Phase 2|
| Integrator | All code artifacts         | Wired, runnable project       | Phase 2|
| Migrator   | Existing codebase          | Refactored/migrated code      | Phase 3|

## Configuration

The swarm uses `claude-opus-4-6` by default. The model is set in `internal/claude/client.go`.

The 10-minute timeout is set in `cmd/swarm/main.go`.
