package e2e

// The completion scripts are run by the real shells, with the fnaze built
// from this source on the PATH: a line is typed as far as the cursor, and
// what the shell would put in is read back. Each shell is a different
// program with different rules about words (an empty one may be dropped,
// quotes are kept), so this is what shows that the scripts pass the line to
// fnaze __complete intact, and quote a pattern. What is offered is decided by
// fnaze and tested in internal/cli.
//
// A shell that is not installed is skipped. The devcontainer has all four
// (.devcontainer/postCreate.sh, devcontainer.json); the runners of CI have
// bash and PowerShell.

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// completionGroup is the group of the project of the completion tests.
const completionGroup = "20261107T003426Z_8980a325-8b57-49b0-8e0a-ed0aaf9eeda7"

// completionCase is a line as typed, with the cursor at its end, and the
// values that fnaze offers for it, as fnaze gives them.
type completionCase struct {
	line string
	want []string
}

var completionCases = []completionCase{
	{"fnaze sh", []string{"show"}},
	{"fnaze status --re", []string{"--require"}},
	{"fnaze completion ", []string{"bash", "zsh", "fish", "powershell"}},
	{"fnaze show ", []string{completionGroup}},
	{"fnaze new ", []string{"en", "ja"}},
	{"fnaze new ja ", []string{"en"}},
	{"fnaze rm ", []string{"src/", "config/"}},
	{"fnaze rm src/", []string{"src/", "src/auth/"}},
	{"fnaze rm src/auth/", []string{"src/auth/", "src/auth/session.go", "src/auth/*_test.go"}},
	{"fnaze rm config/", []string{"config/auth.yaml"}},
	{"fnaze rm --lang ja s", []string{"src/"}},
	{"fnaze log x", nil},
}

// completionShell runs scripts in one shell, in a project, with the fnaze
// that was built.
type completionShell struct {
	t    *testing.T
	path string
	args []string
	root string
	env  []string
}

// newCompletionShell finds a shell, or skips the test. A shell that is there
// but does not run the built fnaze is a failure, not a skip: the test would
// have run and shown nothing.
func newCompletionShell(t *testing.T, name string, args ...string) *completionShell {
	t.Helper()
	path, err := exec.LookPath(name)
	if err != nil {
		t.Skipf("%s is not installed", name)
	}
	if name == "bash" {
		path = gitBash(path)
	}
	root := newRoot(t)
	write(t, root, map[string]string{
		".fnaze/changes/" + completionGroup + ".txt":    "# New\nsrc/\n\tauth/\tAuth\n\t\tsession.go\tExtend\n\t\t*_test.go\tTests\nconfig/auth.yaml\tTTL\n",
		".fnaze/changes/" + completionGroup + "_ja.txt": "# 新しい\nsrc/\tソース\n\tauth/\n\t\tsession.go\n\t\t*_test.go\nconfig/auth.yaml\n",
	})
	// A home of its own: fish wants to write there, and no profile of the
	// user is read.
	home := t.TempDir()
	env := append(environ("PATH", "FNAZE_LANG", "HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME"),
		"PATH="+filepath.Dir(bin)+string(os.PathListSeparator)+os.Getenv("PATH"), "HOME="+home, "TERM=dumb",
		"POWERSHELL_UPDATECHECK=Off", "POWERSHELL_TELEMETRY_OPTOUT=1", "DOTNET_CLI_TELEMETRY_OPTOUT=1")
	s := &completionShell{t: t, path: path, args: args, root: root, env: env}
	if got := strings.TrimSpace(s.run("fnaze --version\n")); got != "fnaze v0.1.0" {
		t.Fatalf("%s does not run the fnaze that was built: %q", name, got)
	}
	return s
}

// gitBash is the bash to use on Windows, where the one that the PATH finds
// first may be the launcher of the Windows Subsystem for Linux, which cannot
// run the fnaze.exe that was built. Git for Windows has a bash of its own.
func gitBash(found string) string {
	if runtime.GOOS != "windows" || !strings.Contains(strings.ToLower(found), `\system32\`) {
		return found
	}
	own := filepath.Join(os.Getenv("ProgramFiles"), "Git", "bin", "bash.exe")
	if _, err := os.Stat(own); err == nil {
		return own
	}
	return found
}

// run runs a script on the standard input of the shell, in the project, and
// returns what it printed.
func (s *completionShell) run(script string) string {
	s.t.Helper()
	cmd := exec.Command(s.path, s.args...)
	cmd.Dir = s.root
	cmd.Env = s.env
	cmd.Stdin = strings.NewReader(script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		s.t.Fatalf("%s: %v\n%s\n%s", s.path, err, stdout.String(), stderr.String())
	}
	// What a shell says on standard error is not a candidate.
	if stderr.Len() > 0 {
		s.t.Logf("%s said on standard error:\n%s", s.path, stderr.String())
	}
	return stdout.String()
}

// script is the completion script of a shell, from the built fnaze.
func (s *completionShell) script(shell string) string {
	s.t.Helper()
	r := fnaze(s.t, s.root, nil, "completion", shell)
	if r.code != 0 {
		s.t.Fatalf("fnaze completion %s:\n%s", shell, r)
	}
	return r.stdout
}

// check reads the output of a run of the cases: a line "### n" starts the
// answer of case n, and each line after it is a value as the shell puts it
// in (before cut, which starts a description). put is how the shell puts in
// a value that fnaze gave.
func check(t *testing.T, cases []completionCase, out, cut string, put func(string) string) {
	t.Helper()
	answers := make([][]string, len(cases))
	current := -1
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if rest, ok := strings.CutPrefix(line, "### "); ok {
			_, _ = fmt.Sscanf(rest, "%d", &current)
			continue
		}
		if current < 0 || line == "" {
			continue
		}
		value, _, _ := strings.Cut(line, cut)
		answers[current] = append(answers[current], value)
	}
	for i, tc := range cases {
		var want []string
		for _, v := range tc.want {
			want = append(want, put(v))
		}
		got := slices.Clone(answers[i])
		slices.Sort(got)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Errorf("%q:\n got  %q\n want %q", tc.line, got, want)
		}
	}
}

// quotePattern is how bash and PowerShell put in a pattern: in single quotes.
func quotePattern(v string) string {
	if strings.Contains(v, "*") {
		return "'" + v + "'"
	}
	return v
}

func asIs(v string) string { return v }

// The single quotes of each shell. Nothing typed here has one, but a line
// that had would otherwise end the string.
func quote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

func quoteFish(s string) string {
	return "'" + strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), "'", `\'`) + "'"
}

func quotePowerShell(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

func TestCompletionInBash(t *testing.T) {
	s := newCompletionShell(t, "bash", "--norc", "--noprofile")
	// bash keeps the quotes of a word as it was typed; the script takes them
	// off before it asks fnaze.
	cases := append(slices.Clone(completionCases), completionCase{"fnaze rm 'src/auth/", []string{"src/auth/", "src/auth/session.go", "src/auth/*_test.go"}})
	var b strings.Builder
	b.WriteString(s.script("bash"))
	for i, tc := range cases {
		words := strings.Split(tc.line, " ") // the last is empty when the line ends with a space
		quoted := make([]string, len(words))
		for j, w := range words {
			quoted[j] = quote(w)
		}
		fmt.Fprintf(&b, "echo '### %d'\nCOMP_WORDS=(%s)\nCOMP_CWORD=%d\n_fnaze\nprintf '%%s\\n' \"${COMPREPLY[@]}\"\n",
			i, strings.Join(quoted, " "), len(words)-1)
	}
	check(t, cases, s.run(b.String()), "\t", quotePattern)
}

// zsh needs its completion system to offer anything, which a script cannot
// start without a terminal. So the function of the script is called as the
// system would call it (with words and CURRENT set) and what it hands to
// _describe is read: zsh itself quotes what it puts in.
func TestCompletionInZsh(t *testing.T) {
	s := newCompletionShell(t, "zsh", "-f")
	script := s.script("zsh")
	syntax := exec.Command(s.path, "-n")
	syntax.Stdin = strings.NewReader(script)
	if out, err := syntax.CombinedOutput(); err != nil {
		t.Fatalf("zsh -n: %v\n%s", err, out)
	}

	var b strings.Builder
	b.WriteString("compdef() { :; }\n")
	b.WriteString("_describe() { shift 3; print -l -- \"${(P@)1}\"; }\n") // -t candidates 'fnaze' <array> [-S '']
	b.WriteString("_files() { :; }\n")
	b.WriteString(script)
	for i, tc := range completionCases {
		words := strings.Split(tc.line, " ")
		quoted := make([]string, len(words))
		for j, w := range words {
			quoted[j] = quote(w)
		}
		fmt.Fprintf(&b, "print '### %d'\nwords=(%s)\nCURRENT=%d\n_fnaze\n", i, strings.Join(quoted, " "), len(words))
	}
	check(t, completionCases, s.run(b.String()), ":", asIs)
}

func TestCompletionInFish(t *testing.T) {
	s := newCompletionShell(t, "fish", "--no-config")
	var b strings.Builder
	b.WriteString("fnaze completion fish | source\n")
	for i, tc := range completionCases {
		fmt.Fprintf(&b, "echo '### %d'\ncomplete -C %s\n", i, quoteFish(tc.line))
	}
	check(t, completionCases, s.run(b.String()), "\t", asIs)
}

func TestCompletionInPowerShell(t *testing.T) {
	s := newCompletionShell(t, "pwsh", "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", "-")
	var b strings.Builder
	b.WriteString(s.script("powershell"))
	b.WriteString("\n")
	for i, tc := range completionCases {
		fmt.Fprintf(&b, "'### %d'\n$line = %s\n(TabExpansion2 $line $line.Length).CompletionMatches | ForEach-Object { $_.CompletionText }\n",
			i, quotePowerShell(tc.line))
	}
	check(t, completionCases, s.run(b.String()), "\t", quotePattern)
}
