// Package core holds every decision of fnaze (docs/reference/commands.md).
// The CLI and the MCP server are thin wrappers around it: they call the same
// methods and print or return the same results, which are also the JSON of
// --json.
package core

import (
	"cmp"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/amisonnet8/fnaze/internal/format"
	"github.com/amisonnet8/fnaze/internal/lang"
	"github.com/amisonnet8/fnaze/internal/name"
	"github.com/amisonnet8/fnaze/internal/project"
)

// Core works on the description files of one root.
type Core struct {
	Root    string           // the directory that holds .fnaze/
	EnvLang string           // the value of FNAZE_LANG
	Now     func() time.Time // the clock of fnaze new
	Random  io.Reader        // the randomness of the UUIDs of fnaze new
}

// Open finds the root from dir upward.
func Open(dir, envLang string) (*Core, error) {
	root, err := project.Find(dir)
	if err != nil {
		return nil, err
	}
	return &Core{Root: root, EnvLang: envLang, Now: time.Now, Random: rand.Reader}, nil
}

// Init makes .fnaze/ in dir (fnaze init) and returns the root.
func Init(dir string) (string, error) {
	return project.Init(dir)
}

// changesPath is the path of a file in .fnaze/changes/, from the root, with
// "/" on every OS.
func changesPath(file string) string {
	return path.Join(project.Dir, project.Changes, file)
}

// group is one group, with the names of its files in name order (en first).
type group struct {
	ID    string
	Files []name.Name
}

func (g *group) langs() []string {
	langs := make([]string, len(g.Files))
	for i, n := range g.Files {
		langs[i] = n.Lang
	}
	return langs
}

func (g *group) file(l string) (name.Name, bool) {
	for _, n := range g.Files {
		if n.Lang == l {
			return n, true
		}
	}
	return name.Name{}, false
}

// changes is what is in .fnaze/changes/.
type changes struct {
	dir    *os.Root
	groups []*group // in name order: the newest is the last
	bad    []string // the names that do not have the form, in name order
}

func (c *Core) open() (*changes, error) {
	dir, err := os.OpenRoot(project.ChangesDir(c.Root))
	if err != nil {
		return nil, err
	}
	entries, err := readDir(dir)
	if err != nil {
		_ = dir.Close()
		return nil, err
	}
	ch := &changes{dir: dir}
	byID := map[string]*group{}
	for _, e := range entries {
		n, ok := name.Parse(e.Name())
		if !ok || e.IsDir() {
			ch.bad = append(ch.bad, e.Name())
			continue
		}
		g := byID[n.Group]
		if g == nil {
			g = &group{ID: n.Group}
			byID[n.Group] = g
			ch.groups = append(ch.groups, g)
		}
		g.Files = append(g.Files, n)
	}
	// readDir sorts by name, so the groups and their files are in name order.
	return ch, nil
}

func readDir(dir *os.Root) ([]os.DirEntry, error) {
	f, err := dir.Open(".")
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }() // only read
	entries, err := f.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(entries, func(a, b os.DirEntry) int { return cmp.Compare(a.Name(), b.Name()) })
	return entries, nil
}

func (ch *changes) close() {
	_ = ch.dir.Close()
}

// ErrNoGroup is returned when a command needs a group and there is none.
var ErrNoGroup = errors.New("there is no group yet: make one with fnaze new")

// find returns the group that spec names: "" for the newest, or the whole
// <datetime>_<UUID>, or a start of it that names only one group.
func (ch *changes) find(spec string) (*group, error) {
	if len(ch.groups) == 0 {
		return nil, ErrNoGroup
	}
	if spec == "" {
		return ch.groups[len(ch.groups)-1], nil
	}
	var found []*group
	for _, g := range ch.groups {
		if g.ID == spec {
			return g, nil
		}
		if strings.HasPrefix(g.ID, spec) {
			found = append(found, g)
		}
	}
	switch len(found) {
	case 0:
		return nil, fmt.Errorf("no group starts with %s", spec)
	case 1:
		return found[0], nil
	}
	ids := make([]string, len(found))
	for i, g := range found {
		ids[i] = g.ID
	}
	return nil, fmt.Errorf("%s starts more than one group: %s", spec, strings.Join(ids, ", "))
}

func (ch *changes) read(n name.Name) (*format.File, []format.Problem, error) {
	data, err := ch.dir.ReadFile(n.File())
	if err != nil {
		return nil, nil, err
	}
	f, problems := format.Parse(data)
	return f, problems, nil
}

// fileOf returns the file of a group in a language, read, for a command that
// writes it: it must have no format error.
func (ch *changes) fileOf(g *group, l string) (name.Name, *format.File, error) {
	n, ok := g.file(l)
	if !ok {
		return name.Name{}, nil, fmt.Errorf("group %s has no file in %s", g.ID, l)
	}
	f, problems, err := ch.read(n)
	if err != nil {
		return name.Name{}, nil, err
	}
	for _, p := range problems {
		if p.Kind == format.KindFormat {
			return name.Name{}, nil, fmt.Errorf("%s has a format error on line %d (%s): fix it first (fnaze status)", changesPath(n.File()), p.Line, p.Message)
		}
	}
	return n, f, nil
}

func (c *Core) oneLang(given string) (string, error) {
	return lang.One(given, c.EnvLang)
}
