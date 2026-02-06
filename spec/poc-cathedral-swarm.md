# POC: Cathedral Swarm — Multi-Agent Orchestration System

## Hypothesis
A hierarchical agent swarm built on Claude Opus 4.6 can autonomously decompose, parallelize, and execute complex software engineering tasks (full-stack scaffolding and codebase migration) with higher throughput and quality than a single-agent approach, by leveraging specialized agent roles, dependency-aware task graphs, and automated review loops.

## Success Criteria
- [ ] Scaffold a working full-stack CRUD app from a 2-sentence natural language description
- [ ] Execute at least 3 specialist agents in parallel during a single run
- [ ] Reviewer agent catches and requests rework on at least one issue per run
- [ ] Migrate a JavaScript project to TypeScript with compilable output in a single run
- [ ] All generated code compiles and passes basic smoke tests
- [ ] End-to-end scaffold completes in under 5 minutes for a simple app

---

## Problem Context

### Current State
- Single-agent LLM workflows execute tasks sequentially, creating bottlenecks on independent work
- Complex projects require context that exceeds single-agent working memory
- No standardized pattern exists for coordinating multiple Claude agents on a shared codebase
- Migration/refactor tasks are tedious and error-prone when done file-by-file

### What We Need to Learn
- Can a DAG-based task graph effectively coordinate agent dependencies without deadlocks or race conditions on shared files?
- Does parallel specialist execution actually reduce wall-clock time vs. sequential single-agent?
- Can agents produce compatible artifacts (APIs, interfaces, schemas) without excessive rework?
- Is the Claude Code `Task` tool a viable primitive for spawning and managing sub-agents?

### Risk if Not Validated
- Investing in a full agent framework without proving the coordination model works wastes engineering effort
- Inability to demonstrate parallel execution undermines the core value proposition of swarms over single agents
- Artifact conflicts (e.g., incompatible API contracts) could make multi-agent worse than single-agent

---

## Approach

### What We're Testing
A Go orchestration layer that:
1. Accepts a high-level goal (build app / migrate codebase)
2. Decomposes it into a dependency-aware task DAG
3. Dispatches tasks to specialist agents via Claude Code sub-agents
4. Manages artifact handoff through the filesystem (shared contracts, schemas, code files)
5. Enforces quality gates via a Reviewer agent before marking tasks complete

### Architecture

#### Task DAG

```
                    ┌─────────────┐
                    │ Orchestrator │
                    │  (DAG + dispatch)
                    └──────┬──────┘
                           │
                    ┌──────┴──────┐
                    │  Architect  │  Phase 1: Design
                    │  (design)   │
                    └──────┬──────┘
                           │ (blocks until complete)
            ┌──────────────┼──────────────┐
            │              │              │
      ┌─────┴─────┐ ┌─────┴─────┐ ┌─────┴─────┐
      │  Backend  │ │ Frontend  │ │ Database  │  Phase 2: Build
      │  Agent    │ │  Agent    │ │  Agent    │  (max 4 concurrent)
      └─────┬─────┘ └─────┬─────┘ └─────┬─────┘
            │              │              │
            └──────┬───────┼──────────────┘
                   │ (auto-wired per code task)
            ┌──────┴──────┐
            │  Reviewer   │  Phase 3: Review
            │  (per task) │
            └──────┬──────┘
                   │ (all reviews pass)
            ┌──────┴──────┐
            │  Architect  │  Phase 4: Validate
            │  (validate) │  Cross-agent coherence check
            └──────┬──────┘
                   │
            ┌──────┴──────┐
            │ Integrator  │  Phase 5: Assemble
            │  Agent      │
            └─────────────┘
```

#### Terminal Layout

Each agent runs in its own tmux window (tab). The orchestrator occupies window 0 and displays a live dashboard. Agents run in Claude Code interactive CLI mode — they auto-execute their prompt on launch, but can pause and ask the user questions if they need clarification.

```
  tmux session: cathedral-swarm
  ┌──────────────────────────────────────────────────────┐
  │ [0:orchestrator] [1:architect] [2:backend] [3:front] │  ← tab bar
  └──────────────────────────────────────────────────────┘

  Window 0 — Orchestrator Dashboard (live DAG view)
  ┌──────────────────────────────────────────────────────┐
  │ === Cathedral Swarm ===                              │
  │ Task               Status      Window   Duration    │
  │ ──────────────────────────────────────────────────── │
  │ architect-design    completed   1        42s         │
  │ backend-api         running     2        18s         │
  │ frontend-ui         running     3        12s         │
  │ review-backend      pending     -        -           │
  │                                                      │
  │ --- Events ---                                       │
  │   architect-design completed (sentinel)              │
  │   task plan expanded, 4 total tasks                  │
  │                                                      │
  │ --- Agent Questions ---                              │
  │   [backend] Which ORM should I use? (window 2)      │
  └──────────────────────────────────────────────────────┘

  Windows 1-N — Individual agent Claude Code interactive sessions
```

#### Orchestration Flow

1. User runs: `swarm "Build a todo app with user accounts"`
2. Orchestrator creates tmux session, occupies window 0
3. **Phase 1 — Design**: Architect agent spawns in window 1, auto-executes the design prompt
4. Orchestrator **blocks** until architect completes (polls for `.done` sentinel)
5. Orchestrator reads `artifacts/contracts/task-plan.yaml`, expands the DAG
6. **Phase 2 — Build**: Sub-agents spawn in windows 2-N (max 4 concurrent), each auto-executing
7. Orchestrator polls: reaps completed tasks, launches newly-ready tasks, surfaces questions
8. **Phase 3 — Review**: Reviewer agents auto-wire for code-producing tasks, reject → retry cycle
9. **Phase 4 — Validate**: Once all reviews pass, architect re-spawns to validate cross-agent coherence (do implementations honor contracts collectively?)
10. If architect rejects: specific tasks re-enter the build→review cycle with architect feedback
11. **Phase 5 — Assemble**: Integrator agent assembles final output from all validated artifacts

### Communication Model
- **Artifact Bus**: Shared filesystem directory (`artifacts/`) with structured outputs
  - `artifacts/contracts/` — API contracts, interface definitions (produced by Architect, consumed by all agents)
  - `artifacts/schemas/` — Database schemas (produced by DB Agent, consumed by Backend)
  - `artifacts/code/backend/` — Backend source code
  - `artifacts/code/frontend/` — Frontend source code
  - `artifacts/reviews/` — Review feedback files (produced by Reviewer, consumed by task owners)
  - `artifacts/shared-context/` — Runtime shared state for cross-agent awareness
- **Shared Runtime Context**: Agents can read sibling artifacts at runtime, not just at launch. Each agent is instructed to check `artifacts/shared-context/` before making interface decisions (endpoint signatures, data shapes, shared types). Agents write key decisions here as they work, enabling runtime coordination without direct message passing.
- **Task Graph**: In-memory DAG tracking task status, dependencies, assigned agent, tmux window ID, and output artifacts

### Technical Constraints
- Claude Code CLI and tmux must be available in the execution environment
- All agents run in Claude Code **interactive CLI mode** — they can use tools, show live diffs, and pause for user input
- Agents receive their full prompt at launch (written to a temp file, referenced via positional arg) and auto-execute immediately
- Maximum **4 agents running concurrently** — dispatcher holds remaining tasks until slots free up
- Agents share context at runtime via the filesystem (`artifacts/shared-context/`), not just at launch
- API rate limits and context window budgets must be tracked per agent session

### Observability
- The orchestrator dashboard (window 0) displays:
  - Live DAG status table (task, status, window ID, duration)
  - Recent event log (completions, launches, rejections)
  - Pending agent questions (surfaced from agent windows)
- **Context budget tracking**: each agent session's remaining Claude API context window is monitored and surfaced in the dashboard, so the user knows when an agent is approaching its limit

### Out of Scope
- Production deployment or CI/CD integration
- Real-time WebSocket dashboard (Phase 4 stretch — not part of POC)
- More than 2 migration strategies (POC validates JS→TS only)
- Authentication, authorization, or multi-user scenarios in generated apps
- Agent self-modification or learning across runs

---

## Implementation

### Phase 1: Core Orchestration + Architect + Backend (MVP)

#### Steps
1. **Define the Task Graph schema** — Go structs for Task, TaskGraph (DAG), TaskStatus, AgentRole
2. **Build the Orchestrator** — Accepts a goal string, calls Architect agent to produce a task decomposition, builds DAG
3. **Implement Agent Dispatcher** — Walks the DAG, identifies ready tasks (all deps resolved), spawns sub-agents via goroutines + Anthropic API
4. **Implement Architect Agent** — Takes a goal, produces: tech stack decision, API contract, data model, task breakdown
5. **Implement Backend Agent** — Consumes API contract + data model, produces API endpoints + business logic
6. **Wire artifact handoff** — Architect writes contracts to `artifacts/`, Backend reads them before generating code
7. **Validate** — Run with prompt: *"Build a todo app with user accounts"* — verify Backend output matches Architect contract

#### Phase 2: Full Agent Roster + Parallel Execution
8. **Implement Frontend Agent** — Consumes API contract, produces React components + pages
9. **Implement Database Agent** — Consumes data model, produces schema migrations + seed data
10. **Implement Reviewer Agent** — Reads generated code, checks for: compilation errors, contract mismatches, missing error handling. Outputs review feedback files
11. **Implement rework loop** — If Reviewer rejects, Orchestrator re-dispatches task to original agent with feedback attached
12. **Enable parallel dispatch** — Orchestrator launches all ready tasks concurrently (Frontend + Backend + DB in parallel after Architect completes)
13. **Implement Integrator Agent** — Reads all code artifacts, wires imports, fixes interface mismatches, produces final runnable project

#### Phase 3: Migration Capabilities
14. **Implement Migrator Agent** — Analyzes existing JS codebase, identifies files/patterns to transform
15. **Build migration task decomposition** — Orchestrator creates per-file or per-module migration tasks
16. **Parallel migration execution** — Multiple Migrator agent instances refactor files concurrently
17. **Post-migration validation** — Reviewer agent runs `tsc --noEmit` and reports type errors
18. **Validate** — Run against a sample JS project, verify compilable TS output

### Code Structure

```
cathedral-swarm/
├── cmd/
│   └── swarm/
│       └── main.go               # Entry point — accepts goal, launches tmux session
├── internal/
│   ├── orchestrator/
│   │   ├── orchestrator.go       # Core loop — architect phase, poll loop, DAG display
│   │   ├── graph.go              # DAG construction, traversal, status tracking
│   │   └── dispatcher.go         # Agent dispatch — role mapping, concurrency cap
│   ├── agent/
│   │   ├── agent.go              # Agent interface (Role + Launch)
│   │   ├── prompt.go             # Prompt file writing, claude CLI arg builders
│   │   ├── architect.go          # System design + contract generation
│   │   ├── frontend.go           # Frontend code generation
│   │   ├── backend.go            # API + business logic generation
│   │   └── reviewer.go           # Quality gates + review feedback
│   ├── artifact/
│   │   └── artifact.go           # Read/write/list artifacts on filesystem
│   ├── model/
│   │   └── model.go              # Task, TaskStatus, AgentRole types
│   └── tmux/
│       └── tmux.go               # Tmux session/window/pane management
├── prompts/                      # System prompt markdown files per agent role
│   ├── architect.md
│   ├── backend.md
│   ├── frontend.md
│   └── reviewer.md
├── artifacts/                    # Runtime artifact bus (gitignored)
│   ├── contracts/
│   ├── schemas/
│   ├── code/
│   ├── reviews/
│   └── shared-context/           # Cross-agent runtime decisions
├── go.mod
└── go.sum
```

### Key Implementation Detail: Agent Invocation

Each agent launches as a Claude Code interactive CLI session in its own tmux window. The orchestration layer:
1. Writes the full prompt (task description + relevant artifacts) to a temp file
2. Launches `claude --append-system-prompt <system> --allowedTools ... "<read prompt file>"` in a new tmux window
3. Claude auto-submits the first message and begins working with full TUI (live diffs, tool calls visible)
4. If the agent needs clarification, it pauses in its window — the orchestrator surfaces this in the dashboard
5. On completion, the agent writes a `.done` sentinel file; the orchestrator reaps it on the next poll cycle

### Dependencies
- Go >= 1.22
- `gopkg.in/yaml.v3` — YAML parsing for architect task plans
- `tmux` — terminal multiplexer (runtime dependency)
- `claude` — Claude Code CLI (runtime dependency, requires Claude API key with Opus 4.6 access)

### Estimated Effort
**Medium (M)** — ~3-5 days for Phase 1-2, ~2 days for Phase 3

---

## Results

### Observations
- [ ] *To be filled after POC execution*

### Performance/Metrics

| Metric                              | Expected   | Actual  |
| ----------------------------------- | ---------- | ------- |
| Scaffold time (simple CRUD app)     | < 5 min    | [value] |
| Agents running in parallel (max)    | >= 3       | [value] |
| Review rework cycles per run        | 1-2        | [value] |
| Migration: files converted per min  | >= 10      | [value] |
| Generated code compiles cleanly     | Yes        | [value] |
| Contract mismatches post-integration| 0          | [value] |

### Issues Encountered
- [ ] *To be filled after POC execution*

---

## Decision

**Outcome:** [PROCEED / PIVOT / ABANDON]

**Rationale:** [To be determined after POC execution]

**Next Steps:**
- [ ] Execute Phase 1 — Core orchestration + Architect + Backend agents
- [ ] Validate with "Build a todo app with user accounts" prompt
- [ ] Measure parallel execution throughput vs. sequential baseline
- [ ] If PROCEED: expand to Phase 2 full roster, then Phase 3 migration
- [ ] If PIVOT: identify which coordination patterns failed and redesign
