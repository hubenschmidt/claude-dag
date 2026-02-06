package orchestrator

import (
	"fmt"
	"log"
	"time"

	"github.com/hubenschmidt/cathedral-swarm/internal/agent"
	"github.com/hubenschmidt/cathedral-swarm/internal/model"
)

const (
	maxConcurrent = 4
	staggerDelay  = 3 * time.Second // delay between launches so each TUI can initialize
)

// Dispatcher maps agent roles to agent instances and launches them in tmux panes.
type Dispatcher struct {
	agents  map[model.AgentRole]agent.Agent
	session string
}

// NewDispatcher creates a dispatcher bound to the given tmux session.
func NewDispatcher(session string, agents []agent.Agent) *Dispatcher {
	m := make(map[model.AgentRole]agent.Agent, len(agents))
	for _, a := range agents {
		m[a.Role()] = a
	}
	return &Dispatcher{agents: m, session: session}
}

// LaunchReady finds pending-and-ready tasks, launching up to maxConcurrent
// total running tasks. Staggers launches so each Claude TUI has time to init.
func (d *Dispatcher) LaunchReady(g *Graph) error {
	slots := maxConcurrent - g.RunningCount()
	if slots <= 0 {
		return nil
	}

	ready := g.ReadyTasks()
	if len(ready) == 0 {
		return nil
	}

	if len(ready) > slots {
		ready = ready[:slots]
	}

	log.Printf("[dispatch] launching %d task(s) (%d slots available)", len(ready), slots)

	for i, task := range ready {
		// Stagger after the first launch so each TUI can start before the next
		if i > 0 {
			time.Sleep(staggerDelay)
		}
		if err := d.launchTask(g, task); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dispatcher) launchTask(g *Graph, task *model.Task) error {
	a, ok := d.agents[task.Role]
	if !ok {
		log.Printf("[dispatch] skipping task %s: no agent for role %s", task.ID, task.Role)
		_ = g.SetStatus(task.ID, model.StatusFailed)
		_ = g.SetError(task.ID, fmt.Sprintf("no agent for role %s", task.Role))
		return nil
	}

	paneID, err := a.Launch(d.session, task)
	if err != nil {
		log.Printf("[dispatch] failed to launch task %s: %v", task.ID, err)
		_ = g.SetStatus(task.ID, model.StatusFailed)
		_ = g.SetError(task.ID, err.Error())
		return nil
	}

	_ = g.SetStatus(task.ID, model.StatusRunning)
	_ = g.SetPaneID(task.ID, paneID)
	task.StartedAt = time.Now().Unix()

	log.Printf("[dispatch] -> %s (%s) in pane %s", task.ID, task.Role, paneID)
	return nil
}
