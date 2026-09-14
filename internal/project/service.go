package project

import (
	"crypto/rand"
	"fmt"
)

// Service is the project domain's business-logic layer. It sits between
// callers (currently cmd/devflow, later the application/UI layer) and a
// Repository, adding the rules that must hold regardless of which storage
// backend is in use: every project gets a unique, service-assigned ID, and
// every write is validated before it reaches storage.
type Service struct {
	repo Repository
}

// NewProjectService constructs a Service backed by the given Repository.
//
// Parameters:
//   - repo: the storage backend to delegate persistence to (e.g. a
//     *JSONFileStorage).
func NewProjectService(repo Repository) *Service {
	return &Service{repo: repo}
}

// generateID produces a random 32-character hex string suitable for use as
// a Project.ID. It returns an empty string if the system's random source
// could not be read, which callers should treat as a failure to generate an
// ID rather than a valid empty ID.
func generateID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", b)
}

// GetAllProjects returns every registered project.
//
// Returns an error if the underlying Repository could not be read.
func (s *Service) GetAllProjects() ([]Project, error) {
	return s.repo.GetAllProjects()
}

// GetProjectByID returns the project matching id.
//
// Parameters:
//   - id: the Project.ID to look up.
//
// Returns an error if no such project exists or the underlying Repository
// could not be read.
func (s *Service) GetProjectByID(id string) (*Project, error) {
	return s.repo.GetProjectByID(id)
}

// AddProject validates and registers a new project.
//
// Parameters:
//   - project: the project to register. Its ID field is overwritten with a
//     freshly generated value regardless of what the caller set, so
//     callers should never pre-assign an ID.
//
// Returns an error if the project fails ValidateProject (e.g. missing name
// or path), or if the underlying Repository could not be written.
func (s *Service) AddProject(project *Project) error {
	project.ID = generateID()
	err := ValidateProject(project)
	if err != nil {
		return err
	}
	return s.repo.AddProject(project)
}

// UpdateProject validates and persists changes to an existing project.
//
// Parameters:
//   - project: the new state for the project, matched by its existing ID.
//
// Returns an error if the project fails ValidateProject, if no project
// with that ID exists, or if the underlying Repository could not be
// written.
func (s *Service) UpdateProject(project *Project) error {
	err := ValidateProject(project)
	if err != nil {
		return err
	}
	return s.repo.UpdateProject(project)
}

// DeleteProject removes the project matching id from the registry.
//
// Parameters:
//   - id: the Project.ID to remove.
//
// Returns an error if no project with that ID exists, or if the underlying
// Repository could not be written.
func (s *Service) DeleteProject(id string) error {
	return s.repo.DeleteProject(id)
}

// MarkProjectFavorite marks the project matching id as a favorite.
//
// Parameters:
//   - id: the Project.ID to mark as a favorite.
//
// Returns an error if no project with that ID exists, or if the underlying
// Repository could not be written.
func (s *Service) MarkProjectFavorite(id string) error {
	return s.repo.MarkProjectFavorite(id)
}

// UnmarkProjectFavorite removes the favorite marking from the project
// matching id.
//
// Parameters:
//   - id: the Project.ID to unmark as a favorite.
//
// Returns an error if no project with that ID exists, or if the underlying
// Repository could not be written.
func (s *Service) UnmarkProjectFavorite(id string) error {
	return s.repo.UnmarkProjectFavorite(id)
}
