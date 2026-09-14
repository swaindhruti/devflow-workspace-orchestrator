package project

// AddCommand adds a new command to a project
func (p *Project) AddCommand(cmd Command) {
	p.RunCommands = append(p.RunCommands, cmd)
}

// UpdateCommand updates an existing command in a project
func (p *Project) UpdateCommand(updatedCmd Command) {
	for i, cmd := range p.RunCommands {
		if cmd.ID == updatedCmd.ID {
			p.RunCommands[i] = updatedCmd
			return
		}
	}
}

// DeleteCommand deletes a command from a project by its ID
func (p *Project) DeleteCommand(cmdID string) {
	for i, cmd := range p.RunCommands {
		if cmd.ID == cmdID {
			p.RunCommands = append(p.RunCommands[:i], p.RunCommands[i+1:]...)
			return
		}
	}
}

// GetAllCommands returns all commands associated with a project
func (p *Project) GetAllCommands() []Command {
	return p.RunCommands
}
