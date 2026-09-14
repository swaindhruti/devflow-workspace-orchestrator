package projects

import (
	"encoding/json"
	"errors"
	"os"
)

type JSONFileStorage struct {
	FilePath string
}

// readProjects reads the projects from the JSON file and returns them as a slice of Projects
func (s *JSONFileStorage) readProjects() ([]Project, error) {
	data, err := os.ReadFile(s.FilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Project{}, nil // Return empty slice if file does not exist
		}
		return nil, err
	}

	if len(data) == 0 {
		return []Project{}, nil // Return empty slice if file is empty
	}

	var projects []Project

	err = json.Unmarshal(data, &projects)
	if err != nil {
		return nil, err
	}

	return projects, nil
}

// writeProjects writes the given projects to the JSON file
func (s *JSONFileStorage) writeProjects(projects []Project) error {
	data, err := json.MarshalIndent(projects, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(s.FilePath, data, 0644) // Write the JSON data to the file with appropriate permissions

	return err
}

// GetAllProjects retrieves all projects from the JSON file
func (s *JSONFileStorage) GetAllProjects() ([]Project, error) {
	return s.readProjects()
}

// GetProjectByID retrieves a project by its ID from the JSON file
func (s *JSONFileStorage) GetProjectByID(id string) (*Project, error) {
	projects, err := s.readProjects()
	if err != nil {
		return nil, err
	}

	for _, project := range projects {
		if project.ID == id {
			return &project, nil
		}
	}

	return nil, errors.New("project not found")
}

// AddProject adds a new project to the JSON file
func (s *JSONFileStorage) AddProject(project *Project) error {
	projects, err := s.readProjects()
	if err != nil {
		return err
	}

	projects = append(projects, *project)

	return s.writeProjects(projects)
}

// UpdateProject updates an existing project in the JSON file
func (s *JSONFileStorage) UpdateProject(updatedProject *Project) error {
	projects, err := s.readProjects()
	if err != nil {
		return err
	}

	for i := range projects {
		if projects[i].ID == updatedProject.ID {
			projects[i] = *updatedProject
			return s.writeProjects(projects)
		}
	}

	return errors.New("project not found")

}

// DeleteProject deletes a project by its ID from the JSON file
func (s *JSONFileStorage) DeleteProject(id string) error {
	projects, err := s.readProjects()
	if err != nil {
		return err
	}

	for i, project := range projects {
		if project.ID == id {
			projects = append(projects[:i], projects[i+1:]...) // Remove the project from the slice
			return s.writeProjects(projects)
		}
	}

	return errors.New("project not found")
}

// MArkProjectFavorite marks a project as favorite by its ID in the JSON file
func (s *JSONFileStorage) MarkProjectFavorite(id string) error {
	projects, err := s.readProjects()
	if err != nil {
		return err
	}

	for i, project := range projects {
		if project.ID == id {
			projects[i].IsFavorite = true // Mark the project as favorite
			return s.writeProjects(projects)
		}
	}

	return errors.New("project not found")
}

// UnmarkProjectFavorite unmarks a project as favorite by its ID in the JSON file
func (s *JSONFileStorage) UnmarkProjectFavorite(id string) error {
	projects, err := s.readProjects()
	if err != nil {
		return err
	}

	for i, project := range projects {
		if project.ID == id {
			projects[i].IsFavorite = false // Unmark the project as favorite
			return s.writeProjects(projects)
		}
	}

	return errors.New("project not found")
}
