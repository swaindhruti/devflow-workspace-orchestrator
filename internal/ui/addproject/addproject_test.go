package addproject

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	appPkg "github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/app"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/config"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/shellexec"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/screen"
)

// newTestApp builds a real App backed by a disposable temp registry, the
// same pattern internal/ui/dashboard's tests use, so this screen is
// tested against real App behavior rather than a mock of it.
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

// fakeBack is a minimal screen.Screen test double standing in for
// "the screen to return to," so tests can confirm cancel/submit hand
// off to it (and call its Init) without depending on dashboard.
type fakeBack struct {
	initCalled bool
}

func (f *fakeBack) Init() tea.Cmd                           { f.initCalled = true; return nil }
func (f *fakeBack) Update(tea.Msg) (screen.Screen, tea.Cmd) { return f, nil }
func (f *fakeBack) View() string                            { return "back" }

func TestNewStartsOnBrowseStep(t *testing.T) {
	m := New(newTestApp(t), &fakeBack{})

	if m.step != stepBrowse {
		t.Errorf("expected New to start on stepBrowse, got %v", m.step)
	}
}

func TestInitReturnsPickerLoadCommand(t *testing.T) {
	m := New(newTestApp(t), &fakeBack{})

	if cmd := m.Init(); cmd == nil {
		t.Error("expected Init to return the picker's directory-load command")
	}
}

func TestQCancelsFromBrowseStep(t *testing.T) {
	back := &fakeBack{}
	m := New(newTestApp(t), back)

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})

	if got != screen.Screen(back) {
		t.Fatal("expected q to return the back screen")
	}
	if !back.initCalled {
		t.Error("expected the back screen's Init to have been called")
	}
}

func TestSChoosesCurrentDirectoryAndAdvancesToDetails(t *testing.T) {
	m := New(newTestApp(t), &fakeBack{})
	dir := t.TempDir()
	m.picker.CurrentDirectory = dir

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	model := got.(Model)

	if model.step != stepDetails {
		t.Fatal("expected s to advance to stepDetails")
	}
	if model.path != dir {
		t.Errorf("expected path %q, got %q", dir, model.path)
	}
	wantName := filepath.Base(dir)
	if model.inputs[fieldName].Value() != wantName {
		t.Errorf("expected name to be prefilled with %q, got %q", wantName, model.inputs[fieldName].Value())
	}
}

// atDetailsStep returns a Model already past the browse step, as if the
// user had pressed "s" in dir, ready for detail-step tests.
func atDetailsStep(t *testing.T, back screen.Screen) Model {
	t.Helper()
	m := New(newTestApp(t), back)
	m.picker.CurrentDirectory = t.TempDir()
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	return got.(Model)
}

func TestTabCyclesFocusThroughFieldsWithWraparound(t *testing.T) {
	m := atDetailsStep(t, &fakeBack{})

	if m.focus != fieldName {
		t.Fatalf("expected initial focus on fieldName, got %d", m.focus)
	}

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = got.(Model)
	if m.focus != fieldDescription {
		t.Errorf("expected focus to advance to fieldDescription, got %d", m.focus)
	}

	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = got.(Model)
	if m.focus != fieldTags {
		t.Errorf("expected focus to advance to fieldTags, got %d", m.focus)
	}

	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = got.(Model)
	if m.focus != fieldName {
		t.Errorf("expected focus to wrap back to fieldName, got %d", m.focus)
	}

	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = got.(Model)
	if m.focus != fieldTags {
		t.Errorf("expected shift+tab to wrap backward to fieldTags, got %d", m.focus)
	}
}

func TestEscCancelsFromDetailsStep(t *testing.T) {
	back := &fakeBack{}
	m := atDetailsStep(t, back)

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if got != screen.Screen(back) {
		t.Fatal("expected esc to return the back screen")
	}
	if !back.initCalled {
		t.Error("expected the back screen's Init to have been called")
	}
}

func TestSubmitRejectsEmptyName(t *testing.T) {
	m := atDetailsStep(t, &fakeBack{})
	m.inputs[fieldName].SetValue("")
	m.focus = fieldTags // enter only submits from the last field

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := got.(Model)

	if model.err == nil {
		t.Fatal("expected an error for an empty name")
	}
	if model.step != stepDetails {
		t.Error("expected to stay on stepDetails after a validation error")
	}
}

func TestEnterOnNonLastFieldMovesFocusInstead(t *testing.T) {
	m := atDetailsStep(t, &fakeBack{})
	m.inputs[fieldName].SetValue("my-project")

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := got.(Model)

	if model.focus != fieldDescription {
		t.Errorf("expected enter on the name field to move focus to description, got %d", model.focus)
	}
}

func TestSubmitAddsProjectAndReturnsToBack(t *testing.T) {
	a := newTestApp(t)
	back := &fakeBack{}
	dir := t.TempDir()

	m := New(a, back)
	m.picker.CurrentDirectory = dir
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	model := got.(Model)

	model.inputs[fieldName].SetValue("alpha")
	model.inputs[fieldDescription].SetValue("a test project")
	model.inputs[fieldTags].SetValue("go, cli")
	model.focus = fieldTags // simulate having tabbed to the last field

	got, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if got != screen.Screen(back) {
		t.Fatal("expected a successful submit to return the back screen")
	}
	if !back.initCalled {
		t.Error("expected the back screen's Init to have been called")
	}

	projects, err := a.Projects().GetAllProjects()
	if err != nil {
		t.Fatalf("failed to list projects: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project registered, got %d", len(projects))
	}
	p := projects[0]
	if p.Name != "alpha" || p.Path != dir {
		t.Errorf("expected name=alpha path=%q, got name=%q path=%q", dir, p.Name, p.Path)
	}
	if p.Description != "a test project" {
		t.Errorf("expected description to be persisted, got %q", p.Description)
	}
	if len(p.TechStack) != 2 || p.TechStack[0] != "go" || p.TechStack[1] != "cli" {
		t.Errorf("expected tags [go cli], got %v", p.TechStack)
	}
}

func TestParseTags(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"blank", "   ", nil},
		{"single", "go", []string{"go"}},
		{"multiple with spacing", "go, docker ,, cli", []string{"go", "docker", "cli"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTags(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("parseTags(%q) = %v, want %v", tt.in, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseTags(%q)[%d] = %q, want %q", tt.in, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestViewShowsBrowsePathAndHelp(t *testing.T) {
	m := New(newTestApp(t), &fakeBack{})
	dir := t.TempDir()
	m.picker.CurrentDirectory = dir

	view := m.View()
	if !strings.Contains(view, dir) {
		t.Errorf("expected view to show the current browse directory %q", dir)
	}
	if !strings.Contains(view, "choose this directory") {
		t.Error("expected browse-step help text")
	}
}

func TestViewShowsFormFieldsAndError(t *testing.T) {
	m := atDetailsStep(t, &fakeBack{})
	m.err = errors.New("boom")

	view := m.View()
	for _, label := range []string{"Name:", "Description:", "Tags:"} {
		if !strings.Contains(view, label) {
			t.Errorf("expected view to contain field label %q", label)
		}
	}
	if !strings.Contains(view, "error:") {
		t.Error("expected the error to be rendered")
	}
}
