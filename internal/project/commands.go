package project

// AddCommand appends a new command to the project's saved command list.
//
// Parameters:
//   - cmd: the command to add. Callers are responsible for setting its ID
//     and ProjectID before calling AddCommand; this method does not
//     generate or validate them.
func (p *Project) AddCommand(cmd Command) {
	p.RunCommands = append(p.RunCommands, cmd)
}

// UpdateCommand replaces the saved command whose ID matches updatedCmd.ID
// with the given values. If no command with that ID exists, UpdateCommand
// is a no-op.
//
// Parameters:
//   - updatedCmd: the new state for the command, matched by its existing
//     ID.
func (p *Project) UpdateCommand(updatedCmd Command) {
	for i, cmd := range p.RunCommands {
		if cmd.ID == updatedCmd.ID {
			p.RunCommands[i] = updatedCmd
			return
		}
	}
}

// DeleteCommand removes the saved command matching cmdID from the
// project's command list. If no command with that ID exists, DeleteCommand
// is a no-op.
//
// Parameters:
//   - cmdID: the Command.ID to remove.
func (p *Project) DeleteCommand(cmdID string) {
	for i, cmd := range p.RunCommands {
		if cmd.ID == cmdID {
			p.RunCommands = append(p.RunCommands[:i], p.RunCommands[i+1:]...)
			return
		}
	}
}

// GetAllCommands returns every command saved for the project.
func (p *Project) GetAllCommands() []Command {
	return p.RunCommands
}
