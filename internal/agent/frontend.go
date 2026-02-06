package agent

import (
	"fmt"

	"github.com/hubenschmidt/claude-dag/internal/artifact"
	"github.com/hubenschmidt/claude-dag/internal/model"
)

type Frontend struct{}

func NewFrontend() *Frontend { return &Frontend{} }

func (f *Frontend) Role() model.AgentRole { return model.RoleFrontend }

func (f *Frontend) Launch(session string, task *model.Task) (string, error) {
	system, err := LoadPrompt("frontend")
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
Write your own key decisions (component interfaces, API client shapes) to artifacts/shared-context/.

Write all code files to artifacts/code/frontend/ directory ONLY. Do NOT modify any file outside artifacts/.
When completely finished, run: touch artifacts/code/frontend/.done.%s
Then STOP.`, task.Description, contractCtx, task.ID)

	if task.Feedback != "" {
		prompt += fmt.Sprintf("\n\n--- PREVIOUS ATTEMPT WAS REJECTED ---\nAttempt %d/%d. Reviewer feedback:\n%s\n\nFix the issues listed above.", task.Attempts+1, model.MaxAttempts, task.Feedback)
	}

	return launchInteractive(session, task.ID, system, prompt)
}
