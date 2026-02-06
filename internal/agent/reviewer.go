package agent

import (
	"fmt"
	"strings"

	"github.com/hubenschmidt/claude-dag/internal/artifact"
	"github.com/hubenschmidt/claude-dag/internal/model"
)

type Reviewer struct{}

func NewReviewer() *Reviewer { return &Reviewer{} }

func (r *Reviewer) Role() model.AgentRole { return model.RoleReviewer }

func (r *Reviewer) Launch(session string, task *model.Task) (string, error) {
	system, err := LoadPrompt("reviewer")
	if err != nil {
		return "", err
	}

	codeCtx, err := buildReviewContext(task.ArtifactDirs)
	if err != nil {
		return "", fmt.Errorf("build review context: %w", err)
	}

	prompt := fmt.Sprintf(`Review this code and write your verdict to artifacts/reviews/%s.md ONLY. Do NOT modify any file outside artifacts/.
When completely finished, run: touch artifacts/reviews/.done.%s
Then STOP.

%s`, task.ID, task.ID, codeCtx)

	return launchInteractive(session, task.ID, system, prompt)
}

func buildReviewContext(dirs []string) (string, error) {
	var b strings.Builder
	for _, dir := range dirs {
		content, err := artifact.ReadDir(dir)
		if err != nil {
			return "", err
		}
		b.WriteString(content)
	}
	return b.String(), nil
}
