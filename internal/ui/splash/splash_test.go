package splash

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/screen"
)

// fakeScreen is a minimal screen.Screen test double that records whether
// Init was called, so tests can confirm the splash screen actually
// activates the screen it hands control to rather than just returning it.
type fakeScreen struct {
	initCalled bool
}

func (f *fakeScreen) Init() tea.Cmd {
	f.initCalled = true
	return nil
}

func (f *fakeScreen) Update(tea.Msg) (screen.Screen, tea.Cmd) { return f, nil }
func (f *fakeScreen) View() string                            { return "fake" }

func TestInitReturnsTickCommand(t *testing.T) {
	// A tiny duration so this test doesn't block on the real production
	// delay (displayDuration) to observe the tick fire.
	m := Model{next: &fakeScreen{}, duration: time.Millisecond}

	cmd := m.Init()
	if cmd == nil {
		t.Fatal("expected a non-nil command")
	}

	msg := cmd()
	if _, ok := msg.(tickMsg); !ok {
		t.Fatalf("expected tickMsg, got %T", msg)
	}
}

func TestUpdateTransitionsOnKeyPress(t *testing.T) {
	next := &fakeScreen{}
	m := New(next)

	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if got != screen.Screen(next) {
		t.Fatalf("expected Update to return the next screen")
	}
	if !next.initCalled {
		t.Error("expected the next screen's Init to have been called")
	}
	if cmd != nil {
		// New's fakeScreen.Init returns nil, so the propagated Cmd should
		// also be nil.
		t.Errorf("expected nil Cmd from fakeScreen.Init, got %v", cmd)
	}
}

func TestUpdateTransitionsOnTick(t *testing.T) {
	next := &fakeScreen{}
	m := New(next)

	got, _ := m.Update(tickMsg{})

	if got != screen.Screen(next) {
		t.Fatal("expected Update to return the next screen on tickMsg")
	}
	if !next.initCalled {
		t.Error("expected the next screen's Init to have been called")
	}
}

func TestUpdateIgnoresOtherMessages(t *testing.T) {
	next := &fakeScreen{}
	m := New(next)

	got, cmd := m.Update(struct{}{})

	if _, ok := got.(Model); !ok {
		t.Fatalf("expected Update to keep returning the splash Model, got %T", got)
	}
	if cmd != nil {
		t.Errorf("expected nil Cmd for an unrelated message, got %v", cmd)
	}
	if next.initCalled {
		t.Error("did not expect the next screen to be activated for an unrelated message")
	}
}

func TestUpdateTracksWindowSize(t *testing.T) {
	m := New(&fakeScreen{})

	got, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	model, ok := got.(Model)
	if !ok {
		t.Fatalf("expected Update to return Model, got %T", got)
	}
	if model.width != 80 || model.height != 24 {
		t.Errorf("expected width/height 80/24, got %d/%d", model.width, model.height)
	}
}

func TestViewContainsBannerAndHint(t *testing.T) {
	m := New(&fakeScreen{})

	view := m.View()

	if !strings.Contains(view, "press any key") {
		t.Error("expected view to contain the continue hint")
	}
}

func TestViewCentersOnceSizeKnown(t *testing.T) {
	m := New(&fakeScreen{})

	sized, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model := sized.(Model)

	view := model.View()
	lines := strings.Split(view, "\n")
	if len(lines) != 24 {
		t.Errorf("expected the placed view to fill the terminal height (24 lines), got %d", len(lines))
	}
}
