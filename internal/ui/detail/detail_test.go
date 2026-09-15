package detail

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/project"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/screen"
)

// fakeBack is a minimal screen.Screen test double standing in for "the
// screen to return to," so tests can confirm esc hands off to it (and
// calls its Init) without depending on dashboard.
type fakeBack struct {
	initCalled bool
}

func (f *fakeBack) Init() tea.Cmd                           { f.initCalled = true; return nil }
func (f *fakeBack) Update(tea.Msg) (screen.Screen, tea.Cmd) { return f, nil }
func (f *fakeBack) View() string                            { return "back" }

func TestInitReturnsNilCommand(t *testing.T) {
	m := New(nil, &fakeBack{}, project.Project{Name: "alpha"})

	if cmd := m.Init(); cmd != nil {
		t.Errorf("expected nil Cmd (nothing to load yet), got %v", cmd)
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
