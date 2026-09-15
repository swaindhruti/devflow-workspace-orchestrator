package detail

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	appPkg "github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/app"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/config"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/docker"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/git"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/project"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/shellexec"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/screen"
)

// newTestApp builds a real App backed by a disposable temp registry,
// the same pattern internal/ui/dashboard's tests use, so the parts of
// this screen that go through App are tested against real App behavior
// rather than a mock of it.
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

// fakeBack is a minimal screen.Screen test double standing in for "the
// screen to return to," so tests can confirm esc hands off to it (and
// calls its Init) without depending on dashboard.
type fakeBack struct {
	initCalled bool
}

func (f *fakeBack) Init() tea.Cmd                           { f.initCalled = true; return nil }
func (f *fakeBack) Update(tea.Msg) (screen.Screen, tea.Cmd) { return f, nil }
func (f *fakeBack) View() string                            { return "back" }

func TestInitReturnsLoadContextCommand(t *testing.T) {
	m := New(newTestApp(t), &fakeBack{}, project.Project{Name: "alpha"})

	if cmd := m.Init(); cmd == nil {
		t.Error("expected Init to return the context-loading command")
	}
}

func TestInitCommandReportsProjectContext(t *testing.T) {
	a := newTestApp(t)
	p, err := a.AddProject("alpha", t.TempDir())
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}

	m := New(a, &fakeBack{}, *p)
	cmd := m.Init()

	msg, ok := cmd().(contextLoadedMsg)
	if !ok {
		t.Fatalf("expected contextLoadedMsg, got %T", cmd())
	}
	if msg.err != nil {
		t.Fatalf("did not expect an error, got %v", msg.err)
	}
	if msg.ctx.Project.ID != p.ID {
		t.Errorf("expected the context's project to match, got ID %q", msg.ctx.Project.ID)
	}
}

func TestUpdateStoresLoadedContext(t *testing.T) {
	m := New(newTestApp(t), &fakeBack{}, project.Project{Name: "alpha"})

	got, cmd := m.Update(contextLoadedMsg{ctx: appPkg.ProjectContext{Project: project.Project{Name: "alpha"}}})

	model, ok := got.(Model)
	if !ok {
		t.Fatalf("expected Update to return Model, got %T", got)
	}
	if model.ctx == nil {
		t.Fatal("expected ctx to be stored")
	}
	if cmd != nil {
		t.Error("expected no follow-up command")
	}
}

func TestUpdateStoresContextLoadError(t *testing.T) {
	m := New(newTestApp(t), &fakeBack{}, project.Project{Name: "alpha"})

	got, _ := m.Update(contextLoadedMsg{err: errors.New("boom")})

	model := got.(Model)
	if model.ctxErr == nil {
		t.Fatal("expected ctxErr to be stored")
	}
	if model.ctx != nil {
		t.Error("expected ctx to remain nil after a load error")
	}
}

func TestEscReturnsToBack(t *testing.T) {
	back := &fakeBack{}
	m := New(nil, back, project.Project{Name: "alpha"})

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if got != screen.Screen(back) {
		t.Fatal("expected esc to return the back screen")
	}
	if !back.initCalled {
		t.Error("expected the back screen's Init to have been called")
	}
}

func TestUpdateIgnoresOtherKeys(t *testing.T) {
	back := &fakeBack{}
	m := New(nil, back, project.Project{Name: "alpha"})

	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	if _, ok := got.(Model); !ok {
		t.Fatalf("expected an unrelated key to keep the detail screen active, got %T", got)
	}
	if cmd != nil {
		t.Error("expected no command for an unrelated key")
	}
	if back.initCalled {
		t.Error("did not expect the back screen to be activated for an unrelated key")
	}
}

func TestUpdateTracksWindowSize(t *testing.T) {
	m := New(nil, &fakeBack{}, project.Project{Name: "alpha"})

	got, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	model, ok := got.(Model)
	if !ok {
		t.Fatalf("expected Update to return Model, got %T", got)
	}
	if model.width != 80 || model.height != 24 {
		t.Errorf("expected width/height 80/24, got %d/%d", model.width, model.height)
	}
}

func TestViewShowsNamePathAndKeyHints(t *testing.T) {
	m := New(nil, &fakeBack{}, project.Project{Name: "alpha", Path: "/projects/alpha"})

	view := m.View()
	if !strings.Contains(view, "alpha") {
		t.Error("expected the project name in view")
	}
	if !strings.Contains(view, "/projects/alpha") {
		t.Error("expected the project path in view")
	}
	if !strings.Contains(view, "back") {
		t.Error("expected the esc-back key hint in view")
	}
}

func TestViewShowsDescriptionAndStackWhenPresent(t *testing.T) {
	m := New(nil, &fakeBack{}, project.Project{
		Name:        "alpha",
		Description: "a test project",
		TechStack:   []string{"go", "docker"},
	})

	view := m.View()
	if !strings.Contains(view, "a test project") {
		t.Error("expected the description in view")
	}
	if !strings.Contains(view, "go, docker") {
		t.Error("expected the tech stack in view")
	}
}

func TestViewOmitsDescriptionAndStackWhenAbsent(t *testing.T) {
	m := New(nil, &fakeBack{}, project.Project{Name: "alpha"})

	view := m.View()
	if strings.Contains(view, "Stack:") {
		t.Error("expected no stack line when TechStack is empty")
	}
}

func TestViewShowsFavoriteStatus(t *testing.T) {
	favorited := New(nil, &fakeBack{}, project.Project{Name: "alpha", IsFavorite: true})
	if !strings.Contains(favorited.View(), "Favorited") {
		t.Error("expected a favorited project to show its favorite status")
	}

	notFavorited := New(nil, &fakeBack{}, project.Project{Name: "alpha", IsFavorite: false})
	if !strings.Contains(notFavorited.View(), "Not favorited") {
		t.Error("expected a non-favorited project to say so")
	}
}

func TestViewShowsLoadingBeforeContextArrives(t *testing.T) {
	m := New(nil, &fakeBack{}, project.Project{Name: "alpha"})

	if !strings.Contains(m.View(), "Loading") {
		t.Error("expected a loading notice before contextLoadedMsg arrives")
	}
}

func TestViewShowsContextLoadError(t *testing.T) {
	m := New(nil, &fakeBack{}, project.Project{Name: "alpha"})
	m.ctxErr = errors.New("boom")

	if !strings.Contains(m.View(), "failed to load project context") {
		t.Error("expected the load error to be shown")
	}
}

func TestViewShowsGitAndDockerSectionsOnceLoaded(t *testing.T) {
	p := project.Project{Name: "alpha", HasDocker: true}
	m := New(nil, &fakeBack{}, p)
	m.ctx = &appPkg.ProjectContext{
		Project:    p,
		GitStatus:  &git.Status{Branch: "main", Clean: true},
		Containers: []docker.Container{{Name: "alpha-web", Status: "Up 2 hours", State: "running"}},
		Images:     []docker.Image{{Repository: "alpha-web", Tag: "latest", Size: "245MB"}},
	}

	view := m.View()
	if !strings.Contains(view, "Git") || !strings.Contains(view, "main") || !strings.Contains(view, "Clean") {
		t.Errorf("expected the Git section to render branch and clean status, got %q", view)
	}
	if !strings.Contains(view, "Docker") || !strings.Contains(view, "alpha-web") || !strings.Contains(view, "245MB") {
		t.Errorf("expected the Docker section to render container and image info, got %q", view)
	}
}

func TestViewOmitsDockerSectionWhenNotConfigured(t *testing.T) {
	p := project.Project{Name: "alpha", HasDocker: false}
	m := New(nil, &fakeBack{}, p)
	m.ctx = &appPkg.ProjectContext{Project: p, GitStatus: &git.Status{Branch: "main", Clean: true}}

	if strings.Contains(m.View(), "Docker") {
		t.Error("expected no Docker section when the project has no Docker setup")
	}
}

func TestRenderGitSectionNotARepo(t *testing.T) {
	got := renderGitSection(appPkg.ProjectContext{})

	if !strings.Contains(got, "Not a Git repository") {
		t.Errorf("expected a not-a-repo message, got %q", got)
	}
}

func TestRenderGitSectionDetachedHead(t *testing.T) {
	got := renderGitSection(appPkg.ProjectContext{GitStatus: &git.Status{Detached: true, Clean: true}})

	if !strings.Contains(got, "detached HEAD") {
		t.Errorf("expected a detached-HEAD label, got %q", got)
	}
}

func TestRenderGitSectionDirtyShowsCounts(t *testing.T) {
	got := renderGitSection(appPkg.ProjectContext{
		GitStatus: &git.Status{Branch: "main", Staged: 1, Unstaged: 2, Untracked: 3},
	})

	if !strings.Contains(got, "Dirty") || !strings.Contains(got, "1 staged") || !strings.Contains(got, "2 unstaged") || !strings.Contains(got, "3 untracked") {
		t.Errorf("expected dirty status with counts, got %q", got)
	}
}

func TestRenderGitSectionShowsAheadBehindOnlyWhenNonzero(t *testing.T) {
	clean := renderGitSection(appPkg.ProjectContext{GitStatus: &git.Status{Branch: "main", Clean: true}})
	if strings.Contains(clean, "Ahead") {
		t.Errorf("expected no ahead/behind line when both are zero, got %q", clean)
	}

	ahead := renderGitSection(appPkg.ProjectContext{GitStatus: &git.Status{Branch: "main", Clean: true, Ahead: 2, Behind: 1}})
	if !strings.Contains(ahead, "Ahead 2, behind 1") {
		t.Errorf("expected an ahead/behind line, got %q", ahead)
	}
}

func TestRenderDockerSectionShowsPlaceholdersWhenEmpty(t *testing.T) {
	got := renderDockerSection(appPkg.ProjectContext{})

	if !strings.Contains(got, "No containers found") || !strings.Contains(got, "No images found") {
		t.Errorf("expected empty-state placeholders, got %q", got)
	}
}
