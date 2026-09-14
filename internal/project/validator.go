package project

import "errors"

// ValidateProject checks if the project details are valid
func ValidateProject(project *Project) error {
	if project.Name == "" {
		return errors.New("project name is required")
	}
	if project.Path == "" {
		return errors.New("project path is required")
	}
	return nil
}
