package project

import (
	"os"
	"testing"
)

// Creates a temporary JSON file for testing of the storage layer
func CreateTestStorage(t *testing.T) *JSONFileStorage {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "projects-test.json")

	if err != nil {
		t.Fatal(err)
	}

	return &JSONFileStorage{
		FilePath: tmpFile.Name(),
	}
}

func TestAddProject(t *testing.T) {
	storage := CreateTestStorage(t)

	project := Project{
		ID:   "1",
		Name: "Devflow",
		Path: "/tmp/devflow",
	}

	err := storage.AddProject(&project)
	if err != nil {
		t.Fatal(err)
	}

	projects, err := storage.GetAllProjects()
	if err != nil {
		t.Fatal(err)
	}

	if len(projects) != 1 {
		t.Errorf("expected 1 project got %d", len(projects))
	}
}

func TestUpdateProject(t *testing.T) {
	storage := CreateTestStorage(t)

	project := Project{
		ID:   "1",
		Name: "Old Name",
		Path: "/tmp/devflow",
	}

	err := storage.AddProject(&project)
	if err != nil {
		t.Fatal(err)
	}

	project.Name = "New Name"

	err = storage.UpdateProject(&project)

	if err != nil {
		t.Fatal(err)
	}

	result, err := storage.GetProjectByID("1")

	if result.Name != "New Name" {
		t.Errorf("Update Failed")
	}
}

func TestDeleteProject(t *testing.T) {

	storage := CreateTestStorage(t)

	project := Project{
		ID:   "1",
		Name: "Devflow",
		Path: "/tmp/devflow",
	}

	err := storage.AddProject(&project)
	if err != nil {
		t.Fatal(err)
	}

	err = storage.DeleteProject("1")
	if err != nil {
		t.Fatal(err)
	}

	projects, _ := storage.GetAllProjects()

	if len(projects) != 0 {
		t.Errorf("Deletion Failed")
	}

}

func TestGetAllProjects(t *testing.T) {
	storage := CreateTestStorage(t)

	projects := []Project{
		{
			ID:   "1",
			Name: "p1",
		},
		{
			ID:   "2",
			Name: "p2",
		},
		{
			ID:   "3",
			Name: "p3",
		},
		{
			ID:   "4",
			Name: "p4",
		},
		{
			ID:   "5",
			Name: "p5",
		},
	}

	err := storage.writeProjects(projects)
	if err != nil {
		t.Fatal(err)
	}

	projects, _ = storage.GetAllProjects()

	lengthOfProjects := len(projects)
	if lengthOfProjects != 5 {
		t.Errorf("Expected 5 projects but got %d", lengthOfProjects)
	}

}
