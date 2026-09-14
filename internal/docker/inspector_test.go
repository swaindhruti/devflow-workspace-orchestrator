package docker

import (
	"errors"
	"strings"
	"testing"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/shellexec"
)

const samplePSOutput = `{"ID":"abc123","Names":"myapp-web-1","Image":"myapp-web:latest","Status":"Up 2 hours","State":"running","Ports":"0.0.0.0:8080->80/tcp","CreatedAt":"2026-09-10 12:00:00 +0000 UTC"}
{"ID":"def456","Names":"myapp-db-1","Image":"postgres:16","Status":"Exited (0) 3 minutes ago","State":"exited","Ports":"","CreatedAt":"2026-09-10 12:00:00 +0000 UTC"}
`

const sampleImagesOutput = `{"ID":"img1","Repository":"myapp-web","Tag":"latest","Size":"245MB","CreatedAt":"2026-09-10 12:00:00 +0000 UTC"}
`

func TestListContainersParsesAndMaps(t *testing.T) {
	exec := shellexec.NewFakeExecutor()
	exec.Responses["docker"] = shellexec.Response{Stdout: []byte(samplePSOutput)}

	containers, err := ListContainers(exec, "myapp", true)
	if err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}

	if len(containers) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(containers))
	}

	want := Container{
		ID: "abc123", Name: "myapp-web-1", Image: "myapp-web:latest",
		Status: "Up 2 hours", State: "running", Ports: "0.0.0.0:8080->80/tcp",
		CreatedAt: "2026-09-10 12:00:00 +0000 UTC",
	}
	if containers[0] != want {
		t.Errorf("first container = %+v, want %+v", containers[0], want)
	}
}

func TestListContainersFilterConstruction(t *testing.T) {
	tests := []struct {
		name          string
		composeScoped bool
		wantFilter    string
	}{
		{name: "compose scoped uses label filter", composeScoped: true, wantFilter: "label=com.docker.compose.project=myapp"},
		{name: "manual identifier uses name filter", composeScoped: false, wantFilter: "name=myapp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := shellexec.NewFakeExecutor()
			exec.Responses["docker"] = shellexec.Response{Stdout: []byte("")}

			if _, err := ListContainers(exec, "myapp", tt.composeScoped); err != nil {
				t.Fatalf("did not expect error but got %v", err)
			}

			if len(exec.Calls) != 1 {
				t.Fatalf("expected 1 call, got %d", len(exec.Calls))
			}
			if !containsArg(exec.Calls[0].Args, tt.wantFilter) {
				t.Errorf("expected args to contain filter %q, got %v", tt.wantFilter, exec.Calls[0].Args)
			}
		})
	}
}

func TestListContainersReturnsEmptySliceForNoMatches(t *testing.T) {
	exec := shellexec.NewFakeExecutor()
	exec.Responses["docker"] = shellexec.Response{Stdout: []byte("\n")}

	containers, err := ListContainers(exec, "myapp", true)
	if err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}
	if len(containers) != 0 {
		t.Errorf("expected no containers, got %d", len(containers))
	}
}

func TestListContainersWrapsExecutorError(t *testing.T) {
	exec := shellexec.NewFakeExecutor()
	exec.Responses["docker"] = shellexec.Response{
		Stderr: []byte("Cannot connect to the Docker daemon"),
		Err:    errors.New("exit status 1"),
	}

	_, err := ListContainers(exec, "myapp", true)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "Cannot connect to the Docker daemon") {
		t.Errorf("expected error to include docker's stderr, got %v", err)
	}
}

func TestListContainersReturnsErrorOnMalformedOutput(t *testing.T) {
	exec := shellexec.NewFakeExecutor()
	exec.Responses["docker"] = shellexec.Response{Stdout: []byte("not-json")}

	_, err := ListContainers(exec, "myapp", true)
	if err == nil {
		t.Fatal("expected a parse error, got nil")
	}
}

func TestListImagesParsesAndMaps(t *testing.T) {
	exec := shellexec.NewFakeExecutor()
	exec.Responses["docker"] = shellexec.Response{Stdout: []byte(sampleImagesOutput)}

	images, err := ListImages(exec, "myapp", true)
	if err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}

	if len(images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(images))
	}

	want := Image{ID: "img1", Repository: "myapp-web", Tag: "latest", Size: "245MB", CreatedAt: "2026-09-10 12:00:00 +0000 UTC"}
	if images[0] != want {
		t.Errorf("image = %+v, want %+v", images[0], want)
	}
}

func TestListImagesFilterConstruction(t *testing.T) {
	tests := []struct {
		name          string
		composeScoped bool
		wantFilter    string
	}{
		{name: "compose scoped uses label filter", composeScoped: true, wantFilter: "label=com.docker.compose.project=myapp"},
		{name: "manual identifier uses reference filter", composeScoped: false, wantFilter: "reference=myapp*"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := shellexec.NewFakeExecutor()
			exec.Responses["docker"] = shellexec.Response{Stdout: []byte("")}

			if _, err := ListImages(exec, "myapp", tt.composeScoped); err != nil {
				t.Fatalf("did not expect error but got %v", err)
			}

			if len(exec.Calls) != 1 {
				t.Fatalf("expected 1 call, got %d", len(exec.Calls))
			}
			if !containsArg(exec.Calls[0].Args, tt.wantFilter) {
				t.Errorf("expected args to contain filter %q, got %v", tt.wantFilter, exec.Calls[0].Args)
			}
		})
	}
}

// containsArg reports whether want is present anywhere in args, used to
// assert a filter value was passed to the docker CLI without depending on
// its exact position among the other arguments.
func containsArg(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}
