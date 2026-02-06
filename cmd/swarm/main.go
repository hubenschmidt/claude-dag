package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"

	"github.com/hubenschmidt/cathedral-swarm/internal/agent"
	"github.com/hubenschmidt/cathedral-swarm/internal/orchestrator"
	"github.com/hubenschmidt/cathedral-swarm/internal/tmux"
)

const sessionName = "cathedral-swarm"

func main() {
	goal := strings.Join(os.Args[1:], " ")
	if goal == "" {
		fmt.Fprintln(os.Stderr, "usage: swarm <goal description>")
		fmt.Fprintln(os.Stderr, "  example: swarm Build a todo app with user accounts")
		os.Exit(1)
	}

	preflight()

	// If SWARM_INSIDE=1, we're already in the tmux session — run the orchestrator.
	// Otherwise, create the tmux session and re-exec ourselves inside it.
	if os.Getenv("SWARM_INSIDE") != "1" {
		launchInTmux(goal)
		return
	}

	runOrchestrator(goal)
}

// launchInTmux creates a tmux session running this binary as pane 0, then attaches.
func launchInTmux(goal string) {
	_ = tmux.KillSession(sessionName)

	bin, err := os.Executable()
	if err != nil {
		log.Fatalf("resolve executable: %v", err)
	}

	// Create session with the orchestrator as pane 0's command
	innerCmd := fmt.Sprintf("SWARM_INSIDE=1 %s %s", shellEscape(bin), shellEscape(goal))
	if err := tmux.CreateSessionWithCmd(sessionName, innerCmd); err != nil {
		log.Fatalf("create tmux session: %v", err)
	}

	// Keep dead panes visible so user can read agent output
	_ = tmux.ConfigureSession(sessionName)

	// Attach — this replaces our process with tmux attach
	attach := exec.Command("tmux", "attach", "-t", sessionName)
	attach.Stdin = os.Stdin
	attach.Stdout = os.Stdout
	attach.Stderr = os.Stderr
	if err := attach.Run(); err != nil {
		log.Fatalf("tmux attach: %v", err)
	}
}

func runOrchestrator(goal string) {
	agents := []agent.Agent{
		agent.NewArchitect(),
		agent.NewBackend(),
		agent.NewFrontend(),
		agent.NewReviewer(),
	}

	orch := orchestrator.New(sessionName, agents)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		log.Println("interrupted, killing tmux session...")
		_ = tmux.KillSession(sessionName)
		os.Exit(1)
	}()

	start := time.Now()
	log.Println("=== Cathedral Swarm ===")
	log.Printf("Goal: %s", goal)

	err := orch.Run(ctx, goal)
	elapsed := time.Since(start)

	if err != nil {
		log.Printf("swarm failed after %s: %v", elapsed, err)
		printSummary(orch.Graph())
		fmt.Println("\nPress Enter to exit...")
		fmt.Scanln()
		os.Exit(1)
	}

	log.Printf("swarm completed in %s", elapsed)
	printSummary(orch.Graph())
	fmt.Println("\nPress Enter to exit...")
	fmt.Scanln()
}

func preflight() {
	missing := []string{}
	for _, bin := range []string{"tmux", "claude"} {
		if _, err := exec.LookPath(bin); err != nil {
			missing = append(missing, bin)
		}
	}
	if len(missing) == 0 {
		return
	}
	fmt.Fprintf(os.Stderr, "required commands not found: %s\n", strings.Join(missing, ", "))
	fmt.Fprintln(os.Stderr, "install with: sudo apt-get install tmux  (or brew install tmux)")
	fmt.Fprintln(os.Stderr, "claude: https://docs.anthropic.com/en/docs/claude-code")
	os.Exit(1)
}

func printSummary(g *orchestrator.Graph) {
	fmt.Println("\n=== Task Summary ===")
	for _, t := range g.Tasks() {
		pane := ""
		if t.PaneID != "" {
			pane = fmt.Sprintf(" [pane %s]", t.PaneID)
		}
		fmt.Printf("  [%s] %s (%s)%s: %s\n", t.Status, t.ID, t.Role, pane, t.Description)
	}
	fmt.Println("\nArtifacts written to ./artifacts/")
}

func shellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
