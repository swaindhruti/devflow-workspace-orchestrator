package projects

import "testing"

func TestAddComand(t *testing.T) {
	project := Project{}
	cmd := Command{
		ID:          "1",
		Name:        "Test Command",
		Command:     "echo Hello World",
		Path:        "/home/user/devflow",
		Description: "A test command",
	}

	project.AddCommand(cmd)

	if len(project.RunCommands) != 1 {
		t.Errorf("expected 1 command, got %d", len(project.RunCommands))
	}

	if project.RunCommands[0].ID != cmd.ID {
		t.Errorf("expected command ID %s, got %s", cmd.ID, project.RunCommands[0].ID)
	}
}

func TestUpdateCommand(t *testing.T) {
	project := Project{
		RunCommands: []Command{
			{ID: "1", Name: "Test Command", Command: "echo Hello World"},
		},
	}

	updatedCmd := Command{
		ID:          "1",
		Name:        "Updated Command",
		Command:     "echo Updated",
		Description: "An updated command",
	}

	project.UpdateCommand(updatedCmd)

	if project.RunCommands[0].Name != updatedCmd.Name {
		t.Errorf("expected command name %s, got %s", updatedCmd.Name, project.RunCommands[0].Name)
	}

	if project.RunCommands[0].Command != updatedCmd.Command {
		t.Errorf("expected command %s, got %s", updatedCmd.Command, project.RunCommands[0].Command)
	}
}

func TestGetAllCommands(t *testing.T) {
	project := Project{
		RunCommands: []Command{
			{ID: "1", Name: "Command 1", Command: "echo Command 1"},
			{ID: "2", Name: "Command 2", Command: "echo Command 2"},
		},
	}

	commands := project.GetAllCommands()

	if len(commands) != 2 {
		t.Errorf("expected 2 commands, got %d", len(commands))
	}

	if commands[0].ID != "1" || commands[1].ID != "2" {
		t.Errorf("unexpected command IDs: got %s and %s", commands[0].ID, commands[1].ID)
	}
}
