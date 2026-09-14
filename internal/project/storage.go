package project

import (
	"encoding/json"
	"errors"
	"os"
)

// JSONFileStorage is a Repository implementation that persists all
// projects as a single JSON array in a flat file on disk. It is DevFlow's
// initial (Phase 1) storage backend, chosen for being simple, human
// readable, and easy to inspect or hand-edit; see docs/system-design.md for
// the persistence roadmap.
//
// Every method performs a full read-modify-write of the file: it reads the
// entire project list, mutates it in memory, and writes the whole list back
// out. This keeps the implementation simple and is adequate at the scale of
// a personal project registry, but is not designed for concurrent writers
// or large project counts.
type JSONFileStorage struct {
	// FilePath is the location of the JSON file that stores all projects.
	FilePath string
}

// readProjects loads and decodes the full project list from FilePath.
//
// Returns an empty (non-nil) slice, with no error, if the file does not yet
// exist or is empty — both are treated as "no projects registered yet"
// rather than a failure, so callers can use JSONFileStorage before any
// project has ever been written. Returns an error if the file exists but
// could not be read or contains invalid JSON.
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

// writeProjects encodes projects as indented JSON and overwrites FilePath
// with the result.
//
// Parameters:
//   - projects: the complete project list to persist. This replaces the
//     file's entire prior contents.
//
// Returns an error if the data could not be marshaled or the file could
// not be written.
func (s *JSONFileStorage) writeProjects(projects []Project) error {
	data, err := json.MarshalIndent(projects, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(s.FilePath, data, 0644) // Write the JSON data to the file with appropriate permissions

	return err
}

// GetAllProjects implements Repository by returning every project stored
// in the JSON file.
func (s *JSONFileStorage) GetAllProjects() ([]Project, error) {
	return s.readProjects()
}

// GetProjectByID implements Repository by scanning the JSON file's project
// list for an entry whose ID matches id.
//
// Parameters:
//   - id: the Project.ID to look up.
//
// Returns the matching project, or an error if the file could not be read
// or no project with that ID is present.
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

// AddProject implements Repository by appending project to the stored list
// and rewriting the JSON file.
//
// Parameters:
//   - project: the project to add. Its ID is expected to already be set
//     and unique; JSONFileStorage does not check for duplicate IDs.
//
// Returns an error if the file could not be read or rewritten.
func (s *JSONFileStorage) AddProject(project *Project) error {
	projects, err := s.readProjects()
	if err != nil {
		return err
	}

	projects = append(projects, *project)

	return s.writeProjects(projects)
}

// UpdateProject implements Repository by replacing the stored project
// whose ID matches updatedProject.ID with the given values.
//
// Parameters:
//   - updatedProject: the new state for the project, matched by ID.
//
// Returns an error if the file could not be read or rewritten, or if no
// stored project has a matching ID.
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

// DeleteProject implements Repository by removing the stored project whose
// ID matches id and rewriting the JSON file without it.
//
// Parameters:
//   - id: the Project.ID to remove.
//
// Returns an error if the file could not be read or rewritten, or if no
// stored project has a matching ID.
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

// MarkProjectFavorite implements Repository by setting IsFavorite to true
// on the stored project whose ID matches id and rewriting the JSON file.
//
// Parameters:
//   - id: the Project.ID to mark as a favorite.
//
// Returns an error if the file could not be read or rewritten, or if no
// stored project has a matching ID.
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

// UnmarkProjectFavorite implements Repository by setting IsFavorite to
// false on the stored project whose ID matches id and rewriting the JSON
// file.
//
// Parameters:
//   - id: the Project.ID to unmark as a favorite.
//
// Returns an error if the file could not be read or rewritten, or if no
// stored project has a matching ID.
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
