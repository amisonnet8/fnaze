package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
)

// groupForm is the form of a group that fnaze new makes: the datetime in
// UTC, and a random UUID (version 4, variant 10).
var groupForm = regexp.MustCompile(`^[0-9]{8}T[0-9]{6}Z_[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// expect checks a run of the built fnaze.
func expect(t *testing.T, got result, code int, stdout, stderr string) {
	t.Helper()
	if got.code != code || got.stdout != stdout || got.stderr != stderr {
		t.Errorf("got:\n%s\nwant:\n%s", got, result{code, stdout, stderr})
	}
}

// lines joins lines, each ending in "\n".
func lines(ls ...string) string {
	if len(ls) == 0 {
		return ""
	}
	return strings.Join(ls, "\n") + "\n"
}

// TestCommands runs every command of the built fnaze, as one user would in a
// project: from a subdirectory, with FNAZE_LANG set, then from elsewhere
// with -C.
func TestCommands(t *testing.T) {
	root := newRoot(t)
	sub := filepath.Join(root, "src")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	env := []string{"FNAZE_LANG=ja,en"}
	// The text output has paths from the current directory, src/.
	fromSub := func(p string) string { return filepath.FromSlash("../" + p) }

	// init: only one .fnaze/ in a project, found from src/ upward.
	expect(t, fnaze(t, sub, nil, "init"), 2, "", "fnaze: "+root+" already has .fnaze/: a project has only one\n")

	// new: the languages of FNAZE_LANG, in its order, the same group.
	r := fnaze(t, sub, env, "new")
	names := changes(t, root)
	if len(names) != 2 {
		t.Fatalf("new made %q\n%s", names, r)
	}
	group := strings.TrimSuffix(names[0], ".txt")
	if !groupForm.MatchString(group) || names[1] != group+"_ja.txt" {
		t.Fatalf("new made %q", names)
	}
	en := ".fnaze/changes/" + group + ".txt"
	ja := ".fnaze/changes/" + group + "_ja.txt"
	expect(t, r, 0, lines(fromSub(ja), fromSub(en)), "")
	for _, f := range []string{en, ja} {
		if got := read(t, root, f); got != "# \n" {
			t.Errorf("%s: %q, want the empty title", f, got)
		}
	}
	// The datetime is when it was made, in UTC.
	made, err := time.Parse("20060102T150405Z", group[:16])
	if err != nil || time.Since(made) < 0 || time.Since(made) > time.Hour {
		t.Errorf("the datetime of %s is not now: %v", group, err)
	}

	// set: the first language of FNAZE_LANG, or --lang; nothing printed. A
	// new path goes under the deepest directory entry above it, with skeleton
	// lines for the levels that are missing.
	expect(t, fnaze(t, sub, env, "set", "src/auth/session.go", "期限を延ばす"), 0, "", "")
	expect(t, fnaze(t, sub, env, "set", "--lang", "en", "src/auth/session.go", "Extend the lifetime",
		"--detail", "Sessions expired too soon", "--detail", "Reported by users"), 0, "", "")
	expect(t, fnaze(t, sub, env, "set", "src/auth/session.go", "アクセスのたびに期限を延ばす", "--lang=ja", "--detail", "短すぎた"), 0, "", "")
	expect(t, fnaze(t, sub, env, "set", "--lang", "en", "docs/old.md", "Removed"), 0, "", "")
	if got, want := read(t, root, en), "# \nsrc/\n\tauth/\n\t\tsession.go\tExtend the lifetime\n\t\t\t| Sessions expired too soon\n\t\t\t| Reported by users\ndocs/\n\told.md\tRemoved\n"; got != want {
		t.Errorf("%s:\n%s\nwant:\n%s", en, got, want)
	}
	if got, want := read(t, root, ja), "# \nsrc/\n\tauth/\n\t\tsession.go\tアクセスのたびに期限を延ばす\n\t\t\t| 短すぎた\n"; got != want {
		t.Errorf("%s:\n%s\nwant:\n%s", ja, got, want)
	}

	// status: the empty titles and the language mismatch, from src/; the exit
	// code is 1 only with --require. The message is the same as in JSON, so
	// the file it names is from the root.
	report := lines(
		fromSub(en)+":1: empty-title: the title is empty",
		fromSub(ja)+": lang-mismatch: the path lines differ from those of "+en,
		fromSub(ja)+":1: empty-title: the title is empty",
	)
	expect(t, fnaze(t, sub, env, "status"), 0, report, "")
	expect(t, fnaze(t, sub, env, "status", "--require"), 1, report, "")

	// The titles are written by hand, and the ja file gets docs/old.md.
	write(t, root, map[string]string{
		en: "# Extend the session lifetime\n" + strings.TrimPrefix(read(t, root, en), "# \n"),
		ja: "# セッションの期限を延ばす\n" + strings.TrimPrefix(read(t, root, ja), "# \n"),
	})
	expect(t, fnaze(t, sub, env, "set", "docs/old.md", "消した"), 0, "", "")
	expect(t, fnaze(t, sub, env, "status", "--require"), 0, "", "")
	expect(t, fnaze(t, sub, env, "status", "--require", "--all"), 0, "", "")

	// show: the language of FNAZE_LANG, or --lang.
	expect(t, fnaze(t, sub, env, "show"), 0, lines(
		"# セッションの期限を延ばす",
		"",
		"src/",
		"  auth/",
		"    session.go  アクセスのたびに期限を延ばす",
		"                | 短すぎた",
		"docs/",
		"  old.md        消した",
	), "")
	expect(t, fnaze(t, sub, env, "show", "--lang", "en", group[:21]), 0, lines(
		"# Extend the session lifetime",
		"",
		"src/",
		"  auth/",
		"    session.go  Extend the lifetime",
		"                | Sessions expired too soon",
		"                | Reported by users",
		"docs/",
		"  old.md        Removed",
	), "")

	// log: the datetime in the local time zone, which this process and the
	// built fnaze have from the same environment, and the number of entries
	// (skeleton lines included).
	local := made.Local().Format("2006-01-02 15:04")
	expect(t, fnaze(t, sub, env, "log"), 0, lines(group[:21]+"  "+local+"  en,ja  5  セッションの期限を延ばす"), "")
	expect(t, fnaze(t, sub, nil, "log"), 0, lines(group[:21]+"  "+local+"  en,ja  5  Extend the session lifetime"), "")

	// rm: the entry with its children and the skeleton lines left empty.
	expect(t, fnaze(t, sub, env, "rm", "--lang", "en", "src/auth/session.go"), 0, "", "")
	if got, want := read(t, root, en), "# Extend the session lifetime\ndocs/\n\told.md\tRemoved\n"; got != want {
		t.Errorf("%s:\n%s\nwant:\n%s", en, got, want)
	}

	// A second group: new with no language takes those of the newest group,
	// and the path lines of its newest files. It can be made in the same
	// second, and then it is not always after the first in name order.
	r = fnaze(t, root, nil, "new")
	names = changes(t, root)
	if len(names) != 4 {
		t.Fatalf("new made %q\n%s", names, r)
	}
	second := strings.TrimSuffix(names[2], ".txt")
	if second == group {
		second = strings.TrimSuffix(names[0], ".txt")
	}
	if !groupForm.MatchString(second) || !slices.Contains(names, second+"_ja.txt") || second[:16] < group[:16] {
		t.Fatalf("new made %q", names)
	}
	newest := max(group, second)
	expect(t, r, 0, lines(filepath.FromSlash(".fnaze/changes/"+second+".txt"), filepath.FromSlash(".fnaze/changes/"+second+"_ja.txt")), "")
	if got, want := read(t, root, ".fnaze/changes/"+second+"_ja.txt"), "# \nsrc/\n\tauth/\n\t\tsession.go\ndocs/\n\told.md\n"; got != want {
		t.Errorf("the new ja file:\n%s\nwant:\n%s", got, want)
	}

	// -C from elsewhere, and --json: paths from the root, datetimes in UTC.
	elsewhere, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var log struct {
		Groups []struct {
			Group    string   `json:"group"`
			Datetime string   `json:"datetime"`
			Langs    []string `json:"langs"`
			Title    string   `json:"title"`
			Entries  int      `json:"entries"`
		} `json:"groups"`
	}
	decode(t, fnaze(t, elsewhere, nil, "log", "-C", sub, "--json"), &log)
	first := slices.IndexFunc(log.Groups, func(g struct {
		Group    string   `json:"group"`
		Datetime string   `json:"datetime"`
		Langs    []string `json:"langs"`
		Title    string   `json:"title"`
		Entries  int      `json:"entries"`
	}) bool {
		return g.Group == group
	})
	if len(log.Groups) != 2 || log.Groups[0].Group != newest || first < 0 ||
		log.Groups[first].Datetime != made.Format(time.RFC3339) || !slices.Equal(log.Groups[first].Langs, []string{"en", "ja"}) ||
		log.Groups[first].Title != "Extend the session lifetime" || log.Groups[first].Entries != 2 {
		t.Errorf("log --json: %+v", log)
	}
	var status struct {
		Items []struct {
			Group, File, Kind string
			Line              int
		} `json:"items"`
	}
	decode(t, fnaze(t, elsewhere, nil, "status", "-C", root, "--json", group), &status)
	if len(status.Items) != 1 || status.Items[0].File != ja || status.Items[0].Kind != "lang-mismatch" || status.Items[0].Group != group {
		t.Errorf("status --json: %+v", status)
	}
	var set struct{ Group, Lang, Path, Desc, Detail string }
	decode(t, fnaze(t, elsewhere, nil, "set", "-C", root, "--json", "--group", group, "--detail", "a\nb", "lib/", "Library"), &set)
	if set.Group != group || set.Lang != "en" || set.Path != "lib/" || set.Desc != "Library" || set.Detail != "a\nb" {
		t.Errorf("set --json: %+v", set)
	}

	// Errors: exit code 2 and one line on standard error, in every command.
	errorCases := []struct {
		dir    string
		args   []string
		stderr string
	}{
		{elsewhere, []string{"show"}, "fnaze: no .fnaze/ found in " + elsewhere + " or above it: run fnaze init in the root of the project"},
		{root, []string{"show", "nope"}, "fnaze: no group starts with nope"},
		{root, []string{"nope"}, `fnaze: unknown command "nope" (see fnaze --help)`},
		{root, []string{"status", "--all", group}, "fnaze: --all and a group cannot be given together"},
		{root, []string{"set", "--lang", "fr", "a.go", "A"}, "fnaze: group " + newest + " has no file in fr"},
		{root, []string{"rm", "nope.go"}, "fnaze: .fnaze/changes/" + newest + ".txt: no entry for nope.go"},
	}
	for _, c := range errorCases {
		r := fnaze(t, c.dir, nil, c.args...)
		if r.code != 2 || r.stdout != "" || r.stderr != c.stderr+"\n" {
			t.Errorf("fnaze %s:\n%s\nwant exit 2 and %q", strings.Join(c.args, " "), r, c.stderr)
		}
	}
}

// decode reads the JSON that a run printed, which must have succeeded.
func decode(t *testing.T, r result, v any) {
	t.Helper()
	if r.code != 0 || r.stderr != "" {
		t.Fatalf("%s", r)
	}
	if err := json.Unmarshal([]byte(r.stdout), v); err != nil {
		t.Fatalf("%v\n%s", err, r)
	}
}

// TestUsageAndVersion checks what the built fnaze says about itself.
func TestUsageAndVersion(t *testing.T) {
	dir := t.TempDir()
	expect(t, fnaze(t, dir, nil, "--version"), 0, "fnaze v0.1.0\n", "")
	expect(t, fnaze(t, dir, nil, "--help"), 0, lines(
		"usage: fnaze <command> [arguments]",
		"",
		"commands:",
		"  init        create .fnaze/changes/",
		"  new         create a new group, one file for each language",
		"  title       write the title of a file",
		"  set         write one entry",
		"  rm          remove one entry",
		"  status      check a group",
		"  show        show a file of a group as a tree",
		"  log         list the groups, newest first",
		"  mcp         run as an MCP server over standard input and output",
		"  completion  print a shell completion script",
		"",
		"common options: -C <dir>  --json",
		"see fnaze <command> --help, and fnaze --version",
	), "")
	expect(t, fnaze(t, dir, nil, "set", "--help"), 0,
		"usage: fnaze set [--lang <language>] [--group <group>] <path> <desc> [--detail <detail>]... [-C <dir>] [--json]\n", "")
	// With nothing to do, the usage goes to standard error.
	r := fnaze(t, dir, nil)
	if r.code != 2 || r.stdout != "" || !strings.HasPrefix(r.stderr, "usage: fnaze <command>") {
		t.Errorf("fnaze with no command:\n%s", r)
	}
}

// TestCompletionScripts checks that the built fnaze prints the scripts of
// internal/cli/completions/ as they are, and that the command they call
// answers from the .fnaze/ of the line.
func TestCompletionScripts(t *testing.T) {
	dir := t.TempDir()
	for shell, file := range map[string]string{"bash": "fnaze.bash", "zsh": "fnaze.zsh", "fish": "fnaze.fish", "powershell": "fnaze.ps1"} {
		expect(t, fnaze(t, dir, nil, "completion", shell), 0, read(t, "..", "internal/cli/completions/"+file), "")
	}
	expect(t, fnaze(t, dir, nil, "completion", "tcsh"), 2, "", "fnaze: completion: unknown shell \"tcsh\" (use bash, zsh, fish or powershell)\n")
}
