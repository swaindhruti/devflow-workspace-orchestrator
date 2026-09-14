package app

import (
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/docker"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/git"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/project"
)

// ProjectContext aggregates everything the dashboard needs to display
// about one project in a single read: the stored project record, its live
// Git status, and (when Docker is enabled) its containers and images.
// Gathering this spans project, git, and docker — exactly the kind of
// cross-domain read no single domain package should perform itself.
type ProjectContext struct {
	// Project is the stored project record.
	Project project.Project
	// GitStatus is the project's live Git status, or nil if its
	// directory is not a Git repository (or status could not be read
	// for any other reason).
	GitStatus *git.Status
	// Containers is the project's Docker containers, or nil if the
	// project has no Docker setup (Project.HasDocker is false) or they
	// could not be listed.
	Containers []docker.Container
	// Images is the project's Docker images, under the same nil
	// conditions as Containers.
	Images []docker.Image
}

// ProjectContext gathers display-ready context for one project: its
// stored record, live Git status, and, if it has Docker enabled, its
// containers and images.
//
// Parameters:
//   - id: the Project.ID to gather context for.
//
// Returns an error only if the project record itself could not be loaded.
// Every other piece is best-effort: a project whose directory is not a
// Git repository simply gets a nil GitStatus, and a Docker inspection
// failure (e.g. the docker CLI is not installed, as in this project's own
// dev environment) simply leaves Containers/Images nil, rather than
// failing the whole read over one missing or misbehaving piece of
// optional context.
func (a *App) ProjectContext(id string) (ProjectContext, error) {
	p, err := a.projects.GetProjectByID(id)
	if err != nil {
		return ProjectContext{}, err
	}

	ctx := ProjectContext{Project: *p}

	if status, err := a.gitSvc.Status(p.Path); err == nil {
		ctx.GitStatus = &status
	}

	if p.HasDocker {
		composeScoped := p.DockerComposeFile != ""

		if containers, err := docker.ListContainers(a.exec, p.DockerIdentifier, composeScoped); err == nil {
			ctx.Containers = containers
		}
		if images, err := docker.ListImages(a.exec, p.DockerIdentifier, composeScoped); err == nil {
			ctx.Images = images
		}
	}

	return ctx, nil
}
