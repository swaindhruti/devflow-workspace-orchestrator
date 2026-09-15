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

// Domain mascot glyphs and colors. Each of DevFlow's four domains — git,
// docker, tmux, and the project registry itself — gets a one-character
// glyph and its own accent color. newMascot uses these to build the four
// corner characters placed by View, so the corners double as an
// at-a-glance legend of DevFlow's feature set. Glyphs are chosen from
// narrow, single-cell Unicode symbols (not emoji, which often render
// double-width and break the mascots' fixed layout) and colors are
// AdaptiveColor so each mascot stays legible in both light and dark
// terminals.
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

// mascotWidth and mascotHeight are the fixed dimensions (in terminal
// cells) of every character rendered by newMascot. Keeping every mascot
// identical in size is what makes the corner layout math in View a
// simple subtraction rather than a per-mascot measurement.
const (
	mascotWidth  = 9
	mascotHeight = 7
)

// mascotBody is the shared block-art robot body every domain mascot is
// built from: a rounded head with two eyes, a small mouth, and two
// planted feet. Only the antenna glyph on top (see newMascot) and the
// color vary per domain, so the four corner characters read as one
// consistent cast — like a small crew representing DevFlow's
// capabilities — rather than four unrelated icons.
var mascotBody = []string{
	" ╭─────╮ ",
	" │ ◕ ◕ │ ",
	" │  ▾  │ ",
	" ╰┬───┬╯ ",
	"   │ │   ",
}

// newMascot renders one domain's corner character: mascotBody topped
// with an antenna holding up the domain's glyph, styled in the domain's
// color. It is a single fixed frame — no wave/blink animation — since
// the mascots need to stay cheap to render (precomputed strings, no
// tea.Tick, no per-frame state) and a moving character in only one
// corner while the other three stood still would read as inconsistent.
//
// Parameters:
//   - glyph: the single-cell symbol identifying the domain (e.g.
//     gitGlyph). Must be exactly one terminal cell wide so the antenna
//     stays centered over mascotBody.
//   - style: the (already color-configured) style to render the mascot
//     in — see gitColor/dockerColor/tmuxColor/projectsColor.
//
// Returns the styled, multi-line mascot string, mascotWidth cells wide
// and mascotHeight lines tall.
func newMascot(glyph string, style lipgloss.Style) string {
	pad := strings.Repeat(" ", (mascotWidth-1)/2)
	lines := append([]string{
		pad + glyph + pad,
		pad + "│" + pad,
	}, mascotBody...)
	return style.Render(strings.Join(lines, "\n"))
}

// minWidthForMascots and minHeightForMascots are the smallest terminal
// dimensions at which View still has room to place all four corner
// mascots without them colliding with the centered banner. Below this,
// View falls back to the plain centered layout used before the corner
// mascots existed.
const (
	minWidthForMascots  = 60
	minHeightForMascots = 30
)

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

// View renders the DevFlow banner (as a purple-to-yellow gradient), plus
// the tagline and continue hint, centered in the terminal once its size
// is known. Once the terminal is large enough (see
// minWidthForMascots/minHeightForMascots), it also places a static,
// color-coded block-art character in each of the four corners — git
// top-left, docker top-right, tmux bottom-left, projects bottom-right —
// so the splash doubles as an at-a-glance legend of DevFlow's feature
// set, and reads as a small crew greeting the user rather than a bare
// logo screen.
func (m Model) View() string {
	bannerLines := strings.Split(banner.Render("DEVFLOW", "██", "  "), "\n")
	gradientBanner := theme.Gradient(bannerLines, gradientFrom, gradientTo)

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		gradientBanner,
		"",
		theme.SubtleStyle.Render(tagline),
		"",
		theme.HelpStyle.Render("press any key to continue"),
	)

	if m.width == 0 || m.height == 0 {
		return content
	}

	if m.width < minWidthForMascots || m.height < minHeightForMascots {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	gitMascot := newMascot(gitGlyph, lipgloss.NewStyle().Foreground(gitColor))
	dockerMascot := newMascot(dockerGlyph, lipgloss.NewStyle().Foreground(dockerColor))
	tmuxMascot := newMascot(tmuxGlyph, lipgloss.NewStyle().Foreground(tmuxColor))
	projectsMascot := newMascot(projectsGlyph, lipgloss.NewStyle().Foreground(projectsColor))

	spacer := strings.Repeat(" ", m.width-2*mascotWidth)
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, gitMascot, spacer, dockerMascot)
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, tmuxMascot, spacer, projectsMascot)

	middleHeight := m.height - 2*mascotHeight - 2
	middle := lipgloss.Place(m.width, middleHeight, lipgloss.Center, lipgloss.Center, content)

	return strings.Join([]string{topRow, "", middle, "", bottomRow}, "\n")
}
