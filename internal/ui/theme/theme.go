// Package theme defines the shared Lip Gloss color tokens and base styles
// every DevFlow screen builds its rendering from, so views stay visually
// consistent without each one re-picking colors independently. Colors are
// lipgloss.AdaptiveColor values (a light-terminal color and a
// dark-terminal color per token) so the UI reads correctly in either
// terminal background, since DevFlow has no way to know which a given
// user's terminal uses.
package theme

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	colorful "github.com/lucasb-eyer/go-colorful"
)

// Primary is DevFlow's accent color, used for active selections, titles,
// and anything that should draw the eye first. It's the same violet
// family as the splash screen's banner gradient (internal/ui/splash),
// so the rest of the app reads as a continuation of that first
// impression rather than switching palettes once the splash hands off.
var Primary = lipgloss.AdaptiveColor{Light: "#6D28D9", Dark: "#A78BFA"}

// Accent is DevFlow's secondary brand color — the warm yellow half of
// the splash's violet-to-yellow gradient — used sparingly for
// highlights that should stand out from Primary itself (e.g. a
// favorite marker, a call-to-action), not as a replacement for it.
var Accent = lipgloss.AdaptiveColor{Light: "#A16207", Dark: "#FACC15"}

// Muted is used for secondary text: descriptions, hints, and anything
// that should visually recede behind Primary content.
var Muted = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}

// Success indicates a healthy/clean/running state (e.g. a clean Git
// working tree, a running container).
var Success = lipgloss.AdaptiveColor{Light: "#15803D", Dark: "#4ADE80"}

// Warning indicates a state worth noticing but not alarming (e.g. an
// unstaged change, a stopped container).
var Warning = lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#FBBF24"}

// Danger indicates a failure state (e.g. a failed command, an
// unreachable project path).
var Danger = lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#F87171"}

// TitleStyle renders a screen's primary heading.
var TitleStyle = lipgloss.NewStyle().Bold(true).Foreground(Primary)

// SubtleStyle renders secondary or hint text.
var SubtleStyle = lipgloss.NewStyle().Foreground(Muted)

// HelpStyle renders keybinding hints, typically pinned to the bottom of a
// screen.
var HelpStyle = lipgloss.NewStyle().Foreground(Muted).Italic(true)

// Gradient renders each line of lines in a color smoothly interpolated
// between from and to — the first line closest to from, the last line
// closest to to — for decorative multi-line text such as a banner.
//
// Parameters:
//   - lines: the text to render, one entry per line, already laid out
//     (e.g. banner.Render's output split on "\n").
//   - from: the color the first line is rendered closest to. Must be a
//     hex color (e.g. "#7C3AED"); AdaptiveColor's single-value ANSI/hex
//     forms don't apply here, since blending needs concrete RGB to
//     interpolate through — a gradient is a decorative flourish, not
//     body text, so a fixed truecolor pair (rather than full light/dark
//     adaptive blending) is an acceptable simplification.
//   - to: the color the last line is rendered closest to, same
//     constraints as from.
//
// Returns the styled lines joined by "\n". If from or to is not a valid
// hex color, Gradient falls back to returning lines unstyled rather than
// rendering everything in a meaningless color.
func Gradient(lines []string, from, to lipgloss.Color) string {
	start, errFrom := colorful.Hex(string(from))
	end, errTo := colorful.Hex(string(to))
	if errFrom != nil || errTo != nil {
		return strings.Join(lines, "\n")
	}

	styled := make([]string, len(lines))
	for i, line := range lines {
		t := 0.0
		if len(lines) > 1 {
			t = float64(i) / float64(len(lines)-1)
		}
		blended := start.BlendLuv(end, t)
		styled[i] = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(blended.Hex())).Render(line)
	}

	return strings.Join(styled, "\n")
}
