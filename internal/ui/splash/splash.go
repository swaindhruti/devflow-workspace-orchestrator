// Package splash implements DevFlow's startup screen: the DevFlow banner
// shown when the terminal app opens, the way many terminal applications
// greet the user before their main interface appears.
package splash

import (
	"strings"
	"time"

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

// Domain mascot colors — one accent per DevFlow domain (git, docker,
// tmux, and the project registry itself). All four mascots share the
// exact same pixel shape (see mascotBitmapFrames); color is what tells
// them apart, the same way the reference "little robot" design this was
// modeled on reads as a single character purely through its fill color.
var (
	gitColor      = lipgloss.AdaptiveColor{Light: "#C2410C", Dark: "#FB923C"}
	dockerColor   = lipgloss.AdaptiveColor{Light: "#0369A1", Dark: "#38BDF8"}
	tmuxColor     = lipgloss.AdaptiveColor{Light: "#15803D", Dark: "#4ADE80"}
	projectsColor = lipgloss.AdaptiveColor{Light: "#A21CAF", Dark: "#E879F9"}
)

// mascotBitmapFrames are the two block-art animation frames every corner
// mascot cycles between, using the same dot-matrix technique
// banner.Render uses for the DEVFLOW wordmark: each cell of the grid
// expands to a filled ("█") or empty (" ") pair of terminal columns (two
// columns per cell keeps the shape roughly square, since terminal cells
// are taller than they are wide). The shape — a wide blocky head with a
// small tab at each top corner, two square eyes, and two short feet — is
// modeled on a simple blocky robot mascot reference.
//
// Frame 0 is the idle pose: eyes open (the two empty squares), feet
// planted evenly. Frame 1 is the blink-and-wave pose: eyes shut (filled
// solid, a common pixel-art blink convention) and one foot stepped over,
// reading as a small wave/bob. Cycling between the two on a timer (see
// frameMsg/tickFrame) is what gives the corner mascots their motion.
var mascotBitmapFrames = [2][]string{
	{
		"##.......##",
		"###########",
		"###########",
		"##..###..##",
		"##..###..##",
		"###########",
		"...........",
		"..##...##..",
	},
	{
		"##.......##",
		"###########",
		"###########",
		"###########",
		"###########",
		"###########",
		"...........",
		"...##......",
	},
}

// mascotBitmapWidth is the number of grid cells across each row of
// mascotBitmapFrames. mascotWidth/mascotHeight are the resulting
// rendered size in terminal cells (mascotBitmapWidth doubled, since each
// grid cell expands to two terminal columns) — used by View to lay the
// four corners out without measuring each mascot individually.
const (
	mascotBitmapWidth = 11
	mascotWidth       = mascotBitmapWidth * 2
	mascotHeight      = 8
)

// renderMascot expands animation frame index frame of mascotBitmapFrames
// into block-art text and renders it in style.
//
// Parameters:
//   - frame: which entry of mascotBitmapFrames to render (0 or 1).
//   - style: the (already color-configured) style to render the mascot
//     in — see gitColor/dockerColor/tmuxColor/projectsColor.
//
// Returns the styled, multi-line mascot string, mascotWidth cells wide
// and mascotHeight lines tall.
func renderMascot(frame int, style lipgloss.Style) string {
	bitmap := mascotBitmapFrames[frame]
	lines := make([]string, len(bitmap))
	for i, row := range bitmap {
		var b strings.Builder
		for _, cell := range row {
			if cell == '#' {
				b.WriteString("██")
			} else {
				b.WriteString("  ")
			}
		}
		lines[i] = b.String()
	}
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

// frameInterval is how long each mascot animation frame is held before
// advancing to the next — see frameMsg/tickFrame.
const frameInterval = 700 * time.Millisecond

// frameMsg is sent by tickFrame to advance the corner mascots' animation
// by one frame. It carries no data; Update only needs its occurrence.
type frameMsg struct{}

// tickFrame schedules the next frameMsg, frameInterval from now. Model
// calls this both from Init (to start the animation loop) and from
// Update's frameMsg case (to keep it going) — the splash screen's
// keypress-driven transition to the next screen is unaffected by this,
// since frameMsg is a distinct message type from tea.KeyMsg.
func tickFrame() tea.Cmd {
	return tea.Tick(frameInterval, func(time.Time) tea.Msg { return frameMsg{} })
}

// Model is the splash screen. It shows the DevFlow banner and hands
// control to the configured next screen.Screen as soon as the user
// presses any key — there is no automatic timeout, since a splash that
// disappears before it can be read defeats its own purpose. The corner
// mascots animate on their own timer (frame), but that timer only
// advances frame; it never triggers the screen transition itself.
type Model struct {
	next          screen.Screen
	width, height int
	frame         int
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

// Init starts the corner mascots' animation loop (see
// frameMsg/tickFrame). The splash's screen transition itself stays
// purely keypress-driven — there is no timer or data load involved in
// that — this only keeps the mascots blinking and waving while the user
// reads the screen.
func (m Model) Init() tea.Cmd {
	return tickFrame()
}

// Update advances to the configured next screen on any keypress, tracks
// the terminal size (from tea.WindowSizeMsg) so View can center the
// banner, and advances the corner mascots' animation frame on each
// frameMsg (rescheduling the next one so the animation keeps going).
// Every other message is ignored.
func (m Model) Update(msg tea.Msg) (screen.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case frameMsg:
		m.frame = (m.frame + 1) % len(mascotBitmapFrames)
		return m, tickFrame()

	case tea.KeyMsg:
		return m.next, m.next.Init()

	default:
		return m, nil
	}
}

// View renders the DevFlow banner (as a purple-to-yellow gradient), plus
// the tagline and continue hint, centered in the terminal once its size
// is known. Once the terminal is large enough (see
// minWidthForMascots/minHeightForMascots), it also places a block-art
// mascot — animated between mascotBitmapFrames' two frames — in each of
// the four corners: git top-left, docker top-right, tmux bottom-left,
// projects bottom-right, so the splash reads as a small crew greeting
// the user rather than a bare logo screen.
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

	gitMascot := renderMascot(m.frame, lipgloss.NewStyle().Foreground(gitColor))
	dockerMascot := renderMascot(m.frame, lipgloss.NewStyle().Foreground(dockerColor))
	tmuxMascot := renderMascot(m.frame, lipgloss.NewStyle().Foreground(tmuxColor))
	projectsMascot := renderMascot(m.frame, lipgloss.NewStyle().Foreground(projectsColor))

	spacer := strings.Repeat(" ", m.width-2*mascotWidth)
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, gitMascot, spacer, dockerMascot)
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, tmuxMascot, spacer, projectsMascot)

	middleHeight := m.height - 2*mascotHeight - 2
	middle := lipgloss.Place(m.width, middleHeight, lipgloss.Center, lipgloss.Center, content)

	return strings.Join([]string{topRow, "", middle, "", bottomRow}, "\n")
}
