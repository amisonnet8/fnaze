// Package cli is the fnaze command line: it turns arguments into calls to
// internal/core and results into output, and nothing more, so that the CLI
// and the MCP server never behave differently (docs/reference/commands.md).
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/amisonnet8/fnaze/internal/core"
)

// Exit codes (docs/reference/commands.md, "Common").
const (
	exitOK       = 0
	exitReported = 1 // status --require reported something
	exitError    = 2
)

// envLang is the environment variable of the languages.
const envLang = "FNAZE_LANG"

// Env is what the command line runs with.
type Env struct {
	Getenv  func(string) string
	Stdout  io.Writer
	Stderr  io.Writer
	Version string           // see BuildVersion; "" for none
	Now     func() time.Time // the clock of new; nil for time.Now
	Random  io.Reader        // the randomness of new; nil for crypto/rand
}

// order is the order of the commands in the usage.
var order = []string{"init", "new", "set", "rm", "status", "show", "log"}

// usage is what fnaze --help prints.
var usage = func() string {
	var b strings.Builder
	b.WriteString("usage: fnaze <command> [arguments]\n\ncommands:\n")
	for _, name := range order {
		fmt.Fprintf(&b, "  %-8s%s\n", name, commands[name].summary)
	}
	b.WriteString("\ncommon options: -C <dir>  --json\n")
	b.WriteString("see fnaze <command> --help, and fnaze --version\n")
	return b.String()
}()

// command is one subcommand.
type command struct {
	usage    string   // the synopsis after "usage: "
	summary  string   // what it does, in the usage
	options  []string // the options it accepts
	args     []string // the names of its positional arguments
	min      int      // how many of args are required
	variadic bool     // the last argument takes all the rest
	run      func(r *runner, p *parsed) (int, error)
}

var commands = map[string]*command{
	"init": {
		usage: "fnaze init [-C <dir>]", summary: "create .fnaze/changes/",
		options: []string{"-C"},
		run:     (*runner).init,
	},
	"new": {
		usage: "fnaze new [language...] [-C <dir>] [--json]", summary: "create a new group, one file for each language",
		options: []string{"-C", "--json"}, args: []string{"language"}, variadic: true,
		run: (*runner).new,
	},
	"set": {
		usage:   "fnaze set [--lang <language>] [--group <group>] <path> <desc> [--detail <detail>]... [-C <dir>] [--json]",
		summary: "write one entry",
		options: []string{"--lang", "--group", "--detail", "-C", "--json"}, args: []string{"path", "desc"}, min: 2,
		run: (*runner).set,
	},
	"rm": {
		usage:   "fnaze rm [--lang <language>] [--group <group>] <path> [-C <dir>] [--json]",
		summary: "remove one entry",
		options: []string{"--lang", "--group", "-C", "--json"}, args: []string{"path"}, min: 1,
		run: (*runner).rm,
	},
	"status": {
		usage:   "fnaze status [--require] [--all | group] [-C <dir>] [--json]",
		summary: "check a group",
		options: []string{"--require", "--all", "-C", "--json"}, args: []string{"group"},
		run: (*runner).status,
	},
	"show": {
		usage:   "fnaze show [--lang <language>] [group] [-C <dir>] [--json]",
		summary: "show a file of a group as a tree",
		options: []string{"--lang", "-C", "--json"}, args: []string{"group"},
		run: (*runner).show,
	},
	"log": {
		usage:   "fnaze log [--lang <language>] [-C <dir>] [--json]",
		summary: "list the groups, newest first",
		options: []string{"--lang", "-C", "--json"},
		run:     (*runner).log,
	},
}

// Run runs the command line args (without the program name) and returns the
// exit code. Errors go to Stderr as one line "fnaze: …".
func Run(args []string, env Env) int {
	if len(args) == 0 {
		_, _ = io.WriteString(env.Stderr, usage) // nowhere else to report a failure
		return exitError
	}
	switch args[0] {
	case "-h", "--help", "help":
		return say(env, usage)
	case "--version":
		return say(env, "fnaze "+version(env.Version)+"\n")
	}

	name := args[0]
	cmd, ok := commands[name]
	if !ok {
		return fail(env, fmt.Errorf("unknown command %q (see fnaze --help)", name))
	}
	p, err := parse(args[1:], cmd.options)
	if err != nil {
		return fail(env, fmt.Errorf("%s: %w (usage: %s)", name, err, cmd.usage))
	}
	if p.help {
		return say(env, "usage: "+cmd.usage+"\n")
	}
	switch {
	case len(p.args) < cmd.min:
		return fail(env, fmt.Errorf("%s: missing <%s> (usage: %s)", name, cmd.args[len(p.args)], cmd.usage))
	case len(p.args) > len(cmd.args) && !cmd.variadic:
		return fail(env, fmt.Errorf("%s: too many arguments (usage: %s)", name, cmd.usage))
	}

	r := &runner{env: env, dir: p.value("-C")}
	if r.dir == "" {
		r.dir = "."
	}
	code, err := cmd.run(r, p)
	if err != nil {
		return fail(env, err)
	}
	return code
}

// BuildVersion is the version for Env.Version: set, the one the release build
// set, or else the version of the module that Go recorded in the binary
// (go install …@v0.1.0 records v0.1.0). read is debug.ReadBuildInfo. Go
// records "(devel)" when it knows no version; that is none.
func BuildVersion(set string, read func() (*debug.BuildInfo, bool)) string {
	if set != "" {
		return set
	}
	if bi, ok := read(); ok && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return ""
}

// version is the version to show: Env.Version, else "(devel)".
func version(v string) string {
	if v != "" {
		return v
	}
	return "(devel)"
}

// say writes s to Stdout and ends with exit code 0, or 2 if it cannot.
func say(env Env, s string) int {
	if _, err := io.WriteString(env.Stdout, s); err != nil {
		return fail(env, err)
	}
	return exitOK
}

func fail(env Env, err error) int {
	_, _ = fmt.Fprintln(env.Stderr, "fnaze:", err) // nowhere else to report a failure
	return exitError
}

// runner runs one command.
type runner struct {
	env Env
	dir string // the directory to work in: -C, or "."
}

// open finds the root from the directory to work in.
func (r *runner) open() (*core.Core, error) {
	c, err := core.Open(r.dir, r.env.Getenv(envLang))
	if err != nil {
		return nil, err
	}
	if r.env.Now != nil {
		c.Now = r.env.Now
	}
	if r.env.Random != nil {
		c.Random = r.env.Random
	}
	return c, nil
}

// relFunc returns what turns a path from the root of c into one from the
// directory to work in, with the separator of the OS, so that it can be
// opened as it is. A path it cannot turn is left as it is.
func (r *runner) relFunc(c *core.Core) func(string) string {
	base, err := filepath.Abs(r.dir)
	return func(p string) string {
		if err != nil {
			return p
		}
		rel, rerr := filepath.Rel(base, filepath.Join(c.Root, filepath.FromSlash(p)))
		if rerr != nil {
			return p
		}
		return rel
	}
}

// writeJSON writes v as JSON indented by 2 spaces. HTML characters are kept
// as they are, since descriptions are text for people, not for a web page.
func (r *runner) writeJSON(v any) error {
	enc := json.NewEncoder(r.env.Stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// output writes v as JSON with --json, else the text that text makes (nil
// for none).
func (r *runner) output(p *parsed, v any, text func(b *strings.Builder)) error {
	if p.has("--json") {
		return r.writeJSON(v)
	}
	if text == nil {
		return nil
	}
	var b strings.Builder
	text(&b)
	_, err := io.WriteString(r.env.Stdout, b.String())
	return err
}

func (r *runner) init(_ *parsed) (int, error) {
	_, err := core.Init(r.dir)
	return exitOK, err
}

func (r *runner) new(p *parsed) (int, error) {
	c, err := r.open()
	if err != nil {
		return 0, err
	}
	res, err := c.New(p.args)
	if err != nil {
		return 0, err
	}
	return exitOK, r.output(p, res, func(b *strings.Builder) { newText(b, res, r.relFunc(c)) })
}

func (r *runner) set(p *parsed) (int, error) {
	c, err := r.open()
	if err != nil {
		return 0, err
	}
	res, err := c.Set(p.value("--lang"), p.value("--group"), p.args[0], p.args[1], p.opts["--detail"], p.has("--detail"))
	if err != nil {
		return 0, err
	}
	return exitOK, r.output(p, res, nil)
}

func (r *runner) rm(p *parsed) (int, error) {
	c, err := r.open()
	if err != nil {
		return 0, err
	}
	res, err := c.Rm(p.value("--lang"), p.value("--group"), p.args[0])
	if err != nil {
		return 0, err
	}
	return exitOK, r.output(p, res, nil)
}

func (r *runner) status(p *parsed) (int, error) {
	c, err := r.open()
	if err != nil {
		return 0, err
	}
	res, err := c.Status(arg(p, 0), p.has("--all"))
	if err != nil {
		return 0, err
	}
	if err := r.output(p, res, func(b *strings.Builder) { statusText(b, res, r.relFunc(c)) }); err != nil {
		return 0, err
	}
	if p.has("--require") && len(res.Items) > 0 {
		return exitReported, nil
	}
	return exitOK, nil
}

func (r *runner) show(p *parsed) (int, error) {
	c, err := r.open()
	if err != nil {
		return 0, err
	}
	res, err := c.Show(p.value("--lang"), arg(p, 0))
	if err != nil {
		return 0, err
	}
	return exitOK, r.output(p, res, func(b *strings.Builder) { showText(b, res) })
}

func (r *runner) log(p *parsed) (int, error) {
	c, err := r.open()
	if err != nil {
		return 0, err
	}
	res, err := c.Log(p.value("--lang"))
	if err != nil {
		return 0, err
	}
	return exitOK, r.output(p, res, func(b *strings.Builder) { logText(b, res) })
}

// arg is the i-th positional argument, or "" when it is not given.
func arg(p *parsed, i int) string {
	if i < len(p.args) {
		return p.args[i]
	}
	return ""
}
