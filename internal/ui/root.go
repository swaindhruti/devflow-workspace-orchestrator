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
//
// It also remembers the last known terminal size (width/height) and
// re-delivers it, as a synthetic tea.WindowSizeMsg, to whichever screen
// ends up active after a keypress. Bubble Tea itself only sends a real
// tea.WindowSizeMsg on startup and on an actual terminal resize — never
// when a screen internally swaps itself out for another one — so
// without this, a screen a user transitions into mid-session (e.g. from
// a dashboard keybinding) would start out believing the terminal is
// 0x0. Most screens don't depend on that (they just center content once
// they learn a real size), but some — a bubbles/filepicker, for
// instance, which sizes its visible window directly off its Height
// field — render essentially nothing without it.
type rootModel struct {
	active        screen.Screen
	width, height int
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
// regardless of which screen is active — tracks the terminal's size as
// tea.WindowSizeMsg arrives, and otherwise delegates to the active
// screen, adopting whatever Screen it returns as the new active screen.
//
// After a keypress specifically (the only kind of message that drives a
// screen transition in this codebase today), it additionally re-sends
// the last known size to the (possibly new) active screen — see the
// rootModel doc comment for why. This means a screen's Update may be
// called twice for one keypress; every Screen in this codebase is
// written to tolerate being called with an unrelated message (the
// default case in their switches is a no-op), so this is safe, and the
// extra call is cheap enough on a single-user terminal UI not to matter.
func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}

	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width, m.height = sizeMsg.Width, sizeMsg.Height
	}

	next, cmd := m.active.Update(msg)
	m.active = next

	if _, ok := msg.(tea.KeyMsg); ok && m.width > 0 && m.height > 0 {
		var sizeCmd tea.Cmd
		m.active, sizeCmd = m.active.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		cmd = tea.Batch(cmd, sizeCmd)
	}

	return m, cmd
}

// View delegates to the active screen's View.
func (m rootModel) View() string {
	return m.active.View()
}
