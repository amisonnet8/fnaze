package format

import (
	"slices"
	"testing"
)

const base = "# t\nsrc/\n\tauth/\tx\n\t\ta.go\ty\nconfig/auth.yaml\tz\n"

func parse(t *testing.T, s string) *File {
	t.Helper()
	f, problems := Parse([]byte(s))
	if len(problems) != 0 {
		t.Fatalf("problems %v", problems)
	}
	return f
}

func TestSet(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		desc      string
		detail    []string
		setDetail bool
		want      string
	}{
		{"under the deepest directory, with skeleton lines", "src/auth/k/b.go", "D", nil, false,
			"# t\nsrc/\n\tauth/\tx\n\t\ta.go\ty\n\t\tk/\n\t\t\tb.go\tD\nconfig/auth.yaml\tz\n"},
		{"after the siblings", "src/b.go", "D", nil, false,
			"# t\nsrc/\n\tauth/\tx\n\t\ta.go\ty\n\tb.go\tD\nconfig/auth.yaml\tz\n"},
		{"a path written in one line is not a directory entry", "config/db.yaml", "E", nil, false,
			base + "config/\n\tdb.yaml\tE\n"},
		{"a new directory at the top", "new/", "N", nil, false, base + "new/\tN\n"},
		{"replace a desc and keep the detail", "src/auth/a.go", " Y2 ", nil, false,
			"# t\nsrc/\n\tauth/\tx\n\t\ta.go\tY2\nconfig/auth.yaml\tz\n"},
		{"give a skeleton line a desc", "src/", "S", nil, false,
			"# t\nsrc/\tS\n\tauth/\tx\n\t\ta.go\ty\nconfig/auth.yaml\tz\n"},
		{"empty a desc", "src/auth/", "", nil, false,
			"# t\nsrc/\n\tauth/\n\t\ta.go\ty\nconfig/auth.yaml\tz\n"},
		{"a path written in one line", "config/auth.yaml", "Z", []string{"why"}, true,
			"# t\nsrc/\n\tauth/\tx\n\t\ta.go\ty\nconfig/auth.yaml\tZ\n\t| why\n"},
		{"one line for each detail, and for each newline", "src/auth/a.go", "y", []string{"l1\nl2 ", "", "l3"}, true,
			"# t\nsrc/\n\tauth/\tx\n\t\ta.go\ty\n\t\t\t| l1\n\t\t\t| l2\n\t\t\t|\n\t\t\t| l3\nconfig/auth.yaml\tz\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := parse(t, base)
			e, err := f.Set(tt.path, tt.desc, tt.detail, tt.setDetail)
			if err != nil {
				t.Fatal(err)
			}
			if e.Path != tt.path {
				t.Errorf("entry %s", e.Path)
			}
			if got := string(f.Bytes()); got != tt.want {
				t.Errorf("got\n%q\nwant\n%q", got, tt.want)
			}
			if _, problems := Parse(f.Bytes()); len(problems) != 0 {
				t.Errorf("what was written has problems %v", problems)
			}
		})
	}
}

// When a directory and a deeper one are both written at the top, the new
// entry goes under the deeper.
func TestSetUnderTheDeepest(t *testing.T) {
	f := parse(t, "# t\nsrc/\tx\nsrc/auth/\ty\n")
	if _, err := f.Set("src/auth/a.go", "z", nil, false); err != nil {
		t.Fatal(err)
	}
	if got := string(f.Bytes()); got != "# t\nsrc/\tx\nsrc/auth/\ty\n\ta.go\tz\n" {
		t.Errorf("got %q", got)
	}
}

func TestSetDetail(t *testing.T) {
	f := parse(t, "# t\na\tx\n\t| old\n")
	if _, err := f.Set("a", "x", nil, false); err != nil || !slices.Equal(f.Entries[0].Detail, []string{"old"}) {
		t.Errorf("without setDetail: %q, %v", f.Entries[0].Detail, err)
	}
	if _, err := f.Set("a", "x", []string{"", " \n"}, true); err != nil || f.Entries[0].Detail != nil {
		t.Errorf("only empty lines: %q, %v", f.Entries[0].Detail, err)
	}
	if got := string(f.Bytes()); got != "# t\na\tx\n" {
		t.Errorf("got %q", got)
	}
}

func TestSetFails(t *testing.T) {
	for _, path := range []string{"", "/a", "a//b", "./a", "a/../b", "*/a", "a/ b", "a /b", " a", "| x", "a/|", "a\tb", "a\nb"} {
		f := parse(t, base)
		if _, err := f.Set(path, "d", nil, false); err == nil {
			t.Errorf("path %q: no error", path)
		}
		if got := string(f.Bytes()); got != base {
			t.Errorf("path %q changed the file: %q", path, got)
		}
	}
	for _, desc := range []string{"a\nb", "a\rb"} {
		f := parse(t, base)
		if _, err := f.Set("a", desc, nil, false); err == nil {
			t.Errorf("desc %q: no error", desc)
		}
	}
	// What CheckPath allows: a pattern in the last part, "|" not at the start
	// of a part.
	for _, path := range []string{"*_test.go", "gen-*/", "a|b", "a/b|"} {
		if err := CheckPath(path); err != nil {
			t.Errorf("%q: %v", path, err)
		}
	}
}

func TestSetTitle(t *testing.T) {
	f := parse(t, base)
	if err := f.SetTitle(" \t新しい  題 \t"); err != nil {
		t.Fatal(err)
	}
	// Only the spaces at the start and the end go; the entries stay.
	if got, want := string(f.Bytes()), "# 新しい  題\nsrc/\n\tauth/\tx\n\t\ta.go\ty\nconfig/auth.yaml\tz\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if f.Title != "新しい  題" {
		t.Errorf("Title %q", f.Title)
	}
	// An empty title is "# ".
	for _, title := range []string{"", "  "} {
		if err := f.SetTitle(title); err != nil {
			t.Fatal(err)
		}
		if got := string(f.Bytes()); got != "# \nsrc/\n\tauth/\tx\n\t\ta.go\ty\nconfig/auth.yaml\tz\n" {
			t.Errorf("title %q: got %q", title, got)
		}
	}
}

func TestSetTitleFails(t *testing.T) {
	for _, title := range []string{"a\nb", "a\rb", "a\n"} {
		f := parse(t, base)
		if err := f.SetTitle(title); err == nil || err.Error() != "a title cannot have a newline" {
			t.Errorf("title %q: %v", title, err)
		}
		if got := string(f.Bytes()); got != base {
			t.Errorf("title %q changed the file: %q", title, got)
		}
	}
}

func TestFind(t *testing.T) {
	f := parse(t, base)
	for _, p := range []string{"src/", "src/auth/", "src/auth/a.go", "config/auth.yaml"} {
		if e := f.Find(p); e == nil || e.Path != p {
			t.Errorf("Find(%s) = %v", p, e)
		}
	}
	for _, p := range []string{"config/", "src", "src/auth", "src/auth/b.go", "auth/"} {
		if e := f.Find(p); e != nil {
			t.Errorf("Find(%s) = %s", p, e.Path)
		}
	}
}

func TestRemove(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		path    string
		removed int
		want    string
	}{
		{"a directory with a desc stays", base, "src/auth/a.go", 1,
			"# t\nsrc/\n\tauth/\tx\nconfig/auth.yaml\tz\n"},
		{"with its children", base, "src/", 3, "# t\nconfig/auth.yaml\tz\n"},
		{"skeleton lines left with no children go", "# t\nsrc/\n\tauth/\n\t\ta.go\ty\nb\tz\n", "src/auth/a.go", 3, "# t\nb\tz\n"},
		{"a skeleton line with another child stays", "# t\nsrc/\n\ta\tx\n\tb\ty\n", "src/a", 1, "# t\nsrc/\n\tb\ty\n"},
		{"a directory with a detail stays", "# t\nsrc/\n\t| why\n\ta\tx\n", "src/a", 1, "# t\nsrc/\n\t| why\n"},
		{"stops at the first that stays", "# t\nsrc/\tS\n\tauth/\n\t\ta.go\ty\n", "src/auth/a.go", 2, "# t\nsrc/\tS\n"},
		{"with its details", "# t\na\tx\n\t| why\nb\ty\n", "a", 1, "# t\nb\ty\n"},
		{"a path written in one line", base, "config/auth.yaml", 1, "# t\nsrc/\n\tauth/\tx\n\t\ta.go\ty\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := parse(t, tt.in)
			removed, err := f.Remove(tt.path)
			if err != nil || removed != tt.removed {
				t.Errorf("removed %d, %v; want %d", removed, err, tt.removed)
			}
			if got := string(f.Bytes()); got != tt.want {
				t.Errorf("got\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}

func TestRemoveFails(t *testing.T) {
	for _, p := range []string{"src", "src/auth", "config/", "x", ""} {
		f := parse(t, base)
		if _, err := f.Remove(p); err == nil {
			t.Errorf("%q: no error", p)
		}
		if got := string(f.Bytes()); got != base {
			t.Errorf("%q changed the file: %q", p, got)
		}
	}
}

func TestCount(t *testing.T) {
	if got := parse(t, base).Count(); got != 4 {
		t.Errorf("got %d", got)
	}
	if got := (&File{}).Count(); got != 0 {
		t.Errorf("empty file: %d", got)
	}
}
