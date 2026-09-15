// Package detail implements DevFlow's project detail screen: a
// single-project view opened from the dashboard, showing everything
// about one registered project in one place. This first version shows
// only the project's already-known static fields (name, path,
// description, tech stack, favorite status); live Git status, Docker
// containers/images, and saved-command management are added in later
// changes, once this screen's navigation and layout are in place.
package detail

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/app"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/project"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/screen"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/theme"
)

// Model is the project detail screen.
type Model struct {
	app     *app.App
	back    screen.Screen
	project project.Project

	width, height int
}

// New constructs a detail screen for p.
//
// Parameters:
//   - a: the wired application layer — unused by this first version,
//     kept so later changes (fetching git/Docker context) don't need to
//     change every call site's signature.
//   - back: the screen to return to on "esc". Model calls back.Init()
//     itself at the moment of transition, the same way every other
//     screen.Screen transition in this codebase works (see e.g.
//     internal/ui/splash).
//   - p: the project to show details for.
func New(a *app.App, back screen.Screen, p project.Project) Model {
	return Model{app: a, back: back, project: p}
}

// Init has nothing to kick off yet: this first version renders only
// data it already has.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update tracks the terminal size (from tea.WindowSizeMsg) so View can
// center the panel, and returns to m.back on "esc". Every other message
// is ignored.
func (m Model) Update(msg tea.Msg) (screen.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "esc" {
			return m.back, m.back.Init()
		}
		return m, nil

	default:
		return m, nil
	}
}

// keyHints is this screen's keybinding legend.
var keyHints = theme.KeyHints([][2]string{
	{"esc", "back"},
})

// View renders the project's static fields as a single bordered,
// centered panel (see theme.Panel) — name, path, description (if any),
// tech stack (if any), and favorite status — plus the keybinding
// legend.
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
	lines = append(lines, "", favorite, "", keyHints)

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return m.centered(theme.Panel(content, m.width))
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
