package cli

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/amisonnet8/fnaze/internal/core"
)

// The expected widths were computed with Python's unicodedata (East Asian
// Width; W and F are 2, categories Mn, Me and Cf are 0, the rest 1), not with
// this package.
func TestTextWidth(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want int
	}{
		{"empty", "", 0},
		{"ASCII", "abc", 3},
		{"wide", "日本語", 6},
		{"halfwidth katakana", "ｱｲｳ", 3},
		{"fullwidth letter", "Ａ", 2},
		{"ambiguous counts as 1", "○①α", 3},
		{"e and combining acute (U+0301)", "e\xcc\x81", 1},
		{"zero width joiner (U+200D)", "a\xe2\x80\x8db", 2},
		{"a and combining enclosing circle (U+20DD, Me)", "a\xe2\x83\x9d", 1},
		{"emoji", "😀", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := textWidth(tt.s); got != tt.want {
				t.Errorf("textWidth(%q) = %d, want %d", tt.s, got, tt.want)
			}
		})
	}
}

func TestShortNames(t *testing.T) {
	tests := []struct {
		name string
		ids  []string
		want []string
	}{
		{"one group", []string{"20261107T003426Z_8980d0c2-5b1e-4f5e-9a5e-1c2d3e4f5a6b"}, []string{"20261107T003426Z_8980"}},
		{
			"different datetimes: the datetime and four characters",
			[]string{"20261107T003426Z_8980d0c2-5b1e-4f5e-9a5e-1c2d3e4f5a6b", "20261107T021530Z_2c4c1a2b-3c4d-4e5f-8a6b-7c8d9e0f1a2b"},
			[]string{"20261107T003426Z_8980", "20261107T021530Z_2c4c"},
		},
		{
			"the same five characters: up to the sixth",
			[]string{"20261108T000000Z_abcd1111-2222-4333-8444-555566667777", "20261108T000000Z_abcd2111-2222-4333-8444-555566667777"},
			[]string{"20261108T000000Z_abcd1", "20261108T000000Z_abcd2"},
		},
		{
			"only another group's start matters",
			[]string{"20261108T000000Z_abcd1111-2222-4333-8444-555566667777", "20261108T000000Z_abcd1112-2222-4333-8444-555566667777", "20261108T000000Z_ffff1111-2222-4333-8444-555566667777"},
			[]string{"20261108T000000Z_abcd1111", "20261108T000000Z_abcd1112", "20261108T000000Z_ffff"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shortNames(tt.ids); !slices.Equal(got, tt.want) {
				t.Errorf("shortNames = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStatusText(t *testing.T) {
	res := &core.StatusResult{Items: []core.StatusItem{
		{File: ".fnaze/changes/a.txt", Line: 3, Kind: "empty-desc", Message: "src/a.go has no desc"},
		{File: ".fnaze/changes/notes.txt", Kind: "format", Message: "the name does not have the form"},
	}}
	var b strings.Builder
	statusText(&b, res, func(p string) string { return "../" + p })
	want := "../.fnaze/changes/a.txt:3: empty-desc: src/a.go has no desc\n" +
		"../.fnaze/changes/notes.txt: format: the name does not have the form\n"
	if b.String() != want {
		t.Errorf("got\n%s\nwant\n%s", b.String(), want)
	}
}

func TestShowText(t *testing.T) {
	tests := []struct {
		name string
		res  *core.ShowResult
		want string
	}{
		{
			// The example of docs/reference/commands.md.
			"example",
			&core.ShowResult{Title: "Extend the session lifetime on access instead of fixing it", Entries: []core.ShowEntry{
				{Name: "src/", Children: []core.ShowEntry{
					{Name: "auth/", Desc: "Change the session lifetime from fixed to extended", Children: []core.ShowEntry{
						{Name: "session.go", Desc: "Extend the lifetime on every access"},
						{Name: "*_test.go", Desc: "Add tests for the extension"},
					}},
				}},
				{Name: "config/auth.yaml", Desc: "Set the default session.ttl to 30m", Detail: "Users reported sessions expiring too soon"},
				{Name: "docs/old.md", Desc: "Removed; its content moved to docs/auth.md"},
			}},
			"# Extend the session lifetime on access instead of fixing it\n" +
				"\n" +
				"src/\n" +
				"  auth/           Change the session lifetime from fixed to extended\n" +
				"    session.go    Extend the lifetime on every access\n" +
				"    *_test.go     Add tests for the extension\n" +
				"config/auth.yaml  Set the default session.ttl to 30m\n" +
				"                  | Users reported sessions expiring too soon\n" +
				"docs/old.md       Removed; its content moved to docs/auth.md\n",
		},
		{
			// Computed with Python's unicodedata: 認証/ is 5 columns, "  鍵.go" 7.
			"wide characters",
			&core.ShowResult{Title: "t", Entries: []core.ShowEntry{
				{Name: "認証/", Desc: "a", Children: []core.ShowEntry{{Name: "鍵.go", Desc: "b"}}},
				{Name: "x.md", Desc: "c"},
			}},
			"# t\n\n認証/    a\n  鍵.go  b\nx.md     c\n",
		},
		{
			"a detail of several lines, one of them empty, and an entry with no desc",
			&core.ShowResult{Title: "t", Entries: []core.ShowEntry{
				{Name: "a.go", Desc: "d", Detail: "one\n\nthree"},
				{Name: "b.go"},
			}},
			"# t\n\na.go  d\n      | one\n      |\n      | three\nb.go\n",
		},
		{"no entries: the title alone", &core.ShowResult{Title: "t"}, "# t\n"},
		{"empty title", &core.ShowResult{}, "# \n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b strings.Builder
			showText(&b, tt.res)
			if b.String() != tt.want {
				t.Errorf("got\n%s\nwant\n%s", b.String(), tt.want)
			}
		})
	}
}

// setLocal sets time.Local for one test. The test must not be parallel.
func setLocal(t *testing.T, loc *time.Location) {
	t.Helper()
	old := time.Local
	time.Local = loc
	t.Cleanup(func() { time.Local = old })
}

func TestLogText(t *testing.T) {
	setLocal(t, time.FixedZone("", 9*60*60))
	utc := func(s string) time.Time {
		d, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	res := &core.LogResult{Groups: []core.LogGroup{
		{Group: "20261108T000000Z_abcd1112-2222-4333-8444-555566667777", Datetime: utc("2026-11-08T00:00:00Z"), Langs: []string{"ja"}},
		{Group: "20261108T000000Z_abcd1111-2222-4333-8444-555566667777", Datetime: utc("2026-11-08T00:00:00Z"), Langs: []string{"ja"}},
		{Group: "20261107T021530Z_2c4c1a2b-3c4d-4e5f-8a6b-7c8d9e0f1a2b", Datetime: utc("2026-11-07T02:15:30Z"), Langs: []string{"en", "ja"}, Title: "Log session extensions", Entries: 3},
		{Group: "20261107T003426Z_8980d0c2-5b1e-4f5e-9a5e-1c2d3e4f5a6b", Datetime: utc("2026-11-07T00:34:26Z"), Langs: []string{"en", "ja"}, Title: "Extend the session lifetime on access instead of fixing it", Entries: 6},
	}}
	// printf '%-25s  %-16s  %-5s  %1s  %s\n' … | sed 's/ *$//'
	want := "20261108T000000Z_abcd1112  2026-11-08 09:00  ja     0\n" +
		"20261108T000000Z_abcd1111  2026-11-08 09:00  ja     0\n" +
		"20261107T021530Z_2c4c      2026-11-07 11:15  en,ja  3  Log session extensions\n" +
		"20261107T003426Z_8980      2026-11-07 09:34  en,ja  6  Extend the session lifetime on access instead of fixing it\n"
	var b strings.Builder
	logText(&b, res)
	if b.String() != want {
		t.Errorf("got\n%s\nwant\n%s", b.String(), want)
	}
}

func TestLogTextCountsAlignRight(t *testing.T) {
	setLocal(t, time.UTC)
	res := &core.LogResult{Groups: []core.LogGroup{
		{Group: "20261108T000000Z_abcd1111-2222-4333-8444-555566667777", Datetime: time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC), Langs: []string{"en"}, Title: "b", Entries: 12},
		{Group: "20261107T000000Z_abcd1111-2222-4333-8444-555566667777", Datetime: time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC), Langs: []string{"en"}, Title: "a", Entries: 3},
	}}
	// printf '%-21s  %-16s  %-2s  %2s  %s\n' …
	want := "20261108T000000Z_abcd  2026-11-08 00:00  en  12  b\n" +
		"20261107T000000Z_abcd  2026-11-07 00:00  en   3  a\n"
	var b strings.Builder
	logText(&b, res)
	if b.String() != want {
		t.Errorf("got\n%s\nwant\n%s", b.String(), want)
	}
}
