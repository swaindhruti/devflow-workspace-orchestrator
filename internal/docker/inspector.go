package docker

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/shellexec"
)

// dockerPSRow mirrors the JSON keys `docker ps --format '{{json .}}'`
// emits per container, one row per line of output (NDJSON). Field names
// match Docker's own Go template variable names, not Go naming convention,
// since they are dictated by the docker CLI's output format and must match
// exactly for json.Unmarshal to populate them.
type dockerPSRow struct {
	ID        string `json:"ID"`
	Names     string `json:"Names"`
	Image     string `json:"Image"`
	Status    string `json:"Status"`
	State     string `json:"State"`
	Ports     string `json:"Ports"`
	CreatedAt string `json:"CreatedAt"`
}

// dockerImagesRow mirrors the JSON keys `docker images --format
// '{{json .}}'` emits per image, one row per line of output (NDJSON).
type dockerImagesRow struct {
	ID         string `json:"ID"`
	Repository string `json:"Repository"`
	Tag        string `json:"Tag"`
	Size       string `json:"Size"`
	CreatedAt  string `json:"CreatedAt"`
}

// ListContainers returns every container belonging to one project.
//
// Parameters:
//   - exec: the Executor used to invoke the docker CLI.
//   - identifier: the project's Docker identifier
//     (project.Project.DockerIdentifier).
//   - composeScoped: true if identifier is a Docker Compose project name,
//     in which case containers are filtered by the
//     com.docker.compose.project label Compose attaches automatically;
//     false if identifier is a manual identifier for a plain-Dockerfile
//     project with no compose label to rely on, in which case containers
//     are filtered by name prefix instead.
//
// Returns every matching container, including stopped ones (`docker ps
// -a`), or an error if the docker CLI could not be invoked or its output
// could not be parsed. A project with no matching containers yet is not an
// error: it returns an empty, nil slice.
func ListContainers(exec shellexec.Executor, identifier string, composeScoped bool) ([]Container, error) {
	stdout, stderr, err := exec.Run("", "docker",
		"ps", "-a",
		"--filter", containerFilter(identifier, composeScoped),
		"--format", "{{json .}}",
	)
	if err != nil {
		return nil, fmt.Errorf("docker ps failed: %w (%s)", err, strings.TrimSpace(string(stderr)))
	}

	var containers []Container
	for _, line := range splitNonEmptyLines(stdout) {
		var row dockerPSRow
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, fmt.Errorf("failed to parse docker ps output: %w", err)
		}
		containers = append(containers, Container{
			ID:        row.ID,
			Name:      row.Names,
			Image:     row.Image,
			Status:    row.Status,
			State:     row.State,
			Ports:     row.Ports,
			CreatedAt: row.CreatedAt,
		})
	}

	return containers, nil
}

// ListImages returns every image belonging to one project, identified the
// same way as ListContainers.
//
// Parameters:
//   - exec: the Executor used to invoke the docker CLI.
//   - identifier: the project's Docker identifier.
//   - composeScoped: true to filter by the com.docker.compose.project
//     label (images Compose built carry it the same way containers do);
//     false to filter by image reference (repository name) prefix
//     instead, for plain-Dockerfile projects.
//
// Returns every matching image, or an error if the docker CLI could not be
// invoked or its output could not be parsed. A project with no matching
// images yet is not an error: it returns an empty, nil slice.
func ListImages(exec shellexec.Executor, identifier string, composeScoped bool) ([]Image, error) {
	stdout, stderr, err := exec.Run("", "docker",
		"images",
		"--filter", imageFilter(identifier, composeScoped),
		"--format", "{{json .}}",
	)
	if err != nil {
		return nil, fmt.Errorf("docker images failed: %w (%s)", err, strings.TrimSpace(string(stderr)))
	}

	var images []Image
	for _, line := range splitNonEmptyLines(stdout) {
		var row dockerImagesRow
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, fmt.Errorf("failed to parse docker images output: %w", err)
		}
		images = append(images, Image{
			ID:         row.ID,
			Repository: row.Repository,
			Tag:        row.Tag,
			Size:       row.Size,
			CreatedAt:  row.CreatedAt,
		})
	}

	return images, nil
}

// containerFilter builds the value passed to `docker ps --filter` to scope
// containers to one project. See ListContainers for what composeScoped
// controls.
func containerFilter(identifier string, composeScoped bool) string {
	if composeScoped {
		return "label=com.docker.compose.project=" + identifier
	}
	return "name=" + identifier
}

// imageFilter builds the value passed to `docker images --filter` to scope
// images to one project. See ListImages for what composeScoped controls.
func imageFilter(identifier string, composeScoped bool) string {
	if composeScoped {
		return "label=com.docker.compose.project=" + identifier
	}
	return "reference=" + identifier + "*"
}

// splitNonEmptyLines splits NDJSON command output into individual lines,
// skipping blank lines (in particular the trailing empty line left by the
// command's final newline).
func splitNonEmptyLines(output []byte) []string {
	var lines []string
	for _, line := range strings.Split(string(output), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines
}
