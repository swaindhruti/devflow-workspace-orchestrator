package app

import (
	"fmt"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/runner"
)

// RunCommand executes one of a project's saved commands as a tracked
// process, without the caller needing to know whether the command came
// from the project's plain RunCommands or its Docker-specific
// DockerCommands, or how each kind is actually run: plain commands go
// straight through the shared runner.Runner, Docker commands go through
// docker.Service.Execute (which itself runs through the same runner, just
// with the project-directory working-directory fallback docker commands
// need — see docker.Service.Execute).
//
// Parameters:
//   - projectID: the project the command belongs to.
//   - commandID: the Command.ID to run. RunCommands is searched first,
//     then DockerCommands.
//
// Returns the started runner.Process, which the caller can hold onto and
// poll for status and captured output, or an error if the project or
// command could not be found, or the command could not be started.
func (a *App) RunCommand(projectID, commandID string) (*runner.Process, error) {
	p, err := a.projects.GetProjectByID(projectID)
	if err != nil {
		return nil, err
	}

	for _, cmd := range p.RunCommands {
		if cmd.ID == commandID {
			dir := cmd.Path
			if dir == "" {
				dir = p.Path
			}
			return a.runner.Start(cmd.Command, dir)
		}
	}

	for _, cmd := range p.DockerCommands {
		if cmd.ID == commandID {
			return a.dockerSvc.Execute(cmd, p.Path)
		}
	}

	return nil, fmt.Errorf("command not found: %s", commandID)
}
