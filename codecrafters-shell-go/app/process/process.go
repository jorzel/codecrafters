package process

import (
	"errors"
	"io"
	"os/exec"
)

type Runner struct{}

func New() *Runner {
	return &Runner{}
}

func (r *Runner) Run(path, name string, args []string, stdout, stderr io.Writer) error {
	cmd := &exec.Cmd{
		Path:   path,
		Args:   append([]string{name}, args...),
		Stdout: stdout,
		Stderr: stderr,
	}
	err := cmd.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return nil
	}
	return err
}
