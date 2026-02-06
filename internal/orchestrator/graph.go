package orchestrator

import (
	"fmt"
	"sync"

	"github.com/hubenschmidt/claude-dag/internal/model"
)

type Graph struct {
	mu    sync.RWMutex
	tasks map[string]*model.Task
	order []string // insertion order
}

func NewGraph() *Graph {
	return &Graph{
		tasks: make(map[string]*model.Task),
	}
}

func (g *Graph) AddTask(t *model.Task) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.tasks[t.ID]; exists {
		return fmt.Errorf("task %s already exists", t.ID)
	}

	for _, dep := range t.DependsOn {
		if _, exists := g.tasks[dep]; !exists {
			return fmt.Errorf("dependency %s not found for task %s", dep, t.ID)
		}
	}

	t.Status = model.StatusPending
	g.tasks[t.ID] = t
	g.order = append(g.order, t.ID)
	return nil
}

func (g *Graph) ReadyTasks() []*model.Task {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var ready []*model.Task
	for _, id := range g.order {
		t := g.tasks[id]
		if t.Status != model.StatusPending {
			continue
		}
		if g.depsResolved(t) {
			ready = append(ready, t)
		}
	}
	return ready
}

func (g *Graph) depsResolved(t *model.Task) bool {
	for _, dep := range t.DependsOn {
		if g.tasks[dep].Status != model.StatusCompleted {
			return false
		}
	}
	return true
}

func (g *Graph) SetStatus(id string, status model.TaskStatus) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	t, ok := g.tasks[id]
	if !ok {
		return fmt.Errorf("task %s not found", id)
	}
	t.Status = status
	return nil
}

func (g *Graph) SetResult(id, result string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	t, ok := g.tasks[id]
	if !ok {
		return fmt.Errorf("task %s not found", id)
	}
	t.Result = result
	return nil
}

func (g *Graph) SetError(id, errMsg string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	t, ok := g.tasks[id]
	if !ok {
		return fmt.Errorf("task %s not found", id)
	}
	t.Error = errMsg
	return nil
}

func (g *Graph) Get(id string) (*model.Task, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	t, ok := g.tasks[id]
	return t, ok
}

func (g *Graph) AllCompleted() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, t := range g.tasks {
		if t.Status != model.StatusCompleted {
			return false
		}
	}
	return true
}

func (g *Graph) HasFailed() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, t := range g.tasks {
		if t.Status == model.StatusFailed {
			return true
		}
	}
	return false
}

func (g *Graph) Tasks() []*model.Task {
	g.mu.RLock()
	defer g.mu.RUnlock()

	tasks := make([]*model.Task, 0, len(g.order))
	for _, id := range g.order {
		tasks = append(tasks, g.tasks[id])
	}
	return tasks
}

// RejectTask marks a task as rejected with feedback. If under max attempts,
// resets to pending so it re-enters the dispatch queue.
func (g *Graph) RejectTask(id, feedback string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	t, ok := g.tasks[id]
	if !ok {
		return fmt.Errorf("task %s not found", id)
	}

	t.Attempts++
	t.Feedback = feedback

	if t.Attempts >= model.MaxAttempts {
		t.Status = model.StatusFailed
		t.Error = fmt.Sprintf("rejected %d times, giving up", t.Attempts)
		return nil
	}

	t.Status = model.StatusPending
	return nil
}

// SetPaneID stores the tmux pane ID on a task.
func (g *Graph) SetPaneID(id, paneID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	t, ok := g.tasks[id]
	if !ok {
		return fmt.Errorf("task %s not found", id)
	}
	t.PaneID = paneID
	return nil
}

// RunningCount returns the number of tasks currently in running status.
func (g *Graph) RunningCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	count := 0
	for _, t := range g.tasks {
		if t.Status == model.StatusRunning {
			count++
		}
	}
	return count
}

// SetFeedback stores review feedback on a task.
func (g *Graph) SetFeedback(id, feedback string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	t, ok := g.tasks[id]
	if !ok {
		return fmt.Errorf("task %s not found", id)
	}
	t.Feedback = feedback
	return nil
}
