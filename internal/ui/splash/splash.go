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

// coworkerMascot is a small static block-art character — a friendly robot
// face — rendered beside the banner so the splash reads as "a coworker
// greeting you" rather than a bare logo. It is deliberately a single
// fixed frame (no wave/blink animation): the user asked for the corner
// badges below to stay unanimated for performance and memory efficiency,
// and a moving mascot next to four static badges would read as
// inconsistent, so the whole splash stays static.
var coworkerMascot = []string{
	"╭─────╮",
	"│ ◕ ◕ │",
	"│  ▾  │",
	"╰┬───┬╯",
	"  │ │  ",
}

// mascotStyle renders coworkerMascot in the app's primary accent color —
// present enough to read as a character, but flat (not gradient) so it
// doesn't compete with the banner for attention.
var mascotStyle = lipgloss.NewStyle().Foreground(theme.Primary)

// Domain badge glyphs and colors. Each of DevFlow's four domains — git,
// docker, tmux, and the project registry itself — gets a one-character
// glyph and its own accent color, so the four corner badges (built with
// newBadge and placed in View) act as an at-a-glance legend of the app's
// feature set. Glyphs are chosen from narrow, single-cell Unicode symbols (not
// emoji, which often render double-width and break the badges' fixed
// layout) and colors are AdaptiveColor so each badge stays legible in
// both light and dark terminals.
const (
	gitGlyph      = "⎇" // branch symbol
	dockerGlyph   = "▦" // stacked containers
	tmuxGlyph     = "⊞" // split panes
	projectsGlyph = "⌂" // project home
)

var (
	gitColor      = lipgloss.AdaptiveColor{Light: "#C2410C", Dark: "#FB923C"}
	dockerColor   = lipgloss.AdaptiveColor{Light: "#0369A1", Dark: "#38BDF8"}
	tmuxColor     = lipgloss.AdaptiveColor{Light: "#15803D", Dark: "#4ADE80"}
	projectsColor = lipgloss.AdaptiveColor{Light: "#A21CAF", Dark: "#E879F9"}
)

// badgeWidth and badgeHeight are the fixed dimensions (in terminal cells)
// of every badge rendered by newBadge. Keeping every badge identical in
// size is what makes the corner layout math in View a simple subtraction
// rather than a per-badge measurement.
const (
	badgeWidth  = 7
	badgeHeight = 3
)

// minWidthForBadges and minHeightForBadges are the smallest terminal
// dimensions at which View still has room to place all four corner
// badges without them colliding with the centered banner/mascot. Below
// this, View falls back to the plain centered layout used before the
// corner badges existed.
const (
	minWidthForBadges  = 60
	minHeightForBadges = 20
)

// newBadge renders one domain corner badge: glyph boxed in a fixed
// badgeWidth x badgeHeight frame, styled in the given color.
//
// Parameters:
//   - glyph: the single-cell symbol identifying the domain (e.g.
//     gitGlyph). Must be exactly one terminal cell wide for the box to
//     stay badgeWidth cells across every row.
//   - style: the (already color-configured) style to render the badge
//     in — see gitColor/dockerColor/tmuxColor/projectsColor.
//
// Returns the styled, multi-line badge string.
func newBadge(glyph string, style lipgloss.Style) string {
	lines := []string{
		"╭─────╮",
		"│  " + glyph + "  │",
		"╰─────╯",
	}
	return style.Render(strings.Join(lines, "\n"))
}

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
// the coworker mascot, plus the tagline and continue hint, centered in
// the terminal once its size is known. Once the terminal is large enough
// (see minWidthForBadges/minHeightForBadges), it also places a static,
// color-coded domain badge in each of the four corners — git top-left,
// docker top-right, tmux bottom-left, projects bottom-right — so the
// splash doubles as an at-a-glance legend of DevFlow's feature set.
func (m Model) View() string {
	bannerLines := strings.Split(banner.Render("DEVFLOW", "██", "  "), "\n")
	gradientBanner := theme.Gradient(bannerLines, gradientFrom, gradientTo)
	mascot := mascotStyle.Render(strings.Join(coworkerMascot, "\n"))

	lockup := lipgloss.JoinHorizontal(lipgloss.Center, mascot, "   ", gradientBanner)

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

	if m.width < minWidthForBadges || m.height < minHeightForBadges {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	gitBadge := newBadge(gitGlyph, lipgloss.NewStyle().Foreground(gitColor))
	dockerBadge := newBadge(dockerGlyph, lipgloss.NewStyle().Foreground(dockerColor))
	tmuxBadge := newBadge(tmuxGlyph, lipgloss.NewStyle().Foreground(tmuxColor))
	projectsBadge := newBadge(projectsGlyph, lipgloss.NewStyle().Foreground(projectsColor))

	spacer := strings.Repeat(" ", m.width-2*badgeWidth)
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, gitBadge, spacer, dockerBadge)
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, tmuxBadge, spacer, projectsBadge)

	middleHeight := m.height - 2*badgeHeight - 2
	middle := lipgloss.Place(m.width, middleHeight, lipgloss.Center, lipgloss.Center, content)

	return strings.Join([]string{topRow, "", middle, "", bottomRow}, "\n")
}
