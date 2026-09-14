package project

import (
	"errors"
	"testing"
)

type mockRepository struct {
	projects []Project
}

func (m *mockRepository) GetAllProjects() ([]Project, error) {
	return m.projects, nil
}

func (m *mockRepository) GetProjectByID(id string) (*Project, error) {
	for _, p := range m.projects {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, errors.New("project not found")
}

func (m *mockRepository) AddProject(project *Project) error {
	m.projects = append(m.projects, *project)
	return nil
}

func (m *mockRepository) UpdateProject(project *Project) error {
	for i, p := range m.projects {
		if p.ID == project.ID {
			m.projects[i] = *project
			return nil
		}
	}
	return errors.New("project not found")
}

func (m *mockRepository) DeleteProject(id string) error {
	for i, p := range m.projects {
		if p.ID == id {
			m.projects = append(m.projects[:i], m.projects[i+1:]...)
			return nil
		}
	}
	return errors.New("project not found")
}

func (m *mockRepository) MarkProjectFavorite(id string) error {
	for i, p := range m.projects {
		if p.ID == id {
			m.projects[i].IsFavorite = true
			return nil
		}
	}
	return errors.New("project not found")
}

func (m *mockRepository) UnmarkProjectFavorite(id string) error {
	for i, p := range m.projects {
		if p.ID == id {
			m.projects[i].IsFavorite = false
			return nil
		}
	}
	return errors.New("project not found")
}

func TestService_AddProject(t *testing.T) {
	mockRepo := &mockRepository{}
	service := NewProjectService(mockRepo)

	project := &Project{Name: "Test Project", Path: "/test/path"}
	err := service.AddProject(project)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(mockRepo.projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(mockRepo.projects))
	}

	if mockRepo.projects[0].ID == "" {
		t.Fatal("expected project ID to be generated, got empty string")
	}
}

func TestService_AddProject_InvalidName(t *testing.T) {
	mockRepo := &mockRepository{}
	service := NewProjectService(mockRepo)

	project := &Project{Name: "", Path: "/test/path"}
	err := service.AddProject(project)
	if err == nil {
		t.Fatalf("expected error for invalid project name, got nil")
	}

	if len(mockRepo.projects) != 0 {
		t.Fatalf("expected 0 projects, got %d", len(mockRepo.projects))
	}
}

func TestService_AddProject_InvalidPath(t *testing.T) {
	mockRepo := &mockRepository{}
	service := NewProjectService(mockRepo)

	project := &Project{Name: "Test Project", Path: ""}
	err := service.AddProject(project)
	if err == nil {
		t.Fatalf("expected error for invalid project path, got nil")
	}

	if len(mockRepo.projects) != 0 {
		t.Fatalf("expected 0 projects, got %d", len(mockRepo.projects))
	}
}

func TestService_AddProject_Valid(t *testing.T) {
	mockRepo := &mockRepository{}
	service := NewProjectService(mockRepo)

	project := &Project{Name: "Valid Project", Path: "/valid/path"}
	err := service.AddProject(project)
	if err != nil {
		t.Fatalf("expected no error for valid project, got %v", err)
	}

	if len(mockRepo.projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(mockRepo.projects))
	}

	if mockRepo.projects[0].Name != "Valid Project" {
		t.Fatalf("expected project name to be 'Valid Project', got '%s'", mockRepo.projects[0].Name)
	}
}

func TestService_GetProjectByID_Found(t *testing.T) {
	mockRepo := &mockRepository{
		projects: []Project{
			{ID: "1", Name: "Project 1", Path: "/path/1", IsFavorite: false},
			{ID: "2", Name: "Project 2", Path: "/path/2", IsFavorite: true},
		},
	}
	service := NewProjectService(mockRepo)

	project, err := service.GetProjectByID("1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if project.ID != "1" {
		t.Fatalf("expected project ID to be '1', got '%s'", project.ID)
	}
}

func TestService_GetProjectByID_NotFound(t *testing.T) {
	mockRepo := &mockRepository{
		projects: []Project{
			{ID: "1", Name: "Project 1", Path: "/path/1", IsFavorite: false},
			{ID: "2", Name: "Project 2", Path: "/path/2", IsFavorite: true},
		},
	}
	service := NewProjectService(mockRepo)

	_, err := service.GetProjectByID("3")
	if err == nil {
		t.Fatalf("expected error for non-existent project, got nil")
	}
}

func TestService_UpdateProject_Valid(t *testing.T) {
	mockRepo := &mockRepository{
		projects: []Project{
			{ID: "1", Name: "Project 1", Path: "/path/1", IsFavorite: false},
		},
	}
	service := NewProjectService(mockRepo)
	updatedProject := &Project{ID: "1", Name: "Updated Project 1", Path: "/path/1", IsFavorite: true}
	err := service.UpdateProject(updatedProject)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if mockRepo.projects[0].Name != "Updated Project 1" {
		t.Fatalf("expected project name to be 'Updated Project 1', got '%s'", mockRepo.projects[0].Name)
	}

	if !mockRepo.projects[0].IsFavorite {
		t.Fatalf("expected project to be marked as favorite")
	}
}

func TestService_UpdateProject_Invalid(t *testing.T) {
	mockRepo := &mockRepository{
		projects: []Project{
			{ID: "1", Name: "Project 1", Path: "/path/1", IsFavorite: false},
		},
	}
	service := NewProjectService(mockRepo)
	invalidProject := &Project{ID: "1", Name: "", Path: "/path/1", IsFavorite: true}
	err := service.UpdateProject(invalidProject)
	if err == nil {
		t.Fatalf("expected error for invalid project name, got nil")
	}

	if mockRepo.projects[0].Name != "Project 1" {
		t.Fatalf("expected project name to remain 'Project 1', got '%s'", mockRepo.projects[0].Name)
	}
}

func TestService_DeleteProject(t *testing.T) {
	mockRepo := &mockRepository{
		projects: []Project{
			{ID: "1", Name: "Project 1", Path: "/path/1", IsFavorite: false},
			{ID: "2", Name: "Project 2", Path: "/path/2", IsFavorite: true},
		},
	}
	service := NewProjectService(mockRepo)

	err := service.DeleteProject("1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(mockRepo.projects) != 1 {
		t.Fatalf("expected 1 project after deletion, got %d", len(mockRepo.projects))
	}

	if mockRepo.projects[0].ID != "2" {
		t.Fatalf("expected remaining project ID to be '2', got '%s'", mockRepo.projects[0].ID)
	}
}

func TestService_MarkProjectFavorite(t *testing.T) {
	mockRepo := &mockRepository{
		projects: []Project{
			{ID: "1", Name: "Project 1", Path: "/path/1", IsFavorite: false},
		},
	}
	service := NewProjectService(mockRepo)

	err := service.MarkProjectFavorite("1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !mockRepo.projects[0].IsFavorite {
		t.Fatalf("expected project to be marked as favorite")
	}
}

func TestService_UnmarkProjectFavorite(t *testing.T) {
	mockRepo := &mockRepository{
		projects: []Project{
			{ID: "1", Name: "Project 1", Path: "/path/1", IsFavorite: true},
		},
	}
	service := NewProjectService(mockRepo)

	err := service.UnmarkProjectFavorite("1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if mockRepo.projects[0].IsFavorite {
		t.Fatalf("expected project to be unmarked as favorite")
	}
}
