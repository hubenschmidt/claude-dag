package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// PaneInfo holds metadata about a tmux pane.
type PaneInfo struct {
	ID   string
	Dead bool
}

// PromptDir returns a stable temp directory for prompt files.
func PromptDir() string {
	dir := filepath.Join(os.TempDir(), "claude-dag")
	os.MkdirAll(dir, 0o755)
	return dir
}

// CreateSession creates a detached tmux session with a default shell.
func CreateSession(name string) error {
	return run("new-session", "-d", "-s", name, "-x", "200", "-y", "50")
}

// CreateSessionWithCmd creates a detached tmux session whose pane 0 runs cmd.
func CreateSessionWithCmd(name, cmd string) error {
	return run("new-session", "-d", "-s", name, "-x", "200", "-y", "50", cmd)
}

// ConfigureSession sets session-level options.
// Dead panes auto-close so they don't clutter the layout.
func ConfigureSession(name string) error {
	_ = run("set-option", "-t", name, "remain-on-exit", "off")
	_ = run("set-option", "-t", name, "history-limit", "50000")
	return nil
}

// NewWindow creates a named tmux window (tab) in the session without stealing focus.
func NewWindow(session, name, cmd string) (string, error) {
	out, err := output("new-window", "-d", "-t", session, "-n", name, "-P", "-F", "#{pane_id}", cmd)
	if err != nil {
		return "", fmt.Errorf("new window: %w", err)
	}
	paneID := strings.TrimSpace(out)
	return paneID, nil
}

// WritePromptFile writes a prompt to a temp file and returns the path.
func WritePromptFile(taskID, promptText string) (string, error) {
	promptPath := filepath.Join(PromptDir(), taskID+".md")
	if err := os.WriteFile(promptPath, []byte(promptText), 0o644); err != nil {
		return "", fmt.Errorf("write prompt file: %w", err)
	}
	return promptPath, nil
}

// NewAutoWindow creates a named tmux window running cmd, waits for the TUI
// to initialize, then types the message and presses Enter separately to
// ensure the TUI processes them correctly.
func NewAutoWindow(session, name, cmd, initialMsg string) (string, error) {
	paneID, err := NewWindow(session, name, cmd)
	if err != nil {
		return "", err
	}

	// Wait for Claude Code TUI to fully initialize
	time.Sleep(5 * time.Second)

	// Send text literally (without key-name interpretation)
	if err := run("send-keys", "-t", paneID, "-l", initialMsg); err != nil {
		return "", fmt.Errorf("send text: %w", err)
	}

	// Brief pause so TUI processes the text before Enter
	time.Sleep(500 * time.Millisecond)

	// Submit
	if err := run("send-keys", "-t", paneID, "Enter"); err != nil {
		return "", fmt.Errorf("send enter: %w", err)
	}

	return paneID, nil
}

// IsPaneAlive returns true if the pane's process is still running.
func IsPaneAlive(paneID string) bool {
	out, err := output("list-panes", "-a", "-F", "#{pane_id} #{pane_dead}", "-f", fmt.Sprintf("#{==:#{pane_id},%s}", paneID))
	if err != nil {
		return false
	}
	trimmed := strings.TrimSpace(out)
	if trimmed == "" {
		return false
	}
	return !strings.HasSuffix(trimmed, "1")
}

// KillSession destroys the entire tmux session.
func KillSession(name string) error {
	return run("kill-session", "-t", name)
}

// ListPanes returns info about all panes in a session.
func ListPanes(session string) ([]PaneInfo, error) {
	out, err := output("list-panes", "-t", session, "-F", "#{pane_id} #{pane_dead}")
	if err != nil {
		return nil, fmt.Errorf("list panes: %w", err)
	}

	var panes []PaneInfo
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		panes = append(panes, PaneInfo{
			ID:   parts[0],
			Dead: parts[1] == "1",
		})
	}
	return panes, nil
}

// SendKeys sends keystrokes to a pane.
func SendKeys(paneID, keys string) error {
	return run("send-keys", "-t", paneID, keys, "Enter")
}

func run(args ...string) error {
	cmd := exec.Command("tmux", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux %s: %w: %s", args[0], err, string(out))
	}
	return nil
}

func output(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("tmux %s: %w", args[0], err)
	}
	return string(out), nil
}
