package core

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/amisonnet8/fnaze/internal/format"
	"github.com/amisonnet8/fnaze/internal/lang"
	"github.com/amisonnet8/fnaze/internal/name"
)

// The results of the commands. They are the JSON of --json and of the MCP
// tools (docs/reference/commands.md, "JSON"): every key is always there, with
// "" and [] for empty values.

// NewResult is the result of New.
type NewResult struct {
	Group string    `json:"group"`
	Files []NewFile `json:"files"`
}

// NewFile is one file that New made.
type NewFile struct {
	Lang string `json:"lang"`
	Path string `json:"path"`
}

// SetResult is the result of Set: the entry after writing.
type SetResult struct {
	Group  string `json:"group"`
	Lang   string `json:"lang"`
	Path   string `json:"path"`
	Desc   string `json:"desc"`
	Detail string `json:"detail"`
}

// RmResult is the result of Rm.
type RmResult struct {
	Group   string `json:"group"`
	Lang    string `json:"lang"`
	Path    string `json:"path"`
	Removed int    `json:"removed"`
}

// StatusResult is the result of Status.
type StatusResult struct {
	Items []StatusItem `json:"items"`
}

// StatusItem is one thing that Status reports.
type StatusItem struct {
	Group   string `json:"group"`
	File    string `json:"file"`
	Line    int    `json:"line"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

// KindLangMismatch is the kind of the files of one group whose path lines
// differ.
const KindLangMismatch = "lang-mismatch"

// ShowResult is the result of Show.
type ShowResult struct {
	Group   string      `json:"group"`
	Lang    string      `json:"lang"`
	File    string      `json:"file"`
	Title   string      `json:"title"`
	Entries []ShowEntry `json:"entries"`
}

// ShowEntry is one entry of Show, with its children.
type ShowEntry struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	Desc     string      `json:"desc"`
	Detail   string      `json:"detail"`
	Children []ShowEntry `json:"children"`
}

// LogResult is the result of Log.
type LogResult struct {
	Groups []LogGroup `json:"groups"`
}

// LogGroup is one group of Log.
type LogGroup struct {
	Group    string    `json:"group"`
	Datetime time.Time `json:"datetime"` // UTC
	Langs    []string  `json:"langs"`
	Title    string    `json:"title"`
	Entries  int       `json:"entries"`
}

// New makes a new group (fnaze new) with one file for each language: langs,
// or else those of the newest group, or else those of FNAZE_LANG, or else en.
// Each file starts from the newest file in its language, without the title,
// the descs and the details.
func (c *Core) New(langs []string) (*NewResult, error) {
	ch, err := c.open()
	if err != nil {
		return nil, err
	}
	defer ch.close()
	if len(langs) == 0 {
		switch {
		case len(ch.groups) > 0:
			langs = ch.groups[len(ch.groups)-1].langs()
		case len(lang.FromEnv(c.EnvLang)) > 0:
			langs = lang.FromEnv(c.EnvLang)
		default:
			langs = []string{lang.Default}
		}
	}
	if err := lang.CheckGroup(langs); err != nil {
		return nil, err
	}
	contents := make([][]byte, len(langs))
	for i, l := range langs {
		if contents[i], err = ch.skeleton(l); err != nil {
			return nil, err
		}
	}
	id, err := name.NewGroup(c.Now(), c.Random)
	if err != nil {
		return nil, err
	}
	res := &NewResult{Group: id, Files: []NewFile{}}
	var made []string
	for i, l := range langs {
		file := name.Name{Group: id, Lang: l}.File()
		if err := ch.create(file, contents[i]); err != nil {
			// Leave no group with only some of its languages.
			for _, m := range made {
				_ = ch.dir.Remove(m)
			}
			return nil, err
		}
		made = append(made, file)
		res.Files = append(res.Files, NewFile{Lang: l, Path: changesPath(file)})
	}
	return res, nil
}

// skeleton returns what a new file in language l starts as.
func (ch *changes) skeleton(l string) ([]byte, error) {
	for i := len(ch.groups) - 1; i >= 0; i-- {
		if n, ok := ch.groups[i].file(l); ok {
			f, _, err := ch.read(n)
			if err != nil {
				return nil, err
			}
			return f.Skeleton().Bytes(), nil
		}
	}
	return (&format.File{}).Bytes(), nil
}

func (ch *changes) create(file string, data []byte) error {
	f, err := ch.dir.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	return errors.Join(err, f.Close())
}

func (ch *changes) write(file string, data []byte) error {
	return ch.dir.WriteFile(file, data, 0o644)
}

// Set writes one entry to the file of a group in a language (fnaze set).
// detail replaces the detail only when setDetail is true.
func (c *Core) Set(lng, groupSpec, path, desc string, detail []string, setDetail bool) (*SetResult, error) {
	l, err := c.oneLang(lng)
	if err != nil {
		return nil, err
	}
	ch, err := c.open()
	if err != nil {
		return nil, err
	}
	defer ch.close()
	g, err := ch.find(groupSpec)
	if err != nil {
		return nil, err
	}
	n, f, err := ch.fileOf(g, l)
	if err != nil {
		return nil, err
	}
	e, err := f.Set(path, desc, detail, setDetail)
	if err != nil {
		return nil, err
	}
	if err := ch.write(n.File(), f.Bytes()); err != nil {
		return nil, err
	}
	return &SetResult{Group: g.ID, Lang: l, Path: e.Path, Desc: e.Desc, Detail: strings.Join(e.Detail, "\n")}, nil
}

// Rm removes one entry from the file of a group in a language (fnaze rm).
func (c *Core) Rm(lng, groupSpec, path string) (*RmResult, error) {
	l, err := c.oneLang(lng)
	if err != nil {
		return nil, err
	}
	ch, err := c.open()
	if err != nil {
		return nil, err
	}
	defer ch.close()
	g, err := ch.find(groupSpec)
	if err != nil {
		return nil, err
	}
	n, f, err := ch.fileOf(g, l)
	if err != nil {
		return nil, err
	}
	removed, err := f.Remove(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", changesPath(n.File()), err)
	}
	if err := ch.write(n.File(), f.Bytes()); err != nil {
		return nil, err
	}
	return &RmResult{Group: g.ID, Lang: l, Path: path, Removed: removed}, nil
}

// Status checks a group (groupSpec, or the newest), or every group when all
// is true (fnaze status). The names in .fnaze/changes/ that do not have the
// form are reported whichever group is checked. With no group at all, only
// those are reported.
func (c *Core) Status(groupSpec string, all bool) (*StatusResult, error) {
	if all && groupSpec != "" {
		return nil, errors.New("--all and a group cannot be given together")
	}
	ch, err := c.open()
	if err != nil {
		return nil, err
	}
	defer ch.close()
	res := &StatusResult{Items: []StatusItem{}}
	for _, b := range ch.bad {
		res.Items = append(res.Items, StatusItem{File: changesPath(b), Kind: string(format.KindFormat),
			Message: "the name does not have the form <datetime>_<UUID>[_<language>].txt"})
	}
	groups := ch.groups
	if !all {
		groups = nil
		if groupSpec != "" || len(ch.groups) > 0 {
			g, err := ch.find(groupSpec)
			if err != nil {
				return nil, err
			}
			groups = []*group{g}
		}
	}
	for _, g := range groups {
		items, err := ch.check(g)
		if err != nil {
			return nil, err
		}
		res.Items = append(res.Items, items...)
	}
	slices.SortStableFunc(res.Items, func(a, b StatusItem) int {
		if a.File != b.File {
			return strings.Compare(a.File, b.File)
		}
		return a.Line - b.Line
	})
	return res, nil
}

func (ch *changes) check(g *group) ([]StatusItem, error) {
	var items []StatusItem
	if err := lang.CheckGroup(g.langs()); err != nil {
		items = append(items, StatusItem{Group: g.ID, File: changesPath(g.Files[0].File()), Kind: string(format.KindFormat), Message: err.Error()})
	}
	var first []string
	for i, n := range g.Files {
		f, problems, err := ch.read(n)
		if err != nil {
			return nil, err
		}
		file := changesPath(n.File())
		for _, p := range problems {
			items = append(items, StatusItem{Group: g.ID, File: file, Line: p.Line, Kind: string(p.Kind), Message: p.Message})
		}
		lines := f.PathLines()
		if i == 0 {
			first = lines
		} else if !slices.Equal(lines, first) {
			items = append(items, StatusItem{Group: g.ID, File: file, Kind: KindLangMismatch,
				Message: fmt.Sprintf("the path lines differ from those of %s", changesPath(g.Files[0].File()))})
		}
	}
	return items, nil
}

// Show returns the file of a group in a language as a tree (fnaze show).
func (c *Core) Show(lng, groupSpec string) (*ShowResult, error) {
	l, err := c.oneLang(lng)
	if err != nil {
		return nil, err
	}
	ch, err := c.open()
	if err != nil {
		return nil, err
	}
	defer ch.close()
	g, err := ch.find(groupSpec)
	if err != nil {
		return nil, err
	}
	n, ok := g.file(l)
	if !ok {
		return nil, fmt.Errorf("group %s has no file in %s", g.ID, l)
	}
	f, _, err := ch.read(n)
	if err != nil {
		return nil, err
	}
	return &ShowResult{Group: g.ID, Lang: l, File: changesPath(n.File()), Title: f.Title, Entries: showEntries(f.Entries)}, nil
}

func showEntries(entries []*format.Entry) []ShowEntry {
	out := make([]ShowEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, ShowEntry{Name: e.Name, Path: e.Path, Desc: e.Desc, Detail: strings.Join(e.Detail, "\n"), Children: showEntries(e.Children)})
	}
	return out
}

// Log lists the groups, newest first, with their title in a language (fnaze
// log). A group with no file in the language has an empty title.
func (c *Core) Log(lng string) (*LogResult, error) {
	l, err := c.oneLang(lng)
	if err != nil {
		return nil, err
	}
	ch, err := c.open()
	if err != nil {
		return nil, err
	}
	defer ch.close()
	res := &LogResult{Groups: []LogGroup{}}
	for i := len(ch.groups) - 1; i >= 0; i-- {
		g := ch.groups[i]
		lg := LogGroup{Group: g.ID, Datetime: name.Datetime(g.ID).UTC(), Langs: g.langs()}
		if n, ok := g.file(l); ok {
			f, _, err := ch.read(n)
			if err != nil {
				return nil, err
			}
			lg.Title, lg.Entries = f.Title, f.Count()
		}
		res.Groups = append(res.Groups, lg)
	}
	return res, nil
}
