package orchestrator

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/hubenschmidt/cathedral-swarm/internal/agent"
	"github.com/hubenschmidt/cathedral-swarm/internal/artifact"
	"github.com/hubenschmidt/cathedral-swarm/internal/model"
	"github.com/hubenschmidt/cathedral-swarm/internal/tmux"
)

// Roles that produce code and should be auto-reviewed.
var reviewableRoles = map[model.AgentRole]bool{
	model.RoleBackend:  true,
	model.RoleFrontend: true,
	model.RoleDatabase: true,
}

const (
	pollInterval = 3 * time.Second
	maxWaves     = 50
)

// Orchestrator manages the DAG of tasks, launching them as tmux windows
// and polling for completion.
type Orchestrator struct {
	session    string
	dispatcher *Dispatcher
	graph      *Graph
	events     []string // recent log events shown in the DAG display
}

// New creates an orchestrator bound to a tmux session.
func New(session string, agents []agent.Agent) *Orchestrator {
	return &Orchestrator{
		session:    session,
		dispatcher: NewDispatcher(session, agents),
		graph:      NewGraph(),
	}
}

// Run executes the full 5-phase orchestration.
func (o *Orchestrator) Run(ctx context.Context, goal string) error {
	log.Printf("[orchestrator] goal: %s", goal)

	// Phase 1 — Design: launch architect
	o.logEvent("phase 1: architect design")
	archTask := &model.Task{
		ID:          "architect-design",
		Role:        model.RoleArchitect,
		Description: goal,
		OutputDir:   "contracts",
	}
	if err := o.graph.AddTask(archTask); err != nil {
		return fmt.Errorf("add architect task: %w", err)
	}
	if err := o.dispatcher.LaunchReady(o.graph); err != nil {
		return fmt.Errorf("launch architect: %w", err)
	}
	if err := o.pollUntilDone(ctx, "architect-design"); err != nil {
		return fmt.Errorf("architect design: %w", err)
	}

	// Expand the architect's task plan into the DAG
	if err := o.expandTaskPlan(); err != nil {
		return fmt.Errorf("expand task plan: %w", err)
	}
	o.logEvent("task plan expanded, %d total tasks", len(o.graph.Tasks()))

	// Phase 2+3 — Build + Review: poll loop for sub-agents and reviewers
	o.logEvent("phase 2-3: build + review")
	if err := o.pollLoop(ctx); err != nil {
		return fmt.Errorf("build/review phase: %w", err)
	}

	// Phase 4 — Validate: architect reviews cross-agent coherence
	o.logEvent("phase 4: architect validation")
	if err := o.runArchitectValidation(ctx); err != nil {
		return fmt.Errorf("architect validation: %w", err)
	}

	// Phase 5 — Assemble: integrator wires everything together
	// TODO: implement integrator agent launch
	o.logEvent("all phases complete")
	return nil
}

// runArchitectValidation spawns the architect in validation mode to check
// cross-agent coherence. If rejected, specific tasks re-enter build→review.
func (o *Orchestrator) runArchitectValidation(ctx context.Context) error {
	reviewDeps := o.reviewTaskIDs()

	valTask := &model.Task{
		ID:        "architect-validate",
		Role:      model.RoleArchitect,
		DependsOn: reviewDeps,
		Description: "Validate that all implementations honor the original contracts",
		OutputDir: "reviews",
	}
	if err := o.graph.AddTask(valTask); err != nil {
		return fmt.Errorf("add validation task: %w", err)
	}
	if err := o.dispatcher.LaunchReady(o.graph); err != nil {
		return fmt.Errorf("launch validation: %w", err)
	}
	if err := o.pollUntilDone(ctx, "architect-validate"); err != nil {
		return fmt.Errorf("validation poll: %w", err)
	}

	// Read the validation verdict
	verdict, err := artifact.Read("reviews", "architect-validate.md")
	if err != nil {
		return fmt.Errorf("read validation verdict: %w", err)
	}

	if parseVerdict(verdict) == "APPROVED" {
		o.logEvent("architect validation: APPROVED")
		return nil
	}

	// Rejected — extract feedback and re-queue affected tasks
	feedback := extractFeedback(verdict)
	o.logEvent("architect validation: REJECTED — re-entering build/review")

	// Reset all code-producing tasks with the architect's feedback
	for _, t := range o.graph.Tasks() {
		if !reviewableRoles[t.Role] {
			continue
		}
		_ = o.graph.RejectTask(t.ID, feedback)
		// Also reset corresponding reviewer
		if t.ReviewTaskID != "" {
			_ = o.graph.SetStatus(t.ReviewTaskID, model.StatusPending)
			_ = o.graph.SetResult(t.ReviewTaskID, "")
		}
	}

	// Reset the validation task itself so it can re-run after fixes
	_ = o.graph.SetStatus("architect-validate", model.StatusPending)
	_ = o.graph.SetResult("architect-validate", "")

	// Re-enter build→review→validate loop
	if err := o.pollLoop(ctx); err != nil {
		return fmt.Errorf("rework build/review: %w", err)
	}
	return o.runArchitectValidation(ctx)
}

// reviewTaskIDs returns IDs of all reviewer tasks in the graph.
func (o *Orchestrator) reviewTaskIDs() []string {
	var ids []string
	for _, t := range o.graph.Tasks() {
		if t.Role != model.RoleReviewer {
			continue
		}
		ids = append(ids, t.ID)
	}
	return ids
}

// pollUntilDone blocks until the named task's pane exits.
func (o *Orchestrator) pollUntilDone(ctx context.Context, taskID string) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		o.printDAG()
		o.reapFinished()

		task, ok := o.graph.Get(taskID)
		if !ok {
			return fmt.Errorf("task %s not found", taskID)
		}

		if task.Status == model.StatusCompleted {
			return nil
		}
		if task.Status == model.StatusFailed {
			return fmt.Errorf("task %s failed: %s", taskID, task.Error)
		}

		time.Sleep(pollInterval)
	}
}

// pollLoop is the main orchestration loop for build+review phases.
func (o *Orchestrator) pollLoop(ctx context.Context) error {
	for i := range maxWaves {
		if err := ctx.Err(); err != nil {
			return err
		}

		o.printDAG()
		o.reapFinished()
		o.processReviews()

		if o.graph.AllCompleted() {
			log.Println("[orchestrator] all tasks completed")
			o.printDAG()
			return nil
		}

		// On permanent failure, ask user for feedback instead of exiting
		if o.graph.HasFailed() {
			o.printDAG()
			if !o.promptUserForRetry() {
				return fmt.Errorf("one or more tasks failed permanently")
			}
			continue
		}

		// Launch any newly-ready tasks
		if err := o.dispatcher.LaunchReady(o.graph); err != nil {
			return fmt.Errorf("wave %d: %w", i+1, err)
		}

		// Check for deadlock: nothing running, nothing ready, not all completed
		if !o.hasRunning() && len(o.graph.ReadyTasks()) == 0 && !o.graph.AllCompleted() {
			return fmt.Errorf("deadlock: no running or ready tasks, but not all completed")
		}

		time.Sleep(pollInterval)
	}

	return fmt.Errorf("exceeded max polling iterations (%d)", maxWaves)
}

// promptUserForRetry shows failed tasks and asks the user for feedback.
// Returns true if the user provided feedback and tasks were reset for retry.
func (o *Orchestrator) promptUserForRetry() bool {
	fmt.Println("--- Failed Tasks ---")
	for _, t := range o.graph.Tasks() {
		if t.Status != model.StatusFailed {
			continue
		}
		fmt.Printf("  %s: %s\n", t.ID, t.Error)
	}
	fmt.Println()
	fmt.Print("Enter feedback to retry failed tasks (or 'q' to quit): ")

	var input string
	fmt.Scanln(&input)

	if input == "" || input == "q" {
		return false
	}

	for _, t := range o.graph.Tasks() {
		if t.Status != model.StatusFailed {
			continue
		}
		t.Attempts = 0
		t.Status = model.StatusPending
		t.Feedback = input
		t.Error = ""
		o.logEvent("user retry: %s", t.ID)

		// Also reset corresponding reviewer if one exists
		if t.ReviewTaskID == "" {
			continue
		}
		rev, ok := o.graph.Get(t.ReviewTaskID)
		if !ok {
			continue
		}
		rev.Status = model.StatusPending
		rev.Result = ""
	}

	return true
}

// reapFinished checks running tasks for completion via .done sentinel files
// or pane death (whichever comes first).
func (o *Orchestrator) reapFinished() {
	for _, t := range o.graph.Tasks() {
		if t.Status != model.StatusRunning {
			continue
		}

		// Check for .done sentinel file
		if t.OutputDir != "" {
			donePath := filepath.Join(artifact.BaseDir, t.OutputDir, ".done")
			if _, err := os.Stat(donePath); err == nil {
				_ = o.graph.SetStatus(t.ID, model.StatusCompleted)
				o.logEvent("task %s completed (sentinel)", t.ID)
				continue
			}
		}

		// Fallback: pane exited
		if t.PaneID != "" && !tmux.IsPaneAlive(t.PaneID) {
			_ = o.graph.SetStatus(t.ID, model.StatusCompleted)
			o.logEvent("task %s completed (pane %s exited)", t.ID, t.PaneID)
		}
	}
}

// processReviews checks completed reviewer tasks and either approves or
// rejects the code task they reviewed.
func (o *Orchestrator) processReviews() {
	for _, t := range o.graph.Tasks() {
		if t.Role != model.RoleReviewer || t.Status != model.StatusCompleted {
			continue
		}
		if t.ReviewTaskID == "" {
			continue
		}

		reviewed, ok := o.graph.Get(t.ReviewTaskID)
		if !ok || reviewed.Status != model.StatusCompleted {
			continue
		}

		// Read the review artifact to determine verdict
		reviewContent, err := artifact.Read("reviews", t.ID+".md")
		if err != nil {
			log.Printf("[orchestrator] could not read review for %s: %v", t.ID, err)
			continue
		}

		verdict := parseVerdict(reviewContent)
		if verdict == "APPROVED" {
			o.logEvent("review APPROVED: %s", t.ReviewTaskID)
			continue
		}

		feedback := extractFeedback(reviewContent)
		o.logEvent("review REJECTED: %s (attempt %d/%d)", t.ReviewTaskID, reviewed.Attempts+1, model.MaxAttempts)

		_ = o.graph.RejectTask(t.ReviewTaskID, feedback)
		_ = o.graph.SetStatus(t.ID, model.StatusPending)
		_ = o.graph.SetResult(t.ID, "")
	}
}

func (o *Orchestrator) logEvent(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	o.events = append(o.events, msg)
	// Keep last 10 events
	if len(o.events) > 10 {
		o.events = o.events[len(o.events)-10:]
	}
}

// hasRunning returns true if any task is in running status.
func (o *Orchestrator) hasRunning() bool {
	for _, t := range o.graph.Tasks() {
		if t.Status == model.StatusRunning {
			return true
		}
	}
	return false
}

// printDAG renders the current DAG status table to stdout.
func (o *Orchestrator) printDAG() {
	fmt.Print("\033[2J\033[H") // clear screen
	fmt.Println("=== Cathedral Swarm ===")
	fmt.Println()
	fmt.Printf("%-30s %-12s %-8s %-10s\n", "Task", "Status", "Window", "Duration")
	fmt.Println(strings.Repeat("-", 65))

	now := time.Now().Unix()
	for _, t := range o.graph.Tasks() {
		pane := "-"
		if t.PaneID != "" {
			pane = t.PaneID
		}

		dur := "-"
		if t.StartedAt > 0 {
			elapsed := time.Duration(now-t.StartedAt) * time.Second
			dur = elapsed.Truncate(time.Second).String()
		}

		fmt.Printf("%-30s %-12s %-8s %-10s\n", t.ID, t.Status, pane, dur)
	}

	// Show recent events
	if len(o.events) > 0 {
		fmt.Println()
		fmt.Println("--- Events ---")
		for _, e := range o.events {
			fmt.Printf("  %s\n", e)
		}
	}
	fmt.Println()
}

func parseVerdict(result string) string {
	upper := strings.ToUpper(strings.TrimSpace(result))
	if strings.HasPrefix(upper, "APPROVED") {
		return "APPROVED"
	}
	return "REJECTED"
}

func extractFeedback(result string) string {
	idx := strings.Index(strings.ToUpper(result), "REJECTED:")
	if idx == -1 {
		return result
	}
	return strings.TrimSpace(result[idx+9:])
}

type taskPlanEntry struct {
	ID          string   `yaml:"id"`
	Role        string   `yaml:"role"`
	Description string   `yaml:"description"`
	DependsOn   []string `yaml:"depends_on"`
}

// taskPlanWrapper handles the case where the architect wraps the list in a "tasks:" key.
type taskPlanWrapper struct {
	Tasks []taskPlanEntry `yaml:"tasks"`
}

func (o *Orchestrator) expandTaskPlan() error {
	raw, err := artifact.Read("contracts", "task-plan.yaml")
	if err != nil {
		return fmt.Errorf("read task plan: %w", err)
	}

	entries, err := parseTaskPlan(raw)
	if err != nil {
		return fmt.Errorf("parse task plan: %w", err)
	}

	roleMap := map[string]model.AgentRole{
		"backend":    model.RoleBackend,
		"frontend":   model.RoleFrontend,
		"database":   model.RoleDatabase,
		"reviewer":   model.RoleReviewer,
		"integrator": model.RoleIntegrator,
		"migrator":   model.RoleMigrator,
	}

	for _, e := range entries {
		role, ok := roleMap[strings.ToLower(e.Role)]
		if !ok {
			log.Printf("[orchestrator] skipping unknown role %q in task plan", e.Role)
			continue
		}

		deps := e.DependsOn
		if len(deps) == 0 {
			deps = []string{"architect-design"}
		}

		task := &model.Task{
			ID:           e.ID,
			Role:         role,
			Description:  e.Description,
			DependsOn:    deps,
			ArtifactDirs: artifactDirsForRole(role),
			OutputDir:    outputDirForRole(role),
		}
		if err := o.graph.AddTask(task); err != nil {
			return fmt.Errorf("add task %s: %w", e.ID, err)
		}

		// Auto-wire a reviewer for code-producing tasks
		if !reviewableRoles[role] {
			continue
		}
		reviewID := "review-" + e.ID
		reviewTask := &model.Task{
			ID:           reviewID,
			Role:         model.RoleReviewer,
			Description:  fmt.Sprintf("Review code produced by task %s", e.ID),
			DependsOn:    []string{e.ID},
			ArtifactDirs: append([]string{outputDirForRole(role)}, "contracts"),
			OutputDir:    "reviews",
			ReviewTaskID: e.ID,
		}
		if err := o.graph.AddTask(reviewTask); err != nil {
			return fmt.Errorf("add review task %s: %w", reviewID, err)
		}
		task.ReviewTaskID = reviewID
	}

	return nil
}

// parseTaskPlan handles both formats: bare list and {tasks: [...]}.
func parseTaskPlan(raw string) ([]taskPlanEntry, error) {
	var entries []taskPlanEntry
	if err := yaml.Unmarshal([]byte(raw), &entries); err == nil && len(entries) > 0 {
		return entries, nil
	}

	var wrapped taskPlanWrapper
	if err := yaml.Unmarshal([]byte(raw), &wrapped); err != nil {
		return nil, err
	}
	if len(wrapped.Tasks) == 0 {
		return nil, fmt.Errorf("task plan is empty")
	}
	return wrapped.Tasks, nil
}

func artifactDirsForRole(role model.AgentRole) []string {
	m := map[model.AgentRole][]string{
		model.RoleBackend:    {"contracts"},
		model.RoleFrontend:   {"contracts"},
		model.RoleDatabase:   {"contracts"},
		model.RoleReviewer:   {"contracts", "code/backend"},
		model.RoleIntegrator: {"contracts", "code/backend", "code/frontend"},
		model.RoleMigrator:   {"contracts"},
	}
	return m[role]
}

func outputDirForRole(role model.AgentRole) string {
	m := map[model.AgentRole]string{
		model.RoleBackend:    "code/backend",
		model.RoleFrontend:   "code/frontend",
		model.RoleDatabase:   "schemas",
		model.RoleReviewer:   "reviews",
		model.RoleIntegrator: "code/integrated",
		model.RoleMigrator:   "code/migrated",
	}
	return m[role]
}

// Graph returns the internal task graph for external inspection.
func (o *Orchestrator) Graph() *Graph {
	return o.graph
}
