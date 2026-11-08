// Package format reads and writes one description file
// (docs/reference/file-format.md): its title, its entries with their descs
// and details, and the problems that status reports about the file alone
// (format errors, an empty title, empty descs).
//
// Reading follows crube's structure.txt: spaces at the end of a line (a CR
// included) are removed, and blank lines are ignored. Writing gives the
// canonical form: LF, no blank lines, no spaces at the end of lines except in
// the empty title "# ".
package format

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"unicode"
)

// Kind is the kind of a problem, as status reports it.
type Kind string

// The kinds of problems found in one file. The kinds that need more than one
// file (lang-mismatch) are found elsewhere.
const (
	KindFormat     Kind = "format"
	KindEmptyTitle Kind = "empty-title"
	KindEmptyDesc  Kind = "empty-desc"
)

// Problem is one thing status reports about a file.
type Problem struct {
	Line    int // 1-based; 0 when it is not about one line
	Kind    Kind
	Message string
}

// File is a description file.
type File struct {
	Title   string // without the "# "; "" for an empty title
	Entries []*Entry
}

// Entry is one entry and what is written under it.
type Entry struct {
	Name     string   // as written: it may have "/" in the middle, and a directory ends in "/"
	Path     string   // from the root
	Desc     string   // "" when there is none
	Detail   []string // one element for each detail line ("" for a line of "|" alone)
	Children []*Entry
	Line     int // where it was read; 0 for an entry that was not read
}

// IsDir reports whether the entry is a directory.
func (e *Entry) IsDir() bool {
	return strings.HasSuffix(e.Name, "/")
}

var bom = []byte("\xef\xbb\xbf")

// Parse reads a description file. It returns what it could read even when
// there are problems, so that every problem of the file is reported at once.
// The problems are in the order of their lines.
func Parse(data []byte) (*File, []Problem) {
	p := &parser{f: &File{}, seen: map[string]int{}}
	if bytes.HasPrefix(data, bom) {
		p.add(1, KindFormat, "the file starts with a BOM")
		data = data[len(bom):]
	}
	titled := false
	for i, raw := range strings.Split(string(data), "\n") {
		n := i + 1
		line := strings.TrimRightFunc(raw, unicode.IsSpace)
		if line == "" {
			continue
		}
		if !titled {
			titled = true
			if p.title(n, line) {
				continue
			}
		}
		p.body(n, line)
	}
	if !titled {
		p.add(1, KindFormat, `no title line: the first line must start with "# "`)
	}
	p.emptyDescs(p.f.Entries)
	slices.SortStableFunc(p.problems, func(a, b Problem) int { return a.Line - b.Line })
	return p.f, p.problems
}

type parser struct {
	f        *File
	problems []Problem
	stack    []*Entry       // stack[d] is the last entry read at depth d
	seen     map[string]int // the line of each path read so far
}

func (p *parser) add(line int, kind Kind, format string, args ...any) {
	p.problems = append(p.problems, Problem{Line: line, Kind: kind, Message: fmt.Sprintf(format, args...)})
}

// title reads the first line that is not blank. It reports false when the line
// is not a title at all, so that it is read as an entry instead.
func (p *parser) title(n int, line string) bool {
	switch {
	case line == "#":
		p.add(n, KindEmptyTitle, "the title is empty")
	case strings.HasPrefix(line, "# "):
		p.f.Title = strings.TrimSpace(line[2:])
	case strings.HasPrefix(line, "#"):
		p.add(n, KindFormat, `the title must start with "# "`)
	default:
		p.add(n, KindFormat, `no title line: the first line must start with "# "`)
		return false
	}
	return true
}

func (p *parser) body(n int, line string) {
	body := strings.TrimLeft(line, "\t")
	depth := len(line) - len(body)
	if body[0] == ' ' {
		p.add(n, KindFormat, "the indentation has a space in it")
		return
	}
	if body == "|" || strings.HasPrefix(body, "| ") {
		p.detail(n, depth, body)
		return
	}
	p.entry(n, depth, body)
}

func (p *parser) detail(n, depth int, body string) {
	if len(p.stack) == 0 {
		p.add(n, KindFormat, "a detail line with no entry before it")
		return
	}
	e := p.stack[len(p.stack)-1]
	if depth != len(p.stack) {
		p.add(n, KindFormat, "a detail line must be indented one level deeper than its entry (line %d)", e.Line)
		return
	}
	text := ""
	if len(body) > 1 {
		text = body[2:]
	}
	e.Detail = append(e.Detail, text)
}

func (p *parser) entry(n, depth int, body string) {
	if depth > len(p.stack) {
		p.add(n, KindFormat, "indented more than one level deeper than the entry before it")
		return
	}
	var parent *Entry
	if depth > 0 {
		parent = p.stack[depth-1]
		if !parent.IsDir() {
			p.add(n, KindFormat, "a file entry cannot have children (line %d)", parent.Line)
			return
		}
	}
	name, desc, _ := strings.Cut(body, "\t")
	e := &Entry{Name: name, Path: name, Desc: strings.TrimSpace(desc), Line: n}
	if parent != nil {
		e.Path = parent.Path + name
		parent.Children = append(parent.Children, e)
	} else {
		p.f.Entries = append(p.f.Entries, e)
	}
	p.stack = append(p.stack[:depth], e)
	if msg := checkName(name); msg != "" {
		p.add(n, KindFormat, "%s", msg)
	} else if msg := checkPattern(e.Path); msg != "" {
		p.add(n, KindFormat, "%s", msg)
	}
	if first, ok := p.seen[e.Path]; ok {
		p.add(n, KindFormat, "%s is also on line %d", e.Path, first)
	} else {
		p.seen[e.Path] = n
	}
}

// checkName returns why a name as written cannot be used, or "".
func checkName(name string) string {
	if strings.HasPrefix(name, "/") {
		return fmt.Sprintf(`%s: a path cannot start with "/"`, name)
	}
	for part := range strings.SplitSeq(strings.TrimSuffix(name, "/"), "/") {
		switch part {
		case "":
			return fmt.Sprintf("%s: a path cannot have an empty part", name)
		case ".", "..":
			return fmt.Sprintf(`%s: a path cannot have "." or ".." as a part`, name)
		}
	}
	return ""
}

// checkPattern returns why a path from the root has "*" where it cannot, or "".
// Only the last part of a path can be a pattern, so a directory pattern has no
// children.
func checkPattern(path string) string {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	if slices.ContainsFunc(parts[:len(parts)-1], func(s string) bool { return strings.Contains(s, "*") }) {
		return fmt.Sprintf(`%s: only the last part of a path can have "*"`, path)
	}
	return ""
}

func (p *parser) emptyDescs(entries []*Entry) {
	for _, e := range entries {
		if len(e.Children) == 0 && e.Desc == "" {
			p.add(e.Line, KindEmptyDesc, "%s has no desc", e.Path)
		}
		p.emptyDescs(e.Children)
	}
}

// Bytes writes the file in the canonical form.
func (f *File) Bytes() []byte {
	var b bytes.Buffer
	b.WriteString("# " + strings.TrimSpace(f.Title) + "\n")
	writeEntries(&b, f.Entries, 0)
	return b.Bytes()
}

func writeEntries(b *bytes.Buffer, entries []*Entry, depth int) {
	indent := strings.Repeat("\t", depth)
	for _, e := range entries {
		b.WriteString(indent + e.Name)
		if desc := strings.TrimSpace(e.Desc); desc != "" {
			b.WriteString("\t" + desc)
		}
		b.WriteByte('\n')
		for _, d := range e.Detail {
			if d = strings.TrimRightFunc(d, unicode.IsSpace); d == "" {
				b.WriteString(indent + "\t|\n")
			} else {
				b.WriteString(indent + "\t| " + d + "\n")
			}
		}
		writeEntries(b, e.Children, depth+1)
	}
}

// Skeleton returns a copy of the file with only its path lines: the title, the
// descs and the details are removed (what fnaze new starts a file from).
func (f *File) Skeleton() *File {
	return &File{Entries: skeleton(f.Entries)}
}

func skeleton(entries []*Entry) []*Entry {
	out := make([]*Entry, 0, len(entries))
	for _, e := range entries {
		out = append(out, &Entry{Name: e.Name, Path: e.Path, Children: skeleton(e.Children)})
	}
	return out
}

// PathLines returns the path lines of the file in order, each as its
// indentation and the name as written. The files of one group must have the
// same path lines.
func (f *File) PathLines() []string {
	var lines []string
	var walk func([]*Entry, int)
	walk = func(entries []*Entry, depth int) {
		for _, e := range entries {
			lines = append(lines, strings.Repeat("\t", depth)+e.Name)
			walk(e.Children, depth+1)
		}
	}
	walk(f.Entries, 0)
	return lines
}
