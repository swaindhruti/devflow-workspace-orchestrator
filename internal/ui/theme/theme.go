// Package theme defines the shared Lip Gloss color tokens and base styles
// every DevFlow screen builds its rendering from, so views stay visually
// consistent without each one re-picking colors independently. Colors are
// lipgloss.AdaptiveColor values (a light-terminal color and a
// dark-terminal color per token) so the UI reads correctly in either
// terminal background, since DevFlow has no way to know which a given
// user's terminal uses.
package theme

import "github.com/charmbracelet/lipgloss"

// Primary is DevFlow's accent color, used for the banner, active
// selections, and anything that should draw the eye first.
var Primary = lipgloss.AdaptiveColor{Light: "#0057B8", Dark: "#4FC3F7"}

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
