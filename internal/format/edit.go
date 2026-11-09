package format

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// CheckPath returns an error unless path, from the root, can be written as an
// entry: the file format allows it, and each of its parts can be written on a
// line of its own (fnaze set writes missing directories as skeleton lines).
func CheckPath(path string) error {
	if path == "" {
		return errors.New("a path cannot be empty")
	}
	if strings.ContainsAny(path, "\t\n\r") {
		return fmt.Errorf("%q: a path cannot have a tab or a newline", path)
	}
	if msg := checkName(path); msg != "" {
		return errors.New(msg)
	}
	if msg := checkPattern(path); msg != "" {
		return errors.New(msg)
	}
	for part := range strings.SplitSeq(strings.TrimSuffix(path, "/"), "/") {
		if part != strings.TrimSpace(part) {
			return fmt.Errorf("%q: a part of a path cannot start or end with a space", path)
		}
		if part == "|" || strings.HasPrefix(part, "| ") {
			return fmt.Errorf(`%q: a part of a path cannot be "|" or start with "| "`, path)
		}
	}
	return nil
}

// Find returns the entry of path, or nil.
func (f *File) Find(path string) *Entry {
	chain := f.chain(path)
	if len(chain) == 0 || chain[len(chain)-1].Path != path {
		return nil
	}
	return chain[len(chain)-1]
}

// chain returns the entries from the top down to the deepest one whose path
// is path or a directory above it.
func (f *File) chain(path string) []*Entry {
	var chain []*Entry
	entries := f.Entries
	for {
		var next *Entry
		for _, e := range entries {
			if e.Path == path || e.IsDir() && strings.HasPrefix(path, e.Path) {
				// The deepest match is the longest path; on a tie (a format
				// error) the first one.
				if next == nil || len(e.Path) > len(next.Path) {
					next = e
				}
			}
		}
		if next == nil || next.Path == path {
			if next != nil {
				chain = append(chain, next)
			}
			return chain
		}
		chain = append(chain, next)
		entries = next.Children
	}
}

// Set writes the entry of path (fnaze set). An existing entry gets desc, and
// detail too when setDetail is true. A new entry goes under the deepest
// directory entry above it, after its siblings, with a skeleton line for each
// missing directory in between. Each element of detail is one line, a newline
// in it starts another, and a detail of only empty lines is removed. The path
// must pass CheckPath.
func (f *File) Set(path, desc string, detail []string, setDetail bool) (*Entry, error) {
	if err := CheckPath(path); err != nil {
		return nil, err
	}
	if strings.ContainsAny(desc, "\n\r") {
		return nil, errors.New("a desc cannot have a newline")
	}
	e := f.Find(path)
	if e == nil {
		e = f.add(path)
	}
	e.Desc = strings.TrimSpace(desc)
	if setDetail {
		e.Detail = detailLines(detail)
	}
	return e, nil
}

// SetTitle writes the title (fnaze title), without the spaces at its start
// and end. An empty title is written as "# ".
func (f *File) SetTitle(title string) error {
	if strings.ContainsAny(title, "\n\r") {
		return errors.New("a title cannot have a newline")
	}
	f.Title = strings.TrimSpace(title)
	return nil
}

func (f *File) add(path string) *Entry {
	chain := f.chain(path)
	siblings := &f.Entries
	parent := ""
	if len(chain) > 0 {
		siblings = &chain[len(chain)-1].Children
		parent = chain[len(chain)-1].Path
	}
	rest := strings.TrimPrefix(path, parent)
	dirs := strings.SplitAfter(rest, "/")
	if dirs[len(dirs)-1] == "" {
		dirs = dirs[:len(dirs)-1]
	}
	var e *Entry
	for _, name := range dirs {
		e = &Entry{Name: name, Path: parent + name}
		*siblings = append(*siblings, e)
		siblings = &e.Children
		parent = e.Path
	}
	return e
}

func detailLines(values []string) []string {
	var lines []string
	empty := true
	for _, v := range values {
		for l := range strings.SplitSeq(v, "\n") {
			l = strings.TrimRightFunc(l, unicode.IsSpace)
			lines = append(lines, l)
			empty = empty && l == ""
		}
	}
	if empty {
		return nil
	}
	return lines
}

// Remove removes the entry of path with its children and details, and then
// the directory entries above it that are left with no children, no desc and
// no detail (fnaze rm). It returns how many entries it removed.
func (f *File) Remove(path string) (int, error) {
	chain := f.chain(path)
	if len(chain) == 0 || chain[len(chain)-1].Path != path {
		return 0, fmt.Errorf("no entry for %s", path)
	}
	removed := 0
	for i := len(chain) - 1; i >= 0; i-- {
		e := chain[i]
		if i < len(chain)-1 && (len(e.Children) > 0 || e.Desc != "" || len(e.Detail) > 0) {
			break
		}
		siblings := &f.Entries
		if i > 0 {
			siblings = &chain[i-1].Children
		}
		*siblings = remove(*siblings, e)
		removed += count(e)
	}
	return removed, nil
}

func remove(entries []*Entry, e *Entry) []*Entry {
	out := entries[:0]
	for _, x := range entries {
		if x != e {
			out = append(out, x)
		}
	}
	return out
}

// count returns how many entries e is, with its children.
func count(e *Entry) int {
	n := 1
	for _, c := range e.Children {
		n += count(c)
	}
	return n
}

// Count returns how many entries the file has, children included.
func (f *File) Count() int {
	n := 0
	for _, e := range f.Entries {
		n += count(e)
	}
	return n
}
