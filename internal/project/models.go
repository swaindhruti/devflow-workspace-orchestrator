// Package project implements DevFlow's project registry domain: the data
// model for a registered project and its saved commands, a Repository
// abstraction over how projects are persisted, a JSON-file-backed
// implementation of that abstraction, a Service that layers validation and
// ID generation on top of a Repository, and validation rules for project
// data. Every file in this package concerns the same domain concept (a
// "project"), following a domain-first layout rather than splitting model,
// storage, and service logic into separate top-level packages.
package project

// Project represents a single workspace registered with DevFlow: a local
// directory the user works in, along with the metadata and saved commands
// DevFlow needs to present and operate on it.
type Project struct {
	// ID uniquely identifies the project. It is assigned by Service.AddProject
	// (via GenerateID) and must never be set by callers before that point.
	ID string `json:"id"`
	// Name is the human-readable project name shown in the dashboard.
	Name string `json:"name"`
	// Description is an optional free-text summary of the project.
	Description string `json:"description"`
	// Path is the absolute filesystem path to the project's local
	// directory. It is required: Service.AddProject and Service.UpdateProject
	// both reject a Project with an empty Path via ValidateProject.
	Path string `json:"path"`
	// TechStack lists the technologies or languages associated with the
	// project (e.g. "go", "react"), used for search and filtering.
	TechStack []string `json:"tech_stack"`
	// IsFavorite marks the project as a favorite for quick access in the
	// dashboard. It is toggled independently of UpdateProject via
	// Service.MarkProjectFavorite and Service.UnmarkProjectFavorite.
	IsFavorite bool `json:"favorite"`

	// RunCommands holds the reusable shell commands saved for this
	// project (e.g. "go test ./...", "npm run dev"). Manage this slice
	// through the AddCommand, UpdateCommand, DeleteCommand, and
	// GetAllCommands helpers in commands.go rather than mutating it
	// directly, so callers have one consistent place to look for that
	// logic.
	RunCommands []Command `json:"commands"`

	// HasDocker records whether the project directory contains a
	// Dockerfile and/or a docker-compose file. It is set by the
	// internal/docker package's detector, not by this package, since
	// detection requires filesystem access this package deliberately
	// does not perform.
	HasDocker bool `json:"has_docker"`
	// DockerComposeFile is the filename of the compose file detected in
	// the project directory (e.g. "docker-compose.yml"), or "" if the
	// project has no compose file (a plain Dockerfile only, or no Docker
	// setup at all).
	DockerComposeFile string `json:"docker_compose_file,omitempty"`
	// DockerIdentifier scopes Docker CLI queries (docker ps, docker
	// images) to this project. When DockerComposeFile is set, it
	// defaults to Docker Compose's own project-name convention (the
	// lowercased project directory name) so containers/images already
	// carry the matching com.docker.compose.project label with no extra
	// setup. It can be overridden here, and is the only way to scope a
	// plain-Dockerfile project (with no compose file) that has no such
	// label to filter by.
	DockerIdentifier string `json:"docker_identifier,omitempty"`
	// DockerCommands holds the reusable docker/compose commands saved
	// for this project (e.g. "up", "down", "build"), managed the same
	// way as RunCommands. Populated with defaults once Docker is
	// detected, and editable like any other saved command afterward.
	DockerCommands []Command `json:"docker_commands,omitempty"`
}

// Command represents a single saved, reusable shell command scoped to a
// project, such as a build, test, or run step.
type Command struct {
	// ID uniquely identifies the command within its project.
	ID string `json:"id"`
	// ProjectID is the ID of the Project this command belongs to.
	ProjectID string `json:"project_id"`
	// Name is a short, human-readable label for the command (e.g. "Run
	// tests"), shown in the UI instead of the raw command string.
	Name string `json:"name"`
	// Command is the actual shell command text to execute (e.g.
	// "go test ./...").
	Command string `json:"command"`
	// Path is an optional working directory the command should run from,
	// relative to or overriding the project's own Path.
	Path string `json:"path"`
	// Description is an optional free-text note about what the command
	// does.
	Description string `json:"description"`
}
