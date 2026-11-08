// Package project finds the root of a project (the directory that holds
// .fnaze/) and makes .fnaze/ (docs/reference/file-format.md, "Location";
// docs/reference/commands.md, "fnaze init").
package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Dir is the name of the directory that fnaze keeps its files in, and Changes
// the directory of the description files inside it.
const (
	Dir     = ".fnaze"
	Changes = "changes"
)

// ErrNotFound is returned by Find when no .fnaze/ is found.
var ErrNotFound = errors.New("no " + Dir + "/ found")

// Find returns the root: the first directory, from dir upward, that holds a
// .fnaze/ directory.
func Find(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for d := abs; ; {
		if fi, err := os.Stat(filepath.Join(d, Dir)); err == nil && fi.IsDir() {
			return d, nil
		}
		parent := filepath.Dir(d)
		if parent == d {
			return "", fmt.Errorf("%w in %s or above it: run fnaze init in the root of the project", ErrNotFound, abs)
		}
		d = parent
	}
}

// ChangesDir returns the directory of the description files of a root.
func ChangesDir(root string) string {
	return filepath.Join(root, Dir, Changes)
}

// Init makes .fnaze/changes/ in dir, and returns dir as the root. It fails if
// a .fnaze/ is found in dir or above it: a project has only one.
func Init(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	if fi, err := os.Stat(abs); err != nil {
		return "", err
	} else if !fi.IsDir() {
		return "", fmt.Errorf("%s is not a directory", abs)
	}
	if root, err := Find(abs); err == nil {
		return "", fmt.Errorf("%s already has %s/: a project has only one", root, Dir)
	} else if !errors.Is(err, ErrNotFound) {
		return "", err
	}
	if err := os.Mkdir(filepath.Join(abs, Dir), 0o755); err != nil { //nolint:gosec // .fnaze/ is shared with the people of the project, like the rest of the working tree
		return "", err
	}
	if err := os.Mkdir(ChangesDir(abs), 0o755); err != nil { //nolint:gosec // as above
		return "", err
	}
	return abs, nil
}
