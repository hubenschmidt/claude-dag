package agent

import (
	"fmt"

	"github.com/hubenschmidt/claude-dag/internal/artifact"
	"github.com/hubenschmidt/claude-dag/internal/model"
)

type Architect struct{}

func NewArchitect() *Architect { return &Architect{} }

func (a *Architect) Role() model.AgentRole { return model.RoleArchitect }

func (a *Architect) Launch(session string, task *model.Task) (string, error) {
	system, err := LoadPrompt("architect")
	if err != nil {
		return "", err
	}

	// Validation mode: architect reviews completed sub-agent work
	if task.ID == "architect-validate" {
		return a.launchValidation(session, task, system)
	}

	prompt := fmt.Sprintf(`Design the architecture for: %s

Write exactly three files to artifacts/contracts/: api-contract.yaml, data-model.yaml, task-plan.yaml.
Do NOT write code or files anywhere else — only artifacts/contracts/.
After writing all three files, run: touch artifacts/contracts/.done.%s
Then STOP. Do not implement anything. Your job ends at design.`, task.Description, task.ID)

	return launchInteractive(session, task.ID, system, prompt)
}

func (a *Architect) launchValidation(session string, task *model.Task, system string) (string, error) {
	contractCtx, err := artifact.ReadDir("contracts")
	if err != nil {
		return "", fmt.Errorf("read contracts: %w", err)
	}

	codeCtx, err := readCodeArtifacts()
	if err != nil {
		return "", fmt.Errorf("read code artifacts: %w", err)
	}

	prompt := fmt.Sprintf(`You are validating that all sub-agent implementations honor the original contracts.

=== Original Contracts ===
%s

=== Implemented Code ===
%s

Check for:
1. API contract mismatches (endpoints, request/response shapes)
2. Data model inconsistencies between backend and frontend
3. Missing or incompatible interfaces between components

If everything is coherent, write "APPROVED" to artifacts/reviews/architect-validate.md
If there are issues, write "REJECTED:" followed by specific issues and which task(s) need rework to artifacts/reviews/architect-validate.md

After writing the review, also write a README.md to the project root (artifacts/README.md) with:
- A one-line project description
- Prerequisites (runtime, dependencies)
- How to install/build
- How to run (both backend and frontend if applicable)
- Example usage (curl commands, URLs to visit, etc.)

Keep it concise and practical — just enough for someone to clone and run.

After writing both files, run: touch artifacts/reviews/.done.%s
Then STOP.`, contractCtx, codeCtx, task.ID)

	return launchInteractive(session, task.ID, system, prompt)
}

func readCodeArtifacts() (string, error) {
	var result string
	for _, dir := range []string{"code/backend", "code/frontend", "schemas"} {
		content, err := artifact.ReadDir(dir)
		if err != nil {
			continue // dir may not exist if no agent produced it
		}
		result += content
	}
	return result, nil
}
