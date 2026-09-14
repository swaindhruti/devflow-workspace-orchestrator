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
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/screen"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/theme"
)

// Model is the project dashboard screen.
type Model struct {
	app           *app.App
	projects      []project.Project
	cursor        int
	err           error
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

// Update handles the project-loaded result and keyboard navigation
// (up/k, down/j to move the selection, q to quit).
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
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.projects)-1 {
				m.cursor++
			}
		case "q":
			return m, tea.Quit
		}
		return m, nil

	default:
		return m, nil
	}
}

// View renders the project list, or an empty-state or error message in
// place of it when there is nothing to show yet.
func (m Model) View() string {
	title := theme.TitleStyle.Render("DevFlow — Projects")

	if m.err != nil {
		return lipgloss.JoinVertical(lipgloss.Left, title, "",
			theme.SubtleStyle.Render(fmt.Sprintf("failed to load projects: %v", m.err)))
	}

	if len(m.projects) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left, title, "",
			theme.SubtleStyle.Render("No projects registered yet."))
	}

	rows := make([]string, len(m.projects))
	for i, p := range m.projects {
		rows[i] = renderRow(p, i == m.cursor)
	}

	help := theme.HelpStyle.Render("↑/k up · ↓/j down · q quit")

	return lipgloss.JoinVertical(lipgloss.Left, title, "", strings.Join(rows, "\n"), "", help)
}

// renderRow formats one project's list entry: a selection marker, its
// favorite star (if any), name, tech stack (if any), path, and a Docker
// badge (if HasDocker). The whole row is styled with the theme's title
// style when selected, so the active row visually stands out.
func renderRow(p project.Project, selected bool) string {
	marker := "  "
	if selected {
		marker = "> "
	}

	fav := " "
	if p.IsFavorite {
		fav = "★"
	}

	stack := ""
	if len(p.TechStack) > 0 {
		stack = " [" + strings.Join(p.TechStack, ", ") + "]"
	}

	docker := ""
	if p.HasDocker {
		docker = " 🐳"
	}

	line := fmt.Sprintf("%s%s %-24s%s  %s%s", marker, fav, p.Name, stack, p.Path, docker)

	if selected {
		return theme.TitleStyle.Render(line)
	}
	return line
}
