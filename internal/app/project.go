package app

import (
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/docker"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/project"
)

// AddProject registers a new project and, if its directory has a
// Dockerfile or compose file, immediately populates its Docker detection
// fields and default commands — the choreography of "add, then detect,
// then persist the combined result" that project.Service and
// docker.DetectDockerSetup deliberately don't perform themselves, since
// each stays decoupled from the other's concern.
//
// Parameters:
//   - name: the new project's display name.
//   - path: the new project's local directory. Docker detection reads
//     this path; if it does not exist or cannot be read, detection is
//     silently skipped rather than failing registration (see below).
//
// Returns the newly registered project, or an error if it failed
// validation (see project.ValidateProject) or could not be persisted. A
// failure to detect Docker is not treated as an error: registration has
// already succeeded by that point, and Docker fields can always be filled
// in later via RefreshDockerInfo once the path is reachable.
func (a *App) AddProject(name, path string) (*project.Project, error) {
	p := &project.Project{Name: name, Path: path}
	if err := a.projects.AddProject(p); err != nil {
		return nil, err
	}

	if err := a.applyDockerDetection(p); err == nil {
		if err := a.projects.UpdateProject(p); err != nil {
			return nil, err
		}
	}

	return p, nil
}

// RefreshDockerInfo re-runs Docker detection for an already-registered
// project and persists the result. Unlike AddProject's best-effort
// detection, a detection failure here is returned to the caller, since
// refreshing was explicitly requested and silently doing nothing would be
// surprising.
//
// Parameters:
//   - id: the Project.ID to refresh.
//
// Returns the updated project, or an error if the project does not exist,
// detection failed (e.g. its directory does not exist or is unreadable),
// or the update could not be persisted.
func (a *App) RefreshDockerInfo(id string) (*project.Project, error) {
	p, err := a.projects.GetProjectByID(id)
	if err != nil {
		return nil, err
	}

	if err := a.applyDockerDetection(p); err != nil {
		return nil, err
	}

	if err := a.projects.UpdateProject(p); err != nil {
		return nil, err
	}

	return p, nil
}

// applyDockerDetection runs docker.DetectDockerSetup for p.Path and copies
// the result onto p's Docker fields, generating default commands when
// Docker is detected. It does not persist p; callers are responsible for
// that.
func (a *App) applyDockerDetection(p *project.Project) error {
	result, err := docker.DetectDockerSetup(p.Path)
	if err != nil {
		return err
	}

	p.HasDocker = result.HasDocker
	p.DockerComposeFile = result.ComposeFile
	if result.Identifier != "" {
		p.DockerIdentifier = result.Identifier
	}
	if result.HasDocker {
		p.DockerCommands = docker.DefaultCommands(result, p.DockerIdentifier)
	}

	return nil
}
