package splash

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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

func TestInitStartsMascotAnimationTicker(t *testing.T) {
	m := New(&fakeScreen{})

	if cmd := m.Init(); cmd == nil {
		t.Error("expected Init to return a Cmd that starts the corner mascots' animation ticker")
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

func TestUpdateAdvancesMascotFrameOnTickAndReschedules(t *testing.T) {
	m := New(&fakeScreen{})

	got, cmd := m.Update(frameMsg{})

	model, ok := got.(Model)
	if !ok {
		t.Fatalf("expected Update to return Model, got %T", got)
	}
	if model.frame != 1 {
		t.Errorf("expected frame to advance from 0 to 1, got %d", model.frame)
	}
	if cmd == nil {
		t.Error("expected Update to reschedule the next animation tick")
	}
}

func TestUpdateWrapsMascotFrameAroundFrameCount(t *testing.T) {
	m := New(&fakeScreen{})
	m.frame = len(mascotBitmapFrames) - 1

	got, _ := m.Update(frameMsg{})

	model := got.(Model)
	if model.frame != 0 {
		t.Errorf("expected frame to wrap around to 0, got %d", model.frame)
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

// mascotTopRowPrefix is the expansion of mascotBitmapFrames[0]'s first
// row's leading tab ("##" -> "████"), used below to detect whether a
// corner mascot was actually placed at the very start of the view.
const mascotTopRowPrefix = "████"

func TestViewIncludesDomainMascotsWhenTerminalLargeEnough(t *testing.T) {
	m := New(&fakeScreen{})

	sized, _ := m.Update(tea.WindowSizeMsg{Width: minWidthForMascots, Height: minHeightForMascots})
	view := sized.(Model).View()

	lines := strings.Split(view, "\n")
	if len(lines) != minHeightForMascots {
		t.Fatalf("expected mascotted view to still fill the terminal height (%d lines), got %d", minHeightForMascots, len(lines))
	}
	if !strings.HasPrefix(lines[0], mascotTopRowPrefix) {
		t.Errorf("expected the top-left corner mascot at the start of line 0, got %q", lines[0])
	}
}

func TestViewOmitsDomainMascotsWhenTerminalTooSmall(t *testing.T) {
	m := New(&fakeScreen{})

	sized, _ := m.Update(tea.WindowSizeMsg{Width: minWidthForMascots - 1, Height: minHeightForMascots})
	view := sized.(Model).View()

	lines := strings.Split(view, "\n")
	if strings.HasPrefix(lines[0], mascotTopRowPrefix) {
		t.Error("expected no corner mascot on a terminal narrower than minWidthForMascots")
	}
	if !strings.Contains(view, "press any key") {
		t.Error("expected the plain centered content to still render")
	}
}

func TestRenderMascotHasFixedDimensions(t *testing.T) {
	got := renderMascot(0, lipgloss.NewStyle())

	lines := strings.Split(got, "\n")
	if len(lines) != mascotHeight {
		t.Fatalf("expected %d lines, got %d", mascotHeight, len(lines))
	}
	for i, line := range lines {
		if w := lipgloss.Width(line); w != mascotWidth {
			t.Errorf("line %d: expected width %d, got %d (%q)", i, mascotWidth, w, line)
		}
	}
}

func TestRenderMascotFramesDiffer(t *testing.T) {
	idle := renderMascot(0, lipgloss.NewStyle())
	blinkWave := renderMascot(1, lipgloss.NewStyle())

	if idle == blinkWave {
		t.Error("expected the idle and blink/wave frames to render differently")
	}
}
