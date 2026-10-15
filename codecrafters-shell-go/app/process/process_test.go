package process_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/codecrafters-io/shell-starter-go/app/process"
)

func TestRunProgramExitingWithNonZeroCodeIsNotAnError(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := process.New().Run("/bin/sh", "sh", []string{"-c", "exit 3"}, &stdout, &stderr)

	if err != nil {
		t.Errorf("Run returned %v, want nil for a program that ran and exited with 3", err)
	}
}

func TestRunProgramThatCannotStartIsAnError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	missing := filepath.Join(t.TempDir(), "missing")

	err := process.New().Run(missing, "missing", nil, &stdout, &stderr)

	if err == nil {
		t.Errorf("Run(%q) returned nil, want an error", missing)
	}
}
