// Package e2e checks fnaze as a user meets it: the fnaze built from this
// source is run as a program (every command, the MCP server over standard
// input and output, the completion scripts in the real shells), and the
// examples in docs/ are checked against what fnaze really prints.
package e2e

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/amisonnet8/fnaze/internal/cli"
)

// bin is the fnaze built from this source by TestMain.
var bin string

// TestMain builds fnaze once, as a release would be (with its version set),
// into a directory of its own, and removes it after the tests.
func TestMain(m *testing.M) {
	os.Exit(func() int {
		dir, err := os.MkdirTemp("", "fnaze-e2e-")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		defer func() { _ = os.RemoveAll(dir) }()
		name := "fnaze"
		if runtime.GOOS == "windows" {
			name += ".exe" // a name without it does not match PATHEXT
		}
		bin = filepath.Join(dir, name)
		build := exec.Command("go", "build", "-ldflags=-X main.version=v0.1.0", "-o", bin, "../cmd/fnaze")
		if out, err := build.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "go build: %v\n%s", err, out)
			return 1
		}
		return m.Run()
	}())
}

// result is what one run of fnaze did.
type result struct {
	code           int
	stdout, stderr string
}

func (r result) String() string {
	return fmt.Sprintf("exit %d\nstdout:\n%sstderr:\n%s", r.code, r.stdout, r.stderr)
}

// fnaze runs the built fnaze in dir, with the environment of the test
// without FNAZE_LANG, plus env ("K=V").
func fnaze(t *testing.T, dir string, env []string, args ...string) result {
	t.Helper()
	return runProgram(t, bin, dir, append(environ("FNAZE_LANG"), env...), args...)
}

// runProgram runs a program in dir with the environment env (PWD left out:
// it would name another directory).
func runProgram(t *testing.T, path, dir string, env []string, args ...string) result {
	t.Helper()
	cmd := exec.Command(path, args...)
	cmd.Dir = dir
	for _, kv := range env {
		if !strings.HasPrefix(kv, "PWD=") {
			cmd.Env = append(cmd.Env, kv)
		}
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatalf("%s %s: %v", path, strings.Join(args, " "), err)
		}
		code = exit.ExitCode()
	}
	return result{code, stdout.String(), stderr.String()}
}

// environ is the environment of the test without the variables named.
func environ(without ...string) []string {
	var env []string
	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		drop := false
		for _, w := range without {
			drop = drop || strings.EqualFold(k, w)
		}
		if !drop {
			env = append(env, kv)
		}
	}
	return env
}

// The group that fnaze new makes in runCLI: the datetime is
// date -u -d 2026-11-08T01:02:03Z +%Y%m%dT%H%M%SZ, and the UUID is
// uuid.UUID(bytes=bytes(range(16)), version=4) in Python. The docs use it in
// their examples.
const newGroup = "20261108T010203Z_00010203-0405-4607-8809-0a0b0c0d0e0f"

// runCLI runs fnaze in this process, where the clock and the randomness can
// be fixed, so that the examples of new in the docs can be made again: the
// clock at 2026-11-08T01:02:03Z and the bytes 0, 1, …, 15 as the randomness.
// FNAZE_LANG is lang ("" for unset).
func runCLI(lang string, args ...string) result {
	var out, errb bytes.Buffer
	code := cli.Run(args, cli.Env{
		Getenv: func(k string) string {
			if k == "FNAZE_LANG" {
				return lang
			}
			return ""
		},
		Stdout:  &out,
		Stderr:  &errb,
		Version: "v0.1.0",
		Now:     func() time.Time { return time.Date(2026, 11, 8, 1, 2, 3, 0, time.UTC) },
		Random:  bytes.NewReader([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}),
	})
	return result{code, out.String(), errb.String()}
}

// newRoot runs the built fnaze init in a new directory and returns it, with
// its symbolic links resolved: the paths that fnaze prints are compared with
// it (macOS has /var → /private/var, Windows short names).
func newRoot(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if r := fnaze(t, dir, nil, "init"); r.code != 0 || r.stdout != "" || r.stderr != "" {
		t.Fatalf("init:\n%s", r)
	}
	return dir
}

// write makes the files under dir, with the directories above them.
func write(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func read(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// changes lists the names in .fnaze/changes/ of dir.
func changes(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(dir, ".fnaze", "changes"))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}
