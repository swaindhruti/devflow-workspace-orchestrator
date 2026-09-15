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

// mascotBitmapWidth is the number of grid cells across every mascot
// bitmap row (see mascotSpec.Frames). mascotWidth/mascotHeight are the
// rendered size of the art itself in terminal cells (mascotBitmapWidth
// doubled, since each grid cell expands to two terminal columns — see
// renderMascotArt). cornerHeight adds room for the blank line and name
// caption every corner also renders (see renderCorner); View uses
// mascotWidth/cornerHeight to lay the four corners out without measuring
// each one individually.
const (
	mascotBitmapWidth = 13
	mascotWidth       = mascotBitmapWidth * 2
	mascotHeight      = 8
	cornerHeight      = mascotHeight + 2 // + blank line + name row
)

// headRow and gapRow are the bitmap rows shared by every mascot's head
// outline and the blank spacer above its foot — only the topper, eyes,
// and foot rows vary per mascot and per animation frame (see
// mascotSpec.Frames/buildFrame), so each mascot's own "signature" stays
// legible against a consistent head shape.
const (
	headRow = ".###########."
	gapRow  = "............."
)

// buildFrame assembles one full 8-row mascot animation frame from its
// three mascot-specific rows, filling in the shared headRow/gapRow rows
// around them.
//
// Parameters:
//   - topper: the row above the head — an antenna, a flat roof, a
//     folder tab, etc. — the clearest single-glance signal of which
//     mascot this is.
//   - eyes: the row used twice for the eye band. A solid headRow-style
//     string here reads as "eyes shut" (this project's blink
//     convention), anything else as the mascot's particular eye shape.
//   - foot: the row below the gap, showing the mascot's stance.
//
// All three, like every row in this file, must be exactly
// mascotBitmapWidth runes so every mascot renders at the same fixed
// mascotWidth.
func buildFrame(topper, eyes, foot string) []string {
	return []string{topper, headRow, headRow, eyes, eyes, headRow, gapRow, foot}
}

// mascotSpec describes one domain's corner character: its display name,
// its accent color, and its two block-art animation frames.
type mascotSpec struct {
	// Name is the short, DevFlow-flavored nickname rendered under the
	// mascot (e.g. "GitMaster" for git) — playful but still legible as
	// "this corner is about git."
	Name string
	// Color is the mascot's accent color, applied to both its art and
	// its name.
	Color lipgloss.AdaptiveColor
	// Frames holds the mascot's two animation frames (see buildFrame).
	// Frame 0 is its idle pose; frame 1 is its own distinct "in-motion"
	// pose — what specifically changes between the two (an antenna
	// tip, a rocking base, a folder tab flap, ...) differs per mascot,
	// which is what gives each corner its own little animation instead
	// of all four moving in lockstep the same way.
	Frames [2][]string
}

// mascots holds the four corner characters, one per DevFlow domain, in
// the order View places them: top-left, top-right, bottom-left,
// bottom-right.
var mascots = [4]mascotSpec{
	{ // git, top-left: a single center antenna that forks outward
		// between frames (branching), eyes as a round pair, one
		// trunk-like foot that steps a column over.
		Name:  "GitMaster",
		Color: lipgloss.AdaptiveColor{Light: "#C2410C", Dark: "#FB923C"},
		Frames: [2][]string{
			buildFrame(".....#.#.....", ".##..###..##.", "......#......"),
			buildFrame("....#...#....", headRow, ".....#......."),
		},
	},
	{ // docker, top-right: no topper at all (a flush, flat roof, like
		// a container), a wide visor-style eye band, and a wide flat
		// base that rocks side to side between frames.
		Name:  "Dockzilla",
		Color: lipgloss.AdaptiveColor{Light: "#0369A1", Dark: "#38BDF8"},
		Frames: [2][]string{
			buildFrame(gapRow, "##.........##", headRow),
			buildFrame(gapRow, headRow, ".#########..."),
		},
	},
	{ // tmux, bottom-left: twin toppers that slide between an outer
		// and inner stance (panes splitting/merging), four small
		// grid-arranged eyes, and twin feet that slide the same way.
		Name:  "PaneMan",
		Color: lipgloss.AdaptiveColor{Light: "#15803D", Dark: "#4ADE80"},
		Frames: [2][]string{
			buildFrame("...#.....#...", ".##.##.##.##.", ".....#.#....."),
			buildFrame(".....#.#.....", headRow, "....#...#...."),
		},
	},
	{ // projects, bottom-right: an asymmetric folder-tab topper that
		// shifts a column over, a single wide "cyclops" eye, and twin
		// feet that hop inward between frames.
		Name:  "ProjectPal",
		Color: lipgloss.AdaptiveColor{Light: "#A21CAF", Dark: "#E879F9"},
		Frames: [2][]string{
			buildFrame("..##.........", ".####...####.", "...##...##..."),
			buildFrame("...##........", headRow, "....##.##...."),
		},
	},
}

// renderMascotArt expands a mascot animation frame (one of
// mascotSpec.Frames' two []string bitmaps) into block-art text, using
// the same dot-matrix technique banner.Render uses for the DEVFLOW
// wordmark: each bitmap cell becomes a filled ("██") or empty ("  ")
// pair of terminal columns (two columns per cell keeps the shape
// roughly square, since terminal cells are taller than they are wide).
//
// Parameters:
//   - rows: one of a mascotSpec's Frames entries — mascotHeight rows of
//     mascotBitmapWidth cells each.
//   - style: the (already color-configured) style to render the art in.
//
// Returns the styled, multi-line art string, mascotWidth cells wide and
// mascotHeight lines tall.
func renderMascotArt(rows []string, style lipgloss.Style) string {
	lines := make([]string, len(rows))
	for i, row := range rows {
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

// renderCorner renders one full corner block for spec: its art at
// animation frame frameIdx, a blank line, and its name caption
// underneath — both in spec.Color. This is what View places in each of
// the four screen corners.
func renderCorner(spec mascotSpec, frameIdx int) string {
	style := lipgloss.NewStyle().Foreground(spec.Color)
	art := renderMascotArt(spec.Frames[frameIdx], style)
	name := style.Bold(true).Render(spec.Name)
	return lipgloss.JoinVertical(lipgloss.Center, art, "", name)
}

// minWidthForMascots and minHeightForMascots are the smallest terminal
// dimensions at which View still has room to place all four corner
// mascots without them colliding with the centered banner. Below this,
// View falls back to the plain centered layout used before the corner
// mascots existed.
const (
	minWidthForMascots  = 60
	minHeightForMascots = 34
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
		m.frame = (m.frame + 1) % len(mascots[0].Frames)
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
// minWidthForMascots/minHeightForMascots), it also places one of the
// four mascots (see the mascots slice) — its art, animated between its
// two frames, plus its name caption — in each screen corner in mascots'
// order (top-left, top-right, bottom-left, bottom-right), so the splash
// reads as a small crew greeting the user rather than a bare logo
// screen.
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

	topLeft := renderCorner(mascots[0], m.frame)
	topRight := renderCorner(mascots[1], m.frame)
	bottomLeft := renderCorner(mascots[2], m.frame)
	bottomRight := renderCorner(mascots[3], m.frame)

	spacer := strings.Repeat(" ", m.width-2*mascotWidth)
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, topLeft, spacer, topRight)
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, bottomLeft, spacer, bottomRight)

	middleHeight := m.height - 2*cornerHeight - 2
	middle := lipgloss.Place(m.width, middleHeight, lipgloss.Center, lipgloss.Center, content)

	return strings.Join([]string{topRow, "", middle, "", bottomRow}, "\n")
}
