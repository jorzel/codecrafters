package shell_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/codecrafters-io/shell-starter-go/app/shell"
)

// noExecutables is a finder for tests that don't depend on PATH lookup.
type noExecutables struct{}

func (noExecutables) Find(string) (string, bool) { return "", false }

func run(input *strings.Reader, output *bytes.Buffer) int {
	return shell.New(noExecutables{}).Run(input, output)
}

func TestReplRunsEveryCommandFromInput(t *testing.T) {
	input := strings.NewReader("echo a\necho b\n")
	var output bytes.Buffer

	run(input, &output)

	got := output.String()
	for _, want := range []string{"a\n", "b\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("output %q does not contain %q", got, want)
		}
	}
}

func TestExitWithoutArgumentStopsWithCodeZero(t *testing.T) {
	input := strings.NewReader("exit\necho after\n")
	var output bytes.Buffer

	code := run(input, &output)

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if strings.Contains(output.String(), "after") {
		t.Errorf("output %q: commands after exit should not run", output.String())
	}
}

func TestExitWithArgumentStopsWithThatCode(t *testing.T) {
	input := strings.NewReader("exit 3\n")
	var output bytes.Buffer

	code := run(input, &output)

	if code != 3 {
		t.Errorf("exit code = %d, want 3", code)
	}
}

func TestFailingCommandDoesNotStopShell(t *testing.T) {
	input := strings.NewReader("type unknown_command\necho still here\n")
	var output bytes.Buffer

	run(input, &output)

	if !strings.Contains(output.String(), "still here") {
		t.Errorf("output %q: shell stopped after a failing command", output.String())
	}
}

func TestCommandOutputIsFollowedDirectlyByPrompt(t *testing.T) {
	input := strings.NewReader("type unknown_command\n")
	var output bytes.Buffer

	run(input, &output)

	want := "unknown_command: not found\n$ "
	if !strings.Contains(output.String(), want) {
		t.Errorf("output %q does not contain %q", output.String(), want)
	}
}

// fakeFinder finds exactly the executables listed in paths.
type fakeFinder map[string]string

func (f fakeFinder) Find(name string) (string, bool) {
	path, ok := f[name]
	return path, ok
}

func TestTypeReportsFullPathOfExecutable(t *testing.T) {
	finder := fakeFinder{"mycmd": "/opt/bin/mycmd"}
	var output bytes.Buffer

	shell.New(finder).Run(strings.NewReader("type mycmd\n"), &output)

	want := "mycmd is /opt/bin/mycmd\n"
	if !strings.Contains(output.String(), want) {
		t.Errorf("output %q does not contain %q", output.String(), want)
	}
}

func TestTypeReportsBuiltinEvenWhenExecutableWithSameNameExists(t *testing.T) {
	finder := fakeFinder{"echo": "/bin/echo"}
	var output bytes.Buffer

	shell.New(finder).Run(strings.NewReader("type echo\n"), &output)

	want := "echo is a shell builtin\n"
	if !strings.Contains(output.String(), want) {
		t.Errorf("output %q does not contain %q", output.String(), want)
	}
}
