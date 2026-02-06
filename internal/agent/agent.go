package agent

import (
	"github.com/hubenschmidt/cathedral-swarm/internal/model"
)

// Agent launches a Claude Code session in a tmux pane for a given task.
type Agent interface {
	Role() model.AgentRole
	Launch(session string, task *model.Task) (paneID string, err error)
}
