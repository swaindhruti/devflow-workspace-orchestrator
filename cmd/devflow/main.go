package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/config"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/projects"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/runner"
)

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func main() {
	cfg, err := config.Load("devflow.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	storage := &projects.JSONFileStorage{
		FilePath: cfg.Storage.Path,
	}

	os.MkdirAll(filepath.Dir(cfg.Storage.Path), 0755)

	projectSvc := projects.NewProjectService(storage)
	runnerSvc := runner.New(runner.Config{
		Shell:   cfg.Runner.Shell,
		Timeout: cfg.Runner.Timeout,
	})

	project := &projects.Project{
		Name: "DevFlow",
		Path: "/mnt/nvme/projects/devflow",
	}

	err = projectSvc.AddProject(project)
	if err != nil {
		log.Fatalf("failed to add project: %v", err)
	}
	fmt.Printf("Project added: %s (%s)\n", project.Name, shortID(project.ID))

	allProjects, err := projectSvc.GetAllProjects()
	if err != nil {
		log.Fatalf("failed to list projects: %v", err)
	}
	fmt.Println("\nAll Projects:")
	for _, p := range allProjects {
		fmt.Printf("  [%s] %s → %s\n", shortID(p.ID), p.Name, p.Path)
	}

	fmt.Println("\nRunning a demo command...")
	proc, err := runnerSvc.Start("echo 'DevFlow runner works!'", "")
	if err != nil {
		log.Fatalf("failed to start command: %v", err)
	}
	fmt.Printf("  Started: %s (PID: %d)\n", shortID(proc.ID), proc.PID)
}
