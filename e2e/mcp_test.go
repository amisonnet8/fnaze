package e2e

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// mcpSession starts the built fnaze mcp in dir, with FNAZE_LANG set to lang
// ("" for unset), and connects to it over its standard input and output.
func mcpSession(t *testing.T, dir, lang string, args ...string) *mcp.ClientSession {
	t.Helper()
	cmd := exec.Command(bin, append([]string{"mcp"}, args...)...)
	cmd.Dir = dir
	cmd.Env = environ("FNAZE_LANG", "PWD")
	if lang != "" {
		cmd.Env = append(cmd.Env, "FNAZE_LANG="+lang)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "e2e", Version: "0"}, nil).
		Connect(context.Background(), &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

// call calls a tool and returns its text, and whether it is a tool error.
func call(t *testing.T, cs *mcp.ClientSession, tool string, args map[string]any) (string, bool) {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", tool, err)
	}
	if len(res.Content) != 1 {
		t.Fatalf("%s: %d contents", tool, len(res.Content))
	}
	return res.Content[0].(*mcp.TextContent).Text, res.IsError
}

// TestMCPTools checks the tools that the built fnaze mcp offers, with and
// without --allow-write.
func TestMCPTools(t *testing.T) {
	root := newRoot(t)
	for _, tt := range []struct {
		args []string
		want []string
	}{
		{nil, []string{"log", "show", "status"}},
		{[]string{"--allow-write"}, []string{"log", "new_group", "remove_description", "set_description", "show", "status"}},
	} {
		res, err := mcpSession(t, root, "", tt.args...).ListTools(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, tool := range res.Tools {
			got = append(got, tool.Name)
		}
		slices.Sort(got)
		if !slices.Equal(got, tt.want) {
			t.Errorf("fnaze mcp %s: %q, want %q", strings.Join(tt.args, " "), got, tt.want)
		}
	}
}

// TestMCPSameAsCLI checks that each tool of the built fnaze mcp returns, to
// the byte, what the built fnaze prints with --json, and fails with the same
// message without "fnaze: ". The server runs with -C in another directory,
// and FNAZE_LANG=ja, the language of a tool without lang.
func TestMCPSameAsCLI(t *testing.T) {
	cliRoot, mcpRoot := exampleRoot(t), exampleRoot(t)
	cs := mcpSession(t, t.TempDir(), "ja", "--allow-write", "-C", mcpRoot)
	env := []string{"FNAZE_LANG=ja"}
	steps := []struct {
		tool string
		args map[string]any
		cli  []string // without --json
	}{
		{"log", nil, []string{"log"}},
		{"log", map[string]any{"lang": "en"}, []string{"log", "--lang", "en"}},
		{"show", nil, []string{"show"}},
		{"show", map[string]any{"lang": "en", "group": exampleOld[:21]}, []string{"show", "--lang", "en", exampleOld[:21]}},
		{"status", map[string]any{"all": true}, []string{"status", "--all"}},
		{"show", map[string]any{"group": "nope"}, []string{"show", "nope"}},
		{"set_description", map[string]any{"path": "src/auth/session.go", "desc": "延長を数える", "detail": []string{"a", "b"}},
			[]string{"set", "src/auth/session.go", "延長を数える", "--detail", "a", "--detail", "b"}},
		{"set_description", map[string]any{"lang": "en", "group": exampleOld, "path": "lib/x.go", "desc": "X"},
			[]string{"set", "--lang", "en", "--group", exampleOld, "lib/x.go", "X"}},
		{"status", nil, []string{"status"}},
		{"status", map[string]any{"group": exampleOld}, []string{"status", exampleOld}},
		{"remove_description", map[string]any{"lang": "en", "group": exampleOld, "path": "src/"}, []string{"rm", "--lang", "en", "--group", exampleOld, "src/"}},
		{"remove_description", map[string]any{"path": "nope.go"}, []string{"rm", "nope.go"}},
		{"show", map[string]any{"lang": "en", "group": exampleOld}, []string{"show", "--lang", "en", exampleOld}},
	}
	for _, s := range steps {
		got, failed := call(t, cs, s.tool, s.args)
		r := fnaze(t, cliRoot, env, append(s.cli, "--json")...)
		switch {
		case r.code == 2:
			if !failed || "fnaze: "+got+"\n" != r.stderr {
				t.Errorf("%s %v: the tool returned (error %v) %q, the command said %q", s.tool, s.args, failed, got, r.stderr)
			}
		case failed || got != r.stdout:
			t.Errorf("%s %v: the tool returned (error %v):\n%s\nthe command printed:\n%s", s.tool, s.args, failed, got, r)
		}
	}
	// The writes left the same files.
	for _, n := range changes(t, cliRoot) {
		if a, b := read(t, cliRoot, ".fnaze/changes/"+n), read(t, mcpRoot, ".fnaze/changes/"+n); a != b {
			t.Errorf("%s differs:\nthe command wrote:\n%s\nthe tool wrote:\n%s", n, a, b)
		}
	}
}

// TestMCPNewGroup checks new_group of the built fnaze mcp: the languages of
// FNAZE_LANG when there is no group, then those given, and the root looked
// for on each call (it is made after the server starts).
func TestMCPNewGroup(t *testing.T) {
	dir := t.TempDir()
	cs := mcpSession(t, dir, "ja,en", "--allow-write")
	if got, failed := call(t, cs, "log", nil); !failed || !strings.HasPrefix(got, "no .fnaze/ found in ") {
		t.Errorf("log before init: (error %v) %q", failed, got)
	}
	if r := fnaze(t, dir, nil, "init"); r.code != 0 {
		t.Fatal(r)
	}
	type newResult struct {
		Group string `json:"group"`
		Files []struct {
			Lang string `json:"lang"`
			Path string `json:"path"`
		} `json:"files"`
	}
	for _, tt := range []struct {
		args  map[string]any
		langs []string
	}{
		{nil, []string{"ja", "en"}},
		{map[string]any{"langs": []string{"fr"}}, []string{"fr"}},
	} {
		got, failed := call(t, cs, "new_group", tt.args)
		var res newResult
		if err := json.Unmarshal([]byte(got), &res); failed || err != nil || !groupForm.MatchString(res.Group) {
			t.Fatalf("new_group %v: (error %v) %q", tt.args, failed, got)
		}
		var langs []string
		for _, f := range res.Files {
			langs = append(langs, f.Lang)
			name := res.Group + ".txt"
			if f.Lang != "en" {
				name = res.Group + "_" + f.Lang + ".txt"
			}
			if f.Path != ".fnaze/changes/"+name || read(t, dir, filepath.ToSlash(f.Path)) != "# \n" {
				t.Errorf("new_group %v: %s", tt.args, got)
			}
		}
		if !slices.Equal(langs, tt.langs) {
			t.Errorf("new_group %v: the languages %q, want %q", tt.args, langs, tt.langs)
		}
	}
}
