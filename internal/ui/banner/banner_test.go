package banner

import "testing"

func TestRenderSingleLetterExactShape(t *testing.T) {
	// 'L' is simple enough to hand-verify by eye: a vertical bar down the
	// left edge, then a full bottom row.
	got := Render("L", "#", ".")
	want := "#....\n" +
		"#....\n" +
		"#....\n" +
		"#....\n" +
		"#....\n" +
		"#....\n" +
		"#####"

	if got != want {
		t.Errorf("Render(\"L\") =\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderProducesRectangularBlock(t *testing.T) {
	lines := splitLines(Render("DEVFLOW", "#", "."))

	if len(lines) != glyphHeight {
		t.Fatalf("expected %d lines, got %d", glyphHeight, len(lines))
	}

	width := len(lines[0])
	for i, line := range lines {
		if len(line) != width {
			t.Errorf("line %d has length %d, want %d (all lines must align)", i, len(line), width)
		}
	}
}

func TestRenderUnsupportedRuneFallsBackToBlank(t *testing.T) {
	got := Render("A", "#", ".")
	want := Render(" ", "#", ".")

	if got != want {
		t.Errorf("unsupported rune should render identically to a space, got:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderUsesProvidedTokens(t *testing.T) {
	got := Render("O", "X", "_")

	for _, r := range got {
		if r != 'X' && r != '_' && r != '\n' {
			t.Fatalf("expected output to only contain the provided filled/empty tokens, found %q in:\n%s", r, got)
		}
	}
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, r := range s {
		if r == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}
