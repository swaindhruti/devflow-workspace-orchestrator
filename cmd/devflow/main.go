package main

import (
	"fmt"
	"log"
	"os"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/app"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/config"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/project"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/shellexec"
)

// shortID truncates id to its first 8 characters for compact console
// output, or returns it unchanged if it is already that short or
// shorter.
func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// main is a manual smoke test for the app layer, exercised end to end
// until internal/ui exists to replace it: it registers the current
// working directory as a project (exercising App.AddProject's Docker
// auto-detection), lists the registry, gathers cross-domain project
// context (Git status plus Docker containers/images via
// App.ProjectContext), and runs a saved command through App.RunCommand.
func main() {
	cfg, err := config.Load("devflow.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	a, err := app.NewApp(cfg, shellexec.DefaultExecutor{})
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to determine working directory: %v", err)
	}

	proj, err := a.AddProject("DevFlow", wd)
	if err != nil {
		log.Fatalf("failed to add project: %v", err)
	}
	fmt.Printf("Project added: %s (%s)\n", proj.Name, shortID(proj.ID))
	fmt.Printf("  Docker detected: %v\n", proj.HasDocker)

	allProjects, err := a.Projects().GetAllProjects()
	if err != nil {
		log.Fatalf("failed to list projects: %v", err)
	}
	fmt.Println("\nAll Projects:")
	for _, p := range allProjects {
		fmt.Printf("  [%s] %s → %s\n", shortID(p.ID), p.Name, p.Path)
	}

	fmt.Println("\nGathering project context (git status, docker)...")
	ctx, err := a.ProjectContext(proj.ID)
	if err != nil {
		log.Fatalf("failed to gather project context: %v", err)
	}
	if ctx.GitStatus != nil {
		fmt.Printf("  Git branch: %s (clean: %v)\n", ctx.GitStatus.Branch, ctx.GitStatus.Clean)
	} else {
		fmt.Println("  Git status: unavailable (not a git repository, or git is not installed)")
	}
	if proj.HasDocker {
		fmt.Printf("  Docker containers: %d, images: %d\n", len(ctx.Containers), len(ctx.Images))
	}

	fmt.Println("\nRunning a demo command through App.RunCommand...")
	demoCmd := project.Command{ID: "demo", Name: "Demo", Command: "echo 'DevFlow runner works!'"}
	proj.AddCommand(demoCmd)
	if err := a.Projects().UpdateProject(proj); err != nil {
		log.Fatalf("failed to save demo command: %v", err)
	}

	proc, err := a.RunCommand(proj.ID, demoCmd.ID)
	if err != nil {
		log.Fatalf("failed to run demo command: %v", err)
	}
	fmt.Printf("  Started: %s (PID: %d)\n", shortID(proc.ID), proc.PID)
}
