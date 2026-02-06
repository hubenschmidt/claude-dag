# Claude DAG

A multi-agent orchestration system built on Claude Code. Specialist agents collaborate through a dependency-aware task DAG to scaffold full-stack applications from natural language descriptions.

## Screenshots

![Tree view with live agent previews](spec/screenshots/Screenshot%20from%202026-02-06%2004-42-43.png)
*Tmux tree view showing the orchestrator dashboard and four Claude Code agents working in parallel*

![Completed run with task summary](spec/screenshots/Screenshot%20from%202026-02-06%2004-45-21.png)
*Completed orchestration run — DAG status table, event log, and per-agent task summaries*

## Prerequisites

- Go >= 1.22
- [tmux](https://github.com/tmux/tmux)
- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) installed and authenticated

```bash
go version
tmux -V
claude --version
```

## Quick Start

```bash
go build -o dag ./cmd/swarm

./dag Build a todo app with user accounts
```

Or run directly:

```bash
go run ./cmd/swarm Build a REST API in Go for a bookstore
```

## How It Works

Each agent runs in its own tmux window (tab) with the full Claude Code interactive TUI. The orchestrator occupies window 0 and displays a live dashboard.

### 5-Phase Orchestration

1. **Design** — Architect agent produces API contracts, data model, and task plan
2. **Build** — Specialist agents (Backend, Frontend, Database) execute in parallel (max 4 concurrent)
3. **Review** — Reviewer agents auto-wire for each code-producing task
4. **Validate** — Architect re-spawns to verify cross-agent coherence
5. **Assemble** — Integrator wires everything into a runnable project

```
Architect (design)
    │
    ├─> Backend  ─┐
    ├─> Frontend ─┤  (parallel, max 4)
    └─> Database ─┘
          │
       Reviewer (per task)
          │
    Architect (validate)
          │
      Integrator
```

### Interactive Dashboard

The orchestrator dashboard (window 0) shows:
- Live DAG status table (task, status, window, duration)
- Recent event log
- On failure: prompts for user feedback to retry

Navigate between agent windows: `Ctrl-b` then window number, or `Ctrl-b w` for the window picker.

### Shared Runtime Context

Agents share context at runtime via `artifacts/shared-context/`. Each agent reads sibling decisions before making interface choices, enabling cross-agent coordination without direct message passing.

## Output

All artifacts are written to `./artifacts/`:

```
artifacts/
├── contracts/         # API contracts, data models, task plans
├── schemas/           # Database schemas and migrations
├── code/
│   ├── backend/       # Generated backend code
│   └── frontend/      # Generated frontend code
├── reviews/           # Code review feedback
└── shared-context/    # Cross-agent runtime decisions
```

## Project Structure

```
├── cmd/swarm/main.go                # CLI entry point, tmux session setup
├── internal/
│   ├── model/model.go               # Task, AgentRole, TaskStatus types
│   ├── orchestrator/
│   │   ├── orchestrator.go          # 5-phase orchestration loop + DAG display
│   │   ├── graph.go                 # Thread-safe DAG with dependency resolution
│   │   └── dispatcher.go            # Agent dispatch with concurrency cap
│   ├── agent/
│   │   ├── agent.go                 # Agent interface (Role + Launch)
│   │   ├── prompt.go                # Prompt file writing, interactive launch
│   │   ├── architect.go             # Design + validation modes
│   │   ├── backend.go               # API code generation
│   │   ├── frontend.go              # Frontend code generation
│   │   └── reviewer.go              # Code review + quality gates
│   ├── artifact/artifact.go         # Filesystem artifact I/O
│   └── tmux/tmux.go                 # Tmux session/window management
├── prompts/                         # System prompt markdown files per role
├── spec/
│   └── poc-claude-dag.md            # Full POC specification
├── go.mod
└── go.sum
```
