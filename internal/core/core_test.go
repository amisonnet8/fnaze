package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/amisonnet8/fnaze/internal/project"
)

const (
	ex1 = "20261107T003426Z_8980a325-8b57-49b0-8e0a-ed0aaf9eeda7"
	ex2 = "20261107T021530Z_2c4cc818-64d6-45e8-9f3a-5471dba04be7"
	// The group that New makes with testNow and the bytes 0, 1, ..., 15 as
	// randomness (the UUID is from Python: uuid.UUID(bytes=..., version=4)).
	newGroup = "20261108T010203Z_00010203-0405-4607-8809-0a0b0c0d0e0f"
)

var testNow = time.Date(2026, 11, 8, 1, 2, 3, 0, time.UTC)

// newCore makes a root in a temporary directory and copies the given example
// files of docs/examples/changes/ into it.
func newCore(t *testing.T, env string, examples ...string) *Core {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Init(dir); err != nil {
		t.Fatal(err)
	}
	for _, e := range examples {
		data, err := os.ReadFile(filepath.Join("..", "..", "docs", "examples", "changes", e))
		if err != nil {
			t.Fatal(err)
		}
		writeFile(t, dir, e, string(data))
	}
	c, err := Open(dir, env)
	if err != nil {
		t.Fatal(err)
	}
	c.Now = func() time.Time { return testNow }
	c.Random = bytes.NewReader([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15})
	return c
}

func allExamples() []string {
	return []string{ex1 + ".txt", ex1 + "_ja.txt", ex2 + ".txt", ex2 + "_ja.txt"}
}

func writeFile(t *testing.T, root, file, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(project.ChangesDir(root), file), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, root, file string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(project.ChangesDir(root), file))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func files(t *testing.T, root string) []string {
	t.Helper()
	entries, err := os.ReadDir(project.ChangesDir(root))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func toJSON(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestOpenWithoutRoot(t *testing.T) {
	dir := t.TempDir()
	if _, err := Open(dir, ""); !errors.Is(err, project.ErrNotFound) {
		t.Errorf("got %v", err)
	}
}

func TestNewFirstGroup(t *testing.T) {
	c := newCore(t, "")
	res, err := c.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"group":"` + newGroup + `","files":[{"lang":"en","path":".fnaze/changes/` + newGroup + `.txt"}]}`
	if got := toJSON(t, res); got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
	if got := readFile(t, c.Root, newGroup+".txt"); got != "# \n" {
		t.Errorf("content %q", got)
	}
}

func TestNewOneLanguageFromEnv(t *testing.T) {
	c := newCore(t, "ja")
	res, err := c.New(nil)
	if err != nil || len(res.Files) != 1 || res.Files[0].Lang != "ja" {
		t.Errorf("got %+v, %v", res, err)
	}
}

func TestNewLanguagesFromEnv(t *testing.T) {
	c := newCore(t, " ja , en")
	res, err := c.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := toJSON(t, res.Files); got != `[{"lang":"ja","path":".fnaze/changes/`+newGroup+`_ja.txt"},{"lang":"en","path":".fnaze/changes/`+newGroup+`.txt"}]` {
		t.Errorf("got %s", got)
	}
}

// The example of docs/examples/README.md: the second group starts from the
// first. Each file is the skeleton of the newest file in its language.
func TestNewFromNewestFiles(t *testing.T) {
	c := newCore(t, "ko", ex1+".txt", ex1+"_ja.txt")
	res, err := c.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := []string{res.Files[0].Lang, res.Files[1].Lang}; !slices.Equal(got, []string{"en", "ja"}) {
		t.Errorf("languages %q: want those of the newest group", got)
	}
	wantEn := "# \nsrc/\n\tauth/\n\t\tsession.go\n\t\t*_test.go\nconfig/auth.yaml\ndocs/old.md\n"
	if got := readFile(t, c.Root, newGroup+".txt"); got != wantEn {
		t.Errorf("en %q", got)
	}
	if got := readFile(t, c.Root, newGroup+"_ja.txt"); got != wantEn {
		t.Errorf("ja %q", got)
	}
}

// Of two groups with en, the newer is the one a new file starts from.
func TestNewFromTheNewest(t *testing.T) {
	c := newCore(t, "", allExamples()...)
	if _, err := c.New([]string{"en"}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, c.Root, newGroup+".txt"); got != "# \nsrc/\n\tauth/\n\t\tsession.go\n" {
		t.Errorf("en %q", got)
	}
	// The file has empty descs, which do not stop set.
	if _, err := c.Set("en", "", "src/auth/session.go", "x", nil, false); err != nil {
		t.Errorf("set on the new file: %v", err)
	}
}

// A file that is already there is not written over.
func TestNewDoesNotOverwrite(t *testing.T) {
	c := newCore(t, "")
	writeFile(t, c.Root, newGroup+"_ja.txt", "# 前からある\n")
	if _, err := c.New([]string{"en", "ja"}); err == nil {
		t.Fatal("no error")
	}
	if got := readFile(t, c.Root, newGroup+"_ja.txt"); got != "# 前からある\n" {
		t.Errorf("written over: %q", got)
	}
	if got := files(t, c.Root); !slices.Equal(got, []string{newGroup + "_ja.txt"}) {
		t.Errorf("left %q", got)
	}
}

func TestNewFromAnOlderGroup(t *testing.T) {
	c := newCore(t, "", ex1+".txt", ex1+"_ja.txt", ex2+".txt")
	if _, err := c.New([]string{"ja", "fr"}); err != nil {
		t.Fatal(err)
	}
	// ja: from the first group (the newest has no ja). fr: no file yet.
	if got := readFile(t, c.Root, newGroup+"_ja.txt"); got != "# \nsrc/\n\tauth/\n\t\tsession.go\n\t\t*_test.go\nconfig/auth.yaml\ndocs/old.md\n" {
		t.Errorf("ja %q", got)
	}
	if got := readFile(t, c.Root, newGroup+"_fr.txt"); got != "# \n" {
		t.Errorf("fr %q", got)
	}
	if _, err := os.Stat(filepath.Join(project.ChangesDir(c.Root), newGroup+".txt")); err == nil {
		t.Error("en was made without being given")
	}
}

func TestNewFails(t *testing.T) {
	for _, langs := range [][]string{{"ja", "JA"}, {"ja", "ja"}, {"zh-hant"}} {
		c := newCore(t, "")
		if _, err := c.New(langs); err == nil {
			t.Errorf("%q: no error", langs)
		}
		if got := files(t, c.Root); len(got) != 0 {
			t.Errorf("%q: made %q", langs, got)
		}
	}
	c := newCore(t, "ja,JA")
	if _, err := c.New(nil); err == nil {
		t.Error("FNAZE_LANG=ja,JA: no error")
	}
}

// When one file cannot be made, the files made before it are removed.
func TestNewLeavesNoPart(t *testing.T) {
	c := newCore(t, "")
	if err := os.Mkdir(filepath.Join(project.ChangesDir(c.Root), newGroup+"_ja.txt"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := c.New([]string{"en", "ja"}); err == nil {
		t.Fatal("no error")
	}
	if got := files(t, c.Root); !slices.Equal(got, []string{newGroup + "_ja.txt"}) {
		t.Errorf("left %q", got)
	}
}

func TestNewRandomFails(t *testing.T) {
	c := newCore(t, "")
	c.Random = bytes.NewReader(nil)
	if _, err := c.New(nil); err == nil {
		t.Error("no error")
	}
}

func TestFindGroup(t *testing.T) {
	c := newCore(t, "", allExamples()...)
	for spec, want := range map[string]string{
		"":                 ex2,
		ex1:                ex1,
		"20261107T00":      ex1,
		"20261107T021530Z": ex2,
	} {
		res, err := c.Show("", spec)
		if err != nil || res.Group != want {
			t.Errorf("%q: got %v, %v; want %s", spec, res, err, want)
		}
	}
	for _, spec := range []string{"20261107T", "2025", ex1 + "x", "8980a325"} {
		if _, err := c.Show("", spec); err == nil {
			t.Errorf("%q: no error", spec)
		}
	}
}

func TestNoGroup(t *testing.T) {
	c := newCore(t, "")
	if _, err := c.Show("", ""); !errors.Is(err, ErrNoGroup) {
		t.Errorf("show: %v", err)
	}
	if _, err := c.Set("", "", "a", "x", nil, false); !errors.Is(err, ErrNoGroup) {
		t.Errorf("set: %v", err)
	}
	if _, err := c.Rm("", "", "a"); !errors.Is(err, ErrNoGroup) {
		t.Errorf("rm: %v", err)
	}
	if res, err := c.Status("", false); err != nil || toJSON(t, res) != `{"items":[]}` {
		t.Errorf("status: %v, %v", res, err)
	}
	if res, err := c.Status("", true); err != nil || toJSON(t, res) != `{"items":[]}` {
		t.Errorf("status --all: %v, %v", res, err)
	}
	if res, err := c.Log(""); err != nil || toJSON(t, res) != `{"groups":[]}` {
		t.Errorf("log: %v, %v", res, err)
	}
	if _, err := c.Status("2026", false); err == nil {
		t.Error("status of a group that is not there: no error")
	}
}

func TestSet(t *testing.T) {
	c := newCore(t, "ja", allExamples()...)
	res, err := c.Set("", "", "src/auth/session.go", "延長を info に", []string{"a", "b"}, true)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"group":"` + ex2 + `","lang":"ja","path":"src/auth/session.go","desc":"延長を info に","detail":"a\nb"}`
	if got := toJSON(t, res); got != want {
		t.Errorf("got %s", got)
	}
	if got := readFile(t, c.Root, ex2+"_ja.txt"); got != "# セッションの延長をログに出す\nsrc/\n\tauth/\n\t\tsession.go\t延長を info に\n\t\t\t| a\n\t\t\t| b\n" {
		t.Errorf("file %q", got)
	}
	// show joins the detail lines with newlines too.
	shown, err := c.Show("", "")
	if err != nil {
		t.Fatal(err)
	}
	if got := shown.Entries[0].Children[0].Children[0].Detail; got != "a\nb" {
		t.Errorf("show: detail %q", got)
	}
	// The other language is not changed.
	data, _ := os.ReadFile(filepath.Join("..", "..", "docs", "examples", "changes", ex2+".txt"))
	if got := readFile(t, c.Root, ex2+".txt"); got != string(data) {
		t.Errorf("en changed: %q", got)
	}
}

func TestSetLanguageAndGroup(t *testing.T) {
	c := newCore(t, "ja", allExamples()...)
	res, err := c.Set("en", "20261107T00", "new.go", "Add", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Group != ex1 || res.Lang != "en" || res.Detail != "" {
		t.Errorf("got %+v", res)
	}
	if got := readFile(t, c.Root, ex1+".txt"); !strings.HasSuffix(got, "docs/old.md\tRemoved; its content moved to docs/auth.md\nnew.go\tAdd\n") {
		t.Errorf("file %q", got)
	}
}

func TestSetFails(t *testing.T) {
	c := newCore(t, "", allExamples()...)
	broken := "# t\n\t\ta\tx\n"
	writeFile(t, c.Root, newGroup+".txt", broken)
	if _, err := c.Set("", "", "a", "x", nil, false); err == nil {
		t.Error("a file with a format error: no error")
	}
	if got := readFile(t, c.Root, newGroup+".txt"); got != broken {
		t.Errorf("the file was changed: %q", got)
	}
	for _, tt := range []struct{ lang, group, path string }{
		{"fr", ex1, "a"},
		{"zh-hant", ex1, "a"},
		{"", ex1, "/a"},
	} {
		if _, err := c.Set(tt.lang, tt.group, tt.path, "x", nil, false); err == nil {
			t.Errorf("%+v: no error", tt)
		}
	}
}

func TestTitle(t *testing.T) {
	c := newCore(t, "ja", allExamples()...)
	res, err := c.Title("", "", " ログに出す ")
	if err != nil {
		t.Fatal(err)
	}
	want := `{"group":"` + ex2 + `","lang":"ja","title":"ログに出す"}`
	if got := toJSON(t, res); got != want {
		t.Errorf("got %s", got)
	}
	if got := readFile(t, c.Root, ex2+"_ja.txt"); got != "# ログに出す\nsrc/\n\tauth/\n\t\tsession.go\t延長のたびに debug でログを出す\n\t\t\t| 本番で 30m の既定を確かめるため\n" {
		t.Errorf("file %q", got)
	}
	// The other language is not changed.
	data, _ := os.ReadFile(filepath.Join("..", "..", "docs", "examples", "changes", ex2+".txt"))
	if got := readFile(t, c.Root, ex2+".txt"); got != string(data) {
		t.Errorf("en changed: %q", got)
	}

	// The language and the group; an empty title, which status reports.
	res, err = c.Title("en", "20261107T00", "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Group != ex1 || res.Lang != "en" || res.Title != "" {
		t.Errorf("got %+v", res)
	}
	if got := readFile(t, c.Root, ex1+".txt"); !strings.HasPrefix(got, "# \nsrc/\n") {
		t.Errorf("file %q", got)
	}
	st, err := c.Status(ex1, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Items) != 1 || st.Items[0].Kind != "empty-title" || st.Items[0].File != ".fnaze/changes/"+ex1+".txt" || st.Items[0].Line != 1 {
		t.Errorf("status %+v", st.Items)
	}
}

func TestTitleFails(t *testing.T) {
	c := newCore(t, "", allExamples()...)
	broken := "# t\n\t\ta\tx\n"
	writeFile(t, c.Root, newGroup+".txt", broken)
	if _, err := c.Title("", "", "x"); err == nil {
		t.Error("a file with a format error: no error")
	}
	if got := readFile(t, c.Root, newGroup+".txt"); got != broken {
		t.Errorf("the file was changed: %q", got)
	}
	for _, tt := range []struct{ lang, group, title string }{
		{"fr", ex1, "x"},
		{"zh-hant", ex1, "x"},
		{"", "nope", "x"},
		{"", ex1, "a\nb"},
	} {
		if _, err := c.Title(tt.lang, tt.group, tt.title); err == nil {
			t.Errorf("%+v: no error", tt)
		}
	}
	data, _ := os.ReadFile(filepath.Join("..", "..", "docs", "examples", "changes", ex1+".txt"))
	if got := readFile(t, c.Root, ex1+".txt"); got != string(data) {
		t.Errorf("a newline changed the file: %q", got)
	}

	// A failure does not write the file, even one that writing would change
	// (here, by adding the final newline).
	c = newCore(t, "")
	writeFile(t, c.Root, newGroup+".txt", "# t\na.go\td")
	if _, err := c.Title("", "", "a\nb"); err == nil {
		t.Error("a newline: no error")
	}
	if got := readFile(t, c.Root, newGroup+".txt"); got != "# t\na.go\td" {
		t.Errorf("a newline wrote the file: %q", got)
	}
}

func TestRm(t *testing.T) {
	c := newCore(t, "", allExamples()...)
	res, err := c.Rm("", "", "src/auth/session.go")
	if err != nil {
		t.Fatal(err)
	}
	if got := toJSON(t, res); got != `{"group":"`+ex2+`","lang":"en","path":"src/auth/session.go","removed":3}` {
		t.Errorf("got %s", got)
	}
	if got := readFile(t, c.Root, ex2+".txt"); got != "# Log session extensions\n" {
		t.Errorf("file %q", got)
	}
	if _, err := c.Rm("", "", "src/auth/session.go"); err == nil {
		t.Error("removed twice: no error")
	}
	if _, err := c.Rm("fr", "", "src/"); err == nil {
		t.Error("no file in fr: no error")
	}
}

func TestStatusExamples(t *testing.T) {
	c := newCore(t, "", allExamples()...)
	for _, all := range []bool{false, true} {
		if res, err := c.Status("", all); err != nil || len(res.Items) != 0 {
			t.Errorf("all=%v: %v, %v", all, res, err)
		}
	}
}

func TestStatusOneGroup(t *testing.T) {
	c := newCore(t, "")
	writeFile(t, c.Root, newGroup+".txt", "# t\na\n")
	res, err := c.Status("", false)
	if err != nil {
		t.Fatal(err)
	}
	if got := items(res); !slices.Equal(got, []item{{newGroup, ".fnaze/changes/" + newGroup + ".txt", 2, "empty-desc"}}) {
		t.Errorf("got %v", got)
	}
}

type item struct {
	Group, File string
	Line        int
	Kind        string
}

func items(res *StatusResult) []item {
	var out []item
	for _, it := range res.Items {
		if it.Message == "" {
			out = append(out, item{"NO MESSAGE", it.File, it.Line, it.Kind})
			continue
		}
		out = append(out, item{it.Group, it.File, it.Line, it.Kind})
	}
	return out
}

func TestStatus(t *testing.T) {
	c := newCore(t, "", allExamples()...)
	g := newGroup
	writeFile(t, c.Root, g+".txt", "# \nsrc/\n\ta.go\n")
	writeFile(t, c.Root, g+"_ja.txt", "# 題\nsrc/\n\tb.go\tx\n")
	writeFile(t, c.Root, "notes.txt", "")
	writeFile(t, c.Root, ex1+"_en.txt", "")
	writeFile(t, c.Root, ex1+"_fr.txt", "# t\na\n")
	p := func(f string) string { return ".fnaze/changes/" + f }
	res, err := c.Status("", false)
	if err != nil {
		t.Fatal(err)
	}
	want := []item{
		{"", p(ex1 + "_en.txt"), 0, "format"},
		{g, p(g + ".txt"), 1, "empty-title"},
		{g, p(g + ".txt"), 3, "empty-desc"},
		{g, p(g + "_ja.txt"), 0, "lang-mismatch"},
		{"", p("notes.txt"), 0, "format"},
	}
	if got := items(res); !slices.Equal(got, want) {
		t.Errorf("got\n%v\nwant\n%v", got, want)
	}
	// --all also checks the first group, whose fr file has an empty desc and
	// path lines that differ.
	res, err = c.Status("", true)
	if err != nil {
		t.Fatal(err)
	}
	wantAll := append([]item{
		{ex1, p(ex1 + "_fr.txt"), 0, "lang-mismatch"},
		{ex1, p(ex1 + "_fr.txt"), 2, "empty-desc"},
	}, want...)
	slices.SortStableFunc(wantAll, func(a, b item) int {
		if a.File != b.File {
			return strings.Compare(a.File, b.File)
		}
		return a.Line - b.Line
	})
	if got := items(res); !slices.Equal(got, wantAll) {
		t.Errorf("--all got\n%v\nwant\n%v", got, wantAll)
	}
	if _, err := c.Status(ex1, true); err == nil {
		t.Error("--all with a group: no error")
	}
	res, err = c.Status(ex2, false)
	if err != nil || len(res.Items) != 2 {
		t.Errorf("only the bad names for %s: %v, %v", ex2, items(res), err)
	}
}

// Files of one group in two languages that differ only in case can be made by
// hand where the file system tells case apart (Linux). Where it does not
// (macOS, Windows), the second write goes to the first file, so there is
// nothing to check.
func TestStatusLanguagesThatDifferOnlyInCase(t *testing.T) {
	c := newCore(t, "")
	g := newGroup
	for _, f := range []string{g + ".txt", g + "_ja.txt", g + "_JA.txt"} {
		writeFile(t, c.Root, f, "# t\na\tx\n")
	}
	if got := files(t, c.Root); len(got) != 3 {
		t.Skipf("the file system ignores case: %q", got)
	}
	res, err := c.Status("", false)
	if err != nil {
		t.Fatal(err)
	}
	if got := items(res); !slices.Equal(got, []item{{g, ".fnaze/changes/" + g + ".txt", 0, "format"}}) {
		t.Errorf("got %v", got)
	}
}

func TestShow(t *testing.T) {
	c := newCore(t, "", allExamples()...)
	want := `{"group":"` + ex1 + `","lang":"ja","file":".fnaze/changes/` + ex1 + `_ja.txt","title":"認証のセッションの期限を、固定から延長に変えた","entries":[` +
		`{"name":"src/","path":"src/","desc":"","detail":"","children":[` +
		`{"name":"auth/","path":"src/auth/","desc":"期限を固定から延長に変えた","detail":"","children":[` +
		`{"name":"session.go","path":"src/auth/session.go","desc":"アクセスのたびに期限を延ばす","detail":"","children":[]},` +
		`{"name":"*_test.go","path":"src/auth/*_test.go","desc":"延長のテストを足す","detail":"","children":[]}]}]},` +
		`{"name":"config/auth.yaml","path":"config/auth.yaml","desc":"session.ttl の既定を 30m にした","detail":"短すぎてセッションが切れる報告があった","children":[]},` +
		`{"name":"docs/old.md","path":"docs/old.md","desc":"消した。内容は docs/auth.md に移した","detail":"","children":[]}]}`
	res, err := c.Show("ja", ex1)
	if err != nil {
		t.Fatal(err)
	}
	if got := toJSON(t, res); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	if _, err := c.Show("fr", ex1); err == nil {
		t.Error("no file in fr: no error")
	}
}

// The datetimes are UTC whatever the time zone of the user is
// (.claude/rules/datetime.md). Not parallel: it changes time.Local.
func TestLog(t *testing.T) {
	local := time.Local
	time.Local = time.FixedZone("JST", 9*60*60)
	t.Cleanup(func() { time.Local = local })
	c := newCore(t, "ja", ex1+".txt", ex1+"_ja.txt", ex2+".txt")
	res, err := c.Log("")
	if err != nil {
		t.Fatal(err)
	}
	want := `{"groups":[` +
		`{"group":"` + ex2 + `","datetime":"2026-11-07T02:15:30Z","langs":["en"],"title":"","entries":0},` +
		`{"group":"` + ex1 + `","datetime":"2026-11-07T00:34:26Z","langs":["en","ja"],"title":"認証のセッションの期限を、固定から延長に変えた","entries":6}]}`
	if got := toJSON(t, res); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	res, err = c.Log("en")
	if err != nil || res.Groups[0].Title != "Log session extensions" || res.Groups[0].Entries != 3 {
		t.Errorf("en: %+v, %v", res, err)
	}
}
