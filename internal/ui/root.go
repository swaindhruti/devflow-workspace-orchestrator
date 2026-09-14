// Package ui is DevFlow's Bubble Tea terminal interface: the root program
// (this file), and the individual screens under internal/ui/{splash,
// dashboard, ...}. The root package owns no view logic of its own beyond
// delegating to whichever screen.Screen is currently active and handling
// the one keybinding (ctrl+c) that should work no matter which screen the
// user is on.
package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/screen"
)

// rootModel is Bubble Tea's actual tea.Model. It owns the currently
// active screen.Screen and forwards every Init/Update/View call to it,
// adopting whatever Screen Update returns as the new active screen — see
// screen.Screen's doc comment for why that alone is enough to drive
// transitions between screens, with no separate "switch screen" message
// type needed.
type rootModel struct {
	active screen.Screen
}

// newRootModel constructs a rootModel starting on initial.
func newRootModel(initial screen.Screen) rootModel {
	return rootModel{active: initial}
}

// Init delegates to the active screen's Init.
func (m rootModel) Init() tea.Cmd {
	return m.active.Init()
}

// Update handles the one global keybinding — ctrl+c always quits,
// regardless of which screen is active — and otherwise delegates to the
// active screen, adopting whatever Screen it returns as the new active
// screen.
func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}

	next, cmd := m.active.Update(msg)
	m.active = next
	return m, cmd
}

// View delegates to the active screen's View.
func (m rootModel) View() string {
	return m.active.View()
}
