package cli

import (
	"embed"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/amisonnet8/fnaze/internal/core"
)

// Shell completion (docs/reference/commands.md, "fnaze completion").

// The scripts are fixed text. They do not list the commands: at every TAB
// they ask fnaze __complete, which reads the tables of cli.go, so a script
// cannot disagree with the fnaze that is installed.
//
//go:embed completions
var completionScripts embed.FS

// shells are the shells that have a script, in the order they are named.
var shells = []string{"bash", "zsh", "fish", "powershell"}

var scriptFiles = map[string]string{
	"bash":       "completions/fnaze.bash",
	"zsh":        "completions/fnaze.zsh",
	"fish":       "completions/fnaze.fish",
	"powershell": "completions/fnaze.ps1",
}

// completeCommand is the command the scripts call. It is not in the usage:
// it is not for typing.
const completeCommand = "__complete"

// candidateWidth is how many columns of a desc a candidate has: it is for the
// menu of a shell.
const candidateWidth = 60

func (r *runner) completion(p *parsed) (int, error) {
	shell := p.args[0]
	if !slices.Contains(shells, shell) {
		return 0, fmt.Errorf("completion: unknown shell %q (use bash, zsh, fish or powershell)", shell)
	}
	script, err := completionScripts.ReadFile(scriptFiles[shell])
	if err != nil {
		return 0, err // the files are embedded, so this is a build without them
	}
	_, err = r.env.Stdout.Write(script)
	return exitOK, err
}

// candidate is one thing that can come next on a command line.
type candidate struct {
	value, desc string
}

// complete runs fnaze __complete [--word=<word>] -- <words>...: it lists what
// can come next after the words, of what starts with word, one per line as
// value, or value<TAB>desc. The scripts call it at every TAB, so it never
// complains: whatever the words are and whatever .fnaze/ holds, it exits 0
// and says nothing on Stderr.
func complete(args []string, env Env) int {
	var word string
	var words []string // the words after "--"
	for i, a := range args {
		if a == "--" {
			words = args[i+1:]
			break
		}
		if w, ok := strings.CutPrefix(a, "--word="); ok {
			word = w
		}
	}
	var b strings.Builder
	for _, c := range candidatesAt(walk(words), word, env) {
		b.WriteString(c.value)
		if c.desc != "" {
			b.WriteString("\t" + cut(c.desc, candidateWidth))
		}
		b.WriteString("\n")
	}
	_, _ = io.WriteString(env.Stdout, b.String()) // nowhere to report a failure
	return exitOK
}

// position is what a walk over the words of a line finds out about where the
// next word stands.
type position struct {
	name     string   // the command, "" before it
	cmd      *command // nil for a command that fnaze does not have
	seen     map[string]bool
	pending  string   // an option that waits for its value
	args     []string // the positional arguments so far
	dashdash bool     // after "--": no option can come any more
	dir      string   // the value of -C
	lang     string   // the value of --lang
	group    string   // the value of --group
}

// walk reads the words the way parse does, but forgives everything: a line
// that is being typed is wrong more often than not.
func walk(words []string) position {
	p := position{seen: map[string]bool{}}
	set := func(name, value string) {
		switch name {
		case "-C":
			p.dir = value
		case "--lang":
			p.lang = value
		case "--group":
			p.group = value
		}
	}
	for _, w := range words {
		switch {
		case p.pending != "":
			if w == "=" {
				continue // bash splits --lang=ja into three words
			}
			set(p.pending, w)
			p.pending = ""
		case p.name == "":
			p.name = w
			p.cmd = commands[w]
		case w == "--" && !p.dashdash:
			p.dashdash = true
		case !p.dashdash && len(w) >= 2 && w[0] == '-':
			name, value, hasValue := strings.Cut(w, "=")
			if name == "-h" {
				name = "--help"
			}
			p.seen[name] = true
			switch {
			case hasValue:
				set(name, value)
			case valueOptions[name]:
				p.pending = name
			}
		default:
			p.args = append(p.args, w)
		}
	}
	return p
}

// candidatesAt lists what can stand at a position, of what starts with what
// has been typed of the word.
func candidatesAt(p position, word string, env Env) []candidate {
	option := strings.HasPrefix(word, "-") && !p.dashdash
	var list []candidate
	switch {
	case p.name == "" && option:
		list = []candidate{{value: "--help"}, {value: "--version"}}
	case p.name == "":
		for _, name := range order {
			list = append(list, candidate{value: name, desc: commands[name].summary})
		}
	case p.cmd == nil:
	case p.pending == "--lang":
		list = langCandidates(p, env, nil)
	case p.pending == "--group":
		list = groupCandidates(p, env)
	case p.pending != "":
		// A directory (-C) or a text (--detail): the shell knows about the
		// first, and nobody about the second.
	case option:
		for _, name := range append(slices.Clone(p.cmd.options), "--help") {
			if !p.seen[name] || name == "--detail" {
				list = append(list, candidate{value: name})
			}
		}
	case p.name == "completion" && len(p.args) == 0:
		for _, s := range shells {
			list = append(list, candidate{value: s})
		}
	case p.name == "new":
		list = langCandidates(p, env, p.args)
	case (p.name == "status" && !p.seen["--all"] || p.name == "show") && len(p.args) == 0:
		list = groupCandidates(p, env)
	case (p.name == "set" || p.name == "rm") && len(p.args) == 0:
		list = pathCandidates(p, word, env)
	}
	return slices.DeleteFunc(list, func(c candidate) bool { return !strings.HasPrefix(c.value, word) })
}

// open opens the root the line is in (its -C, or the current directory).
// Anything that goes wrong is no candidates: a message at every TAB would
// break the line being typed.
func (p position) open(env Env) *core.Core {
	dir := p.dir
	if dir == "" {
		dir = "."
	}
	c, err := core.Open(dir, env.Getenv(envLang))
	if err != nil {
		return nil
	}
	return c
}

// groupCandidates are the groups, newest first, each described by its title
// in the language of the line.
func groupCandidates(p position, env Env) []candidate {
	c := p.open(env)
	if c == nil {
		return nil
	}
	res, err := c.Log(p.lang)
	if err != nil {
		return nil
	}
	list := make([]candidate, len(res.Groups))
	for i, g := range res.Groups {
		list[i] = candidate{value: g.Group, desc: g.Title}
	}
	return list
}

// langCandidates are the languages of the groups, as the newest group has
// them and then those that only older groups have, without those given.
func langCandidates(p position, env Env, given []string) []candidate {
	c := p.open(env)
	if c == nil {
		return nil
	}
	res, err := c.Log("")
	if err != nil {
		return nil
	}
	var list []candidate
	seen := map[string]bool{}
	for _, g := range given {
		seen[g] = true
	}
	for _, g := range res.Groups {
		for _, l := range g.Langs {
			if !seen[l] {
				seen[l] = true
				list = append(list, candidate{value: l})
			}
		}
	}
	return list
}

// pathCandidates are the entry paths of the target group's file in the
// language of the line that word can go on with: the next level only, so a
// directory ends with "/" and the line goes on into it. Each is described by
// its desc.
func pathCandidates(p position, word string, env Env) []candidate {
	c := p.open(env)
	if c == nil {
		return nil
	}
	res, err := c.Show(p.lang, p.group)
	if err != nil {
		return nil
	}
	var list []candidate
	seen := map[string]bool{}
	var add func(entries []core.ShowEntry)
	add = func(entries []core.ShowEntry) {
		for _, e := range entries {
			if v, ok := nextLevel(e.Path, word); ok && !seen[v] {
				seen[v] = true
				desc := ""
				if v == e.Path {
					desc = e.Desc
				}
				list = append(list, candidate{value: v, desc: desc})
			}
			add(e.Children)
		}
	}
	add(res.Entries)
	return list
}

// nextLevel is path up to the end of the level after word, with its "/": the
// candidate that path gives for word. ok is false when path does not start
// with word.
func nextLevel(path, word string) (string, bool) {
	rest, ok := strings.CutPrefix(path, word)
	if !ok {
		return "", false
	}
	if i := strings.Index(rest, "/"); i >= 0 {
		return word + rest[:i+1], true // the whole path when the "/" is its end
	}
	return path, true
}

// cut shortens s to at most avail columns, ending with "…" when it is cut.
func cut(s string, avail int) string {
	if textWidth(s) <= avail {
		return s
	}
	var b strings.Builder
	w := 0
	for _, r := range s {
		rw := runeWidth(r)
		if w+rw > avail-1 {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	return b.String() + "…"
}
