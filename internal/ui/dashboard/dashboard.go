// Package dashboard implements DevFlow's project dashboard: a scrollable
// list of every registered project, keyboard-navigable, showing each
// project's favorite marker, name, tech stack, path, and whether it has
// Docker configured. It is the screen the splash screen hands off to.
//
// The dashboard only renders fields already stored on project.Project
// (favorite, name, tech stack, path, HasDocker) rather than making a live
// Git or Docker CLI call per row: with many registered projects, shelling
// out per row on every load would make the list slow to open. Live,
// per-call context (Git status, Docker containers/images) belongs to a
// project detail screen for one project at a time — see app.ProjectContext,
// which already exists for exactly that — not the list view.
package dashboard

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/app"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/project"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/addproject"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/banner"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/screen"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/theme"
)

// mode distinguishes the dashboard's input states: ordinary list
// navigation versus waiting on a yes/no answer to a pending delete.
// Keeping this as an explicit field (rather than inferring it from other
// state) is what lets Update route a keypress unambiguously — "d" always
// means something different depending on which mode is active.
type mode int

const (
	modeList mode = iota
	modeConfirmDelete
)

// Model is the project dashboard screen.
type Model struct {
	app      *app.App
	projects []project.Project
	cursor   int
	err      error
	// status is a transient, non-fatal message (e.g. "project deleted",
	// or a favorite-toggle failure) shown in place of the help line
	// until the next action clears or replaces it. Unlike err, a
	// non-empty status never replaces the whole view — the project
	// list stays visible underneath it.
	status        string
	mode          mode
	width, height int
}

// New constructs a dashboard for the given application layer.
//
// Parameters:
//   - a: the wired application layer to load the project registry from.
func New(a *app.App) Model {
	return Model{app: a}
}

// projectsLoadedMsg carries the result of loading the project registry,
// delivered via a tea.Cmd so the initial load doesn't block Bubble Tea's
// event loop.
type projectsLoadedMsg struct {
	projects []project.Project
	err      error
}

// Init kicks off loading the project registry.
func (m Model) Init() tea.Cmd {
	return m.loadProjects
}

// loadProjects is the tea.Cmd Init returns: it reads every registered
// project and reports the result as a projectsLoadedMsg.
func (m Model) loadProjects() tea.Msg {
	projects, err := m.app.Projects().GetAllProjects()
	return projectsLoadedMsg{projects: projects, err: err}
}

// Update handles the project-loaded result and keyboard input. Which
// keys do what depends on m.mode: ordinary navigation
// (up/k/down/j/f/d/q) in modeList, versus a yes/no answer in
// modeConfirmDelete.
func (m Model) Update(msg tea.Msg) (screen.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case projectsLoadedMsg:
		m.projects = msg.projects
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		if m.mode == modeConfirmDelete {
			return m.updateConfirmDelete(msg)
		}
		return m.updateList(msg)

	default:
		return m, nil
	}
}

// updateList handles a keypress while in modeList: cursor movement,
// favorite toggling, entering delete confirmation, opening the
// add-project screen, and quit.
func (m Model) updateList(msg tea.KeyMsg) (screen.Screen, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.projects)-1 {
			m.cursor++
		}

	case "f":
		m.toggleFavorite()

	case "d":
		if m.cursor < len(m.projects) {
			m.mode = modeConfirmDelete
			m.status = ""
		}

	case "a":
		next := addproject.New(m.app, m)
		return next, next.Init()

	case "q":
		return m, tea.Quit
	}
	return m, nil
}

// updateConfirmDelete handles a keypress while in modeConfirmDelete: "y"
// deletes the project the cursor was on when "d" was pressed, "n"/"esc"
// cancels. Any other key is ignored, so a stray keypress can't
// accidentally confirm a delete.
func (m Model) updateConfirmDelete(msg tea.KeyMsg) (screen.Screen, tea.Cmd) {
	switch msg.String() {
	case "y":
		m.deleteSelected()
		m.mode = modeList

	case "n", "esc":
		m.mode = modeList
	}
	return m, nil
}

// toggleFavorite flips the selected project's favorite state, persists
// it via the project service, and updates the in-memory copy to match
// on success. On failure, it leaves the project's state untouched and
// reports the error through m.status rather than m.err, since a failed
// toggle shouldn't take the whole project list off the screen.
func (m *Model) toggleFavorite() {
	if m.cursor >= len(m.projects) {
		return
	}

	p := &m.projects[m.cursor]
	var err error
	if p.IsFavorite {
		err = m.app.Projects().UnmarkProjectFavorite(p.ID)
	} else {
		err = m.app.Projects().MarkProjectFavorite(p.ID)
	}

	if err != nil {
		m.status = fmt.Sprintf("failed to update favorite: %v", err)
		return
	}
	p.IsFavorite = !p.IsFavorite
	m.status = ""
}

// deleteSelected deletes the project the cursor is on via the project
// service and removes it from the in-memory list on success, moving the
// cursor back by one if it would otherwise point past the new end of
// the list. On failure, the list is left untouched and the error is
// reported through m.status.
func (m *Model) deleteSelected() {
	if m.cursor >= len(m.projects) {
		return
	}

	deleted := m.projects[m.cursor]
	if err := m.app.Projects().DeleteProject(deleted.ID); err != nil {
		m.status = fmt.Sprintf("failed to delete project: %v", err)
		return
	}

	m.projects = append(m.projects[:m.cursor], m.projects[m.cursor+1:]...)
	if m.cursor >= len(m.projects) && m.cursor > 0 {
		m.cursor--
	}
	m.status = fmt.Sprintf("deleted %q", deleted.Name)
}

// keyHints is the dashboard's full keybinding legend, shown at the
// bottom of the panel whenever there's no more pressing status or
// confirmation to show instead. It's the answer to "how do I use this
// screen" for a first-time user.
var keyHints = theme.KeyHints([][2]string{
	{"↑/k ↓/j", "navigate"},
	{"a", "add project"},
	{"f", "toggle favorite"},
	{"d", "delete"},
	{"q", "quit"},
})

// confirmPromptStyle renders the "delete this project?" prompt in
// theme.Danger, bold, so it reads as distinctly higher-stakes than the
// ordinary status line it temporarily replaces.
var confirmPromptStyle = lipgloss.NewStyle().Bold(true).Foreground(theme.Danger)

// logoLines is the DEVFLOW block-art wordmark's raw bitmap output,
// computed once at package init since banner.Render's output never
// changes — View re-applies theme.Gradient to it on every render
// (cheap: it's just re-coloring the same fixed text), matching the
// splash screen's own banner so the dashboard reads as a continuation
// of that first impression rather than a different, unbranded screen.
var logoLines = strings.Split(banner.Render("DEVFLOW", "██", "  "), "\n")

// View renders the dashboard as the gradient DEVFLOW wordmark above a
// single bordered, centered panel (see theme.PanelStyle): a subtitle,
// then the project list (or an empty-state/error message in place of
// it), then a footer. The keybinding legend (keyHints) is always part
// of that footer — it's the answer to "how do I use this screen," so
// it shouldn't disappear the moment the user actually does something —
// with a delete confirmation prompt (modeConfirmDelete) or a transient
// status message (m.status) shown as an extra line above it when
// there's one to show. The wordmark plus panel are placed together in
// the center of the terminal once a size is known (see m.centered),
// matching the splash screen's centered first impression instead of
// the list floating against the top-left corner.
func (m Model) View() string {
	logo := theme.Gradient(logoLines, theme.BrandGradientFrom, theme.BrandGradientTo)
	header := theme.SubtleStyle.Render("Your registered projects, all in one place.")

	if m.err != nil {
		errPanel := theme.Panel(lipgloss.JoinVertical(lipgloss.Left, header, "",
			lipgloss.NewStyle().Foreground(theme.Danger).Render(fmt.Sprintf("failed to load projects: %v", m.err))), m.width)
		return m.centered(lipgloss.JoinVertical(lipgloss.Center, logo, "", errPanel))
	}

	var body string
	if len(m.projects) == 0 {
		body = lipgloss.JoinVertical(lipgloss.Center,
			theme.SubtleStyle.Render("No projects yet — let's add your first one."),
			"",
			theme.KeyHints([][2]string{{"a", "add a project"}}),
		)
	} else {
		rows := make([]string, len(m.projects))
		for i, p := range m.projects {
			rows[i] = renderRow(p, i == m.cursor)
		}
		body = strings.Join(rows, "\n")
	}

	footer := keyHints
	switch {
	case m.mode == modeConfirmDelete && m.cursor < len(m.projects):
		prompt := confirmPromptStyle.Render(fmt.Sprintf("Delete %q? (y/n)", m.projects[m.cursor].Name))
		footer = lipgloss.JoinVertical(lipgloss.Left, prompt, "", footer)
	case m.status != "":
		footer = lipgloss.JoinVertical(lipgloss.Left, theme.SubtleStyle.Render(m.status), "", footer)
	}

	panel := theme.Panel(lipgloss.JoinVertical(lipgloss.Left, header, "", body, "", footer), m.width)
	return m.centered(lipgloss.JoinVertical(lipgloss.Center, logo, "", panel))
}

// centered places content in the middle of the terminal once its size
// is known (via a tea.WindowSizeMsg reaching Update — see
// internal/ui/root.go for why the dashboard reliably receives one even
// though it isn't the first screen shown), falling back to returning
// content unplaced if the size isn't known yet.
func (m Model) centered(content string) string {
	if m.width == 0 || m.height == 0 {
		return content
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

// favoriteStarStyle renders an unselected row's favorite star in
// theme.Accent, so favorited projects stand out in the app's yellow
// brand accent. It's only applied to unselected rows: embedding a
// separately-colored substring inside a string that's about to be
// wrapped in another Render call (as selected rows are, in
// theme.TitleStyle) would let the star's own ANSI reset code cut off
// the outer style partway through the line.
var favoriteStarStyle = lipgloss.NewStyle().Foreground(theme.Accent)

// stackStyle renders an unselected row's tech-stack tags in
// theme.Primary (violet), the same safe-to-embed reasoning as
// favoriteStarStyle above: unselected rows aren't re-wrapped in another
// Render call, so it's safe to color just this substring. Without this,
// every unselected row was a single flat, unstyled color — this and the
// Accent-colored star are what give a row its two intentional brand
// colors instead of relying on the terminal's own default foreground.
var stackStyle = lipgloss.NewStyle().Foreground(theme.Primary)

// renderRow formats one project's list entry: a selection marker, its
// favorite star (if any), name, tech stack (if any), path, and a Docker
// badge (if HasDocker). The whole row is styled with the theme's title
// style when selected, so the active row visually stands out.
func renderRow(p project.Project, selected bool) string {
	marker := "  "
	if selected {
		marker = "> "
	}

	starGlyph := " "
	if p.IsFavorite {
		starGlyph = "★"
	}

	stack := ""
	if len(p.TechStack) > 0 {
		stack = " [" + strings.Join(p.TechStack, ", ") + "]"
	}

	docker := ""
	if p.HasDocker {
		docker = " 🐳"
	}

	if selected {
		line := fmt.Sprintf("%s%s %-24s%s  %s%s", marker, starGlyph, p.Name, stack, p.Path, docker)
		return theme.TitleStyle.Render(line)
	}

	fav := starGlyph
	if p.IsFavorite {
		fav = favoriteStarStyle.Render(starGlyph)
	}
	if stack != "" {
		stack = stackStyle.Render(stack)
	}
	return fmt.Sprintf("%s%s %-24s%s  %s%s", marker, fav, p.Name, stack, p.Path, docker)
}
