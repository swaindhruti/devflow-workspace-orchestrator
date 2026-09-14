package project

import "errors"

// ValidateProject checks that a Project carries the minimum data DevFlow
// needs to manage it, and is called by Service before any write to
// storage.
//
// Parameters:
//   - project: the project to validate.
//
// Returns an error naming the first missing required field (Name, then
// Path), or nil if the project is valid.
func ValidateProject(project *Project) error {
	if project.Name == "" {
		return errors.New("project name is required")
	}
	if project.Path == "" {
		return errors.New("project path is required")
	}
	return nil
}
