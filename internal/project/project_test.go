package project

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// tempDir returns a new directory with the symbolic links in its path resolved,
// so that it can be compared with paths that fnaze returns
// (.claude/rules/testing.md).
func tempDir(t *testing.T) string {
	t.Helper()
	d, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestFind(t *testing.T) {
	root := tempDir(t)
	mkdir(t, filepath.Join(root, ".fnaze", "changes"))
	sub := filepath.Join(root, "a", "b")
	mkdir(t, sub)
	for _, dir := range []string{root, sub, filepath.Join(root, "a")} {
		got, err := Find(dir)
		if err != nil || got != root {
			t.Errorf("Find(%s) = %s, %v; want %s", dir, got, err, root)
		}
	}
}

// The nearest .fnaze/ wins, and a file named .fnaze is not one.
func TestFindNearest(t *testing.T) {
	root := tempDir(t)
	mkdir(t, filepath.Join(root, ".fnaze"))
	inner := filepath.Join(root, "inner")
	mkdir(t, filepath.Join(inner, ".fnaze"))
	fileDir := filepath.Join(inner, "file")
	mkdir(t, fileDir)
	if err := os.WriteFile(filepath.Join(fileDir, ".fnaze"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := Find(fileDir); err != nil || got != inner {
		t.Errorf("got %s, %v; want %s", got, err, inner)
	}
}

func TestFindRelative(t *testing.T) {
	root := tempDir(t)
	mkdir(t, filepath.Join(root, ".fnaze"))
	mkdir(t, filepath.Join(root, "sub"))
	t.Chdir(root)
	if got, err := Find("sub"); err != nil || got != root {
		t.Errorf("got %s, %v; want %s", got, err, root)
	}
}

func TestFindNone(t *testing.T) {
	dir := tempDir(t)
	if _, err := os.Stat(filepath.Join(dir, ".fnaze")); err == nil {
		t.Skip("a .fnaze/ is above the temporary directory")
	}
	if got, err := Find(dir); !errors.Is(err, ErrNotFound) {
		t.Errorf("got %s, %v", got, err)
	}
}

func TestInit(t *testing.T) {
	dir := tempDir(t)
	root, err := Init(dir)
	if err != nil || root != dir {
		t.Fatalf("got %s, %v", root, err)
	}
	if fi, err := os.Stat(filepath.Join(dir, ".fnaze", "changes")); err != nil || !fi.IsDir() {
		t.Fatalf(".fnaze/changes/: %v", err)
	}
	if got := ChangesDir(root); got != filepath.Join(dir, ".fnaze", "changes") {
		t.Errorf("ChangesDir = %s", got)
	}
	if got, err := Find(dir); err != nil || got != dir {
		t.Errorf("Find after Init = %s, %v", got, err)
	}
}

func TestInitFails(t *testing.T) {
	root := tempDir(t)
	if _, err := Init(root); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "sub")
	mkdir(t, sub)
	for _, dir := range []string{root, sub, filepath.Join(root, "missing")} {
		if _, err := Init(dir); err == nil {
			t.Errorf("Init(%s): no error", dir)
		}
	}
	if _, err := os.Stat(filepath.Join(sub, ".fnaze")); err == nil {
		t.Error("Init made .fnaze/ under another .fnaze/")
	}
	file := filepath.Join(tempDir(t), "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(file); err == nil {
		t.Error("Init of a file: no error")
	}
}
