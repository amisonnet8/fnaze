package format

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

const example1 = "20261107T003426Z_8980a325-8b57-49b0-8e0a-ed0aaf9eeda7"

func readExample(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "examples", "changes", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// The examples in docs/examples/changes/ are already in the canonical form, so
// reading and writing them gives the same bytes.
func TestExamplesRoundTrip(t *testing.T) {
	names, err := filepath.Glob(filepath.Join("..", "..", "docs", "examples", "changes", "*.txt"))
	if err != nil || len(names) == 0 {
		t.Fatalf("no examples: %v", err)
	}
	for _, name := range names {
		data := readExample(t, filepath.Base(name))
		f, problems := Parse(data)
		if len(problems) != 0 {
			t.Errorf("%s: problems %v", name, problems)
		}
		if got := string(f.Bytes()); got != string(data) {
			t.Errorf("%s: written again as\n%s\nwant\n%s", name, got, data)
		}
	}
}

type flat struct {
	Path   string
	Desc   string
	Detail []string
	Line   int
}

func flatten(entries []*Entry) []flat {
	var out []flat
	for _, e := range entries {
		out = append(out, flat{e.Path, e.Desc, e.Detail, e.Line})
		out = append(out, flatten(e.Children)...)
	}
	return out
}

func TestParseExample(t *testing.T) {
	f, _ := Parse(readExample(t, example1+".txt"))
	if f.Title != "Extend the session lifetime on access instead of fixing it" {
		t.Errorf("title %q", f.Title)
	}
	want := []flat{
		{"src/", "", nil, 2},
		{"src/auth/", "Change the session lifetime from fixed to extended", nil, 3},
		{"src/auth/session.go", "Extend the lifetime on every access", nil, 4},
		{"src/auth/*_test.go", "Add tests for the extension", nil, 5},
		{"config/auth.yaml", "Set the default session.ttl to 30m", []string{"Users reported sessions expiring too soon"}, 6},
		{"docs/old.md", "Removed; its content moved to docs/auth.md", nil, 8},
	}
	if got := flatten(f.Entries); !equalFlat(got, want) {
		t.Errorf("entries\n%v\nwant\n%v", got, want)
	}
}

func equalFlat(a, b []flat) bool {
	return slices.EqualFunc(a, b, func(x, y flat) bool {
		return x.Path == y.Path && x.Desc == y.Desc && x.Line == y.Line && slices.Equal(x.Detail, y.Detail)
	})
}

type at struct {
	Line int
	Kind Kind
}

func TestProblems(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []at
	}{
		{"no line", "", []at{{1, KindFormat}}},
		{"only blank lines", "\n\n", []at{{1, KindFormat}}},
		{"empty title #", "#\n", []at{{1, KindEmptyTitle}}},
		{"empty title # and space", "# \n", []at{{1, KindEmptyTitle}}},
		{"title without space", "#title\na\tx\n", []at{{1, KindFormat}}},
		{"title with tab", "#\ttitle\na\tx\n", []at{{1, KindFormat}}},
		{"no title line", "a\tx\n", []at{{1, KindFormat}}},
		{"no title line, read as an entry", "a\n", []at{{1, KindFormat}, {1, KindEmptyDesc}}},
		{"BOM", "\xef\xbb\xbf# t\na\tx\n", []at{{1, KindFormat}}},
		{"CRLF", "# t\r\na\tx\r\n", nil},
		{"blank lines", "\n# t\n\na\tx\n\n", nil},
		{"no LF at the end", "# t\na\tx", nil},
		{"file without desc", "# t\na.go\n", []at{{2, KindEmptyDesc}}},
		{"file with only spaces as desc", "# t\na.go\t  \n", []at{{2, KindEmptyDesc}}},
		{"file with only a detail", "# t\na.go\n\t| why\n", []at{{2, KindEmptyDesc}}},
		{"directory without children", "# t\nsrc/\n", []at{{2, KindEmptyDesc}}},
		{"skeleton line", "# t\nsrc/\n\ta.go\tx\n", nil},
		{"two levels deeper", "# t\nsrc/\n\t\ta.go\tx\n", []at{{2, KindEmptyDesc}, {3, KindFormat}}},
		{"first entry indented", "# t\n\ta.go\tx\n", []at{{2, KindFormat}}},
		{"space in indentation", "# t\n a.go\tx\n", []at{{2, KindFormat}}},
		{"space after tab", "# t\nsrc/\tx\n\t a.go\tx\n", []at{{3, KindFormat}}},
		{"file with children", "# t\na.go\tx\n\tb\ty\n", []at{{3, KindFormat}}},
		{"same path twice", "# t\na.go\tx\na.go\ty\n", []at{{3, KindFormat}}},
		{"same path written two ways", "# t\nsrc/\tx\n\ta.go\ty\nsrc/a.go\tz\n", []at{{4, KindFormat}}},
		{"file and directory of one name", "# t\na\tx\na/\ty\n", nil},
		{"leading slash", "# t\n/a\tx\n", []at{{2, KindFormat}}},
		{"empty part", "# t\na//b\tx\n", []at{{2, KindFormat}}},
		{"dot", "# t\n./a\tx\n", []at{{2, KindFormat}}},
		{"dot dot", "# t\na/../b\tx\n", []at{{2, KindFormat}}},
		{"dot dot as a child", "# t\nsrc/\n\t../a\tx\n", []at{{3, KindFormat}}},
		{"pattern", "# t\n*_test.go\tx\n", nil},
		{"directory pattern", "# t\ngen-*/\tx\n", nil},
		{"pattern not last", "# t\n*/a.go\tx\n", []at{{2, KindFormat}}},
		{"directory pattern with a child", "# t\ngen-*/\tx\n\ta.go\ty\n", []at{{3, KindFormat}}},
		{"detail with no entry", "# t\n| why\n", []at{{2, KindFormat}}},
		{"detail not indented", "# t\na\tx\n| why\n", []at{{3, KindFormat}}},
		{"detail too deep", "# t\na\tx\n\t\t| why\n", []at{{3, KindFormat}}},
		{"detail after a child", "# t\nsrc/\tx\n\ta\ty\n\t| why\n", []at{{4, KindFormat}}},
		{"detail of a directory then children", "# t\nsrc/\tx\n\t| why\n\ta\ty\n", nil},
		{"bar without space is an entry", "# t\n|x\ty\n", nil},
		{"problems in line order", "# t\na\nb\tx\nb\ty\n", []at{{2, KindEmptyDesc}, {4, KindFormat}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, problems := Parse([]byte(tt.in))
			var got []at
			for _, p := range problems {
				if p.Message == "" {
					t.Errorf("no message: %+v", p)
				}
				got = append(got, at{p.Line, p.Kind})
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v (%v)", got, tt.want, problems)
			}
		})
	}
}

func TestParseValues(t *testing.T) {
	f, problems := Parse([]byte("#   title  \r\nsrc/\t\t\tdir  \n\t|\n\t| first\n\t|   indented\n\ta.go\t\tdesc\twith tab \n|x\ty\n"))
	if len(problems) != 0 {
		t.Fatalf("problems %v", problems)
	}
	if f.Title != "title" {
		t.Errorf("title %q", f.Title)
	}
	want := []flat{
		{"src/", "dir", []string{"", "first", "  indented"}, 2},
		{"src/a.go", "desc\twith tab", nil, 6},
		{"|x", "y", nil, 7},
	}
	if got := flatten(f.Entries); !equalFlat(got, want) {
		t.Errorf("entries\n%v\nwant\n%v", got, want)
	}
}

func TestBytes(t *testing.T) {
	f, _ := Parse([]byte("# t \r\n\nsrc/\t\t x \n\t|\n\t| n  \n\n\ta.go\ty"))
	want := "# t\nsrc/\tx\n\t|\n\t| n\n\ta.go\ty\n"
	if got := string(f.Bytes()); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := string((&File{}).Bytes()); got != "# \n" {
		t.Errorf("empty file %q", got)
	}
}

// What is written is canonical even when the File was not read (fnaze set
// fills it from the command line), so Bytes itself removes the spaces.
func TestBytesOfMadeFile(t *testing.T) {
	f := &File{Title: " t ", Entries: []*Entry{{Name: "a.go", Desc: " x\t", Detail: []string{"n \t", " "}}}}
	want := "# t\na.go\tx\n\t| n\n\t|\n"
	if got := string(f.Bytes()); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// The example of fnaze new in docs/reference/commands.md.
func TestSkeleton(t *testing.T) {
	f, _ := Parse([]byte("# 認証の期限を延長に変えた\nsrc/\n\tauth/\t期限を延長に変えた\n\t\tsession.go\tアクセスのたびに期限を延ばす\n\t\t\t| 短すぎてセッションが切れる報告があった\n"))
	want := "# \nsrc/\n\tauth/\n\t\tsession.go\n"
	if got := string(f.Skeleton().Bytes()); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if f.Entries[0].Children[0].Desc == "" {
		t.Error("Skeleton changed the file it was made from")
	}
}

func TestPathLines(t *testing.T) {
	en, _ := Parse(readExample(t, example1+".txt"))
	ja, _ := Parse(readExample(t, example1+"_ja.txt"))
	want := []string{"src/", "\tauth/", "\t\tsession.go", "\t\t*_test.go", "config/auth.yaml", "docs/old.md"}
	if got := en.PathLines(); !slices.Equal(got, want) {
		t.Errorf("en %q", got)
	}
	if got := ja.PathLines(); !slices.Equal(got, want) {
		t.Errorf("ja %q", got)
	}
}
