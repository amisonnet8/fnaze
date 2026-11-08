package mcpserver

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/amisonnet8/fnaze/internal/core"
)

// The group that new_group makes: see the constants of internal/cli's tests.
const (
	group  = "20261108T010203Z_00010203-0405-4607-8809-0a0b0c0d0e0f"
	fileJa = ".fnaze/changes/" + group + "_ja.txt"
)

// newRoot makes a root with .fnaze/changes/ and returns it.
func newRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if _, err := core.Init(dir); err != nil {
		t.Fatal(err)
	}
	return dir
}

// options are the options of a server on dir, with a fixed clock and
// randomness.
func options(dir string) Options {
	return Options{
		Dir:    dir,
		Now:    func() time.Time { return time.Date(2026, 11, 8, 1, 2, 3, 0, time.UTC) },
		Random: bytes.NewReader([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}),
	}
}

// connect starts the server with o and returns a client session to it.
func connect(t *testing.T, o Options) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	ss, err := New(o).Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ss.Close() })
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

// call calls the tool and returns the text it returns, and whether it is an
// error.
func call(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) (string, bool) {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if len(res.Content) != 1 {
		t.Fatalf("%s: %d contents", name, len(res.Content))
	}
	return res.Content[0].(*mcp.TextContent).Text, res.IsError
}

// ok calls the tool and returns its text; the call must succeed.
func ok(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) string {
	t.Helper()
	text, isErr := call(t, cs, name, args)
	if isErr {
		t.Fatalf("%s: %s", name, text)
	}
	return text
}

func readFile(t *testing.T, dir, file string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(file)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func toolNames(t *testing.T, cs *mcp.ClientSession) []string {
	t.Helper()
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
	}
	slices.Sort(names)
	return names
}

func TestTools(t *testing.T) {
	dir := newRoot(t)
	read := []string{"log", "show", "status"}
	if got := toolNames(t, connect(t, options(dir))); !reflect.DeepEqual(got, read) {
		t.Errorf("without --allow-write: %q, want %q", got, read)
	}
	o := options(dir)
	o.AllowWrite = true
	all := []string{"log", "new_group", "remove_description", "set_description", "show", "status"}
	if got := toolNames(t, connect(t, o)); !reflect.DeepEqual(got, all) {
		t.Errorf("with --allow-write: %q, want %q", got, all)
	}
}

func TestWrite(t *testing.T) {
	dir := newRoot(t)
	o := options(dir)
	o.AllowWrite = true
	o.EnvLang = "ja"
	cs := connect(t, o)

	// The languages from FNAZE_LANG.
	want := "{\n  \"group\": \"" + group + "\",\n  \"files\": [\n    {\n      \"lang\": \"ja\",\n      \"path\": \"" + fileJa + "\"\n    }\n  ]\n}\n"
	if got := ok(t, cs, "new_group", nil); got != want {
		t.Errorf("new_group\n%s\nwant\n%s", got, want)
	}

	ok(t, cs, "set_description", map[string]any{"path": "src/a.go", "desc": "延ばす", "detail": []string{"一つ", "二つ\n三つ"}})
	// Without detail, the detail is kept.
	ok(t, cs, "set_description", map[string]any{"path": "src/a.go", "desc": "延ばす"})
	if got, w := readFile(t, dir, fileJa), "# \nsrc/\n\ta.go\t延ばす\n\t\t| 一つ\n\t\t| 二つ\n\t\t| 三つ\n"; got != w {
		t.Errorf("after set\n%s\nwant\n%s", got, w)
	}
	// An empty list removes it.
	want = "{\n  \"group\": \"" + group + "\",\n  \"lang\": \"ja\",\n  \"path\": \"src/a.go\",\n  \"desc\": \"<延ばす>\",\n  \"detail\": \"\"\n}\n"
	if got := ok(t, cs, "set_description", map[string]any{"lang": "ja", "group": group[:21], "path": "src/a.go", "desc": "<延ばす>", "detail": []string{}}); got != want {
		t.Errorf("set_description\n%s\nwant\n%s", got, want)
	}

	want = "{\n  \"group\": \"" + group + "\",\n  \"lang\": \"ja\",\n  \"path\": \"src/a.go\",\n  \"removed\": 2\n}\n"
	if got := ok(t, cs, "remove_description", map[string]any{"path": "src/a.go"}); got != want {
		t.Errorf("remove_description\n%s\nwant\n%s", got, want)
	}
	if got := readFile(t, dir, fileJa); got != "# \n" {
		t.Errorf("after remove %q", got)
	}
}

func TestRead(t *testing.T) {
	dir := newRoot(t)
	o := options(dir)
	o.AllowWrite = true
	cs := connect(t, o)
	ok(t, cs, "new_group", map[string]any{"langs": []string{"en", "ja"}})
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(fileJa)), []byte("# 題\na.go\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	want := "{\n  \"group\": \"" + group + "\",\n  \"lang\": \"ja\",\n  \"file\": \"" + fileJa + "\",\n  \"title\": \"題\",\n  \"entries\": [\n    {\n      \"name\": \"a.go\",\n      \"path\": \"a.go\",\n      \"desc\": \"\",\n      \"detail\": \"\",\n      \"children\": []\n    }\n  ]\n}\n"
	if got := ok(t, cs, "show", map[string]any{"lang": "ja", "group": group}); got != want {
		t.Errorf("show\n%s\nwant\n%s", got, want)
	}

	if got := ok(t, cs, "status", map[string]any{"all": true}); !bytes.Contains([]byte(got), []byte(`"kind": "empty-desc"`)) ||
		!bytes.Contains([]byte(got), []byte(`"kind": "lang-mismatch"`)) {
		t.Errorf("status %s", got)
	}

	want = "{\n  \"groups\": [\n    {\n      \"group\": \"" + group + "\",\n      \"datetime\": \"2026-11-08T01:02:03Z\",\n      \"langs\": [\n        \"en\",\n        \"ja\"\n      ],\n      \"title\": \"題\",\n      \"entries\": 1\n    }\n  ]\n}\n"
	if got := ok(t, cs, "log", map[string]any{"lang": "ja"}); got != want {
		t.Errorf("log\n%s\nwant\n%s", got, want)
	}

	// A group that is not the newest.
	newer := "20261109T000000Z_abcd1111-2222-4333-8444-555566667777"
	if err := os.WriteFile(filepath.Join(dir, ".fnaze", "changes", newer+"_ja.txt"), []byte("# 新しい\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := ok(t, cs, "show", map[string]any{"lang": "ja"}); !strings.Contains(got, `"title": "新しい"`) {
		t.Errorf("show of the newest %s", got)
	}
	if got := ok(t, cs, "show", map[string]any{"lang": "ja", "group": group[:21]}); !strings.Contains(got, `"title": "題"`) {
		t.Errorf("show of %s %s", group[:21], got)
	}
}

// The errors are those of the commands, without "fnaze: ".
func TestErrors(t *testing.T) {
	dir := newRoot(t)
	o := options(dir)
	o.AllowWrite = true
	cs := connect(t, o)
	tests := []struct {
		name string
		args map[string]any
		want string
	}{
		{"show", nil, "there is no group yet: make one with fnaze new"},
		{"status", map[string]any{"all": true, "group": "x"}, "--all and a group cannot be given together"},
		{"log", map[string]any{"lang": "j-a"}, `"j-a" is not a language: use ASCII letters and digits`},
		{"set_description", map[string]any{"path": "a.go", "desc": "d"}, "there is no group yet: make one with fnaze new"},
		{"remove_description", map[string]any{"path": "a.go"}, "there is no group yet: make one with fnaze new"},
		{"new_group", map[string]any{"langs": []string{"ja", "JA"}}, "ja and JA differ only in case, so they cannot be in one group"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, isErr := call(t, cs, tt.name, tt.args)
			if !isErr || text != tt.want {
				t.Errorf("got %v %q, want an error %q", isErr, text, tt.want)
			}
		})
	}
}

func TestNoRoot(t *testing.T) {
	dir := t.TempDir()
	cs := connect(t, options(dir))
	text, isErr := call(t, cs, "log", nil)
	if want := "no .fnaze/ found in " + dir + " or above it: run fnaze init in the root of the project"; !isErr || text != want {
		t.Errorf("got %v %q, want %q", isErr, text, want)
	}
	// The root is looked for on each call.
	if _, err := core.Init(dir); err != nil {
		t.Fatal(err)
	}
	if got := ok(t, cs, "log", nil); got != "{\n  \"groups\": []\n}\n" {
		t.Errorf("after init %q", got)
	}
}
