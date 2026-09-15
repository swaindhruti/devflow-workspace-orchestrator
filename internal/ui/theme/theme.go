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

// BrandGradientFrom and BrandGradientTo are the two endpoint colors of
// DevFlow's wordmark gradient — the same vivid purple fading into a
// warm yellow used by the splash screen's block-art "DEVFLOW" banner
// (internal/ui/splash) — exported here so any screen that wants to
// reuse that exact wordmark (e.g. a small logo above the dashboard
// panel) renders it identically rather than approximating it with a
// slightly different pair of hex values. These are lipgloss.Color, not
// AdaptiveColor, for the same reason Gradient's from/to parameters are:
// blending needs concrete RGB to interpolate through.
var (
	BrandGradientFrom = lipgloss.Color("#7C3AED")
	BrandGradientTo   = lipgloss.Color("#FACC15")
)

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

// keyStyle renders a single keybinding's key (e.g. "a") in Accent, bold,
// so it reads as a distinct, tappable-looking label rather than plain
// text — used by KeyHints.
var keyStyle = lipgloss.NewStyle().Foreground(Accent).Bold(true)

// keyHintSeparator visually divides one "key description" pair from the
// next in a KeyHints line.
var keyHintSeparator = SubtleStyle.Render("   ")

// KeyHints renders a row of keybinding hints — pairs of (key,
// description), e.g. {"a", "add"} — as a single line, each key
// highlighted in Accent and each description in Muted, for a screen's
// help/footer line. This replaces spelling out raw key names in plain
// text (e.g. "a add · f favorite") with a more scannable, visually
// distinct "key → action" format.
//
// Parameters:
//   - pairs: each entry's first string is the key as the user would
//     press it (e.g. "↑/k", "esc"), the second is a short description
//     of what it does.
//
// Returns the fully rendered, styled line.
func KeyHints(pairs [][2]string) string {
	parts := make([]string, len(pairs))
	for i, pair := range pairs {
		parts[i] = keyStyle.Render(pair[0]) + " " + SubtleStyle.Render(pair[1])
	}
	return strings.Join(parts, keyHintSeparator)
}

// PanelStyle frames a screen's main content in a rounded border using
// Primary, so data/form screens (the dashboard, add-project, and
// whatever follows) read as one cohesive panel instead of bare
// left-aligned text floating in the terminal. It intentionally sets no
// Foreground/Background/Bold of its own: those would wrap the entire
// panel content in one more layer of ANSI styling, and content passed
// to PanelStyle.Render is typically already a mix of independently
// pre-styled substrings (a title, colored rows, key hints) — an outer
// style with its own text color would risk one of those inner
// segments' reset codes cutting it off partway through. Border and
// Padding, by contrast, don't wrap the text itself in SGR codes at all,
// so this composes safely with already-styled content.
var PanelStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(Primary).
	Padding(1, 3)

// Panel renders content inside PanelStyle, capping the whole panel
// (border and padding included) to maxWidth terminal columns whenever
// maxWidth is positive. Without this, a panel's width is simply
// whichever of its content lines is longest — usually fine, but a long
// key-hints line or a wide file listing can easily be wider than a
// small terminal, and PanelStyle alone has no way to know how wide the
// terminal actually is and wrap to fit. Screens should call this
// (passing their tracked terminal width) instead of calling
// PanelStyle.Render directly.
//
// This uses Style.Width, not Style.MaxWidth: lipgloss applies Width
// *before* drawing the border (word-wrapping content to fit, then
// adding the border around the wrapped result), whereas MaxWidth
// truncates the already-bordered output as a flat string — which chops
// the right border character off whenever a content line was too long,
// leaving the panel looking unclosed. Width needs the border's own
// width subtracted first so the *bordered* total comes out to
// maxWidth, not the interior alone.
//
// Parameters:
//   - content: the panel's already-composed, already-styled body.
//   - maxWidth: the terminal's known width in columns, or 0/negative
//     if not yet known (in which case the panel is left at its natural
//     content width, same as calling PanelStyle.Render directly).
func Panel(content string, maxWidth int) string {
	style := PanelStyle
	if maxWidth > 0 {
		if contentWidth := maxWidth - style.GetHorizontalBorderSize(); contentWidth > 0 {
			style = style.Width(contentWidth)
		}
	}
	return style.Render(content)
}

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
