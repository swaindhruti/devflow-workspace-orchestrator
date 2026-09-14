package main

import (
	"log"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/app"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/config"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/shellexec"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui"
)

// main is DevFlow's real entrypoint: it loads configuration, wires the
// application layer, and hands control to the Bubble Tea terminal
// interface (splash screen, then the project dashboard) until the user
// quits.
func main() {
	cfg, err := config.Load("devflow.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	a, err := app.NewApp(cfg, shellexec.DefaultExecutor{})
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}

	if err := ui.Run(a); err != nil {
		log.Fatalf("devflow exited with error: %v", err)
	}
}
