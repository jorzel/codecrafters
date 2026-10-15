package pathlookup_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codecrafters-io/shell-starter-go/app/pathlookup"
)

func TestFindReturnsFullPathOfExecutableInPath(t *testing.T) {
	dir := t.TempDir()
	want := writeFile(t, dir, "mycmd", 0o755)

	got, ok := pathlookup.New(dir).Find("mycmd")

	if !ok {
		t.Fatalf("Find(%q) reported not found, want %q", "mycmd", want)
	}
	if got != want {
		t.Errorf("Find(%q) = %q, want %q", "mycmd", got, want)
	}
}

func TestFindReportsNotFoundWhenNoDirectoryContainsCommand(t *testing.T) {
	dir := t.TempDir()

	got, ok := pathlookup.New(dir).Find("missing")

	if ok {
		t.Errorf("Find(%q) = %q, want not found", "missing", got)
	}
}

func TestFindReportsNotFoundWhenFileIsNotExecutable(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "mycmd", 0o644)

	got, ok := pathlookup.New(dir).Find("mycmd")

	if ok {
		t.Errorf("Find(%q) = %q, want not found for non-executable file", "mycmd", got)
	}
}

func TestFindReportsNotFoundWhenNameIsADirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "mycmd"), 0o755); err != nil {
		t.Fatalf("creating directory: %v", err)
	}

	got, ok := pathlookup.New(dir).Find("mycmd")

	if ok {
		t.Errorf("Find(%q) = %q, want not found for directory", "mycmd", got)
	}
}

func TestFindReturnsFirstMatchInPathOrder(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	want := writeFile(t, first, "mycmd", 0o755)
	writeFile(t, second, "mycmd", 0o755)

	got, ok := pathlookup.New(pathList(first, second)).Find("mycmd")

	if !ok || got != want {
		t.Errorf("Find(%q) = (%q, %v), want (%q, true)", "mycmd", got, ok, want)
	}
}

func TestFindSkipsNonExecutableAndContinuesToNextDirectory(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	writeFile(t, first, "mycmd", 0o644)
	want := writeFile(t, second, "mycmd", 0o755)

	got, ok := pathlookup.New(pathList(first, second)).Find("mycmd")

	if !ok || got != want {
		t.Errorf("Find(%q) = (%q, %v), want (%q, true)", "mycmd", got, ok, want)
	}
}

func TestFindSkipsDirectoryThatDoesNotExist(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	dir := t.TempDir()
	want := writeFile(t, dir, "mycmd", 0o755)

	got, ok := pathlookup.New(pathList(missing, dir)).Find("mycmd")

	if !ok || got != want {
		t.Errorf("Find(%q) = (%q, %v), want (%q, true)", "mycmd", got, ok, want)
	}
}

func pathList(dirs ...string) string {
	return strings.Join(dirs, string(os.PathListSeparator))
}

func writeFile(t *testing.T, dir, name string, perm os.FileMode) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), perm); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
	return path
}
