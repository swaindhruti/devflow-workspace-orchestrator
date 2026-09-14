package shellexec

import "fmt"

// Call records a single invocation observed by a FakeExecutor, so tests can
// assert on exactly which command was run, from which directory, and with
// which arguments.
type Call struct {
	// Dir is the working directory the caller passed to Run.
	Dir string
	// Name is the command name the caller passed to Run.
	Name string
	// Args is the argument list the caller passed to Run.
	Args []string
}

// Response is the canned result a FakeExecutor returns for a matching
// command name.
type Response struct {
	// Stdout is the byte slice returned as the command's captured
	// standard output.
	Stdout []byte
	// Stderr is the byte slice returned as the command's captured
	// standard error.
	Stderr []byte
	// Err is returned as Run's error value, simulating a non-zero exit
	// status or a failure to start the command.
	Err error
}

// FakeExecutor is a test double for Executor. It records every call made to
// it and returns a pre-programmed Response looked up by command name, so
// docker, git, and tmux service tests can verify both the exact CLI
// invocation constructed by the code under test and that code's handling
// of a given canned response, with no real binary installed.
type FakeExecutor struct {
	// Responses maps a command name (e.g. "docker") to the Response
	// returned for every call to that command. Populate this before
	// exercising the code under test.
	Responses map[string]Response
	// Calls accumulates every Run invocation observed, in the order they
	// occurred, for later assertions.
	Calls []Call
}

// NewFakeExecutor constructs a FakeExecutor with its Responses map
// initialized and ready to populate.
func NewFakeExecutor() *FakeExecutor {
	return &FakeExecutor{Responses: make(map[string]Response)}
}

// Run implements Executor by recording the call and returning the Response
// registered for name.
//
// Parameters:
//   - dir: the working directory the caller requested. Recorded on the
//     Call but not acted on, since FakeExecutor never touches the
//     filesystem.
//   - name: the command name to look up in Responses.
//   - args: the arguments the caller passed. Recorded on the Call but not
//     acted on.
//
// Returns the registered Response's Stdout, Stderr, and Err fields. If no
// Response was registered for name, Run returns an error rather than a
// silent zero value, so an un-programmed call in a test fails loudly
// instead of masking a bug as "empty output".
func (f *FakeExecutor) Run(dir, name string, args ...string) (stdout, stderr []byte, err error) {
	f.Calls = append(f.Calls, Call{Dir: dir, Name: name, Args: args})

	resp, ok := f.Responses[name]
	if !ok {
		return nil, nil, fmt.Errorf("shellexec: no fake response registered for command %q", name)
	}

	return resp.Stdout, resp.Stderr, resp.Err
}
