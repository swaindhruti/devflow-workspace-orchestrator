package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/app"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/dashboard"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/splash"
)

// Run starts DevFlow's terminal interface: a splash screen showing the
// DevFlow banner, which hands off to the project dashboard once its intro
// finishes.
//
// Parameters:
//   - a: the wired application layer (see app.NewApp) every screen reads
//     and acts through.
//
// Returns once the user quits the program (ctrl+c, or a screen's own quit
// key), or an error if the Bubble Tea program failed to run.
func Run(a *app.App) error {
	dash := dashboard.New(a)
	intro := splash.New(dash)

	_, err := tea.NewProgram(newRootModel(intro), tea.WithAltScreen()).Run()
	return err
}
