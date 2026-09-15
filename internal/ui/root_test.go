package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/screen"
)

// fakeScreen is a minimal screen.Screen test double for exercising
// rootModel in isolation, without depending on any real screen package.
type fakeScreen struct {
	updateCalled bool
	viewText     string
	nextOnUpdate screen.Screen // if set, Update returns this instead of itself
	lastSize     tea.WindowSizeMsg
	sawSize      bool
}

func (f *fakeScreen) Init() tea.Cmd { return nil }

func (f *fakeScreen) Update(msg tea.Msg) (screen.Screen, tea.Cmd) {
	f.updateCalled = true
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		f.lastSize = sizeMsg
		f.sawSize = true
	}
	if f.nextOnUpdate != nil {
		return f.nextOnUpdate, nil
	}
	return f, nil
}

func (f *fakeScreen) View() string { return f.viewText }

func TestRootModelInitDelegatesToActiveScreen(t *testing.T) {
	m := newRootModel(&fakeScreen{})

	if cmd := m.Init(); cmd != nil {
		t.Errorf("expected nil Cmd from fakeScreen.Init, got %v", cmd)
	}
}

func TestRootModelQuitsOnCtrlC(t *testing.T) {
	active := &fakeScreen{}
	m := newRootModel(active)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("expected a quit command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", cmd())
	}
	if active.updateCalled {
		t.Error("ctrl+c should be handled by root before reaching the active screen")
	}
}

func TestRootModelDelegatesOtherMessages(t *testing.T) {
	active := &fakeScreen{}
	m := newRootModel(active)

	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if !active.updateCalled {
		t.Error("expected the active screen's Update to be called")
	}
}

func TestRootModelAdoptsScreenReturnedByUpdate(t *testing.T) {
	next := &fakeScreen{viewText: "next screen"}
	active := &fakeScreen{nextOnUpdate: next}
	m := newRootModel(active)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	rm, ok := updated.(rootModel)
	if !ok {
		t.Fatalf("expected Update to return a rootModel, got %T", updated)
	}
	if rm.View() != "next screen" {
		t.Errorf("expected root to adopt the screen returned by Update, view = %q", rm.View())
	}
}

func TestRootModelResendsKnownSizeToScreenAdoptedOnKeypress(t *testing.T) {
	next := &fakeScreen{viewText: "next screen"}
	active := &fakeScreen{}
	m := newRootModel(active)

	sized, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = sized.(rootModel)
	active.nextOnUpdate = next

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	rm := updated.(rootModel)

	if _, ok := rm.active.(*fakeScreen); !ok {
		t.Fatalf("expected the root to have adopted next, got %T", rm.active)
	}
	if !next.sawSize {
		t.Fatal("expected the newly-adopted screen to receive a synthetic WindowSizeMsg")
	}
	if next.lastSize.Width != 100 || next.lastSize.Height != 40 {
		t.Errorf("expected the last known size (100x40) to be re-delivered, got %dx%d", next.lastSize.Width, next.lastSize.Height)
	}
}

func TestRootModelDoesNotResendSizeBeforeAnyIsKnown(t *testing.T) {
	next := &fakeScreen{}
	active := &fakeScreen{nextOnUpdate: next}
	m := newRootModel(active)

	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if next.sawSize {
		t.Error("expected no synthetic WindowSizeMsg when the terminal size is still unknown")
	}
}

func TestRootModelViewDelegatesToActiveScreen(t *testing.T) {
	m := newRootModel(&fakeScreen{viewText: "hello"})

	if m.View() != "hello" {
		t.Errorf("View() = %q, want %q", m.View(), "hello")
	}
}
