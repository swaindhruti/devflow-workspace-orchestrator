package project

// Methods for managing projects in the repository
type Repository interface {
	GetAllProjects() ([]Project, error)
	GetProjectByID(id string) (*Project, error)
	AddProject(project *Project) error
	UpdateProject(project *Project) error
	DeleteProject(id string) error
	MarkProjectFavorite(id string) error
	UnmarkProjectFavorite(id string) error
}
