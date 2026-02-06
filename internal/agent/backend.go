package agent

import (
	"fmt"

	"github.com/hubenschmidt/claude-dag/internal/artifact"
	"github.com/hubenschmidt/claude-dag/internal/model"
)

type Backend struct{}

func NewBackend() *Backend { return &Backend{} }

func (b *Backend) Role() model.AgentRole { return model.RoleBackend }

func (b *Backend) Launch(session string, task *model.Task) (string, error) {
	system, err := LoadPrompt("backend")
	if err != nil {
		return "", err
	}

	contractCtx, err := artifact.ReadDir("contracts")
	if err != nil {
		return "", fmt.Errorf("read contracts: %w", err)
	}

	prompt := fmt.Sprintf(`Task: %s

Architect artifacts:
%s

Before making interface decisions, check artifacts/shared-context/ for decisions from other agents.
Write your own key decisions (endpoint signatures, data shapes) to artifacts/shared-context/.

Write all code files to artifacts/code/backend/ directory ONLY. Do NOT modify go.mod, go.sum, or any file outside artifacts/.
When completely finished, run: touch artifacts/code/backend/.done.%s
Then STOP.`, task.Description, contractCtx, task.ID)

	if task.Feedback != "" {
		prompt += fmt.Sprintf("\n\n--- PREVIOUS ATTEMPT WAS REJECTED ---\nAttempt %d/%d. Reviewer feedback:\n%s\n\nFix the issues listed above.", task.Attempts+1, model.MaxAttempts, task.Feedback)
	}

	return launchInteractive(session, task.ID, system, prompt)
}
