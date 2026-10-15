package shell_test

import (
	"bytes"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/codecrafters-io/shell-starter-go/app/shell"
)

// noExecutables is a finder for tests that don't depend on PATH lookup.
type noExecutables struct{}

func (noExecutables) Find(string) (string, bool) { return "", false }

func run(stdin *strings.Reader, stdout *bytes.Buffer, stderr *bytes.Buffer) int {
	return shell.New(noExecutables{}, &fakeRunner{}).Run(stdin, stdout, stderr)
}

func TestReplRunsEveryCommandFromInput(t *testing.T) {
	input := strings.NewReader("echo a\necho b\n")
	var output bytes.Buffer
	var stderr bytes.Buffer

	run(input, &output, &stderr)

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
	var stderr bytes.Buffer

	code := run(input, &output, &stderr)

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
	var stderr bytes.Buffer

	code := run(input, &output, &stderr)

	if code != 3 {
		t.Errorf("exit code = %d, want 3", code)
	}
}

func TestFailingCommandDoesNotStopShell(t *testing.T) {
	input := strings.NewReader("type unknown_command\necho still here\n")
	var output bytes.Buffer
	var stderr bytes.Buffer

	run(input, &output, &stderr)

	if !strings.Contains(output.String(), "still here") {
		t.Errorf("output %q: shell stopped after a failing command", output.String())
	}
}

func TestNotFoundMessageGoesToStderr(t *testing.T) {
	input := strings.NewReader("type unknown_command\n")
	var output bytes.Buffer
	var stderr bytes.Buffer

	run(input, &output, &stderr)

	want := "unknown_command: not found\n"
	if !strings.Contains(stderr.String(), want) {
		t.Errorf("stderr %q does not contain %q", stderr.String(), want)
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
	var stderr bytes.Buffer

	shell.New(finder, &fakeRunner{}).Run(strings.NewReader("type mycmd\n"), &output, &stderr)

	want := "mycmd is /opt/bin/mycmd\n"
	if !strings.Contains(output.String(), want) {
		t.Errorf("output %q does not contain %q", output.String(), want)
	}
}

func TestTypeReportsBuiltinEvenWhenExecutableWithSameNameExists(t *testing.T) {
	finder := fakeFinder{"echo": "/bin/echo"}
	var output bytes.Buffer
	var stderr bytes.Buffer

	shell.New(finder, &fakeRunner{}).Run(strings.NewReader("type echo\n"), &output, &stderr)

	want := "echo is a shell builtin\n"
	if !strings.Contains(output.String(), want) {
		t.Errorf("output %q does not contain %q", output.String(), want)
	}
}

// fakeRunner records every program the shell asks it to run.
type fakeRunner struct {
	calls []runCall
}

type runCall struct {
	path string
	name string
	args []string
}

func (r *fakeRunner) Run(path, name string, args []string, stdout, stderr io.Writer) error {
	r.calls = append(r.calls, runCall{path: path, name: name, args: args})
	return nil
}

func TestRunsExecutableFoundInPathWithItsArguments(t *testing.T) {
	finder := fakeFinder{"mycmd": "/opt/bin/mycmd"}
	runner := &fakeRunner{}
	var output bytes.Buffer
	var stderr bytes.Buffer

	shell.New(finder, runner).Run(strings.NewReader("mycmd a b\n"), &output, &stderr)

	if len(runner.calls) != 1 {
		t.Fatalf("runner called %d times, want 1", len(runner.calls))
	}
	got := runner.calls[0]
	if got.path != "/opt/bin/mycmd" {
		t.Errorf("path = %q, want %q", got.path, "/opt/bin/mycmd")
	}
	if got.name != "mycmd" {
		t.Errorf("name = %q, want %q", got.name, "mycmd")
	}
	if want := []string{"a", "b"}; !slices.Equal(got.args, want) {
		t.Errorf("args = %q, want %q", got.args, want)
	}
}
