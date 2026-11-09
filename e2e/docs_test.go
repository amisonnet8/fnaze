package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
	_ "time/tzdata" // Asia/Tokyo for the example of log, on every OS
)

// block is a fenced code block of a Markdown file.
type block struct {
	body string // the lines inside, each ending in "\n"
	line int    // the line of the opening ```
}

// readBlocks reads the code blocks of a file of the repository.
func readBlocks(t *testing.T, name string) []block {
	t.Helper()
	var blocks []block
	var cur *block
	for i, line := range strings.Split(read(t, "..", name), "\n") {
		switch {
		case cur == nil && strings.HasPrefix(line, "```"):
			cur = &block{line: i + 1}
		case cur != nil && strings.HasPrefix(line, "```"):
			blocks = append(blocks, *cur)
			cur = nil
		case cur != nil:
			cur.body += line + "\n"
		}
	}
	return blocks
}

// docPages are the two languages of a page of docs/: X.md in English and
// X_ja.md in Japanese.
func docPages(name string) []string {
	return []string{name + ".md", name + "_ja.md"}
}

// exampleGroups are the groups of docs/examples/changes/.
const (
	exampleOld = "20261107T003426Z_8980a325-8b57-49b0-8e0a-ed0aaf9eeda7"
	exampleNew = "20261107T021530Z_2c4cc818-64d6-45e8-9f3a-5471dba04be7"
)

// exampleRoot is a root whose .fnaze/changes/ has the files of
// docs/examples/changes/.
func exampleRoot(t *testing.T) string {
	t.Helper()
	root := newRoot(t)
	entries, err := os.ReadDir(filepath.Join("..", "docs", "examples", "changes"))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, e := range entries {
		files[".fnaze/changes/"+e.Name()] = read(t, "..", "docs/examples/changes/"+e.Name())
	}
	write(t, root, files)
	return root
}

// checkBlocks runs check on each code block of the pages, and fails on a
// block that check does not know (it returns false), so that a new example
// gets checked too. Both languages have the same blocks in the same order.
func checkBlocks(t *testing.T, pages []string, count int, check func(t *testing.T, page string, i int, body string) bool) {
	t.Helper()
	for _, page := range pages {
		blocks := readBlocks(t, page)
		if len(blocks) != count {
			t.Errorf("%s has %d code blocks, the test knows %d", page, len(blocks), count)
		}
		for i, b := range blocks {
			t.Run(page+"#"+strings.SplitN(b.body, "\n", 2)[0], func(t *testing.T) {
				if !check(t, page, i, b.body) {
					t.Errorf("%s:%d: a code block the test does not know; add how to check it", page, b.line)
				}
			})
		}
	}
}

// same fails when fnaze did not print what the docs say.
func same(t *testing.T, what string, got result, want string) {
	t.Helper()
	if got.code != 0 || got.stderr != "" || got.stdout != want {
		t.Errorf("%s:\n%s\nthe docs say:\n%s", what, got, want)
	}
}

// TestCommandsExamples checks the examples of docs/reference/commands.md.
func TestCommandsExamples(t *testing.T) {
	newJa := map[string]string{} // the file before fnaze new ja, by page
	checkBlocks(t, docPages("docs/reference/commands"), 9, func(t *testing.T, page string, i int, body string) bool {
		switch i {
		case 0: // what new makes, by its arguments: in a comment
			for _, l := range strings.Split(strings.TrimSuffix(body, "\n"), "\n") {
				if !strings.HasPrefix(l, "fnaze new") {
					t.Errorf("not a command of new: %q", l)
				}
			}
		case 1: // fnaze new en ja
			root := newRoot(t)
			cmd, want, _ := strings.Cut(body, "\n")
			args := strings.Fields(strings.TrimPrefix(cmd, "$ fnaze "))
			same(t, cmd, runCLI("", append(args, "-C", root)...), filepath.FromSlash(want))
		case 2: // the newest ja file
			newJa[page] = body
		case 3: // and what fnaze new ja makes from it
			root := newRoot(t)
			write(t, root, map[string]string{".fnaze/changes/" + exampleOld + "_ja.txt": newJa[page]})
			same(t, "fnaze new ja", runCLI("", "new", "ja", "-C", root), filepath.FromSlash(".fnaze/changes/"+newGroup+"_ja.txt\n"))
			if got := read(t, root, ".fnaze/changes/"+newGroup+"_ja.txt"); got != body {
				t.Errorf("fnaze new ja made:\n%s\nthe docs say:\n%s", got, body)
			}
		case 4: // status: an empty desc on line 3, and a file that is not a description file
			root := newRoot(t)
			write(t, root, map[string]string{
				".fnaze/changes/" + newGroup + ".txt": "# Title\nsrc/\n\ta.go\n",
				".fnaze/changes/notes.txt":            "",
			})
			// The file at the start of a line is a path of the OS; the path
			// in a message is from the root, with "/" on every OS.
			var want strings.Builder
			for _, l := range strings.SplitAfter(body, "\n") {
				i := strings.Index(l, ".txt") + len(".txt")
				if i < len(".txt") {
					i = 0
				}
				want.WriteString(filepath.FromSlash(l[:i]) + l[i:])
			}
			same(t, "fnaze status", runCLI("", "status", "-C", root), want.String())
		case 5: // show of docs/examples/changes/, the older group in en
			same(t, "fnaze show "+exampleOld[:21], runCLI("", "show", exampleOld[:21], "-C", exampleRoot(t)), body)
		case 6: // log of docs/examples/changes/, in TZ=Asia/Tokyo
			tokyo, err := time.LoadLocation("Asia/Tokyo")
			if err != nil {
				t.Fatal(err)
			}
			defer func(l *time.Location) { time.Local = l }(time.Local)
			time.Local = tokyo
			same(t, "fnaze log", runCLI("", "log", "-C", exampleRoot(t)), body)
		case 7: // where to put each completion script: one line for each shell
			var shells []string
			for _, l := range strings.Split(strings.TrimSuffix(body, "\n"), "\n") {
				f := strings.Fields(l)
				if len(f) < 3 || f[0] != "fnaze" || f[1] != "completion" {
					t.Errorf("not a command of completion: %q", l)
					continue
				}
				shells = append(shells, f[2])
			}
			if !slices.Equal(shells, []string{"bash", "zsh", "fish", "powershell"}) {
				t.Errorf("the shells are %q", shells)
			}
		case 8: // the paths that completion offers, with their descs
			checkCompletionExample(t, body)
		default:
			return false
		}
		return true
	})
}

// checkCompletionExample checks the example of what a shell shows for the
// paths of rm: each value with its desc, as fnaze __complete gives them.
func checkCompletionExample(t *testing.T, body string) {
	t.Helper()
	cmd, rest, _ := strings.Cut(body, "\n")
	line, ok := strings.CutSuffix(strings.TrimPrefix(cmd, "$ "), "<TAB>")
	if !ok {
		t.Fatalf("not a line with <TAB>: %q", cmd)
	}
	words := strings.Split(line, " ")
	root := newRoot(t)
	write(t, root, map[string]string{
		".fnaze/changes/" + newGroup + ".txt": "# T\nsrc/\n\tauth/\tAuth\n\t\tsession.go\tExtend\n\t\t*_test.go\tTests\n",
	})
	args := append([]string{"__complete", "--word=" + words[len(words)-1], "--"}, words[1:len(words)-1]...)
	args = append(args, "-C", root)
	var want strings.Builder
	columns := regexp.MustCompile(`  +`)
	for _, l := range strings.Split(strings.TrimSuffix(rest, "\n"), "\n") {
		want.WriteString(columns.ReplaceAllString(l, "\t") + "\n")
	}
	same(t, line, runCLI("", args...), want.String())
}

// TestFileFormatExamples checks the examples of
// docs/reference/file-format.md against docs/examples/changes/, and that
// fnaze reads them without a report.
func TestFileFormatExamples(t *testing.T) {
	checkBlocks(t, docPages("docs/reference/file-format"), 6, func(t *testing.T, page string, i int, body string) bool {
		switch i {
		case 0, 1, 4, 5: // the forms of the names and lines: no example to run
			if !strings.Contains(body, "<") {
				t.Errorf("not a form: %q", body)
			}
		case 2: // the names of a .fnaze/changes/
			want := ".fnaze/changes/\n"
			for _, n := range changes(t, exampleRoot(t)) {
				want += "\t" + n + "\n"
			}
			if body != want {
				t.Errorf("the docs list:\n%s\ndocs/examples/changes/ has:\n%s", body, want)
			}
		case 3: // a file: the older example, in the language of the page
			name := exampleOld + ".txt"
			if strings.HasSuffix(page, "_ja.md") {
				name = exampleOld + "_ja.txt"
			}
			if want := read(t, "..", "docs/examples/changes/"+name); body != want {
				t.Errorf("the docs show:\n%s\ndocs/examples/changes/%s is:\n%s", body, name, want)
			}
		default:
			return false
		}
		return true
	})
}

// TestExampleChanges checks that fnaze reads docs/examples/changes/ without
// a report, and that it writes them as they are: the files are in the form
// that fnaze writes.
func TestExampleChanges(t *testing.T) {
	root := exampleRoot(t)
	same(t, "fnaze status --require --all", runCLI("", "status", "--require", "--all", "-C", root), "")
	// A set that changes nothing writes the file again, in the form fnaze
	// writes.
	for _, f := range []struct{ group, lang, path, desc string }{
		{exampleOld, "en", "docs/old.md", "Removed; its content moved to docs/auth.md"},
		{exampleOld, "ja", "docs/old.md", "消した。内容は docs/auth.md に移した"},
		{exampleNew, "en", "src/auth/session.go", "Log each extension at debug level"},
		{exampleNew, "ja", "src/auth/session.go", "延長のたびに debug でログを出す"},
	} {
		name := ".fnaze/changes/" + f.group + ".txt"
		if f.lang != "en" {
			name = ".fnaze/changes/" + f.group + "_" + f.lang + ".txt"
		}
		before := read(t, root, name)
		same(t, "fnaze set", runCLI("", "set", "--lang", f.lang, "--group", f.group, f.path, f.desc, "-C", root), "")
		if after := read(t, root, name); after != before {
			t.Errorf("%s is not in the form fnaze writes; fnaze wrote:\n%s", name, after)
		}
	}
}

// TestPreCommitExample runs docs/examples/pre-commit as the hook of a git
// repository, with the built fnaze on the PATH: it stops a commit without a
// description file, and one whose description is not fully written.
func TestPreCommitExample(t *testing.T) {
	git := lookPath(t, "git")
	lookPath(t, "sh")
	dir := newRoot(t)
	home := t.TempDir()
	env := append(environ("PATH", "HOME", "GIT_CONFIG_GLOBAL", "GIT_CONFIG_SYSTEM", "GIT_DIR", "GIT_WORK_TREE", "FNAZE_LANG"),
		"PATH="+filepath.Dir(bin)+string(os.PathListSeparator)+os.Getenv("PATH"),
		"HOME="+home, "GIT_CONFIG_GLOBAL="+filepath.Join(home, ".gitconfig"), "GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=e2e", "GIT_AUTHOR_EMAIL=e2e@example.com", "GIT_COMMITTER_NAME=e2e", "GIT_COMMITTER_EMAIL=e2e@example.com")
	run := func(args ...string) result {
		t.Helper()
		return runProgram(t, git, dir, env, args...)
	}
	if r := run("init", "-q"); r.code != 0 {
		t.Fatalf("git init:\n%s", r)
	}
	hook := filepath.Join(dir, ".git", "hooks", "pre-commit")
	write(t, dir, map[string]string{".git/hooks/pre-commit": read(t, "..", "docs/examples/pre-commit")})
	if err := os.Chmod(hook, 0o755); err != nil {
		t.Fatal(err)
	}

	write(t, dir, map[string]string{"a.go": "package a\n"})
	run("add", "a.go")
	r := run("commit", "-q", "-m", "a")
	if r.code == 0 || !strings.Contains(r.stderr, "pre-commit: this commit adds no description in .fnaze/changes/.") {
		t.Errorf("a commit without a description:\n%s", r)
	}

	if r := fnaze(t, dir, nil, "new"); r.code != 0 {
		t.Fatal(r)
	}
	name := changes(t, dir)[0]
	run("add", ".fnaze")
	r = run("commit", "-q", "-m", "a")
	if r.code == 0 || !strings.Contains(r.stdout+r.stderr, name+":1: empty-title: the title is empty") {
		t.Errorf("a commit with an empty title:\n%s", r)
	}

	write(t, dir, map[string]string{".fnaze/changes/" + name: "# Add a\na.go\tAdd a\n"})
	run("add", ".fnaze")
	if r := run("commit", "-q", "-m", "a"); r.code != 0 {
		t.Errorf("a commit with its description:\n%s", r)
	}
}

// TestOwnPreCommit checks that the pre-commit of this repository
// (.githooks/pre-commit, core.hooksPath) is the example as it is, so that what
// this repository runs is what TestPreCommitExample checks.
func TestOwnPreCommit(t *testing.T) {
	if own, example := read(t, "..", ".githooks/pre-commit"), read(t, "..", "docs/examples/pre-commit"); own != example {
		t.Errorf(".githooks/pre-commit differs from docs/examples/pre-commit:\n%s", own)
	}
}

// lookPath finds a program, or skips the test.
func lookPath(t *testing.T, name string) string {
	t.Helper()
	p, err := exec.LookPath(name)
	if err != nil {
		t.Skipf("%s is not installed", name)
	}
	return p
}
