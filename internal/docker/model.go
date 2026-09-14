// Package docker implements DevFlow's Docker/Compose awareness domain: it
// detects whether a registered project has a Docker or Compose setup,
// inspects that project's containers and images scoped to its Docker
// identifier, and composes the default docker/compose commands a project
// gets once Docker is detected. It depends on internal/shellexec for
// synchronous CLI calls (e.g. `docker ps --format json`) and on
// internal/runner for executing the longer-running commands it composes
// (e.g. `docker compose up`), the same way internal/project's commands are
// executed.
package docker

// Container represents a single row from `docker ps`, scoped to one
// project's containers by ListContainers.
type Container struct {
	// ID is the container's short ID as reported by Docker.
	ID string
	// Name is the container's assigned name.
	Name string
	// Image is the image the container was created from (e.g.
	// "myproject-web:latest").
	Image string
	// Status is Docker's human-readable status string (e.g. "Up 2 hours",
	// "Exited (0) 3 minutes ago").
	Status string
	// State is the container's coarse-grained state (e.g. "running",
	// "exited", "paused").
	State string
	// Ports is Docker's human-readable port-mapping summary (e.g.
	// "0.0.0.0:8080->80/tcp").
	Ports string
	// CreatedAt is Docker's human-readable creation timestamp string.
	CreatedAt string
}

// Image represents a single row from `docker images`, scoped to one
// project's images by ListImages.
type Image struct {
	// ID is the image's short ID as reported by Docker.
	ID string
	// Repository is the image's repository name (e.g. "myproject-web").
	Repository string
	// Tag is the image's tag (e.g. "latest").
	Tag string
	// Size is Docker's human-readable image size string (e.g. "245MB").
	Size string
	// CreatedAt is Docker's human-readable creation timestamp string.
	CreatedAt string
}
