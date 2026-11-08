package cli

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/width"

	"github.com/amisonnet8/fnaze/internal/core"
)

// The text output, without --json (docs/reference/commands.md). Nothing is
// colored.

const (
	indent = "  " // one level of show
	gap    = "  " // between two columns
)

// runeWidth is the display width of r: 2 for wide and fullwidth characters
// (East Asian Width W and F), 0 for combining marks and format characters
// (such as U+0301 and U+200D, which take no column of their own), and 1 for
// everything else. Ambiguous characters (A, such as ○ and ①) count as 1.
func runeWidth(r rune) int {
	if unicode.In(r, unicode.Mn, unicode.Me, unicode.Cf) {
		return 0
	}
	switch width.LookupRune(r).Kind() {
	case width.EastAsianWide, width.EastAsianFullwidth:
		return 2
	default:
		return 1
	}
}

// textWidth is the display width of s.
func textWidth(s string) int {
	n := 0
	for _, r := range s {
		n += runeWidth(r)
	}
	return n
}

// pad is s followed by spaces up to the display width w.
func pad(s string, w int) string {
	return s + strings.Repeat(" ", max(w-textWidth(s), 0))
}

// newText is one path of a file made, per line. rel turns a path from the
// root into one from the current directory.
func newText(b *strings.Builder, res *core.NewResult, rel func(string) string) {
	for _, f := range res.Files {
		b.WriteString(rel(f.Path) + "\n")
	}
}

// statusText is one line per item: <file>:<line>: <kind>: <message>, without
// :<line> when the line is 0.
func statusText(b *strings.Builder, res *core.StatusResult, rel func(string) string) {
	for _, it := range res.Items {
		b.WriteString(rel(it.File))
		if it.Line != 0 {
			b.WriteString(":" + strconv.Itoa(it.Line))
		}
		fmt.Fprintf(b, ": %s: %s\n", it.Kind, it.Message)
	}
}

// showRow is one line of show before the descs are aligned: the indented
// name, and the desc or one line of the detail ("| …").
type showRow struct {
	left, right string
}

// showText is the title, an empty line, and the entries as a tree: two spaces
// for each level, the descs in one column two spaces after the widest path
// (in display width), and each detail line in that column after "| ". With no
// entries, it is the title alone.
func showText(b *strings.Builder, res *core.ShowResult) {
	b.WriteString("# " + res.Title + "\n")
	var rows []showRow
	var walk func(entries []core.ShowEntry, depth int)
	walk = func(entries []core.ShowEntry, depth int) {
		for _, e := range entries {
			rows = append(rows, showRow{left: strings.Repeat(indent, depth) + e.Name, right: e.Desc})
			if e.Detail != "" {
				for line := range strings.SplitSeq(e.Detail, "\n") {
					rows = append(rows, showRow{right: strings.TrimRight("| "+line, " ")})
				}
			}
			walk(e.Children, depth+1)
		}
	}
	walk(res.Entries, 0)
	if len(rows) == 0 {
		return
	}
	b.WriteString("\n")
	col := 0
	for _, r := range rows {
		col = max(col, textWidth(r.left))
	}
	col += len(gap)
	for _, r := range rows {
		if r.right == "" {
			b.WriteString(r.left + "\n")
			continue
		}
		b.WriteString(pad(r.left, col) + r.right + "\n")
	}
}

// logText is one line per group: the short name, the datetime in the local
// time zone to the minute, the languages joined with ",", the number of
// entries and the title, in aligned columns.
func logText(b *strings.Builder, res *core.LogResult) {
	ids := make([]string, len(res.Groups))
	for i, g := range res.Groups {
		ids[i] = g.Group
	}
	short := shortNames(ids)
	langs := make([]string, len(res.Groups))
	shortWidth, langsWidth, countWidth := 0, 0, 0
	for i, g := range res.Groups {
		shortWidth = max(shortWidth, len(short[i]))
		langs[i] = strings.Join(g.Langs, ",")
		langsWidth = max(langsWidth, textWidth(langs[i]))
		countWidth = max(countWidth, len(strconv.Itoa(g.Entries)))
	}
	for i, g := range res.Groups {
		line := pad(short[i], shortWidth) + gap + g.Datetime.Local().Format("2006-01-02 15:04") + gap +
			pad(langs[i], langsWidth) + gap + fmt.Sprintf("%*d", countWidth, g.Entries) + gap + g.Title
		b.WriteString(strings.TrimRight(line, " ") + "\n")
	}
}

// shortMin is the length of the shortest short name: the datetime, "_" and
// four characters of the UUID.
const shortMin = len("20060102T150405Z_") + 4

// shortNames returns, for each group, the shortest start of it that no other
// group starts with, and at least shortMin long.
func shortNames(ids []string) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		n := min(shortMin, len(id))
		for ; n < len(id); n++ {
			if !startsAnother(ids, i, id[:n]) {
				break
			}
		}
		out[i] = id[:n]
	}
	return out
}

func startsAnother(ids []string, self int, prefix string) bool {
	for j, other := range ids {
		if j != self && strings.HasPrefix(other, prefix) {
			return true
		}
	}
	return false
}
