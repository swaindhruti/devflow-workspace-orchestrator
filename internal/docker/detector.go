package docker

import (
	"os"
	"path/filepath"
	"strings"
)

// composeFileNames lists the filenames DetectDockerSetup checks for, in
// priority order. Docker Compose itself recognizes several equivalent
// names across its v1/v2 history; DetectDockerSetup mirrors that list so
// detection works regardless of which convention a given project uses.
var composeFileNames = []string{
	"docker-compose.yml",
	"docker-compose.yaml",
	"compose.yml",
	"compose.yaml",
}

// dockerfileName is the filename DetectDockerSetup falls back to checking
// for when no compose file is present.
const dockerfileName = "Dockerfile"

// DetectionResult reports what DetectDockerSetup found in a project
// directory.
type DetectionResult struct {
	// HasDocker is true if either a Dockerfile or a compose file was
	// found in the project directory.
	HasDocker bool
	// ComposeFile is the filename of the compose file found (e.g.
	// "docker-compose.yml"), or "" if the project has no compose file.
	ComposeFile string
	// Identifier is the default Docker Compose project identifier
	// derived from the project directory name. It is only set when
	// ComposeFile is set, since it is meaningless without Compose:
	// Compose derives its own default project name the same way
	// (lowercased base directory name, invalid characters stripped), so
	// containers and images it creates already carry a matching
	// com.docker.compose.project label with no extra configuration. For
	// a plain-Dockerfile project (ComposeFile == ""), callers should set
	// project.DockerIdentifier manually, since there is no compose
	// convention to derive it from.
	Identifier string
}

// DetectDockerSetup inspects a project's directory for Docker and Compose
// configuration.
//
// Parameters:
//   - path: the project's local directory to inspect.
//
// Returns a DetectionResult describing what was found, or an error if path
// could not be read (e.g. it does not exist or is not a directory). Finding
// no Docker or Compose files at all is not itself an error: the returned
// DetectionResult simply has HasDocker set to false.
func DetectDockerSetup(path string) (DetectionResult, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return DetectionResult{}, err
	}

	present := make(map[string]bool, len(entries))
	for _, entry := range entries {
		present[entry.Name()] = true
	}

	for _, composeFile := range composeFileNames {
		if present[composeFile] {
			return DetectionResult{
				HasDocker:   true,
				ComposeFile: composeFile,
				Identifier:  deriveComposeIdentifier(path),
			}, nil
		}
	}

	if present[dockerfileName] {
		return DetectionResult{HasDocker: true}, nil
	}

	return DetectionResult{}, nil
}

// deriveComposeIdentifier derives the default Docker Compose project name
// for the directory at path, following Compose's own normalization rule: a
// project name may only contain lowercase letters, digits, hyphens, and
// underscores. deriveComposeIdentifier lowercases the directory's base name
// and drops every other character.
//
// Parameters:
//   - path: the project directory whose base name is used to derive the
//     identifier.
func deriveComposeIdentifier(path string) string {
	base := strings.ToLower(filepath.Base(filepath.Clean(path)))

	var b strings.Builder
	for _, r := range base {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}

	return b.String()
}
