package project

import "testing"

func TestValidateProject(t *testing.T) {
	tests := []struct {
		name      string
		project   Project
		shouldErr bool
	}{
		{
			name: "valid project",
			project: Project{
				Name: "DevFlow",
				Path: "/home/user/devflow",
			},
			shouldErr: false,
		},
		{
			name: "empty name",
			project: Project{
				Name: "",
				Path: "/home/user/devflow",
			},
			shouldErr: true,
		},
		{
			name: "empty path",
			project: Project{
				Name: "DevFlow",
				Path: "",
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProject(&tt.project)

			if tt.shouldErr && err == nil {
				t.Errorf("expected error but got nil")
			}

			if !tt.shouldErr && err != nil {
				t.Errorf("did not expect error but got %v", err)
			}
		})
	}
}
