// Package screen defines the contract every top-level DevFlow UI view
// implements. It exists as its own small package, separate from the root
// UI package, purely to break an import cycle: the root package must
// import every concrete screen (splash, dashboard, ...) to wire them
// together, and each screen must be able to hand control to another
// screen — if the transition mechanism lived in the root package, every
// screen would need to import it too, creating a cycle. Depending on this
// leaf package instead lets screens stay mutually independent, matching
// the domain-first, self-contained-package preference described in
// docs/architecture.md.
package screen

import tea "github.com/charmbracelet/bubbletea"

// Screen is implemented by each top-level view (splash, dashboard, and
// whatever views follow). The root UI model owns exactly one active
// Screen at a time and forwards Bubble Tea's lifecycle calls to it.
//
// A screen requests a transition simply by returning a different Screen
// value from Update than the one Update was called on — there is no
// separate "switch screen" message type, since the interface's own
// return value already says what should be active next. The screen doing
// the switching is responsible for calling the new Screen's Init itself
// and returning that as its Cmd, the same way Bubble Tea's own Init is
// expected to be called once when a model becomes active.
type Screen interface {
	// Init returns the screen's initial command, run once when it
	// becomes active (e.g. to start a timer or kick off a data load).
	Init() tea.Cmd
	// Update handles one Bubble Tea message and returns the screen that
	// should be active afterward (itself, in the common case of just
	// updating internal state) along with any command to run.
	Update(msg tea.Msg) (Screen, tea.Cmd)
	// View renders the screen's current state as a string for Bubble Tea
	// to draw to the terminal.
	View() string
}
