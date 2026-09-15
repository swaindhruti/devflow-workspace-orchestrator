// Package detail implements DevFlow's project detail screen: a
// single-project view opened from the dashboard, showing everything
// about one registered project in one place — its static fields, live
// Git status, and Docker containers/images (via app.ProjectContext).
// Saved-command management (listing, adding, removing, running one) is
// added in a later change, once this screen's data display is in
// place.
package detail

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

	width, height int
}

// New constructs a detail screen for p.
//
// Parameters:
//   - a: the wired application layer ProjectContext is fetched through.
//   - back: the screen to return to on "esc". Model calls back.Init()
//     itself at the moment of transition, the same way every other
//     screen.Screen transition in this codebase works (see e.g.
//     internal/ui/splash).
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

// Update tracks the terminal size (from tea.WindowSizeMsg) so View can
// center the panel, stores the result of a contextLoadedMsg, and
// returns to m.back on "esc". Every other message is ignored.
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
// tech stack (if any), and favorite status — followed by its live Git
// and Docker sections (see renderGitSection/renderDockerSection): a
// loading notice until contextLoadedMsg arrives, an error line if it
// failed outright, or the sections themselves once ctx is populated —
// plus the keybinding legend.
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

	lines = append(lines, "", keyHints)

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return m.centered(theme.Panel(content, m.width))
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
