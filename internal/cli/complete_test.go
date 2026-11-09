package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// completeRoot makes a root with two groups:
//   - older, in en and fr: "# Old" with a.go
//   - newer (group), in en and ja: the en file has src/auth/session.go, a
//     single entry config/auth.yaml and a pattern; the ja file has src/ only
const older = "20261107T000000Z_abcd1111-2222-4333-8444-555566667777"

func completeRoot(t *testing.T) string {
	t.Helper()
	dir := newRoot(t)
	writeFile(t, dir, ".fnaze/changes/"+older+".txt", "# Old\na.go\td\n")
	writeFile(t, dir, ".fnaze/changes/"+older+"_fr.txt", "# Vieux\na.go\td\n")
	writeFile(t, dir, fileEn, "# New\nsrc/\n\tauth/\tAuth\n\t\tsession.go\tExtend\n\t\t*_test.go\tTests\nconfig/auth.yaml\tTTL\n")
	writeFile(t, dir, fileJa, "# 新しい\nsrc/\tソース\n")
	return dir
}

func TestComplete(t *testing.T) {
	dir := completeRoot(t)
	tests := []struct {
		name  string
		word  string
		words []string
		want  string
	}{
		{"the commands", "", nil,
			"init\tcreate .fnaze/changes/\nnew\tcreate a new group, one file for each language\ntitle\twrite the title of a file\n" +
				"set\twrite one entry\nrm\tremove one entry\nstatus\tcheck a group\nshow\tshow a file of a group as a tree\nlog\tlist the groups, newest first\n" +
				"mcp\trun as an MCP server over standard input and output\ncompletion\tprint a shell completion script\n"},
		{"the commands that start with the word", "s", nil, "set\twrite one entry\nstatus\tcheck a group\nshow\tshow a file of a group as a tree\n"},
		{"the options before a command", "-", nil, "--help\n--version\n"},
		{"an unknown command", "", []string{"add"}, ""},
		{"the options of a command, without those given", "-", []string{"rm", "--json"}, "--lang\n--group\n-C\n--help\n"},
		{"--detail can be given again", "--d", []string{"set", "a", "b", "--detail", "x"}, "--detail\n"},
		{"after --, no option", "-", []string{"show", "--"}, ""},
		{"the shells", "", []string{"completion"}, "bash\nzsh\nfish\npowershell\n"},
		{"after the shell, nothing", "", []string{"completion", "bash"}, ""},
		{"the value of -C is for the shell", "", []string{"set", "-C"}, ""},
		{"the value of --detail is for nobody", "", []string{"set", "a", "b", "--detail"}, ""},

		{"groups, newest first, with the title in the language", "", []string{"show", "-C", dir}, group + "\tNew\n" + older + "\tOld\n"},
		{"groups with the title in --lang; none without a file", "", []string{"show", "--lang", "ja", "-C", dir}, group + "\t新しい\n" + older + "\n"},
		{"--lang=ja in one word", "", []string{"show", "--lang=ja", "-C=" + dir}, group + "\t新しい\n" + older + "\n"},
		{"--lang = ja as bash splits it", "", []string{"show", "--lang", "=", "ja", "-C", dir}, group + "\t新しい\n" + older + "\n"},
		{"groups that start with the word", "20261107", []string{"status", "-C", dir}, older + "\tOld\n"},
		{"status --all takes no group", "", []string{"status", "--all", "-C", dir}, ""},
		{"show takes one group", "", []string{"show", "x", "-C", dir}, ""},
		{"--group of rm", "", []string{"rm", "-C", dir, "--group"}, group + "\tNew\n" + older + "\tOld\n"},
		{"--group of title, with the title in --lang", "", []string{"title", "--lang", "ja", "-C", dir, "--group"}, group + "\t新しい\n" + older + "\n"},
		{"--lang of title", "", []string{"title", "-C", dir, "--lang"}, "en\nja\nfr\n"},
		{"the options of title", "-", []string{"title", "x"}, "--lang\n--group\n-C\n--json\n--help\n"},
		{"the title itself is for nobody", "", []string{"title", "-C", dir}, ""},

		{"languages, as the newest group has them first", "", []string{"set", "-C", dir, "--lang"}, "en\nja\nfr\n"},
		{"languages of new, without those given", "", []string{"new", "-C", dir, "ja"}, "en\nfr\n"},

		{"paths: the first level", "", []string{"set", "-C", dir}, "src/\nconfig/\n"},
		{"paths: the next level, with the desc of an entry", "src/", []string{"set", "-C", dir}, "src/\nsrc/auth/\tAuth\n"},
		{"paths: the files", "src/auth/", []string{"rm", "-C", dir}, "src/auth/\tAuth\nsrc/auth/session.go\tExtend\nsrc/auth/*_test.go\tTests\n"},
		{"paths: a word without its /", "src", []string{"set", "-C", dir}, "src/\n"},
		{"paths: after - as the path, nothing", "", []string{"set", "-", "-C", dir}, ""},
		{"paths: a single entry with a /", "config/", []string{"rm", "-C", dir}, "config/auth.yaml\tTTL\n"},
		{"paths in --lang", "", []string{"rm", "--lang", "ja", "-C", dir}, "src/\tソース\n"},
		{"paths in --group", "", []string{"rm", "--group", older[:21], "-C", dir}, "a.go\td\n"},
		{"after the path of set, nothing", "", []string{"set", "src/", "-C", dir}, ""},
		{"no file in the language: nothing", "", []string{"rm", "--lang", "de", "-C", dir}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, out, errs := run(t, "", append([]string{"__complete", "--word=" + tt.word, "--"}, tt.words...)...)
			want(t, code, out, errs, 0, tt.want, "")
		})
	}
}

// Whatever is wrong, __complete says nothing and exits 0.
func TestCompleteNeverComplains(t *testing.T) {
	dir := completeRoot(t)
	broken := t.TempDir()
	if err := os.MkdirAll(filepath.Join(broken, ".fnaze"), 0o755); err != nil { // no changes/
		t.Fatal(err)
	}
	for _, words := range [][]string{
		{"show", "-C", t.TempDir()}, // no .fnaze/
		{"show", "-C", broken},
		{"set", "-C", broken},
		{"new", "-C", broken},
		{"set", "--lang", "j-a", "-C", dir}, // not a language
		{"set", "--group", "x", "-C", dir},  // no such group
	} {
		code, out, errs := run(t, "", append([]string{"__complete", "--word=", "--"}, words...)...)
		want(t, code, out, errs, 0, "", "")
	}
	code, out, errs := run(t, "", "__complete")
	want(t, code, out, errs, 0, "init\tcreate .fnaze/changes/\nnew\tcreate a new group, one file for each language\ntitle\twrite the title of a file\n"+
		"set\twrite one entry\nrm\tremove one entry\nstatus\tcheck a group\nshow\tshow a file of a group as a tree\nlog\tlist the groups, newest first\n"+
		"mcp\trun as an MCP server over standard input and output\ncompletion\tprint a shell completion script\n", "")
}

// FNAZE_LANG chooses the language when --lang is not on the line.
func TestCompleteLanguageFromEnv(t *testing.T) {
	dir := completeRoot(t)
	code, out, errs := run(t, "ja,en", "__complete", "--word=", "--", "rm", "-C", dir)
	want(t, code, out, errs, 0, "src/\tソース\n", "")
}

// A desc is cut at 60 columns, for the menu of a shell: one of 60 is kept.
func TestCompleteCutsDesc(t *testing.T) {
	dir := newRoot(t)
	desc := strings.Repeat("あ", 31) // 62 columns
	fits := strings.Repeat("あ", 30) // 60 columns
	ascii := strings.Repeat("x", 61)
	writeFile(t, dir, fileEn, "# t\na\t"+desc+"\nb\t"+fits+"\nc\t"+ascii+"\n")
	code, out, errs := run(t, "", "__complete", "--word=", "--", "rm", "-C", dir)
	want(t, code, out, errs, 0, "a\t"+strings.Repeat("あ", 29)+"…\n"+ // 58 columns and the 1 of …
		"b\t"+fits+"\n"+
		"c\t"+strings.Repeat("x", 59)+"…\n", "") // 59 columns and the 1 of …
}

func TestCompletionScripts(t *testing.T) {
	for _, shell := range shells {
		code, out, errs := run(t, "", "completion", shell)
		if code != 0 || errs != "" || !strings.Contains(out, "fnaze __complete \"--word=") {
			t.Errorf("%s: %d %q %q", shell, code, out, errs)
		}
	}
	code, out, errs := run(t, "", "completion", "tcsh")
	want(t, code, out, errs, 2, "", "fnaze: completion: unknown shell \"tcsh\" (use bash, zsh, fish or powershell)\n")
}
