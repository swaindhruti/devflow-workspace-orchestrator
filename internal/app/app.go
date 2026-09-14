// Package app is DevFlow's composition root: it wires the project,
// docker, git, and tmux domain packages together behind a single App
// value, and hosts the cross-domain orchestration none of those packages
// should perform on their own (e.g. "register a project, then detect its
// Docker setup, then persist both" spans project and docker; "gather a
// project's live Git status and Docker containers in one read" spans
// project, git, and docker). Domain packages stay independent and
// decoupled from each other on purpose — see docs/architecture.md — and
// App is the one place allowed to know about all of them at once.
//
// App holds no business rules of its own beyond that orchestration. The
// eventual Bubble Tea UI (internal/ui) is expected to depend on App
// instead of reaching into individual domain packages directly, the same
// way cmd/devflow's entrypoint does from this phase onward.
package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/config"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/docker"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/git"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/project"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/runner"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/shellexec"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/tmux"
)

// App is the wired-together set of domain services DevFlow operates
// against. Construct one with NewApp; do not build it by hand, since its
// fields are unexported to keep callers going through App's own methods
// (and the domain-service accessors below) rather than reaching past it.
type App struct {
	projects  *project.Service
	runner    *runner.Runner
	dockerSvc *docker.Service
	gitSvc    *git.Service
	tmuxSvc   *tmux.Service
	exec      shellexec.Executor
}

// NewApp constructs an App from configuration, wiring every domain
// service to its shared infrastructure: a JSON-backed project registry at
// cfg.Storage.Path, a command runner configured from cfg.Runner, and
// git/tmux/docker CLI access through exec.
//
// Parameters:
//   - cfg: the loaded configuration (see config.Load) controlling storage
//     location and command-runner settings.
//   - exec: the Executor used for every git, tmux, and Docker-inspection
//     CLI call. Pass shellexec.DefaultExecutor{} in production; tests can
//     pass a shellexec.FakeExecutor to exercise App without git, tmux, or
//     docker actually being installed.
//
// Returns the constructed App, or an error if the storage directory
// (cfg.Storage.Path's parent) could not be created.
func NewApp(cfg *config.Config, exec shellexec.Executor) (*App, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.Storage.Path), 0755); err != nil {
		return nil, fmt.Errorf("failed to prepare storage directory: %w", err)
	}

	storage := &project.JSONFileStorage{FilePath: cfg.Storage.Path}
	r := runner.New(runner.Config{Shell: cfg.Runner.Shell, Timeout: cfg.Runner.Timeout})

	return &App{
		projects:  project.NewProjectService(storage),
		runner:    r,
		dockerSvc: docker.NewService(r),
		gitSvc:    git.NewService(exec),
		tmuxSvc:   tmux.NewService(exec),
		exec:      exec,
	}, nil
}

// Projects returns the underlying project registry service, for callers
// that need project CRUD operations App does not itself wrap in
// cross-domain orchestration (e.g. deleting a project, toggling its
// favorite status).
func (a *App) Projects() *project.Service {
	return a.projects
}

// Docker returns the underlying Docker command service, for callers that
// need to compute default commands or execute one directly without going
// through App.RunCommand.
func (a *App) Docker() *docker.Service {
	return a.dockerSvc
}

// Git returns the underlying Git service.
func (a *App) Git() *git.Service {
	return a.gitSvc
}

// Tmux returns the underlying tmux session service.
func (a *App) Tmux() *tmux.Service {
	return a.tmuxSvc
}
