package theme

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestGradientPreservesLineContentAndCount(t *testing.T) {
	lines := []string{"AAA", "BBB", "CCC"}

	got := Gradient(lines, lipgloss.Color("#7C3AED"), lipgloss.Color("#FACC15"))

	gotLines := strings.Split(got, "\n")
	if len(gotLines) != len(lines) {
		t.Fatalf("expected %d lines, got %d", len(lines), len(gotLines))
	}
	for i, want := range lines {
		if !strings.Contains(gotLines[i], want) {
			t.Errorf("line %d = %q, want it to contain %q", i, gotLines[i], want)
		}
	}
}

func TestGradientHandlesSingleLine(t *testing.T) {
	got := Gradient([]string{"solo"}, lipgloss.Color("#7C3AED"), lipgloss.Color("#FACC15"))

	if !strings.Contains(got, "solo") {
		t.Errorf("expected output to contain %q, got %q", "solo", got)
	}
}

func TestGradientFallsBackToPlainTextForInvalidColor(t *testing.T) {
	lines := []string{"one", "two"}

	got := Gradient(lines, lipgloss.Color("not-a-color"), lipgloss.Color("#FACC15"))

	want := strings.Join(lines, "\n")
	if got != want {
		t.Errorf("expected plain fallback %q, got %q", want, got)
	}
}
