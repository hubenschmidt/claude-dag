package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hubenschmidt/cathedral-swarm/internal/tmux"
)

const defaultPromptDir = "prompts"

// LoadPrompt reads a prompt markdown file from the prompts directory.
func LoadPrompt(name string) (string, error) {
	path := filepath.Join(defaultPromptDir, name+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("load prompt %s: %w", name, err)
	}
	return string(data), nil
}

// launchInteractive writes the prompt to a temp file, launches claude in
// interactive mode in a named tmux window, then sends a short "read that
// file" instruction via SendKeys to auto-submit while keeping the full TUI.
func launchInteractive(session, taskID, systemPrompt, promptText string) (string, error) {
	promptPath, err := tmux.WritePromptFile(taskID, promptText)
	if err != nil {
		return "", err
	}

	escaped := shellEscape(systemPrompt)
	cmd := fmt.Sprintf("claude --append-system-prompt %s --allowedTools Edit Read Write Bash Glob Grep", escaped)
	initialMsg := fmt.Sprintf("Read and follow all instructions in %s", promptPath)

	return tmux.NewAutoWindow(session, taskID, cmd, initialMsg)
}

// shellEscape wraps a string in single quotes, escaping internal single quotes.
func shellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
