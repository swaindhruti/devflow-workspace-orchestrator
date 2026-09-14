package shellexec

import (
	"strings"
	"testing"
)

func TestDefaultExecutorRun(t *testing.T) {
	tests := []struct {
		name       string
		cmdName    string
		args       []string
		wantStdout string
		wantErr    bool
	}{
		{
			name:       "successful command captures stdout",
			cmdName:    "sh",
			args:       []string{"-c", "echo hello"},
			wantStdout: "hello\n",
			wantErr:    false,
		},
		{
			name:    "non-zero exit returns an error",
			cmdName: "sh",
			args:    []string{"-c", "exit 1"},
			wantErr: true,
		},
		{
			name:    "unknown binary returns an error",
			cmdName: "devflow-definitely-not-a-real-binary",
			wantErr: true,
		},
	}

	var e DefaultExecutor

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, _, err := e.Run("", tt.cmdName, tt.args...)

			if tt.wantErr && err == nil {
				t.Fatalf("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("did not expect error but got %v", err)
			}
			if tt.wantStdout != "" && string(stdout) != tt.wantStdout {
				t.Fatalf("expected stdout %q, got %q", tt.wantStdout, stdout)
			}
		})
	}
}

func TestDefaultExecutorRunUsesDir(t *testing.T) {
	var e DefaultExecutor

	stdout, _, err := e.Run("/tmp", "pwd")
	if err != nil {
		t.Fatal(err)
	}

	// macOS resolves /tmp to /private/tmp, so accept either form to keep
	// this test portable across platforms.
	got := strings.TrimSpace(string(stdout))
	if got != "/tmp" && !strings.HasSuffix(got, "/tmp") {
		t.Fatalf("expected pwd output ending in /tmp, got %q", got)
	}
}

func TestDefaultExecutorRunCapturesStderrSeparately(t *testing.T) {
	var e DefaultExecutor

	stdout, stderr, err := e.Run("", "sh", "-c", "echo out; echo err >&2")
	if err != nil {
		t.Fatal(err)
	}

	if strings.TrimSpace(string(stdout)) != "out" {
		t.Fatalf("expected stdout %q, got %q", "out", stdout)
	}
	if strings.TrimSpace(string(stderr)) != "err" {
		t.Fatalf("expected stderr %q, got %q", "err", stderr)
	}
}
