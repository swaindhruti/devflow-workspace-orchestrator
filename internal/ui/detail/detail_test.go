package detail

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	appPkg "github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/app"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/config"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/docker"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/git"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/project"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/shellexec"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/ui/screen"
)

// newTestApp builds a real App backed by a disposable temp registry,
// the same pattern internal/ui/dashboard's tests use, so the parts of
// this screen that go through App are tested against real App behavior
// rather than a mock of it.
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

// fakeBack is a minimal screen.Screen test double standing in for "the
// screen to return to," so tests can confirm esc hands off to it (and
// calls its Init) without depending on dashboard.
type fakeBack struct {
	initCalled bool
}

func (f *fakeBack) Init() tea.Cmd                           { f.initCalled = true; return nil }
func (f *fakeBack) Update(tea.Msg) (screen.Screen, tea.Cmd) { return f, nil }
func (f *fakeBack) View() string                            { return "back" }

// switchSection sends one "right" keypress, advancing m.section by one
// (with wraparound) the same way a user pressing → would.
func switchSection(t *testing.T, m Model) Model {
	t.Helper()
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	model, ok := got.(Model)
	if !ok {
		t.Fatalf("expected Update to return Model, got %T", got)
	}
	return model
}

func TestInitReturnsLoadContextCommand(t *testing.T) {
	m := New(newTestApp(t), &fakeBack{}, project.Project{Name: "alpha"})

	if cmd := m.Init(); cmd == nil {
		t.Error("expected Init to return the context-loading command")
	}
}

func TestInitCommandReportsProjectContext(t *testing.T) {
	a := newTestApp(t)
	p, err := a.AddProject("alpha", t.TempDir())
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}

	m := New(a, &fakeBack{}, *p)
	cmd := m.Init()

	msg, ok := cmd().(contextLoadedMsg)
	if !ok {
		t.Fatalf("expected contextLoadedMsg, got %T", cmd())
	}
	if msg.err != nil {
		t.Fatalf("did not expect an error, got %v", msg.err)
	}
	if msg.ctx.Project.ID != p.ID {
		t.Errorf("expected the context's project to match, got ID %q", msg.ctx.Project.ID)
	}
}

func TestUpdateStoresLoadedContext(t *testing.T) {
	m := New(newTestApp(t), &fakeBack{}, project.Project{Name: "alpha"})

	got, cmd := m.Update(contextLoadedMsg{ctx: appPkg.ProjectContext{Project: project.Project{Name: "alpha"}}})

	model, ok := got.(Model)
	if !ok {
		t.Fatalf("expected Update to return Model, got %T", got)
	}
	if model.ctx == nil {
		t.Fatal("expected ctx to be stored")
	}
	if cmd != nil {
		t.Error("expected no follow-up command")
	}
}

func TestUpdateStoresContextLoadError(t *testing.T) {
	m := New(newTestApp(t), &fakeBack{}, project.Project{Name: "alpha"})

	got, _ := m.Update(contextLoadedMsg{err: errors.New("boom")})

	model := got.(Model)
	if model.ctxErr == nil {
		t.Fatal("expected ctxErr to be stored")
	}
	if model.ctx != nil {
		t.Error("expected ctx to remain nil after a load error")
	}
}

func TestEscReturnsToBack(t *testing.T) {
	back := &fakeBack{}
	m := New(nil, back, project.Project{Name: "alpha"})

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if got != screen.Screen(back) {
		t.Fatal("expected esc to return the back screen")
	}
	if !back.initCalled {
		t.Error("expected the back screen's Init to have been called")
	}
}

func TestUpdateIgnoresOtherKeys(t *testing.T) {
	back := &fakeBack{}
	m := New(nil, back, project.Project{Name: "alpha"})

	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	if _, ok := got.(Model); !ok {
		t.Fatalf("expected an unrelated key to keep the detail screen active, got %T", got)
	}
	if cmd != nil {
		t.Error("expected no command for an unrelated key")
	}
	if back.initCalled {
		t.Error("did not expect the back screen to be activated for an unrelated key")
	}
}

func TestUpdateTracksWindowSize(t *testing.T) {
	m := New(nil, &fakeBack{}, project.Project{Name: "alpha"})

	got, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	model, ok := got.(Model)
	if !ok {
		t.Fatalf("expected Update to return Model, got %T", got)
	}
	if model.width != 80 || model.height != 24 {
		t.Errorf("expected width/height 80/24, got %d/%d", model.width, model.height)
	}
}

func TestViewShowsNamePathAndKeyHints(t *testing.T) {
	m := New(nil, &fakeBack{}, project.Project{Name: "alpha", Path: "/projects/alpha"})

	view := m.View()
	if !strings.Contains(view, "alpha") {
		t.Error("expected the project name in view")
	}
	if !strings.Contains(view, "/projects/alpha") {
		t.Error("expected the project path in view")
	}
	if !strings.Contains(view, "back") {
		t.Error("expected the esc-back key hint in view")
	}
}

func TestViewShowsDescriptionAndStackWhenPresent(t *testing.T) {
	m := New(nil, &fakeBack{}, project.Project{
		Name:        "alpha",
		Description: "a test project",
		TechStack:   []string{"go", "docker"},
	})

	view := m.View()
	if !strings.Contains(view, "a test project") {
		t.Error("expected the description in view")
	}
	if !strings.Contains(view, "go, docker") {
		t.Error("expected the tech stack in view")
	}
}

func TestViewOmitsDescriptionAndStackWhenAbsent(t *testing.T) {
	m := New(nil, &fakeBack{}, project.Project{Name: "alpha"})

	view := m.View()
	if strings.Contains(view, "Stack:") {
		t.Error("expected no stack line when TechStack is empty")
	}
}

func TestViewShowsFavoriteStatus(t *testing.T) {
	favorited := New(nil, &fakeBack{}, project.Project{Name: "alpha", IsFavorite: true})
	if !strings.Contains(favorited.View(), "Favorited") {
		t.Error("expected a favorited project to show its favorite status")
	}

	notFavorited := New(nil, &fakeBack{}, project.Project{Name: "alpha", IsFavorite: false})
	if !strings.Contains(notFavorited.View(), "Not favorited") {
		t.Error("expected a non-favorited project to say so")
	}
}

func TestViewShowsLoadingBeforeContextArrives(t *testing.T) {
	m := New(nil, &fakeBack{}, project.Project{Name: "alpha"})
	m = switchSection(t, m) // Overview -> Commands
	m = switchSection(t, m) // Commands -> Git

	if !strings.Contains(m.View(), "Loading") {
		t.Error("expected a loading notice before contextLoadedMsg arrives")
	}
}

func TestViewShowsContextLoadError(t *testing.T) {
	m := New(nil, &fakeBack{}, project.Project{Name: "alpha"})
	m.ctxErr = errors.New("boom")
	m = switchSection(t, m) // Overview -> Commands
	m = switchSection(t, m) // Commands -> Git

	if !strings.Contains(m.View(), "failed to load project context") {
		t.Error("expected the load error to be shown")
	}
}

func TestViewShowsGitSectionOnceLoaded(t *testing.T) {
	p := project.Project{Name: "alpha"}
	m := New(nil, &fakeBack{}, p)
	m.ctx = &appPkg.ProjectContext{
		Project:   p,
		GitStatus: &git.Status{Branch: "main", Clean: true},
	}
	m = switchSection(t, m) // Overview -> Commands
	m = switchSection(t, m) // Commands -> Git

	view := m.View()
	if !strings.Contains(view, "Git") || !strings.Contains(view, "main") || !strings.Contains(view, "Clean") {
		t.Errorf("expected the Git section to render branch and clean status, got %q", view)
	}
}

func TestViewShowsDockerSectionOnceLoaded(t *testing.T) {
	p := project.Project{Name: "alpha", HasDocker: true}
	m := New(nil, &fakeBack{}, p)
	m.ctx = &appPkg.ProjectContext{
		Project:    p,
		Containers: []docker.Container{{Name: "alpha-web", Status: "Up 2 hours", State: "running"}},
		Images:     []docker.Image{{Repository: "alpha-web", Tag: "latest", Size: "245MB"}},
	}
	m = switchSection(t, m) // Overview -> Commands
	m = switchSection(t, m) // Commands -> Git
	m = switchSection(t, m) // Git -> Docker

	view := m.View()
	if !strings.Contains(view, "Docker") || !strings.Contains(view, "alpha-web") || !strings.Contains(view, "245MB") {
		t.Errorf("expected the Docker section to render container and image info, got %q", view)
	}
}

func TestDockerPaneShowsNotConfiguredMessage(t *testing.T) {
	p := project.Project{Name: "alpha", HasDocker: false}
	m := New(nil, &fakeBack{}, p)
	m.ctx = &appPkg.ProjectContext{Project: p, GitStatus: &git.Status{Branch: "main", Clean: true}}
	m = switchSection(t, m) // Overview -> Commands
	m = switchSection(t, m) // Commands -> Git
	m = switchSection(t, m) // Git -> Docker

	view := m.View()
	if !strings.Contains(view, "This project has no Docker") {
		t.Errorf("expected an explanatory message when the project has no Docker setup, got %q", view)
	}
	if strings.Contains(view, "No containers found") {
		t.Error("expected the container/image placeholders not to show when Docker isn't configured at all")
	}
}

func TestRenderGitSectionNotARepo(t *testing.T) {
	got := renderGitSection(appPkg.ProjectContext{})

	if !strings.Contains(got, "Not a Git repository") {
		t.Errorf("expected a not-a-repo message, got %q", got)
	}
}

func TestRenderGitSectionDetachedHead(t *testing.T) {
	got := renderGitSection(appPkg.ProjectContext{GitStatus: &git.Status{Detached: true, Clean: true}})

	if !strings.Contains(got, "detached HEAD") {
		t.Errorf("expected a detached-HEAD label, got %q", got)
	}
}

func TestRenderGitSectionDirtyShowsCounts(t *testing.T) {
	got := renderGitSection(appPkg.ProjectContext{
		GitStatus: &git.Status{Branch: "main", Staged: 1, Unstaged: 2, Untracked: 3},
	})

	if !strings.Contains(got, "Dirty") || !strings.Contains(got, "1 staged") || !strings.Contains(got, "2 unstaged") || !strings.Contains(got, "3 untracked") {
		t.Errorf("expected dirty status with counts, got %q", got)
	}
}

func TestRenderGitSectionShowsAheadBehindOnlyWhenNonzero(t *testing.T) {
	clean := renderGitSection(appPkg.ProjectContext{GitStatus: &git.Status{Branch: "main", Clean: true}})
	if strings.Contains(clean, "Ahead") {
		t.Errorf("expected no ahead/behind line when both are zero, got %q", clean)
	}

	ahead := renderGitSection(appPkg.ProjectContext{GitStatus: &git.Status{Branch: "main", Clean: true, Ahead: 2, Behind: 1}})
	if !strings.Contains(ahead, "Ahead 2, behind 1") {
		t.Errorf("expected an ahead/behind line, got %q", ahead)
	}
}

func TestRenderDockerSectionShowsPlaceholdersWhenEmpty(t *testing.T) {
	got := renderDockerSection(appPkg.ProjectContext{})

	if !strings.Contains(got, "No containers found") || !strings.Contains(got, "No images found") {
		t.Errorf("expected empty-state placeholders, got %q", got)
	}
}

func TestRenderCommandsSectionShowsPlaceholderWhenEmpty(t *testing.T) {
	got := renderCommandsSection(nil, 0)

	if !strings.Contains(got, "Commands") || !strings.Contains(got, "No saved commands") {
		t.Errorf("expected a Commands heading and empty-state placeholder, got %q", got)
	}
}

func TestRenderCommandsSectionListsEachCommand(t *testing.T) {
	got := renderCommandsSection([]project.Command{
		{Name: "test", Command: "go test ./..."},
		{Name: "dev", Command: "npm run dev"},
	}, 0)

	for _, want := range []string{"test", "go test ./...", "dev", "npm run dev"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected commands section to contain %q, got %q", want, got)
		}
	}
}

func TestRenderCommandsSectionHighlightsSelected(t *testing.T) {
	commands := []project.Command{
		{Name: "test", Command: "go test ./..."},
		{Name: "dev", Command: "npm run dev"},
	}

	selectingFirst := renderCommandsSection(commands, 0)
	if !strings.Contains(selectingFirst, "> test:") {
		t.Errorf("expected the first command to be marked selected, got %q", selectingFirst)
	}

	selectingSecond := renderCommandsSection(commands, 1)
	if !strings.Contains(selectingSecond, "> dev:") {
		t.Errorf("expected the second command to be marked selected, got %q", selectingSecond)
	}
}

func TestViewShowsCommandsSection(t *testing.T) {
	p := project.Project{
		Name:        "alpha",
		RunCommands: []project.Command{{Name: "test", Command: "go test ./..."}},
	}
	m := New(nil, &fakeBack{}, p)
	m = switchSection(t, m) // Overview -> Commands

	view := m.View()
	if !strings.Contains(view, "Commands") || !strings.Contains(view, "go test ./...") {
		t.Errorf("expected the view to include the commands section, got %q", view)
	}
}

func TestLeftRightKeysSwitchSection(t *testing.T) {
	m := New(nil, &fakeBack{}, project.Project{Name: "alpha"})

	m = switchSection(t, m)
	if m.section != sectionCommands {
		t.Fatalf("expected one right press to land on sectionCommands, got %v", m.section)
	}

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	model := got.(Model)
	if model.section != sectionOverview {
		t.Errorf("expected left to move back to sectionOverview, got %v", model.section)
	}

	got, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	model = got.(Model)
	if model.section != sectionDocker {
		t.Errorf("expected 'h' from sectionOverview to wrap to sectionDocker, got %v", model.section)
	}

	got, _ = model.Update(tea.KeyMsg{Type: tea.KeyLeft})
	model = got.(Model)
	if model.section != sectionGit {
		t.Errorf("expected left again to move to sectionGit, got %v", model.section)
	}
}

func TestSidebarHighlightsActiveSection(t *testing.T) {
	m := New(nil, &fakeBack{}, project.Project{Name: "alpha"})

	if !strings.Contains(m.renderSidebar(), "› Overview") {
		t.Errorf("expected Overview to be marked active by default, got %q", m.renderSidebar())
	}

	m = switchSection(t, m)
	if !strings.Contains(m.renderSidebar(), "› Commands") {
		t.Errorf("expected Commands to be marked active after switching, got %q", m.renderSidebar())
	}
}

func TestUpDownMoveCommandCursorWithBounds(t *testing.T) {
	p := project.Project{Name: "alpha", RunCommands: []project.Command{
		{Name: "a"}, {Name: "b"}, {Name: "c"},
	}}
	m := New(nil, &fakeBack{}, p)
	m = switchSection(t, m) // Overview -> Commands

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	got, _ = got.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	model := got.(Model)
	if model.cmdCursor != 2 {
		t.Fatalf("expected cursor 2 after two downs, got %d", model.cmdCursor)
	}

	got, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	model = got.(Model)
	if model.cmdCursor != 2 {
		t.Errorf("expected cursor to stay at 2 (bounds check), got %d", model.cmdCursor)
	}

	got, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	model = got.(Model)
	if model.cmdCursor != 1 {
		t.Errorf("expected cursor 1 after one up, got %d", model.cmdCursor)
	}
}

func TestAKeyOpensAddCommandFormFocused(t *testing.T) {
	m := New(nil, &fakeBack{}, project.Project{Name: "alpha"})
	m = switchSection(t, m) // Overview -> Commands, where "a" is active

	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	model := got.(Model)

	if model.mode != modeAddCommand {
		t.Fatal("expected a to open the add-command form")
	}
	if !model.cmdInputs[cmdFieldName].Focused() {
		t.Error("expected the name field to start focused")
	}
	if cmd == nil {
		t.Error("expected a Cmd starting the cursor blink")
	}
	if !strings.Contains(model.View(), "Add Command") {
		t.Error("expected the add-command form heading in view")
	}
}

func TestEscCancelsAddCommandFormWithoutSaving(t *testing.T) {
	m := New(nil, &fakeBack{}, project.Project{Name: "alpha"})
	m = switchSection(t, m) // Overview -> Commands
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	model := got.(Model)

	got, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = got.(Model)

	if model.mode != modeView {
		t.Error("expected esc to return to modeView")
	}
	if len(model.project.RunCommands) != 0 {
		t.Error("expected no command to have been saved")
	}
}

func TestAddCommandFormRejectsEmptyFields(t *testing.T) {
	m := New(newTestApp(t), &fakeBack{}, project.Project{Name: "alpha", ID: "p1"})
	m = switchSection(t, m) // Overview -> Commands
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	model := got.(Model)
	model.cmdFocus = cmdFieldCommand // enter only submits from the last field

	got, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = got.(Model)

	if model.formErr == nil {
		t.Fatal("expected a validation error for empty fields")
	}
	if model.mode != modeAddCommand {
		t.Error("expected to stay on the add-command form after a validation error")
	}
}

func TestAddCommandFormSubmitsAndPersists(t *testing.T) {
	a := newTestApp(t)
	p, err := a.AddProject("alpha", t.TempDir())
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}

	m := New(a, &fakeBack{}, *p)
	m = switchSection(t, m) // Overview -> Commands
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	model := got.(Model)

	for _, r := range "test" {
		got, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		model = got.(Model)
	}
	got, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = got.(Model)
	for _, r := range "go test ./..." {
		got, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		model = got.(Model)
	}

	got, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = got.(Model)

	if model.mode != modeView {
		t.Fatal("expected a successful submit to return to modeView")
	}
	if len(model.project.RunCommands) != 1 || model.project.RunCommands[0].Name != "test" {
		t.Fatalf("expected the command to be added to the in-memory project, got %v", model.project.RunCommands)
	}

	saved, err := a.Projects().GetProjectByID(p.ID)
	if err != nil {
		t.Fatalf("failed to reload project: %v", err)
	}
	if len(saved.RunCommands) != 1 || saved.RunCommands[0].Command != "go test ./..." {
		t.Errorf("expected the command to be persisted, got %v", saved.RunCommands)
	}
	if saved.RunCommands[0].ID == "" {
		t.Error("expected the new command to have a generated ID")
	}
}

func TestDKeyEntersConfirmModeAndYDeletesCommand(t *testing.T) {
	a := newTestApp(t)
	p, err := a.AddProject("alpha", t.TempDir())
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}
	p.AddCommand(project.Command{ID: "cmd-1", ProjectID: p.ID, Name: "test", Command: "go test ./..."})
	if err := a.Projects().UpdateProject(p); err != nil {
		t.Fatalf("failed to seed command: %v", err)
	}

	m := New(a, &fakeBack{}, *p)
	m = switchSection(t, m) // Overview -> Commands

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	model := got.(Model)
	if model.mode != modeConfirmDeleteCommand {
		t.Fatal("expected d to enter delete-confirmation mode")
	}
	if !strings.Contains(model.View(), `Delete command "test"?`) {
		t.Errorf("expected a confirmation prompt naming the command, got %q", model.View())
	}

	got, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	model = got.(Model)

	if model.mode != modeView {
		t.Error("expected confirming a delete to return to modeView")
	}
	if len(model.project.RunCommands) != 0 {
		t.Errorf("expected the command to be removed, got %v", model.project.RunCommands)
	}

	saved, err := a.Projects().GetProjectByID(p.ID)
	if err != nil {
		t.Fatalf("failed to reload project: %v", err)
	}
	if len(saved.RunCommands) != 0 {
		t.Errorf("expected the command to be deleted from storage, got %v", saved.RunCommands)
	}
}

func TestDKeyThenNCancelsWithoutDeletingCommand(t *testing.T) {
	a := newTestApp(t)
	p, err := a.AddProject("alpha", t.TempDir())
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}
	p.AddCommand(project.Command{ID: "cmd-1", ProjectID: p.ID, Name: "test", Command: "go test ./..."})
	if err := a.Projects().UpdateProject(p); err != nil {
		t.Fatalf("failed to seed command: %v", err)
	}

	m := New(a, &fakeBack{}, *p)
	m = switchSection(t, m) // Overview -> Commands
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	got, _ = got.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	model := got.(Model)

	if model.mode != modeView {
		t.Error("expected n to return to modeView")
	}
	if len(model.project.RunCommands) != 1 {
		t.Errorf("expected the command to remain, got %v", model.project.RunCommands)
	}
}

func TestSectionKeyHintsOmitCommandHintsOutsideCommandsSection(t *testing.T) {
	got := sectionKeyHints(sectionOverview, true)

	if strings.Contains(got, "add command") || strings.Contains(got, "select command") || strings.Contains(got, "delete command") {
		t.Errorf("expected no command-specific hints outside sectionCommands, got %q", got)
	}
	if !strings.Contains(got, "switch section") || !strings.Contains(got, "back") {
		t.Errorf("expected the section-switch and back hints regardless, got %q", got)
	}
}

func TestSectionKeyHintsOmitSelectAndDeleteWhenNoCommands(t *testing.T) {
	got := sectionKeyHints(sectionCommands, false)

	if strings.Contains(got, "select command") || strings.Contains(got, "delete command") {
		t.Errorf("expected no select/delete hints when there are no commands, got %q", got)
	}
	if !strings.Contains(got, "add command") {
		t.Errorf("expected the add-command hint regardless, got %q", got)
	}
}

func TestSectionKeyHintsIncludeSelectAndDeleteWhenCommandsExist(t *testing.T) {
	got := sectionKeyHints(sectionCommands, true)

	if !strings.Contains(got, "select command") || !strings.Contains(got, "delete command") {
		t.Errorf("expected select/delete hints when commands exist, got %q", got)
	}
}
