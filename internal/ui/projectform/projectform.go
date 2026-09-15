// Package projectform implements DevFlow's add-project and edit-project
// flows, sharing one screen since they differ only in how they start and
// how they save: adding is a two-step flow that first lets the user
// browse the filesystem to pick a project directory (via
// bubbles/filepicker) before filling in the project's name,
// description, and tech stack and registering it; editing skips
// straight to that same name/description/tags form, prefilled from an
// existing project whose path can't be changed, and saves over it
// instead of registering a new one.
package projectform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/app"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/project"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/screen"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/theme"
)

// step is which half of the add-project flow is currently showing.
type step int

const (
	// stepBrowse shows the directory picker, for choosing the
	// project's path.
	stepBrowse step = iota
	// stepDetails shows the name/description/tags form, for
	// everything else about the project.
	stepDetails
)

// Field indexes into Model.inputs, and into the label/help text that
// corresponds to each one.
const (
	fieldName = iota
	fieldDescription
	fieldTags
	fieldCount
)

var fieldLabels = [fieldCount]string{
	fieldName:        "Name",
	fieldDescription: "Description",
	fieldTags:        "Tags",
}

// Model is the add-project/edit-project screen.
type Model struct {
	app  *app.App
	back screen.Screen

	// editing is nil when this Model is registering a new project.
	// Non-nil, it holds the project being edited: submit saves over it
	// (preserving every field the form doesn't expose — IsFavorite,
	// HasDocker, DockerIdentifier, RunCommands, and so on) instead of
	// calling app.AddProject, and stepBrowse is never entered since the
	// path of an already-registered project isn't editable here.
	editing *project.Project

	step   step
	picker filepicker.Model

	inputs [fieldCount]textinput.Model
	focus  int

	path string
	err  error

	width, height int
}

// newFilePicker builds the directory picker shared by NewAdd's stepBrowse.
func newFilePicker() filepicker.Model {
	fp := filepicker.New()
	// Both left false: Enter always just navigates into a directory,
	// never "selects" one. bubbles/filepicker's own Select/Open
	// keybindings are the same key (enter), so with DirAllowed true
	// the very first directory a user opened would also register as
	// their final choice, with no way to browse further in. Model
	// implements its own explicit "choose this directory" key ("s",
	// see updateBrowse) instead, using m.picker.CurrentDirectory
	// directly rather than bubbles' own selection mechanism.
	fp.DirAllowed = false
	fp.FileAllowed = false
	if home, err := os.UserHomeDir(); err == nil {
		fp.CurrentDirectory = home
	}
	// AutoHeight (the default) sizes the picker off the raw terminal
	// height alone, with no way to account for the panel border,
	// padding, and instructional text View wraps around it — left as
	// default, the picker would claim more rows than are actually left
	// once that chrome is in place. Update takes over sizing explicitly
	// instead (see browseChromeHeight).
	fp.AutoHeight = false
	// Recolor the picker to DevFlow's own violet/yellow palette instead
	// of bubbles' default magenta, so it reads as part of the same app
	// rather than a bolted-on third-party widget.
	fp.Styles.Directory = lipgloss.NewStyle().Foreground(theme.Primary)
	fp.Styles.Cursor = lipgloss.NewStyle().Foreground(theme.Accent)
	fp.Styles.Selected = lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
	return fp
}

// newInputs builds the shared name/description/tags fields, optionally
// prefilled from an existing project (pass nil for a blank add-project
// form).
func newInputs(prefill *project.Project) [fieldCount]textinput.Model {
	var inputs [fieldCount]textinput.Model
	inputs[fieldName] = textinput.New()
	inputs[fieldName].Placeholder = "my-project"
	inputs[fieldName].CharLimit = 64

	inputs[fieldDescription] = textinput.New()
	inputs[fieldDescription].Placeholder = "optional"
	inputs[fieldDescription].CharLimit = 200

	inputs[fieldTags] = textinput.New()
	inputs[fieldTags].Placeholder = "optional, comma separated — e.g. go, docker"
	inputs[fieldTags].CharLimit = 200

	if prefill != nil {
		inputs[fieldName].SetValue(prefill.Name)
		inputs[fieldDescription].SetValue(prefill.Description)
		inputs[fieldTags].SetValue(strings.Join(prefill.TechStack, ", "))
	}

	return inputs
}

// NewAdd constructs an add-project screen, starting on stepBrowse.
//
// Parameters:
//   - a: the wired application layer AddProject registers the new
//     project through.
//   - back: the screen to return to once the flow finishes — either
//     because the user cancelled (esc/"q"), or because the project was
//     registered successfully. Model calls back.Init() itself at the
//     moment of transition, the same way every other screen.Screen
//     transition in this codebase works (see e.g. internal/ui/splash).
func NewAdd(a *app.App, back screen.Screen) Model {
	return Model{
		app:    a,
		back:   back,
		step:   stepBrowse,
		picker: newFilePicker(),
		inputs: newInputs(nil),
	}
}

// NewEdit constructs an edit-project screen for an already-registered
// project, starting straight on stepDetails (prefilled from p) since
// there's no path to browse to — an existing project's path isn't
// editable here.
//
// Parameters:
//   - a: the wired application layer UpdateProject saves changes
//     through.
//   - back: the screen to return to once the flow finishes, same
//     contract as NewAdd's back parameter.
//   - p: the project to edit. A copy is kept (see Model.editing) so
//     submit can save over every field the form doesn't expose.
func NewEdit(a *app.App, back screen.Screen, p project.Project) Model {
	inputs := newInputs(&p)
	inputs[fieldName].Focus()

	return Model{
		app:     a,
		back:    back,
		editing: &p,
		step:    stepDetails,
		inputs:  inputs,
		path:    p.Path,
	}
}

// Init kicks off the directory picker's initial listing in add mode. In
// edit mode there's no picker to load — stepBrowse is never entered —
// so Init instead starts the name field's cursor blinking: NewEdit
// already focused it (a struct field mutation, which sticks, unlike a
// method call on Init's own value receiver), but the blink animation
// itself is driven by a recurring tea.Cmd that has to be returned from
// somewhere Bubble Tea actually schedules it, which Init is.
func (m Model) Init() tea.Cmd {
	if m.editing != nil {
		return textinput.Blink
	}
	return m.picker.Init()
}

// browseChromeHeight is how many terminal rows the browse step's
// non-picker chrome consumes: theme.PanelStyle's border (2) and
// vertical padding (2), plus the title, subtitle, "Browsing: <path>"
// line, their surrounding blank lines, and the help line View adds
// around the picker itself. Update subtracts this from the known
// terminal height to size the picker (see picker.SetHeight below) so
// the whole panel fits within the terminal instead of the picker
// claiming rows that are actually spoken for by the chrome around it.
const browseChromeHeight = 14

// Update dispatches to updateBrowse or updateDetails depending on
// m.step, after handling the one thing both steps share: tracking the
// terminal size (and, since the picker's AutoHeight is off, sizing the
// picker to fit within it — see browseChromeHeight) so View can lay out
// the whole panel without overflowing the terminal.
func (m Model) Update(msg tea.Msg) (screen.Screen, tea.Cmd) {
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width, m.height = sizeMsg.Width, sizeMsg.Height
		if h := m.height - browseChromeHeight; h >= 3 {
			m.picker.SetHeight(h)
		} else {
			m.picker.SetHeight(3)
		}
	}

	if m.step == stepBrowse {
		return m.updateBrowse(msg)
	}
	return m.updateDetails(msg)
}

// updateBrowse handles input while the directory picker is showing.
// "q" cancels back to m.back; "s" chooses the picker's current
// directory as the project path and advances to stepDetails, prefilling
// the name field from the directory's base name; every other message
// (navigation keys, the picker's own async directory-listing messages,
// window resizes) is forwarded to the embedded filepicker.Model.
func (m Model) updateBrowse(msg tea.Msg) (screen.Screen, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "q":
			return m.back, m.back.Init()

		case "s":
			m.path = m.picker.CurrentDirectory
			m.step = stepDetails
			m.inputs[fieldName].SetValue(filepath.Base(m.path))
			return m, m.inputs[fieldName].Focus()
		}
	}

	var cmd tea.Cmd
	m.picker, cmd = m.picker.Update(msg)
	return m, cmd
}

// updateDetails handles input while the name/description/tags form is
// showing. Tab/shift+tab move focus between fields; enter moves focus
// forward too, except on the last field, where it submits; esc cancels
// back to m.back. Any other key is forwarded to the focused
// textinput.Model.
func (m Model) updateDetails(msg tea.Msg) (screen.Screen, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "esc":
		return m.back, m.back.Init()

	case "tab", "shift+tab":
		return m, m.moveFocus(keyMsg.String() == "tab")

	case "enter":
		if m.focus == fieldCount-1 {
			return m.submit()
		}
		return m, m.moveFocus(true)
	}

	var cmd tea.Cmd
	m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
	return m, cmd
}

// moveFocus blurs the currently-focused input, advances (or, if
// forward is false, retreats) m.focus by one slot with wraparound, and
// focuses the new one, returning the tea.Cmd textinput.Focus produces
// (its cursor-blink command).
func (m *Model) moveFocus(forward bool) tea.Cmd {
	m.inputs[m.focus].Blur()
	if forward {
		m.focus = (m.focus + 1) % fieldCount
	} else {
		m.focus = (m.focus - 1 + fieldCount) % fieldCount
	}
	return m.inputs[m.focus].Focus()
}

// submit saves the form. In edit mode (m.editing != nil) it saves a
// copy of the original project with only Name/Description/TechStack
// overwritten from the form, via UpdateProject — every other field
// (IsFavorite, HasDocker, DockerIdentifier, RunCommands, ...) is
// preserved exactly as it was, since the form never exposed them and
// silently wiping them on every edit would be a surprising way to lose
// data. In add mode it registers the project via app.AddProject using
// the picked path and the name field, then — only if description or
// tags were filled in, since AddProject itself only takes a name and a
// path — patches those in with a follow-up UpdateProject.
//
// Either way, on success it hands off to m.back; on any failure (empty
// name, AddProject/UpdateProject erroring — e.g. a duplicate path the
// validator doesn't already catch) it stays on stepDetails and shows
// the error inline instead, so the user's already-typed input isn't
// lost.
func (m Model) submit() (screen.Screen, tea.Cmd) {
	name := strings.TrimSpace(m.inputs[fieldName].Value())
	if name == "" {
		m.err = fmt.Errorf("name is required")
		return m, nil
	}
	description := strings.TrimSpace(m.inputs[fieldDescription].Value())
	tags := parseTags(m.inputs[fieldTags].Value())

	if m.editing != nil {
		updated := *m.editing
		updated.Name = name
		updated.Description = description
		updated.TechStack = tags
		if err := m.app.Projects().UpdateProject(&updated); err != nil {
			m.err = err
			return m, nil
		}
		return m.back, m.back.Init()
	}

	p, err := m.app.AddProject(name, m.path)
	if err != nil {
		m.err = err
		return m, nil
	}
	if description != "" || len(tags) > 0 {
		p.Description = description
		p.TechStack = tags
		if err := m.app.Projects().UpdateProject(p); err != nil {
			m.err = err
			return m, nil
		}
	}

	return m.back, m.back.Init()
}

// parseTags splits a comma-separated tags string into a trimmed,
// non-empty slice — "go, docker ,, " becomes ["go" "docker"], and an
// all-blank input becomes nil rather than a slice of empty strings.
func parseTags(raw string) []string {
	parts := strings.Split(raw, ",")
	tags := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			tags = append(tags, trimmed)
		}
	}
	return tags
}

// browseKeyHints and detailsKeyHints are each step's full keybinding
// legend — the answer to "how do I use this screen" for a first-time
// user, shown at the bottom of the panel.
var (
	browseKeyHints = theme.KeyHints([][2]string{
		{"↑/k ↓/j", "move"},
		{"enter/l", "open"},
		{"h/esc", "up a dir"},
		{"s", "choose this directory"},
		{"q", "cancel"},
	})
	detailsKeyHints = theme.KeyHints([][2]string{
		{"tab/shift+tab", "move field"},
		{"enter", "next field / submit"},
		{"esc", "cancel"},
	})
)

// View renders stepBrowse's directory picker or stepDetails' form,
// whichever is current, as a single bordered, centered panel (see
// theme.PanelStyle) — matching the splash screen's centered first
// impression, and the dashboard's own panel, rather than either step
// floating against the top-left corner.
func (m Model) View() string {
	titleText := "DevFlow — Add Project"
	if m.editing != nil {
		titleText = "DevFlow — Edit Project"
	}
	title := theme.TitleStyle.Render(titleText)

	var content string
	if m.step == stepBrowse {
		content = lipgloss.JoinVertical(lipgloss.Left,
			title,
			theme.SubtleStyle.Render("Step 1 of 2 — browse to your project's folder."),
			"",
			theme.SubtleStyle.Render("Browsing: "+m.picker.CurrentDirectory), "",
			m.picker.View(), "",
			browseKeyHints,
		)
	} else {
		stepLabel := "Step 2 of 2 — only the name is required."
		if m.editing != nil {
			stepLabel = "Update the details below, then submit on the last field."
		}

		var form strings.Builder
		for i, in := range m.inputs {
			marker := "  "
			if i == m.focus {
				marker = "> "
			}
			fmt.Fprintf(&form, "%s%-12s %s\n", marker, fieldLabels[i]+":", in.View())
		}

		sections := []string{
			title,
			theme.SubtleStyle.Render(stepLabel),
			"",
			theme.SubtleStyle.Render("Path: " + m.path), "",
			form.String(),
		}
		if m.err != nil {
			sections = append(sections, lipgloss.NewStyle().Foreground(theme.Danger).Render(fmt.Sprintf("error: %v", m.err)), "")
		}
		sections = append(sections, detailsKeyHints)

		content = lipgloss.JoinVertical(lipgloss.Left, sections...)
	}

	return m.centered(theme.Panel(content, m.width))
}

// centered places content in the middle of the terminal once its size
// is known (via a tea.WindowSizeMsg reaching Update — see
// internal/ui/root.go for why this screen reliably receives one even
// though it's only ever reached mid-session, not shown at startup),
// falling back to returning content unplaced if the size isn't known
// yet.
func (m Model) centered(content string) string {
	if m.width == 0 || m.height == 0 {
		return content
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}
