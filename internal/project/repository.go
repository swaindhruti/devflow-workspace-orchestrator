package project

// Repository defines the persistence operations the project domain needs,
// independent of how or where projects are actually stored. Service depends
// only on this interface, not on any concrete storage implementation, so
// the storage backend (currently JSONFileStorage; potentially SQLite later,
// per the system design's persistence roadmap) can be swapped without
// changing any business logic in Service.
type Repository interface {
	// GetAllProjects returns every registered project, or an error if the
	// underlying storage could not be read.
	GetAllProjects() ([]Project, error)

	// GetProjectByID returns the project matching id, or an error if no
	// such project exists or the underlying storage could not be read.
	//
	// Parameters:
	//   - id: the Project.ID to look up.
	GetProjectByID(id string) (*Project, error)

	// AddProject persists a new project record.
	//
	// Parameters:
	//   - project: the project to store. Callers are expected to have
	//     already assigned a unique ID (Service.AddProject does this).
	//
	// Returns an error if the underlying storage could not be written.
	AddProject(project *Project) error

	// UpdateProject overwrites the stored record whose ID matches
	// updatedProject.ID with the given values.
	//
	// Parameters:
	//   - updatedProject: the new state for the project, matched by ID.
	//
	// Returns an error if no project with that ID exists, or if the
	// underlying storage could not be written.
	UpdateProject(updatedProject *Project) error

	// DeleteProject removes the project matching id from storage.
	//
	// Parameters:
	//   - id: the Project.ID to remove.
	//
	// Returns an error if no project with that ID exists, or if the
	// underlying storage could not be written.
	DeleteProject(id string) error

	// MarkProjectFavorite sets IsFavorite to true for the project
	// matching id.
	//
	// Parameters:
	//   - id: the Project.ID to mark as a favorite.
	//
	// Returns an error if no project with that ID exists, or if the
	// underlying storage could not be written.
	MarkProjectFavorite(id string) error

	// UnmarkProjectFavorite sets IsFavorite to false for the project
	// matching id.
	//
	// Parameters:
	//   - id: the Project.ID to unmark as a favorite.
	//
	// Returns an error if no project with that ID exists, or if the
	// underlying storage could not be written.
	UnmarkProjectFavorite(id string) error
}
