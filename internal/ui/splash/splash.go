// Package splash implements DevFlow's startup screen: the DevFlow banner
// shown briefly when the terminal app opens, the way many terminal
// applications greet the user before their main interface appears.
package splash

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/banner"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/screen"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/theme"
)

// displayDuration is how long the splash screen stays up before
// automatically advancing, for a user who doesn't press a key.
const displayDuration = 1200 * time.Millisecond

// tagline is the short subtitle shown beneath the banner.
const tagline = "Terminal Workspace Orchestrator"

// bannerStyle renders the DEVFLOW block art in the theme's accent color.
var bannerStyle = lipgloss.NewStyle().Bold(true).Foreground(theme.Primary)

// tickMsg marks displayDuration having elapsed.
type tickMsg struct{}

// Model is the splash screen. It shows the DevFlow banner and hands
// control to the configured next screen.Screen as soon as either
// displayDuration elapses or the user presses any key, whichever happens
// first.
type Model struct {
	next          screen.Screen
	duration      time.Duration
	width, height int
}

// New constructs a splash screen that transitions to next once its intro
// finishes.
//
// Parameters:
//   - next: the screen to activate once the splash finishes. Model calls
//     next.Init() itself at the moment of transition, the same way
//     Bubble Tea calls a model's Init once when it first becomes active.
func New(next screen.Screen) Model {
	return Model{next: next, duration: displayDuration}
}

// Init starts the timer that auto-advances the splash screen once the
// configured duration elapses. duration defaults to displayDuration via
// New; tests construct Model directly with a much shorter duration so
// they don't have to block on the real production delay.
func (m Model) Init() tea.Cmd {
	return tea.Tick(m.duration, func(time.Time) tea.Msg { return tickMsg{} })
}

// Update advances to the configured next screen on any keypress or once
// the display timer fires, whichever comes first. It also tracks the
// terminal size (from tea.WindowSizeMsg) so View can center the banner.
// Every other message is ignored: the splash screen has nothing else to
// react to.
func (m Model) Update(msg tea.Msg) (screen.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg, tickMsg:
		return m.next, m.next.Init()

	default:
		return m, nil
	}
}

// View renders the DevFlow banner and tagline, centered in the terminal
// once its size is known.
func (m Model) View() string {
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		bannerStyle.Render(banner.Render("DEVFLOW", "██", "  ")),
		"",
		theme.SubtleStyle.Render(tagline),
		"",
		theme.HelpStyle.Render("press any key to continue"),
	)

	if m.width == 0 || m.height == 0 {
		return content
	}

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}
