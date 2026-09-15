package dashboard

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	appPkg "github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/app"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/config"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/project"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/shellexec"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/projectform"
)

// newTestApp builds a real App backed by a disposable temp registry, the
// same pattern internal/app's own tests use, so the dashboard is tested
// against real App behavior rather than a mock of it.
func newTestApp(t *testing.T) *appPkg.App {
	t.Helper()

	dir := t.TempDir()
	cfg := &config.Config{
		Storage: config.StorageConfig{Path: filepath.Join(dir, "projects.json")},
		Runner:  config.RunnerConfig{Shell: "sh"},
	}

	a, err := appPkg.NewApp(cfg, shellexec.NewFakeExecutor())
	if err != nil {
		t.Fatalf("failed to construct App: %v", err)
	}
	return a
}

func TestInitLoadsProjects(t *testing.T) {
	a := newTestApp(t)
	if _, err := a.AddProject("alpha", t.TempDir()); err != nil {
		t.Fatalf("failed to add project: %v", err)
	}
	if _, err := a.AddProject("beta", t.TempDir()); err != nil {
		t.Fatalf("failed to add project: %v", err)
	}

	m := New(a)
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("expected a non-nil load command")
	}

	msg, ok := cmd().(projectsLoadedMsg)
	if !ok {
		t.Fatalf("expected projectsLoadedMsg, got %T", cmd())
	}
	if msg.err != nil {
		t.Fatalf("did not expect error but got %v", msg.err)
	}
	if len(msg.projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(msg.projects))
	}
}

func TestUpdateHandlesProjectsLoadedMsg(t *testing.T) {
	m := New(newTestApp(t))

	got, _ := m.Update(projectsLoadedMsg{projects: []project.Project{{Name: "alpha"}, {Name: "beta"}}})

	model := got.(Model)
	if len(model.projects) != 2 {
		t.Errorf("expected 2 projects loaded into state, got %d", len(model.projects))
	}
}

func TestUpdateCursorMovement(t *testing.T) {
	m := New(newTestApp(t))
	m.projects = []project.Project{{Name: "a"}, {Name: "b"}, {Name: "c"}}

	// Moving down twice should land on index 2.
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	got, _ = got.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	model := got.(Model)
	if model.cursor != 2 {
		t.Fatalf("expected cursor 2 after two downs, got %d", model.cursor)
	}

	// A third "down" should not move past the last item.
	got, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	model = got.(Model)
	if model.cursor != 2 {
		t.Errorf("expected cursor to stay at 2 (bounds check), got %d", model.cursor)
	}

	// Moving up should decrement, and never go below zero.
	got, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	got, _ = got.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	got, _ = got.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	model = got.(Model)
	if model.cursor != 0 {
		t.Errorf("expected cursor to stop at 0 (bounds check), got %d", model.cursor)
	}
}

func TestUpdateQuitsOnQ(t *testing.T) {
	m := New(newTestApp(t))

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatal("expected a quit command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", cmd())
	}
}

func TestAKeyOpensAddProjectScreen(t *testing.T) {
	m := New(newTestApp(t))

	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})

	if _, ok := got.(projectform.Model); !ok {
		t.Fatalf("expected a to open projectform.Model, got %T", got)
	}
	if cmd == nil {
		t.Error("expected a non-nil Init command from the new screen")
	}
}

func TestEKeyOpensEditProjectScreen(t *testing.T) {
	a := newTestApp(t)
	p, err := a.AddProject("alpha", t.TempDir())
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}

	m := New(a)
	m.projects = []project.Project{*p}

	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})

	if _, ok := got.(projectform.Model); !ok {
		t.Fatalf("expected e to open projectform.Model, got %T", got)
	}
	if cmd == nil {
		t.Error("expected a non-nil Init command from the new screen")
	}
}

func TestEKeyDoesNothingWhenListIsEmpty(t *testing.T) {
	m := New(newTestApp(t))

	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})

	if _, ok := got.(Model); !ok {
		t.Fatalf("expected e on an empty list to leave the dashboard active, got %T", got)
	}
	if cmd != nil {
		t.Error("expected no command when there's nothing to edit")
	}
}

func TestViewShowsEmptyState(t *testing.T) {
	m := New(newTestApp(t))

	view := m.View()
	if !strings.Contains(view, "No projects yet") {
		t.Error("expected empty-state message in view")
	}
	if !strings.Contains(view, "add a project") {
		t.Error("expected empty-state view to explain how to add a project")
	}
}

func TestViewShowsErrorState(t *testing.T) {
	m := New(newTestApp(t))
	got, _ := m.Update(projectsLoadedMsg{err: errors.New("boom")})

	if !strings.Contains(got.View(), "failed to load projects") {
		t.Error("expected error message in view")
	}
}

func TestFKeyTogglesFavorite(t *testing.T) {
	a := newTestApp(t)
	p, err := a.AddProject("alpha", t.TempDir())
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}

	m := New(a)
	m.projects = []project.Project{*p}

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	model := got.(Model)
	if !model.projects[0].IsFavorite {
		t.Fatal("expected project to be marked favorite after pressing f")
	}

	stored, err := a.Projects().GetProjectByID(p.ID)
	if err != nil {
		t.Fatalf("failed to reload project: %v", err)
	}
	if !stored.IsFavorite {
		t.Error("expected the favorite flag to be persisted, not just held in memory")
	}

	got, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	model = got.(Model)
	if model.projects[0].IsFavorite {
		t.Error("expected a second f to unmark the favorite")
	}
}

func TestDKeyEntersConfirmModeAndYDeletes(t *testing.T) {
	a := newTestApp(t)
	p, err := a.AddProject("alpha", t.TempDir())
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}

	m := New(a)
	m.projects = []project.Project{*p}

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	model := got.(Model)
	if model.mode != modeConfirmDelete {
		t.Fatal("expected d to enter delete-confirmation mode")
	}
	if !strings.Contains(model.View(), "Delete \"alpha\"?") {
		t.Errorf("expected a confirmation prompt naming the project, got %q", model.View())
	}

	got, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	model = got.(Model)
	if model.mode != modeList {
		t.Error("expected confirming a delete to return to list mode")
	}
	if len(model.projects) != 0 {
		t.Errorf("expected the project to be removed from the list, got %d remaining", len(model.projects))
	}

	remaining, err := a.Projects().GetAllProjects()
	if err != nil {
		t.Fatalf("failed to list projects: %v", err)
	}
	if len(remaining) != 0 {
		t.Errorf("expected the project to be deleted from storage, got %d remaining", len(remaining))
	}
}

func TestDKeyThenNCancelsWithoutDeleting(t *testing.T) {
	a := newTestApp(t)
	p, err := a.AddProject("alpha", t.TempDir())
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}

	m := New(a)
	m.projects = []project.Project{*p}

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	got, _ = got.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	model := got.(Model)

	if model.mode != modeList {
		t.Error("expected n to return to list mode")
	}
	if len(model.projects) != 1 {
		t.Errorf("expected the project to remain, got %d", len(model.projects))
	}
}

func TestViewShowsProjectsWithFavoriteAndDockerMarkers(t *testing.T) {
	m := New(newTestApp(t))
	m.projects = []project.Project{
		{Name: "alpha", Path: "/projects/alpha", IsFavorite: true, HasDocker: true},
		{Name: "beta", Path: "/projects/beta"},
	}

	view := m.View()

	if !strings.Contains(view, "alpha") || !strings.Contains(view, "beta") {
		t.Error("expected both project names in view")
	}
	if !strings.Contains(view, "★") {
		t.Error("expected a favorite marker for alpha")
	}
	if !strings.Contains(view, "🐳") {
		t.Error("expected a docker badge for alpha")
	}
}
