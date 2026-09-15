// Package detail implements DevFlow's project detail screen: a
// single-project view opened from the dashboard, showing everything
// about one registered project in one place — its static fields, saved
// commands (with add/remove), and live Git status and Docker
// containers/images (via app.ProjectContext). Running a saved command
// with streamed output is added in a later change.
package detail

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/app"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/project"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/screen"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/theme"
)

// mode distinguishes the detail screen's input states: ordinary
// viewing/navigation, filling in the add-command form, or waiting on a
// yes/no answer to a pending command delete. Keeping this as an
// explicit field (rather than inferring it from other state) is what
// lets Update route a keypress unambiguously, the same reasoning
// internal/ui/dashboard's own mode field documents.
type mode int

const (
	modeView mode = iota
	modeAddCommand
	modeConfirmDeleteCommand
)

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

	mode      mode
	cmdCursor int
	status    string

	cmdInputs [cmdFieldCount]textinput.Model
	cmdFocus  int
	formErr   error

	width, height int
}

// New constructs a detail screen for p.
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
	return Model{app: a, back: back, project: p}
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
// the result of a contextLoadedMsg regardless of mode, then dispatches
// any tea.KeyMsg to updateView, updateAddCommand, or
// updateConfirmDeleteCommand depending on m.mode. Every other message
// is ignored.
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

	case tea.KeyMsg:
		switch m.mode {
		case modeAddCommand:
			return m.updateAddCommand(msg)
		case modeConfirmDeleteCommand:
			return m.updateConfirmDeleteCommand(msg)
		default:
			return m.updateView(msg)
		}

	default:
		return m, nil
	}
}

// updateView handles a keypress in modeView: "esc" returns to m.back;
// up/k and down/j move the selected command (cmdCursor); "a" opens the
// add-command form; "d" enters delete confirmation for the selected
// command, if there is one.
func (m Model) updateView(msg tea.KeyMsg) (screen.Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m.back, m.back.Init()

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
	}
	return m, nil
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

// keyHints builds this screen's keybinding legend. The
// select/delete-command hints only appear when there's at least one
// command to select or delete.
func keyHints(hasCommands bool) string {
	pairs := make([][2]string, 0, 4)
	if hasCommands {
		pairs = append(pairs, [2]string{"↑/k ↓/j", "select command"})
	}
	pairs = append(pairs, [2]string{"a", "add command"})
	if hasCommands {
		pairs = append(pairs, [2]string{"d", "delete command"})
	}
	pairs = append(pairs, [2]string{"esc", "back"})
	return theme.KeyHints(pairs)
}

// confirmPromptStyle renders the "delete this command?" prompt in
// theme.Danger, bold, matching internal/ui/dashboard's own delete
// confirmation styling.
var confirmPromptStyle = lipgloss.NewStyle().Bold(true).Foreground(theme.Danger)

// View renders the project's static fields as a single bordered,
// centered panel (see theme.Panel) — name, path, description (if any),
// tech stack (if any), and favorite status. In modeAddCommand, the
// add-command form (renderAddCommandForm) replaces everything below
// that, so the form is the sole focus. Otherwise it continues with the
// Commands section (renderCommandsSection, with the selected command
// highlighted), then the live Git and Docker sections
// (renderGitSection/renderDockerSection): a loading notice until
// contextLoadedMsg arrives, an error line if it failed outright, or the
// sections themselves once ctx is populated. The footer is the
// keybinding legend (keyHints), with a delete confirmation or a
// transient status shown as an extra line above it when there's one to
// show — the same "never hide the instructions" approach
// internal/ui/dashboard's footer uses.
func (m Model) View() string {
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
	lines = append(lines, "", favorite)

	if m.mode == modeAddCommand {
		lines = append(lines, "", m.renderAddCommandForm())
		return m.centered(theme.Panel(lipgloss.JoinVertical(lipgloss.Left, lines...), m.width))
	}

	lines = append(lines, "", renderCommandsSection(p.RunCommands, m.cmdCursor))

	switch {
	case m.ctxErr != nil:
		lines = append(lines, "", lipgloss.NewStyle().Foreground(theme.Danger).Render(fmt.Sprintf("failed to load project context: %v", m.ctxErr)))
	case m.ctx == nil:
		lines = append(lines, "", theme.SubtleStyle.Render("Loading Git and Docker status…"))
	default:
		lines = append(lines, "", renderGitSection(*m.ctx))
		if p.HasDocker {
			lines = append(lines, "", renderDockerSection(*m.ctx))
		}
	}

	footer := keyHints(len(p.RunCommands) > 0)
	switch {
	case m.mode == modeConfirmDeleteCommand && m.cmdCursor < len(p.RunCommands):
		prompt := confirmPromptStyle.Render(fmt.Sprintf("Delete command %q? (y/n)", p.RunCommands[m.cmdCursor].Name))
		footer = lipgloss.JoinVertical(lipgloss.Left, prompt, "", footer)
	case m.status != "":
		footer = lipgloss.JoinVertical(lipgloss.Left, theme.SubtleStyle.Render(m.status), "", footer)
	}
	lines = append(lines, "", footer)

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return m.centered(theme.Panel(content, m.width))
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
