package main

import (
	"bytes"
	"strings"
	"testing"
)

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
