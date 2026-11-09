package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"strings"
	"testing"
	"time"
)

// The group that run makes with fnaze new: the datetime is
// date -u -d 2026-11-08T01:02:03Z +%Y%m%dT%H%M%SZ, and the UUID is
// uuid.UUID(bytes=bytes(range(16)), version=4) in Python.
const (
	group  = "20261108T010203Z_00010203-0405-4607-8809-0a0b0c0d0e0f"
	fileEn = ".fnaze/changes/" + group + ".txt"
	fileJa = ".fnaze/changes/" + group + "_ja.txt"
)

// run runs the command line with FNAZE_LANG set to lang ("" for unset), the
// clock at 2026-11-08T01:02:03Z and the bytes 0, 1, …, 15 as the randomness.
func run(t *testing.T, lang string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errb bytes.Buffer
	code = Run(args, Env{
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
	return code, out.String(), errb.String()
}

// newRoot runs fnaze init in a new directory and returns it.
func newRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if code, out, errs := run(t, "", "init", "-C", dir); code != 0 || out != "" || errs != "" {
		t.Fatalf("init: %d %q %q", code, out, errs)
	}
	return dir
}

func readFile(t *testing.T, dir, file string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(file)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func writeFile(t *testing.T, dir, file, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(file)), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// want checks the result of run.
func want(t *testing.T, code int, stdout, stderr string, wantCode int, wantOut, wantErr string) {
	t.Helper()
	if code != wantCode || stdout != wantOut || stderr != wantErr {
		t.Errorf("got %d\n%s\n%s\nwant %d\n%s\n%s", code, stdout, stderr, wantCode, wantOut, wantErr)
	}
}

func TestParse(t *testing.T) {
	allowed := []string{"--lang", "--detail", "--json", "-C"}
	tests := []struct {
		name     string
		args     []string
		wantOpts map[string][]string
		wantArgs []string
		wantHelp bool
		wantErr  string
	}{
		{"options after the arguments", []string{"a", "b", "--lang", "ja", "--json"}, map[string][]string{"--lang": {"ja"}, "--json": {""}}, []string{"a", "b"}, false, ""},
		{"options between the arguments", []string{"a", "--lang=ja", "b"}, map[string][]string{"--lang": {"ja"}}, []string{"a", "b"}, false, ""},
		{"--detail given again adds a value", []string{"--detail", "x", "--detail=", "--detail", "-y"}, map[string][]string{"--detail": {"x", "", "-y"}}, nil, false, ""},
		{"after -- everything is an argument", []string{"--", "--json", "-C"}, map[string][]string{}, []string{"--json", "-C"}, false, ""},
		{"- alone is an argument", []string{"-"}, map[string][]string{}, []string{"-"}, false, ""},
		{"-h", []string{"a", "-h"}, map[string][]string{}, []string{"a"}, true, ""},
		{"--help", []string{"--help"}, map[string][]string{}, nil, true, ""},
		{"unknown option", []string{"--all"}, nil, nil, false, "unknown option --all"},
		{"another option given twice", []string{"--lang", "a", "--lang", "b"}, nil, nil, false, "--lang is given twice"},
		{"no value", []string{"-C"}, nil, nil, false, "-C needs a value"},
		{"a value for a flag", []string{"--json=1"}, nil, nil, false, "--json takes no value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parse(tt.args, allowed)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("err = %v, want %s", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(p.opts, tt.wantOpts) || !reflect.DeepEqual(p.args, tt.wantArgs) || p.help != tt.wantHelp {
				t.Errorf("got %q %q %v, want %q %q %v", p.opts, p.args, p.help, tt.wantOpts, tt.wantArgs, tt.wantHelp)
			}
		})
	}
}

func TestBuildVersion(t *testing.T) {
	info := func(v string) func() (*debug.BuildInfo, bool) {
		return func() (*debug.BuildInfo, bool) { return &debug.BuildInfo{Main: debug.Module{Version: v}}, true }
	}
	none := func() (*debug.BuildInfo, bool) { return nil, false }
	tests := []struct {
		name string
		set  string
		read func() (*debug.BuildInfo, bool)
		want string
	}{
		{"set by the release build", "v0.2.0", info("v0.1.0"), "v0.2.0"},
		{"go install", "", info("v0.1.0"), "v0.1.0"},
		{"go build in the repository", "", info("(devel)"), ""},
		{"no build info", "", none, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildVersion(tt.set, tt.read); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUsageAndVersion(t *testing.T) {
	// The summaries start two spaces after the longest name, completion (the
	// whole usage is checked in e2e).
	for _, line := range []string{"\n  init        create .fnaze/changes/\n", "\n  completion  print a shell completion script\n"} {
		if !strings.Contains(usage, line) {
			t.Errorf("the usage has no line %q:\n%s", line, usage)
		}
	}
	code, out, errs := run(t, "")
	want(t, code, out, errs, 2, "", usage)
	for _, a := range []string{"--help", "-h", "help"} {
		code, out, errs = run(t, "", a)
		want(t, code, out, errs, 0, usage, "")
	}
	code, out, errs = run(t, "", "--version")
	want(t, code, out, errs, 0, "fnaze v0.1.0\n", "")

	var b bytes.Buffer
	if code := Run([]string{"--version"}, Env{Stdout: &b}); code != 0 || b.String() != "fnaze (devel)\n" {
		t.Errorf("no version: %d %q", code, b.String())
	}

	code, out, errs = run(t, "", "show", "--help")
	want(t, code, out, errs, 0, "usage: fnaze show [--lang <language>] [group] [-C <dir>] [--json]\n", "")
}

func TestCommandLineErrors(t *testing.T) {
	setUsage := "fnaze set [--lang <language>] [--group <group>] <path> <desc> [--detail <detail>]... [-C <dir>] [--json]"
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"unknown command", []string{"add"}, `fnaze: unknown command "add" (see fnaze --help)` + "\n"},
		{"unknown option", []string{"set", "--all"}, "fnaze: set: unknown option --all (usage: " + setUsage + ")\n"},
		{"missing argument", []string{"set", "a"}, "fnaze: set: missing <desc> (usage: " + setUsage + ")\n"},
		{"too many arguments", []string{"set", "a", "b", "c"}, "fnaze: set: too many arguments (usage: " + setUsage + ")\n"},
		{"rm without a path", []string{"rm"}, "fnaze: rm: missing <path> (usage: fnaze rm [--lang <language>] [--group <group>] <path> [-C <dir>] [--json])\n"},
		{"init takes no --json", []string{"init", "--json"}, "fnaze: init: unknown option --json (usage: fnaze init [-C <dir>])\n"},
		{"log takes no group", []string{"log", "x"}, "fnaze: log: too many arguments (usage: fnaze log [--lang <language>] [-C <dir>] [--json])\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, out, errs := run(t, "", tt.args...)
			want(t, code, out, errs, 2, "", tt.wantErr)
		})
	}
}

func TestInit(t *testing.T) {
	dir := newRoot(t)
	if fi, err := os.Stat(filepath.Join(dir, ".fnaze", "changes")); err != nil || !fi.IsDir() {
		t.Fatalf("no .fnaze/changes/: %v", err)
	}
	code, out, errs := run(t, "", "init", "-C", dir)
	want(t, code, out, errs, 2, "", "fnaze: "+dir+" already has .fnaze/: a project has only one\n")
}

func TestNotFound(t *testing.T) {
	dir := t.TempDir()
	code, out, errs := run(t, "", "log", "-C", dir)
	want(t, code, out, errs, 2, "", "fnaze: no .fnaze/ found in "+dir+" or above it: run fnaze init in the root of the project\n")
}

func TestNew(t *testing.T) {
	dir := newRoot(t)
	code, out, errs := run(t, "", "new", "en", "ja", "-C", dir)
	want(t, code, out, errs, 0, filepath.FromSlash(fileEn)+"\n"+filepath.FromSlash(fileJa)+"\n", "")
	if got := readFile(t, dir, fileJa); got != "# \n" {
		t.Errorf("content %q", got)
	}
}

func TestNewPathsFromTheDirectory(t *testing.T) {
	dir := newRoot(t)
	sub := filepath.Join(dir, "src", "auth")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	code, out, errs := run(t, "ja", "new", "-C", sub)
	want(t, code, out, errs, 0, filepath.FromSlash("../../"+fileJa)+"\n", "")
}

func TestNewJSON(t *testing.T) {
	dir := newRoot(t)
	code, out, errs := run(t, "", "new", "--json", "-C", dir, "en", "ja")
	want(t, code, out, errs, 0, `{
  "group": "`+group+`",
  "files": [
    {
      "lang": "en",
      "path": "`+fileEn+`"
    },
    {
      "lang": "ja",
      "path": "`+fileJa+`"
    }
  ]
}
`, "")
}

func TestSetAndRm(t *testing.T) {
	dir := newRoot(t)
	if code, _, errs := run(t, "", "new", "ja", "-C", dir); code != 0 {
		t.Fatal(errs)
	}
	code, out, errs := run(t, "ja", "set", "src/auth/session.go", "延長する", "--detail", "短すぎる", "--detail", "", "--detail", "a\nb", "-C", dir)
	want(t, code, out, errs, 0, "", "")
	// Without --detail, the detail is kept.
	code, out, errs = run(t, "ja", "set", "src/auth/session.go", "延ばす", "-C", dir)
	want(t, code, out, errs, 0, "", "")
	code, out, errs = run(t, "", "set", "--lang", "ja", "--group", group[:21], "docs/a&b.md", "<消した>", "-C", dir, "--json")
	want(t, code, out, errs, 0, `{
  "group": "`+group+`",
  "lang": "ja",
  "path": "docs/a&b.md",
  "desc": "<消した>",
  "detail": ""
}
`, "")
	if got, w := readFile(t, dir, fileJa), "# \nsrc/\n\tauth/\n\t\tsession.go\t延ばす\n\t\t\t| 短すぎる\n\t\t\t|\n\t\t\t| a\n\t\t\t| b\ndocs/\n\ta&b.md\t<消した>\n"; got != w {
		t.Errorf("after set\n%s\nwant\n%s", got, w)
	}

	code, out, errs = run(t, "ja", "rm", "src/auth/session.go", "-C", dir)
	want(t, code, out, errs, 0, "", "")
	code, out, errs = run(t, "ja", "rm", "docs/a&b.md", "-C", dir, "--json")
	want(t, code, out, errs, 0, `{
  "group": "`+group+`",
  "lang": "ja",
  "path": "docs/a&b.md",
  "removed": 2
}
`, "")
	if got := readFile(t, dir, fileJa); got != "# \n" {
		t.Errorf("after rm %q", got)
	}
}

func TestTitle(t *testing.T) {
	dir := newRoot(t)
	code, out, errs := run(t, "", "title", "t", "-C", dir)
	want(t, code, out, errs, 2, "", "fnaze: there is no group yet: make one with fnaze new\n")
	if code, _, errs := run(t, "", "new", "ja", "-C", dir); code != 0 {
		t.Fatal(errs)
	}
	code, out, errs = run(t, "ja", "title", " 期限を延ばす ", "-C", dir)
	want(t, code, out, errs, 0, "", "")
	if got := readFile(t, dir, fileJa); got != "# 期限を延ばす\n" {
		t.Errorf("after title %q", got)
	}
	code, out, errs = run(t, "", "title", "--lang", "ja", "--group", group[:21], "<a&b>", "-C", dir, "--json")
	want(t, code, out, errs, 0, `{
  "group": "`+group+`",
  "lang": "ja",
  "title": "<a&b>"
}
`, "")
	code, out, errs = run(t, "ja", "title", "a\nb", "-C", dir)
	want(t, code, out, errs, 2, "", "fnaze: a title cannot have a newline\n")
	code, out, errs = run(t, "", "title", "t", "-C", dir)
	want(t, code, out, errs, 2, "", "fnaze: group "+group+" has no file in en\n")
	code, out, errs = run(t, "ja", "title", "-C", dir)
	want(t, code, out, errs, 2, "", "fnaze: title: missing <title> (usage: fnaze title [--lang <language>] [--group <group>] <title> [-C <dir>] [--json])\n")
	if got := readFile(t, dir, fileJa); got != "# <a&b>\n" {
		t.Errorf("after the errors %q", got)
	}
}

func TestSetErrors(t *testing.T) {
	dir := newRoot(t)
	code, out, errs := run(t, "", "set", "a.go", "d", "-C", dir)
	want(t, code, out, errs, 2, "", "fnaze: there is no group yet: make one with fnaze new\n")
	if code, _, errs := run(t, "", "new", "ja", "-C", dir); code != 0 {
		t.Fatal(errs)
	}
	code, out, errs = run(t, "", "set", "a.go", "d", "-C", dir)
	want(t, code, out, errs, 2, "", "fnaze: group "+group+" has no file in en\n")
	code, out, errs = run(t, "j-a", "set", "a.go", "d", "-C", dir)
	if code != 2 || out != "" || !strings.HasPrefix(errs, "fnaze: ") {
		t.Errorf("bad FNAZE_LANG: %d %q %q", code, out, errs)
	}
}

func TestStatus(t *testing.T) {
	dir := newRoot(t)
	code, out, errs := run(t, "", "status", "--require", "-C", dir)
	want(t, code, out, errs, 0, "", "")

	if code, _, errs := run(t, "", "new", "en", "ja", "-C", dir); code != 0 {
		t.Fatal(errs)
	}
	writeFile(t, dir, fileEn, "# t\nsrc/a.go\n")
	writeFile(t, dir, fileJa, "# 題\n")
	writeFile(t, dir, ".fnaze/changes/notes.txt", "")
	// In the order of the files: "2" is before "n".
	lines := filepath.FromSlash(fileEn) + ":2: empty-desc: src/a.go has no desc\n" +
		filepath.FromSlash(fileJa) + ": lang-mismatch: the path lines differ from those of " + fileEn + "\n" +
		filepath.FromSlash(".fnaze/changes/notes.txt") + ": format: the name does not have the form <datetime>_<UUID>[_<language>].txt\n"
	code, out, errs = run(t, "", "status", "-C", dir)
	want(t, code, out, errs, 0, lines, "")
	code, out, errs = run(t, "", "status", "--require", "-C", dir)
	want(t, code, out, errs, 1, lines, "")
	// One item is enough for exit code 1.
	writeFile(t, dir, fileJa, "# 題\nsrc/a.go\td\n")
	if err := os.Remove(filepath.Join(dir, ".fnaze", "changes", "notes.txt")); err != nil {
		t.Fatal(err)
	}
	code, out, errs = run(t, "", "status", "--require", "-C", dir)
	want(t, code, out, errs, 1, filepath.FromSlash(fileEn)+":2: empty-desc: src/a.go has no desc\n", "")
	writeFile(t, dir, fileJa, "# 題\n")
	writeFile(t, dir, ".fnaze/changes/notes.txt", "")

	code, out, errs = run(t, "", "status", "--all", group, "-C", dir)
	want(t, code, out, errs, 2, "", "fnaze: --all and a group cannot be given together\n")

	code, out, errs = run(t, "", "status", "--require", "--json", "-C", dir)
	want(t, code, out, errs, 1, `{
  "items": [
    {
      "group": "`+group+`",
      "file": "`+fileEn+`",
      "line": 2,
      "kind": "empty-desc",
      "message": "src/a.go has no desc"
    },
    {
      "group": "`+group+`",
      "file": "`+fileJa+`",
      "line": 0,
      "kind": "lang-mismatch",
      "message": "the path lines differ from those of `+fileEn+`"
    },
    {
      "group": "",
      "file": ".fnaze/changes/notes.txt",
      "line": 0,
      "kind": "format",
      "message": "the name does not have the form <datetime>_<UUID>[_<language>].txt"
    }
  ]
}
`, "")
}

func TestShow(t *testing.T) {
	dir := newRoot(t)
	code, out, errs := run(t, "", "show", "-C", dir)
	want(t, code, out, errs, 2, "", "fnaze: there is no group yet: make one with fnaze new\n")
	if code, _, errs := run(t, "", "new", "en", "-C", dir); code != 0 {
		t.Fatal(errs)
	}
	writeFile(t, dir, fileEn, "# t\nsrc/\n\ta.go\td\n\t\t| why\n")
	code, out, errs = run(t, "", "show", group[:21], "-C", dir)
	want(t, code, out, errs, 0, "# t\n\nsrc/\n  a.go  d\n        | why\n", "")
	code, out, errs = run(t, "", "show", "--json", "-C", dir)
	want(t, code, out, errs, 0, `{
  "group": "`+group+`",
  "lang": "en",
  "file": "`+fileEn+`",
  "title": "t",
  "entries": [
    {
      "name": "src/",
      "path": "src/",
      "desc": "",
      "detail": "",
      "children": [
        {
          "name": "a.go",
          "path": "src/a.go",
          "desc": "d",
          "detail": "why",
          "children": []
        }
      ]
    }
  ]
}
`, "")
	code, out, errs = run(t, "", "show", "--lang", "ja", "-C", dir)
	want(t, code, out, errs, 2, "", "fnaze: group "+group+" has no file in ja\n")

	// A group that is not the newest.
	writeFile(t, dir, ".fnaze/changes/20261109T000000Z_abcd1111-2222-4333-8444-555566667777.txt", "# newer\n")
	code, out, errs = run(t, "", "show", "-C", dir)
	want(t, code, out, errs, 0, "# newer\n", "")
	code, out, errs = run(t, "", "show", group, "-C", dir)
	want(t, code, out, errs, 0, "# t\n\nsrc/\n  a.go  d\n        | why\n", "")
}

func TestLog(t *testing.T) {
	setLocal(t, time.FixedZone("", 9*60*60))
	dir := newRoot(t)
	code, out, errs := run(t, "", "log", "-C", dir)
	want(t, code, out, errs, 0, "", "")
	if code, _, errs := run(t, "", "new", "en", "ja", "-C", dir); code != 0 {
		t.Fatal(errs)
	}
	writeFile(t, dir, fileJa, "# 題\na.go\td\n")
	code, out, errs = run(t, "ja", "log", "-C", dir)
	want(t, code, out, errs, 0, "20261108T010203Z_0001  2026-11-08 10:02  en,ja  1  題\n", "")
	code, out, errs = run(t, "", "log", "--json", "-C", dir)
	want(t, code, out, errs, 0, `{
  "groups": [
    {
      "group": "`+group+`",
      "datetime": "2026-11-08T01:02:03Z",
      "langs": [
        "en",
        "ja"
      ],
      "title": "",
      "entries": 0
    }
  ]
}
`, "")
}

// failingWriter fails every write.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("broken pipe") }

func TestWriteError(t *testing.T) {
	dir := newRoot(t)
	for _, args := range [][]string{{"log", "--json"}, {"new", "en"}} {
		var errb bytes.Buffer
		code := Run(append(args, "-C", dir), Env{Getenv: func(string) string { return "" }, Stdout: failingWriter{}, Stderr: &errb})
		if code != 2 || errb.String() != "fnaze: broken pipe\n" {
			t.Errorf("%s: got %d %q", args, code, errb.String())
		}
	}
}
