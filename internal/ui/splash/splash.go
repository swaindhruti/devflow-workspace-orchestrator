// Package splash implements DevFlow's startup screen: the DevFlow banner
// shown when the terminal app opens, the way many terminal applications
// greet the user before their main interface appears.
package splash

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/banner"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/screen"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/theme"
)

// tagline is the short subtitle shown beneath the banner.
const tagline = "Terminal Workspace Orchestrator"

// gradientFrom and gradientTo are the banner's two endpoint colors — a
// vivid purple fading into a warm yellow — deliberately distinct from the
// app-wide theme.Primary so the splash keeps its own bold, one-time-only
// first impression instead of just repeating the dashboard's palette.
var (
	gradientFrom = lipgloss.Color("#7C3AED")
	gradientTo   = lipgloss.Color("#FACC15")
)

// terminalIcon is a small decorative illustration — a terminal window,
// DevFlow's own subject matter — drawn in the same block-art spirit as
// the banner text. It's kept to 7 rows so it lines up with the banner's
// height when placed beside it, and rendered in a muted tone (see
// iconStyle) so it reads as a supporting flourish rather than competing
// with the gradient wordmark for attention.
var terminalIcon = []string{
	"╭──────────╮",
	"│ ●  ●  ●  │",
	"│          │",
	"│  >_      │",
	"│          │",
	"│          │",
	"╰──────────╯",
}

// iconStyle renders terminalIcon in a muted tone so it stays a quiet
// accent next to the vivid gradient banner rather than distracting from
// it.
var iconStyle = lipgloss.NewStyle().Foreground(theme.Muted)

// Model is the splash screen. It shows the DevFlow banner and hands
// control to the configured next screen.Screen as soon as the user
// presses any key — there is no automatic timeout, since a splash that
// disappears before it can be read defeats its own purpose.
type Model struct {
	next          screen.Screen
	width, height int
}

// New constructs a splash screen that transitions to next once the user
// presses a key.
//
// Parameters:
//   - next: the screen to activate once the splash finishes. Model calls
//     next.Init() itself at the moment of transition, the same way
//     Bubble Tea calls a model's Init once when it first becomes active.
func New(next screen.Screen) Model {
	return Model{next: next}
}

// Init has nothing to kick off: the splash screen is purely
// keypress-driven, so there is no timer or data load to start here.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update advances to the configured next screen on any keypress, and
// tracks the terminal size (from tea.WindowSizeMsg) so View can center
// the banner. Every other message is ignored: the splash screen has
// nothing else to react to.
func (m Model) Update(msg tea.Msg) (screen.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.next, m.next.Init()

	default:
		return m, nil
	}
}

// View renders the DevFlow banner (as a purple-to-yellow gradient) beside
// the terminal icon illustration, plus the tagline and continue hint,
// centered in the terminal once its size is known.
func (m Model) View() string {
	bannerLines := strings.Split(banner.Render("DEVFLOW", "██", "  "), "\n")
	gradientBanner := theme.Gradient(bannerLines, gradientFrom, gradientTo)
	icon := iconStyle.Render(strings.Join(terminalIcon, "\n"))

	lockup := lipgloss.JoinHorizontal(lipgloss.Center, icon, "   ", gradientBanner)

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		lockup,
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
