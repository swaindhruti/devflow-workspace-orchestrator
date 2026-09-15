// Package detail implements DevFlow's project detail screen: a
// single-project view opened from the dashboard, showing everything
// about one registered project in one place — its static fields, saved
// commands (with add/remove), and live Git status and Docker
// containers/images (via app.ProjectContext), and running a saved
// command with its output streamed live. The screen is laid out as a
// sidebar of sections (Overview, Commands, Git, Docker) next to a
// content pane showing only the active one, rather than stacking every
// section in one long scroll: a single vertical stack of all sections
// could easily exceed a modest terminal's height (Bubble Tea's
// alt-screen buffer doesn't scroll), silently pushing the later
// sections — Docker in particular, since it renders last — off the
// bottom. Showing one short section at a time keeps the whole screen
// within the terminal's actual size, and the sidebar doubles as an
// always-visible index of what's available and how to get to it.
// Pressing "r" on a selected command in the Commands section runs it
// through app.RunCommand and shows its live stdout/stderr in a
// scrollable pane (bubbles/viewport), polled from the tracked
// runner.Process until it finishes.
package detail

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/app"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/project"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/runner"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/screen"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/theme"
)

// defaultPollInterval is how often a running command's output/state is
// re-checked once started (see pollProcessCmd). 200ms is frequent enough
// to feel live without polling so fast it wastes CPU on a single-user
// local tool.
const defaultPollInterval = 200 * time.Millisecond

// mode distinguishes the detail screen's input states: ordinary
// viewing/navigation, filling in the add-command form, waiting on a
// yes/no answer to a pending command delete, or watching a saved
// command run. Keeping this as an explicit field (rather than inferring
// it from other state) is what lets Update route a keypress
// unambiguously, the same reasoning internal/ui/dashboard's own mode
// field documents.
type mode int

const (
	modeView mode = iota
	modeAddCommand
	modeConfirmDeleteCommand
	modeRunning
)

// section is which sidebar entry the content pane is currently showing.
// It's the same kind of explicit, cycling state as mode: Update needs to
// know unambiguously which pane a keypress like "a" or "d" applies to,
// and the sidebar needs to know which entry to highlight.
type section int

const (
	sectionOverview section = iota
	sectionCommands
	sectionGit
	sectionDocker
	sectionCount
)

var sectionLabels = [sectionCount]string{
	sectionOverview: "Overview",
	sectionCommands: "Commands",
	sectionGit:      "Git",
	sectionDocker:   "Docker",
}

// Field indexes into Model.cmdInputs, and into their form labels.
const (
	cmdFieldName = iota
	cmdFieldCommand
	cmdFieldCount
)

var cmdFieldLabels = [cmdFieldCount]string{
	cmdFieldName:    "Name",
	cmdFieldCommand: "Command",
}

// Model is the project detail screen.
type Model struct {
	app     *app.App
	back    screen.Screen
	project project.Project

	// ctx is the project's live Git/Docker context, nil until
	// contextLoadedMsg arrives (see Init/Update). ctxErr is set instead
	// if loading it failed outright — note this is distinct from
	// ctx.GitStatus/Containers/Images individually being nil, which
	// ProjectContext treats as normal, best-effort absence (not a Git
	// repo, Docker not configured, ...) rather than an error.
	ctx    *app.ProjectContext
	ctxErr error

	section   section
	mode      mode
	cmdCursor int
	status    string

	cmdInputs [cmdFieldCount]textinput.Model
	cmdFocus  int
	formErr   error

	// runCommand is the saved command being run in modeRunning, captured
	// at launch time so rendering never depends on a live cmdCursor (the
	// user could, in principle, land back on modeView and move the
	// cursor before this run finishes settling — not yet reachable since
	// nothing exits modeRunning early except esc/finish, but keeping the
	// run's own copy avoids relying on that always staying true).
	runCommand project.Command
	// runProc is the in-flight process, nil until runStartedMsg reports
	// it. runErr is set instead if RunCommand itself failed to start the
	// process at all (as opposed to the process starting and then
	// failing, which is a normal runSnap.State of StateFailed).
	runProc *runner.Process
	runErr  error
	// runSnap is the most recent poll result. Whether the run has
	// finished is derived from it (runProc != nil && runSnap.State !=
	// runner.StateRunning) rather than tracked separately, so there's
	// only one source of truth for "is this still going."
	runSnap      runner.Snapshot
	viewport     viewport.Model
	pollInterval time.Duration

	width, height int
}

// New constructs a detail screen for p, starting on the Overview
// section.
//
// Parameters:
//   - a: the wired application layer ProjectContext is fetched through,
//     and commands are added/removed through (via Projects().UpdateProject).
//   - back: the screen to return to on "esc" from modeView. Model calls
//     back.Init() itself at the moment of transition, the same way
//     every other screen.Screen transition in this codebase works (see
//     e.g. internal/ui/splash).
//   - p: the project to show details for.
func New(a *app.App, back screen.Screen, p project.Project) Model {
	return Model{app: a, back: back, project: p, pollInterval: defaultPollInterval}
}

// contextLoadedMsg carries the result of fetching the project's
// Git/Docker context, delivered via a tea.Cmd so the fetch (which shells
// out to git and, if configured, docker) doesn't block Bubble Tea's
// event loop.
type contextLoadedMsg struct {
	ctx app.ProjectContext
	err error
}

// Init kicks off fetching the project's Git/Docker context.
func (m Model) Init() tea.Cmd {
	return m.loadContext
}

// loadContext is the tea.Cmd Init returns: it fetches the project's
// live context and reports the result as a contextLoadedMsg.
func (m Model) loadContext() tea.Msg {
	ctx, err := m.app.ProjectContext(m.project.ID)
	return contextLoadedMsg{ctx: ctx, err: err}
}

// runStartedMsg carries the result of launching a saved command via
// app.RunCommand, delivered via startCommand's tea.Cmd.
type runStartedMsg struct {
	proc *runner.Process
	err  error
}

// processPolledMsg carries one poll result for a running process,
// delivered via pollProcessCmd's tea.Cmd. proc identifies which run this
// reading belongs to, so a stale tick arriving after the user has left
// modeRunning (or, once a later run replaces runProc, from a previous
// run) can be told apart from the current one — see handleProcessPolled.
type processPolledMsg struct {
	proc *runner.Process
	snap runner.Snapshot
}

// startCommand launches the given saved command through App.RunCommand
// and reports the outcome as a runStartedMsg. It's a tea.Cmd (a
// zero-argument closure returning tea.Msg) so the launch — which starts
// an OS process — doesn't block Bubble Tea's event loop, the same
// reasoning loadContext already documents for the Git/Docker fetch.
func startCommand(a *app.App, projectID, commandID string) tea.Cmd {
	return func() tea.Msg {
		proc, err := a.RunCommand(projectID, commandID)
		return runStartedMsg{proc: proc, err: err}
	}
}

// pollProcessCmd waits interval, then takes one Snapshot of proc and
// reports it as a processPolledMsg. It does not re-arm itself — whether
// polling continues is decided entirely by handleProcessPolled, which
// re-issues this same Cmd only while the process is still running. This
// keeps "when to stop polling" in exactly one place rather than split
// between the Cmd and its handler.
func pollProcessCmd(proc *runner.Process, interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return processPolledMsg{proc: proc, snap: proc.Snapshot()}
	})
}

// newCommandInputs builds the blank name/command fields for the
// add-command form.
func newCommandInputs() [cmdFieldCount]textinput.Model {
	var inputs [cmdFieldCount]textinput.Model

	inputs[cmdFieldName] = textinput.New()
	inputs[cmdFieldName].Placeholder = "test"
	inputs[cmdFieldName].CharLimit = 64

	inputs[cmdFieldCommand] = textinput.New()
	inputs[cmdFieldCommand].Placeholder = "go test ./..."
	inputs[cmdFieldCommand].CharLimit = 300

	return inputs
}

// Update tracks the terminal size (from tea.WindowSizeMsg) and stores
// the result of a contextLoadedMsg, runStartedMsg, or processPolledMsg
// regardless of mode, then dispatches any tea.KeyMsg to updateView,
// updateAddCommand, or updateConfirmDeleteCommand depending on m.mode.
// Every other message is ignored.
func (m Model) Update(msg tea.Msg) (screen.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case contextLoadedMsg:
		if msg.err != nil {
			m.ctxErr = msg.err
			return m, nil
		}
		m.ctx = &msg.ctx
		return m, nil

	case runStartedMsg:
		return m.handleRunStarted(msg)

	case processPolledMsg:
		return m.handleProcessPolled(msg)

	case tea.KeyMsg:
		switch m.mode {
		case modeAddCommand:
			return m.updateAddCommand(msg)
		case modeConfirmDeleteCommand:
			return m.updateConfirmDeleteCommand(msg)
		case modeRunning:
			return m.updateRunning(msg)
		default:
			return m.updateView(msg)
		}

	default:
		return m, nil
	}
}

// updateView handles a keypress in modeView. "esc" returns to m.back.
// left/h and right/l (or tab/shift+tab) switch which sidebar section is
// active, the same on every section. The remaining keys are specific to
// the Commands section: up/k and down/j move the selected command
// (cmdCursor), "a" opens the add-command form, and "d" enters delete
// confirmation for the selected command, if there is one. Gating those
// to sectionCommands keeps the footer's advertised hints (see
// sectionKeyHints) always accurate — a key never does something the
// visible hints for the current section didn't mention.
func (m Model) updateView(msg tea.KeyMsg) (screen.Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m.back, m.back.Init()

	case "left", "h", "shift+tab":
		m.section = (m.section - 1 + sectionCount) % sectionCount
		return m, nil

	case "right", "l", "tab":
		m.section = (m.section + 1) % sectionCount
		return m, nil
	}

	if m.section != sectionCommands {
		return m, nil
	}

	switch msg.String() {
	case "up", "k":
		if m.cmdCursor > 0 {
			m.cmdCursor--
		}

	case "down", "j":
		if m.cmdCursor < len(m.project.RunCommands)-1 {
			m.cmdCursor++
		}

	case "a":
		m.mode = modeAddCommand
		m.formErr = nil
		m.cmdFocus = cmdFieldName
		m.cmdInputs = newCommandInputs()
		return m, m.cmdInputs[cmdFieldName].Focus()

	case "d":
		if m.cmdCursor < len(m.project.RunCommands) {
			m.mode = modeConfirmDeleteCommand
			m.status = ""
		}

	case "r":
		if m.cmdCursor < len(m.project.RunCommands) {
			m.mode = modeRunning
			m.runCommand = m.project.RunCommands[m.cmdCursor]
			m.runProc, m.runErr = nil, nil
			m.runSnap = runner.Snapshot{}

			_, contentWidth, boxHeight := m.layoutDims()
			vw, vh := runningViewportSize(contentWidth, boxHeight)
			m.viewport = viewport.New(vw, vh)

			return m, startCommand(m.app, m.project.ID, m.runCommand.ID)
		}
	}
	return m, nil
}

// updateRunning handles a keypress while a saved command is running or
// has just finished. "s" stops it early without leaving the pane — the
// next poll tick observes the resulting StateStopped and
// handleProcessPolled's own poll loop ends itself the same way it would
// for any other finish. "esc" leaves the pane, stopping the process
// first if it's still running: there's no other screen that can reach
// this process, so leaving without stopping it would orphan it with no
// way to stop it later. Anything else is forwarded to the viewport so
// its own scroll bindings (arrow keys, page up/down, ...) work for
// free.
func (m Model) updateRunning(msg tea.KeyMsg) (screen.Screen, tea.Cmd) {
	running := m.runProc != nil && m.runSnap.State == runner.StateRunning

	switch msg.String() {
	case "s":
		if running {
			_ = m.app.StopCommand(m.runProc.ID)
		}
		return m, nil

	case "esc":
		if running {
			_ = m.app.StopCommand(m.runProc.ID)
		}
		m.mode = modeView
		m.runCommand = project.Command{}
		m.runProc, m.runErr = nil, nil
		m.runSnap = runner.Snapshot{}
		return m, nil
	}

	_, contentWidth, boxHeight := m.layoutDims()
	m.viewport.Width, m.viewport.Height = runningViewportSize(contentWidth, boxHeight)

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// handleRunStarted stores the outcome of startCommand. A launch failure
// (the process never started at all — e.g. the shell couldn't be found)
// is kept separate from runSnap so the running pane can tell "never
// started" apart from "started and then failed," which is a normal
// runSnap.State of StateFailed reported through the poll loop instead.
// On success, it fires the first poll immediately so the pane doesn't
// sit showing nothing for a full pollInterval before any output appears.
func (m Model) handleRunStarted(msg runStartedMsg) (screen.Screen, tea.Cmd) {
	if msg.err != nil {
		m.runErr = msg.err
		return m, nil
	}
	m.runProc = msg.proc
	return m, pollProcessCmd(msg.proc, m.pollInterval)
}

// handleProcessPolled stores one poll result and, if the process is
// still running, re-arms the poll loop; otherwise it lets the loop end
// by returning a nil Cmd. msg.proc is checked against m.runProc first —
// a tick that arrives for a run the user has already left (esc) or that
// belongs to a since-replaced run is silently dropped rather than
// resurrecting stale state. This is what makes the loop self-terminating
// with no separate "cancel polling" message: once modeRunning is left,
// nothing holds onto the old proc pointer to match against, so the next
// stray tick (if one was already in flight) is simply ignored.
func (m Model) handleProcessPolled(msg processPolledMsg) (screen.Screen, tea.Cmd) {
	if msg.proc != m.runProc {
		return m, nil
	}

	m.runSnap = msg.snap
	_, contentWidth, boxHeight := m.layoutDims()
	m.viewport.Width, m.viewport.Height = runningViewportSize(contentWidth, boxHeight)
	m.viewport.SetContent(buildRunOutput(msg.snap))
	m.viewport.GotoBottom()

	if msg.snap.State == runner.StateRunning {
		return m, pollProcessCmd(msg.proc, m.pollInterval)
	}
	return m, nil
}

// buildRunOutput composes the text shown in the running pane's scrollable
// output: stdout, followed by a divider and stderr, but only if there's
// any stderr to show. The two streams are captured into separate
// buffers by runner.Process and can't be reconstructed into one true
// chronological interleaving, so this doesn't attempt to fake that —
// it shows them as two distinct blocks instead.
func buildRunOutput(snap runner.Snapshot) string {
	if snap.Stderr == "" {
		return snap.Stdout
	}
	return snap.Stdout + "\n\n── stderr ──\n" + snap.Stderr
}

// updateAddCommand handles a keypress while the add-command form is
// showing. "esc" cancels back to modeView without saving; tab/shift+tab
// move focus between fields; enter moves focus forward too, except on
// the last field, where it submits. Any other key is forwarded to the
// focused textinput.Model.
func (m Model) updateAddCommand(msg tea.KeyMsg) (screen.Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeView
		m.formErr = nil
		return m, nil

	case "tab", "shift+tab":
		return m, m.moveCmdFocus(msg.String() == "tab")

	case "enter":
		if m.cmdFocus == cmdFieldCount-1 {
			return m.submitAddCommand()
		}
		return m, m.moveCmdFocus(true)
	}

	var cmd tea.Cmd
	m.cmdInputs[m.cmdFocus], cmd = m.cmdInputs[m.cmdFocus].Update(msg)
	return m, cmd
}

// moveCmdFocus blurs the currently-focused command-form input, advances
// (or, if forward is false, retreats) m.cmdFocus by one slot with
// wraparound, and focuses the new one, returning the tea.Cmd
// textinput.Focus produces (its cursor-blink command).
func (m *Model) moveCmdFocus(forward bool) tea.Cmd {
	m.cmdInputs[m.cmdFocus].Blur()
	if forward {
		m.cmdFocus = (m.cmdFocus + 1) % cmdFieldCount
	} else {
		m.cmdFocus = (m.cmdFocus - 1 + cmdFieldCount) % cmdFieldCount
	}
	return m.cmdInputs[m.cmdFocus].Focus()
}

// submitAddCommand validates the form (both fields required), then
// appends a new project.Command — with a freshly generated ID via
// project.GenerateID, since Project.AddCommand takes an already-built
// Command and doesn't generate one itself — to a copy of m.project and
// persists it via UpdateProject. On success it updates m.project to
// match, shows a confirmation in m.status, and returns to modeView; on
// any failure it stays on the form with the error shown inline, so the
// user's already-typed input isn't lost.
func (m Model) submitAddCommand() (screen.Screen, tea.Cmd) {
	name := strings.TrimSpace(m.cmdInputs[cmdFieldName].Value())
	commandText := strings.TrimSpace(m.cmdInputs[cmdFieldCommand].Value())
	if name == "" || commandText == "" {
		m.formErr = fmt.Errorf("both name and command are required")
		return m, nil
	}

	updated := m.project
	updated.AddCommand(project.Command{
		ID:        project.GenerateID(),
		ProjectID: m.project.ID,
		Name:      name,
		Command:   commandText,
	})

	if err := m.app.Projects().UpdateProject(&updated); err != nil {
		m.formErr = err
		return m, nil
	}

	m.project = updated
	m.mode = modeView
	m.status = fmt.Sprintf("added command %q", name)
	return m, nil
}

// updateConfirmDeleteCommand handles a keypress while in
// modeConfirmDeleteCommand: "y" deletes the command the cursor was on
// when "d" was pressed, "n"/"esc" cancels. Any other key is ignored, so
// a stray keypress can't accidentally confirm a delete.
func (m Model) updateConfirmDeleteCommand(msg tea.KeyMsg) (screen.Screen, tea.Cmd) {
	switch msg.String() {
	case "y":
		m.deleteSelectedCommand()
		m.mode = modeView
	case "n", "esc":
		m.mode = modeView
	}
	return m, nil
}

// deleteSelectedCommand removes the command the cursor is on via a copy
// of m.project, persisted via UpdateProject. On success it updates
// m.project to match and moves the cursor back by one if it would
// otherwise point past the new end of the list; on failure the project
// is left untouched and the error is reported through m.status.
//
// It copies RunCommands before calling DeleteCommand on the copy,
// rather than deleting directly off m.project: DeleteCommand removes an
// entry by shifting the tail left within the same backing array (the
// standard Go delete-by-append idiom), and m.project.RunCommands would
// share that same backing array with an un-copied `updated` — so an
// in-place delete would silently corrupt m.project's own view of its
// commands even if UpdateProject then failed and this method never
// commits `updated` back to m.project.
func (m *Model) deleteSelectedCommand() {
	if m.cmdCursor >= len(m.project.RunCommands) {
		return
	}
	target := m.project.RunCommands[m.cmdCursor]

	updated := m.project
	updated.RunCommands = append([]project.Command(nil), m.project.RunCommands...)
	updated.DeleteCommand(target.ID)

	if err := m.app.Projects().UpdateProject(&updated); err != nil {
		m.status = fmt.Sprintf("failed to delete command: %v", err)
		return
	}

	m.project = updated
	if m.cmdCursor >= len(m.project.RunCommands) && m.cmdCursor > 0 {
		m.cmdCursor--
	}
	m.status = fmt.Sprintf("deleted command %q", target.Name)
}

// sectionKeyHints builds the footer's keybinding legend for the given
// section: the section-switch hint is always present, and the
// select/add/delete/run-command hints appear only on sectionCommands
// (and select/delete/run only when there's at least one command), so
// the footer never advertises a key that updateView wouldn't currently
// honor.
func sectionKeyHints(s section, hasCommands bool) string {
	pairs := make([][2]string, 0, 6)
	pairs = append(pairs, [2]string{"←/→", "switch section"})
	if s == sectionCommands {
		if hasCommands {
			pairs = append(pairs, [2]string{"↑/k ↓/j", "select command"})
		}
		pairs = append(pairs, [2]string{"a", "add command"})
		if hasCommands {
			pairs = append(pairs, [2]string{"d", "delete command"})
			pairs = append(pairs, [2]string{"r", "run selected"})
		}
	}
	pairs = append(pairs, [2]string{"esc", "back"})
	return theme.KeyHints(pairs)
}

// confirmPromptStyle renders the "delete this command?" prompt in
// theme.Danger, bold, matching internal/ui/dashboard's own delete
// confirmation styling.
var confirmPromptStyle = lipgloss.NewStyle().Bold(true).Foreground(theme.Danger)

// sideBoxStyle frames both the sidebar and the content pane in a
// matching rounded border, so the two read as one cohesive
// bento-style layout rather than two unrelated boxes. Like
// theme.PanelStyle, it sets no Foreground/Background/Bold of its own so
// it composes safely with the already-styled content each pane renders.
var sideBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(theme.Primary).
	Padding(1, 2)

// sizedBox renders content inside sideBoxStyle, capping the whole box
// (border and padding included) to exactly width columns and height
// rows whenever they're positive — the same reasoning as theme.Panel's
// width handling (Style.Width/Height size the pre-border content area,
// so the border's own size has to be subtracted first for the
// *bordered* total to come out right), extended to height so the
// sidebar and content pane always render as two equal-height boxes
// instead of whichever is taller dictating the other's border.
func sizedBox(content string, width, height int) string {
	style := sideBoxStyle
	if width > 0 {
		if contentWidth := width - style.GetHorizontalBorderSize(); contentWidth > 0 {
			style = style.Width(contentWidth)
		}
	}
	if height > 0 {
		if contentHeight := height - style.GetVerticalBorderSize(); contentHeight > 0 {
			style = style.Height(contentHeight)
		}
	}
	return style.Render(content)
}

// clampInt constrains v to [lo, hi].
func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// layoutDims computes the sidebar/content box sizes View lays out, from
// the screen's last known terminal size. It's factored out of View
// (rather than computed only there) so updateRunning and
// handleProcessPolled can derive the same content-box dimensions to
// size the running pane's viewport — see runningViewportSize — without
// duplicating this math or risking it drifting out of sync with what
// View actually renders. Returns all-zero if the terminal size isn't
// known yet, same as View's own previous inline version did.
func (m Model) layoutDims() (sidebarWidth, contentWidth, boxHeight int) {
	if m.width > 0 {
		sidebarWidth = clampInt(m.width/4, 16, 24)
		contentWidth = m.width - sidebarWidth - 1
		if contentWidth < 28 {
			contentWidth = 28
		}
	}
	if m.height > 0 {
		boxHeight = m.height - 4
		if boxHeight < 8 {
			boxHeight = 8
		}
	}
	return sidebarWidth, contentWidth, boxHeight
}

// runningPaneChromeLines is how many of the running pane's lines are
// fixed chrome rather than the scrollable viewport: heading, a blank
// line, another blank line, a status-or-blank line, a third blank line,
// and the keybinding legend — see renderRunningPane. The status line is
// always present (even while running, as an empty line) specifically so
// the viewport's height — and therefore the pane's whole layout — stays
// identical before and after a command finishes, rather than growing by
// one line the moment a status appears.
const runningPaneChromeLines = 6

// runningViewportSize computes the width/height budget for the running
// pane's scrollable viewport from the content box's own outer
// width/height (as returned by layoutDims), using the same
// border/padding accounting sizedBox uses internally, minus
// runningPaneChromeLines. Returns 0, 0 if the content box's size isn't
// known yet (mirrors sizedBox's own "0 means not yet known" contract).
func runningViewportSize(contentWidth, boxHeight int) (width, height int) {
	if contentWidth <= 0 || boxHeight <= 0 {
		return 0, 0
	}

	width = contentWidth - sideBoxStyle.GetHorizontalBorderSize() - sideBoxStyle.GetHorizontalPadding()
	height = boxHeight - sideBoxStyle.GetVerticalBorderSize() - sideBoxStyle.GetVerticalPadding() - runningPaneChromeLines

	if width < 1 {
		width = 1
	}
	if height < 3 {
		height = 3
	}
	return width, height
}

// View renders the screen as a sidebar (renderSidebar) next to a
// content pane showing only the active section (renderSection), both
// boxed to the same height via sizedBox so they read as one layout. In
// modeAddCommand or modeRunning, the content pane is the add-command
// form or the running-command pane instead (renderAddCommandForm /
// renderRunningPane, both of which carry their own complete keybinding
// legend), and the outer footer is skipped entirely rather than showing
// hints like "esc back" that would mean something different in either
// of those. Otherwise the footer is sectionKeyHints for the active
// section, with a delete confirmation or a transient status shown as an
// extra line above it when there's one to show — the same "never hide
// the instructions" approach internal/ui/dashboard's footer uses.
func (m Model) View() string {
	sidebarWidth, contentWidth, boxHeight := m.layoutDims()

	if m.mode == modeRunning {
		m.viewport.Width, m.viewport.Height = runningViewportSize(contentWidth, boxHeight)
	}

	sidebar := m.renderSidebar()
	content := m.renderSection()

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		sizedBox(sidebar, sidebarWidth, boxHeight),
		" ",
		sizedBox(content, contentWidth, boxHeight),
	)

	if m.mode == modeAddCommand || m.mode == modeRunning {
		return m.centered(body)
	}

	full := lipgloss.JoinVertical(lipgloss.Left, body, "", m.renderFooter())
	return m.centered(full)
}

// renderSidebar lists every section, marking the active one with a "›"
// prefix in theme.TitleStyle and the rest in theme.SubtleStyle — the
// always-visible index of what this screen shows and how to reach it.
// The Commands entry additionally shows how many are saved.
func (m Model) renderSidebar() string {
	items := make([]string, 0, sectionCount*2-1)
	for s := section(0); s < sectionCount; s++ {
		label := sectionLabels[s]
		if s == sectionCommands && len(m.project.RunCommands) > 0 {
			label = fmt.Sprintf("%s (%d)", label, len(m.project.RunCommands))
		}

		if s == m.section {
			items = append(items, theme.TitleStyle.Render("› "+label))
		} else {
			items = append(items, theme.SubtleStyle.Render("  "+label))
		}
		if s != sectionCount-1 {
			items = append(items, "")
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, items...)
}

// renderSection renders the content pane for whichever section is
// active. Commands shows the add-command form in place of the commands
// list while modeAddCommand is active, since "a" can only be pressed
// from sectionCommands in the first place (see updateView), so the two
// are always in sync.
func (m Model) renderSection() string {
	switch m.section {
	case sectionCommands:
		if m.mode == modeAddCommand {
			return m.renderAddCommandForm()
		}
		if m.mode == modeRunning {
			return m.renderRunningPane()
		}
		return m.renderCommandsPane()
	case sectionGit:
		return m.renderGitPane()
	case sectionDocker:
		return m.renderDockerPane()
	default:
		return m.renderOverviewPane()
	}
}

// renderOverviewPane shows the project's static fields — name, path,
// description (if any), tech stack (if any), and favorite status — plus
// a hint on how to reach the other sections.
func (m Model) renderOverviewPane() string {
	p := m.project

	lines := []string{
		theme.TitleStyle.Render(p.Name),
		theme.SubtleStyle.Render(p.Path),
	}

	if p.Description != "" {
		lines = append(lines, "", p.Description)
	}

	if len(p.TechStack) > 0 {
		lines = append(lines, "", theme.SubtleStyle.Render("Stack: "+strings.Join(p.TechStack, ", ")))
	}

	favorite := "Not favorited"
	if p.IsFavorite {
		favorite = lipgloss.NewStyle().Foreground(theme.Accent).Render("★ Favorited")
	}
	lines = append(lines, "", favorite, "",
		theme.HelpStyle.Render("←/→ (h/l, tab) to browse Commands, Git, and Docker."))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderCommandsPane shows the project's saved commands
// (renderCommandsSection, with the selected one highlighted) plus a
// hint on how to select, add, or delete one.
func (m Model) renderCommandsPane() string {
	p := m.project
	section := renderCommandsSection(p.RunCommands, m.cmdCursor)

	help := theme.HelpStyle.Render("Press a to save your first command, e.g. go test ./...")
	if len(p.RunCommands) > 0 {
		help = theme.HelpStyle.Render("↑/k ↓/j select   a add another   d delete selected   r run selected")
	}

	return lipgloss.JoinVertical(lipgloss.Left, section, "", help)
}

// renderRunningPane shows the currently selected command's live run: a
// heading naming it, then one of three things depending on how far the
// run has gotten — an error if RunCommand itself failed to start the
// process, a "Starting…" placeholder if it started but no poll result
// has arrived yet, or the scrollable viewport of its combined
// stdout/stderr (buildRunOutput) — followed, once the process has
// finished, by a colored exit-status line. The keybinding legend is
// embedded here (mirroring renderAddCommandForm) rather than left to
// the outer footer, since "esc" means something specific to this pane
// (stop-if-running, then leave) that the section-level footer's plain
// "back" wouldn't convey.
//
// The status line and its surrounding blank lines are always present
// (blank when still running) rather than only appearing once finished,
// so the viewport's reserved height — see runningPaneChromeLines — and
// therefore the whole pane's layout stays identical before and after
// the command settles, instead of the pane growing by a line the
// moment a status appears.
func (m Model) renderRunningPane() string {
	heading := theme.TitleStyle.Render("Running: " + m.runCommand.Name)

	if m.runErr != nil {
		return lipgloss.JoinVertical(lipgloss.Left,
			heading, "",
			lipgloss.NewStyle().Foreground(theme.Danger).Render(fmt.Sprintf("failed to start: %v", m.runErr)),
			"",
			theme.KeyHints([][2]string{{"esc", "back"}}),
		)
	}

	if m.runProc == nil {
		return lipgloss.JoinVertical(lipgloss.Left, heading, "", theme.SubtleStyle.Render("Starting…"))
	}

	var status string
	switch m.runSnap.State {
	case runner.StateCompleted:
		status = lipgloss.NewStyle().Foreground(theme.Success).Render(fmt.Sprintf("Exit code %d — completed", m.runSnap.ExitCode))
	case runner.StateFailed:
		status = lipgloss.NewStyle().Foreground(theme.Danger).Render(fmt.Sprintf("Exit code %d — failed", m.runSnap.ExitCode))
	case runner.StateStopped:
		status = lipgloss.NewStyle().Foreground(theme.Warning).Render("Stopped")
	}

	hints := theme.KeyHints([][2]string{{"↑/k ↓/j", "scroll"}, {"esc", "back to Commands"}})
	if m.runSnap.State == runner.StateRunning {
		hints = theme.KeyHints([][2]string{{"↑/k ↓/j", "scroll"}, {"s", "stop"}, {"esc", "stop & leave"}})
	}

	return lipgloss.JoinVertical(lipgloss.Left, heading, "", m.viewport.View(), "", status, "", hints)
}

// renderGitPane shows the project's live Git status
// (renderGitSection), or a loading/error notice while ctx hasn't
// arrived yet.
func (m Model) renderGitPane() string {
	switch {
	case m.ctxErr != nil:
		return lipgloss.JoinVertical(lipgloss.Left,
			theme.TitleStyle.Render("Git"), "",
			lipgloss.NewStyle().Foreground(theme.Danger).Render(fmt.Sprintf("failed to load project context: %v", m.ctxErr)))
	case m.ctx == nil:
		return lipgloss.JoinVertical(lipgloss.Left,
			theme.TitleStyle.Render("Git"), "",
			theme.SubtleStyle.Render("Loading Git status…"))
	default:
		return lipgloss.JoinVertical(lipgloss.Left,
			renderGitSection(*m.ctx), "",
			theme.HelpStyle.Render("Read-only — reflects the working tree on disk."))
	}
}

// renderDockerPane shows the project's Docker containers/images
// (renderDockerSection) if the project has Docker configured, a
// loading/error notice while ctx hasn't arrived yet, or — if the
// project has no Docker configuration at all — an explanatory message
// instead of an empty-looking pane.
func (m Model) renderDockerPane() string {
	if !m.project.HasDocker {
		return lipgloss.JoinVertical(lipgloss.Left,
			theme.TitleStyle.Render("Docker"), "",
			theme.SubtleStyle.Render("This project has no Docker configuration."),
			theme.HelpStyle.Render("Docker support is detected automatically when a project is registered (a Dockerfile or compose file present in its directory)."))
	}

	switch {
	case m.ctxErr != nil:
		return lipgloss.JoinVertical(lipgloss.Left,
			theme.TitleStyle.Render("Docker"), "",
			lipgloss.NewStyle().Foreground(theme.Danger).Render(fmt.Sprintf("failed to load project context: %v", m.ctxErr)))
	case m.ctx == nil:
		return lipgloss.JoinVertical(lipgloss.Left,
			theme.TitleStyle.Render("Docker"), "",
			theme.SubtleStyle.Render("Loading Docker status…"))
	default:
		return lipgloss.JoinVertical(lipgloss.Left,
			renderDockerSection(*m.ctx), "",
			theme.HelpStyle.Render("Read-only — scoped to this project's Docker identifier."))
	}
}

// renderFooter builds the screen's bottom line(s): sectionKeyHints for
// the active section, with a delete confirmation or a transient status
// shown as an extra line above it when there's one to show.
func (m Model) renderFooter() string {
	hints := sectionKeyHints(m.section, len(m.project.RunCommands) > 0)

	switch {
	case m.mode == modeConfirmDeleteCommand && m.cmdCursor < len(m.project.RunCommands):
		prompt := confirmPromptStyle.Render(fmt.Sprintf("Delete command %q? (y/n)", m.project.RunCommands[m.cmdCursor].Name))
		return lipgloss.JoinVertical(lipgloss.Left, prompt, "", hints)
	case m.status != "":
		return lipgloss.JoinVertical(lipgloss.Left, theme.SubtleStyle.Render(m.status), "", hints)
	default:
		return hints
	}
}

// renderAddCommandForm renders the add-command form: a heading, the
// name/command fields (with the focused one marked), an inline error
// if the last submit attempt failed, and its own keybinding legend.
func (m Model) renderAddCommandForm() string {
	heading := theme.TitleStyle.Render("Add Command")

	var form strings.Builder
	for i, in := range m.cmdInputs {
		marker := "  "
		if i == m.cmdFocus {
			marker = "> "
		}
		fmt.Fprintf(&form, "%s%-10s %s\n", marker, cmdFieldLabels[i]+":", in.View())
	}

	sections := []string{heading, form.String()}
	if m.formErr != nil {
		sections = append(sections, lipgloss.NewStyle().Foreground(theme.Danger).Render(fmt.Sprintf("error: %v", m.formErr)), "")
	}
	sections = append(sections, theme.KeyHints([][2]string{
		{"tab/shift+tab", "move field"},
		{"enter", "next field / submit"},
		{"esc", "cancel"},
	}))

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderCommandsSection formats a project's saved commands as a
// labeled section: a "Commands" heading, then one line per command
// (its name and shell command text, the selected one highlighted in
// theme.TitleStyle the same way internal/ui/dashboard highlights its
// selected row) — or a placeholder line if there are none yet.
func renderCommandsSection(commands []project.Command, selected int) string {
	heading := theme.TitleStyle.Render("Commands")

	if len(commands) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left, heading, theme.SubtleStyle.Render("No saved commands."))
	}

	rows := []string{heading}
	for i, c := range commands {
		if i == selected {
			rows = append(rows, theme.TitleStyle.Render(fmt.Sprintf("> %s: %s", c.Name, c.Command)))
		} else {
			rows = append(rows, fmt.Sprintf("  %s: %s", c.Name, theme.SubtleStyle.Render(c.Command)))
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// renderGitSection formats ctx.GitStatus as a labeled section: a
// "Git" heading, the branch (or "detached HEAD"), a clean/dirty line
// (with staged/unstaged/untracked counts when dirty), and an
// ahead/behind line when either is nonzero. If GitStatus is nil (the
// project's directory isn't a Git repository, or status couldn't be
// read), it renders a single explanatory line instead.
func renderGitSection(ctx app.ProjectContext) string {
	heading := theme.TitleStyle.Render("Git")

	status := ctx.GitStatus
	if status == nil {
		return lipgloss.JoinVertical(lipgloss.Left, heading, theme.SubtleStyle.Render("Not a Git repository."))
	}

	branch := status.Branch
	if status.Detached {
		branch = "detached HEAD"
	}

	var stateLine string
	if status.Clean {
		stateLine = lipgloss.NewStyle().Foreground(theme.Success).Render("Clean")
	} else {
		stateLine = lipgloss.NewStyle().Foreground(theme.Warning).Render(
			fmt.Sprintf("Dirty — %d staged, %d unstaged, %d untracked", status.Staged, status.Unstaged, status.Untracked))
	}

	rows := []string{heading, fmt.Sprintf("Branch: %s", branch), stateLine}
	if status.Ahead > 0 || status.Behind > 0 {
		rows = append(rows, theme.SubtleStyle.Render(fmt.Sprintf("Ahead %d, behind %d", status.Ahead, status.Behind)))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// renderDockerSection formats ctx.Containers and ctx.Images as a
// labeled section: a "Docker" heading, one line per container (name
// and status, colored by whether it's running), and one line per image
// (repository:tag and size). Each list shows a placeholder line
// ("No containers found."/"No images found.") when empty rather than
// omitting the heading, since HasDocker being true means the caller
// already knows Docker is configured for this project and an empty
// list is itself useful information (nothing currently running).
func renderDockerSection(ctx app.ProjectContext) string {
	heading := theme.TitleStyle.Render("Docker")
	rows := []string{heading}

	if len(ctx.Containers) == 0 {
		rows = append(rows, theme.SubtleStyle.Render("No containers found."))
	} else {
		for _, c := range ctx.Containers {
			style := lipgloss.NewStyle().Foreground(theme.Muted)
			if c.State == "running" {
				style = lipgloss.NewStyle().Foreground(theme.Success)
			}
			rows = append(rows, style.Render(fmt.Sprintf("%s — %s", c.Name, c.Status)))
		}
	}

	if len(ctx.Images) == 0 {
		rows = append(rows, theme.SubtleStyle.Render("No images found."))
	} else {
		for _, img := range ctx.Images {
			rows = append(rows, theme.SubtleStyle.Render(fmt.Sprintf("%s:%s — %s", img.Repository, img.Tag, img.Size)))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// centered places content in the middle of the terminal once its size
// is known (via a tea.WindowSizeMsg reaching Update — see
// internal/ui/root.go for why this screen reliably receives one even
// though it's only ever reached mid-session, not shown at startup),
// falling back to returning content unplaced if the size isn't known
// yet.
func (m Model) centered(content string) string {
	if m.width == 0 || m.height == 0 {
		return content
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}
