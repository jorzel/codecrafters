package pathlookup

import (
	"os"
	"path/filepath"
)

func New(path string) *FilesystemExecutableFinder {
	dirs := filepath.SplitList(path)
	return &FilesystemExecutableFinder{pathDirs: dirs}
}

type FilesystemExecutableFinder struct {
	pathDirs []string
}

func (l *FilesystemExecutableFinder) Find(name string) (string, bool) {
	for _, dir := range l.pathDirs {
		fullPath := filepath.Join(dir, name)
		fileInfo, err := os.Stat(fullPath)
		if err != nil {
			continue
		}
		if fileInfo.Mode().IsRegular() && isFileExecutable(fileInfo) {
			return fullPath, true
		}
	}
	return "", false
}

func isFileExecutable(fileInfo os.FileInfo) bool {
	perm := fileInfo.Mode().Perm()
	return perm&0o111 != 0
}
