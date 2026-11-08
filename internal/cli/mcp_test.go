package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/amisonnet8/fnaze/internal/mcpserver"
)

func TestMCPOptions(t *testing.T) {
	r := &runner{env: Env{Getenv: func(k string) string {
		if k == "FNAZE_LANG" {
			return "ja,en"
		}
		return ""
	}, Version: "v0.1.0"}, dir: "sub"}
	for _, args := range [][]string{{}, {"--allow-write"}} {
		p, err := parse(args, commands["mcp"].options)
		if err != nil {
			t.Fatal(err)
		}
		o := r.mcpOptions(p)
		if o.Dir != "sub" || o.EnvLang != "ja,en" || o.AllowWrite != (len(args) == 1) || o.Version != "v0.1.0" {
			t.Errorf("%q: %+v", args, o)
		}
	}
}

// The MCP tools return the same JSON as the commands with --json, to the
// byte, and their errors are the messages of the commands without "fnaze: "
// (docs/reference/commands.md, "fnaze mcp").
func TestMCPSameAsJSON(t *testing.T) {
	cliDir, mcpDir := newRoot(t), newRoot(t)
	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	ss, err := mcpserver.New(mcpserver.Options{
		Dir:        mcpDir,
		AllowWrite: true,
		Now:        func() time.Time { return time.Date(2026, 11, 8, 1, 2, 3, 0, time.UTC) },
		Random:     bytes.NewReader([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}),
	}).Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ss.Close() })
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })

	steps := []struct {
		args []string // the command line, without --json and -C
		tool string
		in   map[string]any
	}{
		{[]string{"show"}, "show", nil}, // an error: no group yet
		{[]string{"new", "en", "ja"}, "new_group", map[string]any{"langs": []string{"en", "ja"}}},
		{[]string{"set", "src/a&b.go", "<延ばす>", "--detail", "一つ", "--detail", "二つ"}, "set_description", map[string]any{"path": "src/a&b.go", "desc": "<延ばす>", "detail": []string{"一つ", "二つ"}}},
		{[]string{"set", "--lang", "ja", "docs/", "文書"}, "set_description", map[string]any{"lang": "ja", "path": "docs/", "desc": "文書"}},
		{[]string{"show", group[:21]}, "show", map[string]any{"group": group[:21]}},
		{[]string{"status", "--all"}, "status", map[string]any{"all": true}},
		{[]string{"log", "--lang", "ja"}, "log", map[string]any{"lang": "ja"}},
		{[]string{"rm", "src/a&b.go"}, "remove_description", map[string]any{"path": "src/a&b.go"}},
		{[]string{"rm", "src/a&b.go"}, "remove_description", map[string]any{"path": "src/a&b.go"}}, // an error: no entry
		{[]string{"show", "--lang", "fr"}, "show", map[string]any{"lang": "fr"}},                   // an error: no file
	}
	for _, s := range steps {
		code, out, errs := run(t, "", append(s.args, "--json", "-C", cliDir)...)
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: s.tool, Arguments: s.in})
		if err != nil {
			t.Fatalf("%s: %v", s.tool, err)
		}
		text := res.Content[0].(*mcp.TextContent).Text
		if code == 0 {
			if res.IsError || text != out {
				t.Errorf("%s: MCP\n%s\nCLI\n%s", s.args, text, out)
			}
			continue
		}
		if !res.IsError || "fnaze: "+text+"\n" != strings.ReplaceAll(errs, cliDir, mcpDir) {
			t.Errorf("%s: MCP error %v %q, CLI %q", s.args, res.IsError, text, errs)
		}
	}
	// And the files are the same.
	for _, f := range []string{fileEn, fileJa} {
		a, _ := os.ReadFile(filepath.Join(cliDir, filepath.FromSlash(f)))
		b, _ := os.ReadFile(filepath.Join(mcpDir, filepath.FromSlash(f)))
		if !bytes.Equal(a, b) {
			t.Errorf("%s: CLI %q, MCP %q", f, a, b)
		}
	}
}
