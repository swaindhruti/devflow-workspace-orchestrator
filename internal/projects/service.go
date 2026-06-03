package projects

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetAllProjects() ([]Project, error) {
	return s.repo.GetAllProjects()
}

func (s *Service) GetProjectByID(id string) (*Project, error) {
	return s.repo.GetProjectByID(id)
}

func (s *Service) AddProject(project *Project) error {
	err := ValidateProject(project)
	if err != nil {
		return err
	}
	return s.repo.AddProject(project)
}

func (s *Service) UpdateProject(project *Project) error {
	err := ValidateProject(project)
	if err != nil {
		return err
	}
	return s.repo.UpdateProject(project)
}

func (s *Service) DeleteProject(id string) error {
	return s.repo.DeleteProject(id)
}

func (s *Service) MarkProjectFavorite(id string) error {
	return s.repo.MarkProjectFavorite(id)
}

func (s *Service) UnmarkProjectFavorite(id string) error {
	return s.repo.UnmarkProjectFavorite(id)
}
