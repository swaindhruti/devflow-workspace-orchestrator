package docker

import (
	"crypto/rand"
	"fmt"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/project"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/runner"
)

// generateID produces a random 32-character hex string suitable for use as
// a project.Command.ID. Mirrors the identical helper in
// internal/project/service.go and internal/runner/runner.go: each package
// that mints its own IDs keeps this tiny helper local rather than sharing
// it through a generic utility package, per the domain-first layout's
// preference for self-contained packages over cross-cutting "utils".
func generateID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", b)
}

// DefaultCommands returns the default docker/compose commands a project
// gets once Docker is detected. Each returned command has a freshly
// generated ID; ProjectID is left blank for the caller to fill in once the
// project's own ID is known, the same contract project.Project.AddCommand
// already expects of its caller.
//
// Parameters:
//   - result: the DetectionResult from DetectDockerSetup for the project.
//   - identifier: the project's Docker identifier
//     (project.Project.DockerIdentifier), used to tag the image built for
//     a plain-Dockerfile project. Unused when result.ComposeFile is set,
//     since Compose derives its own image names.
//
// Returns nil if result.HasDocker is false. Otherwise returns compose
// commands (up/down/build/restart/logs) when result.ComposeFile is set, or
// Dockerfile-only commands (build/run) when it is not, since compose
// actions like "restart" and "logs" are meaningless without a compose
// file to name the services.
func DefaultCommands(result DetectionResult, identifier string) []project.Command {
	if !result.HasDocker {
		return nil
	}

	if result.ComposeFile != "" {
		return composeCommands(result.ComposeFile)
	}

	return dockerfileCommands(identifier)
}

// composeCommands builds the default command set for a project that has a
// detected compose file: start, stop, build, restart, and follow logs for
// every service the compose file defines.
func composeCommands(composeFile string) []project.Command {
	return []project.Command{
		newCommand("Up", fmt.Sprintf("docker compose -f %s up -d", composeFile),
			"Start all services in the background."),
		newCommand("Down", fmt.Sprintf("docker compose -f %s down", composeFile),
			"Stop and remove all services."),
		newCommand("Build", fmt.Sprintf("docker compose -f %s build", composeFile),
			"Build or rebuild service images."),
		newCommand("Restart", fmt.Sprintf("docker compose -f %s restart", composeFile),
			"Restart all running services."),
		newCommand("Logs", fmt.Sprintf("docker compose -f %s logs -f", composeFile),
			"Follow log output from all services."),
	}
}

// dockerfileCommands builds the default command set for a project that has
// a plain Dockerfile and no compose file: build the image and run a
// throwaway container from it, tagged with the project's Docker
// identifier since there is no compose-managed image name to rely on.
func dockerfileCommands(identifier string) []project.Command {
	return []project.Command{
		newCommand("Build", fmt.Sprintf("docker build -t %s .", identifier),
			"Build the image from the project's Dockerfile."),
		newCommand("Run", fmt.Sprintf("docker run --rm -it %s", identifier),
			"Run a container from the built image."),
	}
}

// newCommand constructs a project.Command with a freshly generated ID for
// use as one of DefaultCommands' entries.
func newCommand(name, command, description string) project.Command {
	return project.Command{
		ID:          generateID(),
		Name:        name,
		Command:     command,
		Description: description,
	}
}

// Service executes docker/compose commands for a project. It holds no
// storage of its own — detection results and saved commands live on
// project.Project, persisted through project.Service — Service's only job
// is running a command and handing back the tracked process.
type Service struct {
	runner *runner.Runner
}

// NewService constructs a Service that executes commands through r, the
// same shared runner.Runner used for a project's own RunCommands, so
// docker actions are tracked and streamed identically to any other
// project command rather than through separate process-management logic.
//
// Parameters:
//   - r: the runner.Runner to start docker/compose commands through.
func NewService(r *runner.Runner) *Service {
	return &Service{runner: r}
}

// Execute starts a saved docker command as a tracked, asynchronous
// process.
//
// Parameters:
//   - cmd: the command to run, typically one of a project's
//     DockerCommands.
//   - projectPath: the project's own directory. Used as the command's
//     working directory whenever cmd.Path is not set, since the compose
//     commands DefaultCommands generates pass a relative
//     "-f <composeFile>" argument that must be resolved from the project
//     directory.
//
// Returns the started runner.Process, which the caller can poll (Get) or
// list (List) for status and captured output, or an error if the command
// could not be started.
func (s *Service) Execute(cmd project.Command, projectPath string) (*runner.Process, error) {
	dir := cmd.Path
	if dir == "" {
		dir = projectPath
	}
	return s.runner.Start(cmd.Command, dir)
}
