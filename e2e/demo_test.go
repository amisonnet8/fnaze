package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode"

	"golang.org/x/text/width"
)

// demoLang is one of the two images: docs/assets/demo.svg at the top of
// README.md, and demo_ja.svg at the top of README_ja.md.
type demoLang struct {
	suffix string // of the file name
	env    string // FNAZE_LANG
	title  string // written with fnaze title
	desc   string // written with fnaze set
}

var demoLangs = []demoLang{
	{"", "", "Make the session TTL configurable per client", "Read the TTL from the client's settings"},
	{"_ja", "ja", "セッションの期限を、クライアントごとに設定できるようにした", "期限をクライアントの設定から読む"},
}

// demoSession is what the images show, in the root of docs/examples/changes/:
// each scene is a screen, and each line of it the arguments of one fnaze run.
// The Japanese image runs the same commands with FNAZE_LANG=ja.
func demoSession(l demoLang) [][][]string {
	return [][][]string{
		{{"new"}, {"title", l.title}, {"set", "src/auth/session.go", l.desc}, {"status"}},
		{{"show"}},
		{{"log"}},
	}
}

// TestDemoSVG keeps docs/assets/demo.svg and demo_ja.svg the same as what
// fnaze prints: it runs demoSession and draws it. fnaze runs in this process
// (runCLI), with the clock and the randomness of the examples of new, and log
// shows the times in UTC. With UPDATE_DEMO=1 (qsoku demo), it writes the
// files instead of comparing.
func TestDemoSVG(t *testing.T) {
	defer func(l *time.Location) { time.Local = l }(time.Local)
	time.Local = time.UTC
	for _, l := range demoLangs {
		name := "docs/assets/demo" + l.suffix + ".svg"
		t.Run(name, func(t *testing.T) {
			root := exampleRoot(t)
			var scenes [][]demoStep
			for _, runs := range demoSession(l) {
				var scene []demoStep
				for _, args := range runs {
					r := runCLI(l.env, append(args, "-C", root)...)
					if r.stderr != "" {
						t.Fatalf("fnaze %q: %s", args, r)
					}
					scene = append(scene, demoStep{cmd: demoCommand(args), out: splitOutput(filepath.ToSlash(r.stdout))})
				}
				scenes = append(scenes, scene)
			}
			got := drawDemo(scenes)

			path := filepath.Join("..", filepath.FromSlash(name))
			if os.Getenv("UPDATE_DEMO") == "1" {
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(want) != got {
				t.Errorf("%s is not what fnaze prints now: run qsoku demo, and look at the new image before committing it", name)
			}
		})
	}
}

// demoCommand is the command line as typed: an argument with a space is put
// in double quotes.
func demoCommand(args []string) string {
	words := []string{"fnaze"}
	for _, a := range args {
		if strings.ContainsAny(a, " '") {
			a = `"` + a + `"`
		}
		words = append(words, a)
	}
	return strings.Join(words, " ")
}

// splitOutput is the lines of an output, without the last newline.
func splitOutput(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

type demoStep struct {
	cmd string
	out []string
}

// demoWidth is the columns that s takes in a terminal, as fnaze counts them
// (internal/cli): 2 for wide and fullwidth characters, 0 for marks and format
// characters, 1 for the rest.
func demoWidth(s string) int {
	n := 0
	for _, r := range s {
		n += demoRuneWidth(r)
	}
	return n
}

func demoRuneWidth(r rune) int {
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

// The look of the terminal: sizes in pixels, times in seconds.
const (
	demoFont      = 14.0
	demoCol       = 8.4 // the advance of one column (0.6 em, that of common monospace fonts)
	demoLine      = 20.0
	demoPad       = 20.0
	demoBar       = 36.0
	demoMinWidth  = 640.0
	demoStart     = 0.6  // before the first scene
	demoTypeCol   = 0.04 // to type one column
	demoAfterType = 0.35 // from the end of typing to the output
	demoReadLine  = 0.2  // the time to read one line of output
	demoReadMin   = 0.8
	demoSceneEnd  = 1.2 // between the last output of a scene and the next scene
	demoLoopEnd   = 1.5 // after the last scene, before it all starts again
)

// demoAnim is one CSS animation: the element is hidden before from and after
// to (seconds), shown in between. A negative to shows it to the end.
type demoAnim struct {
	class    string
	from, to float64
}

// drawDemo draws the scenes as an SVG of a terminal window that types each
// command, shows its output, and goes to the next scene, in a loop. Each
// character is put at its column (wide characters take two), so the columns
// line up whatever width the font gives to wide characters.
func drawDemo(scenes [][]demoStep) string {
	cols, rows := 0, 0
	for _, scene := range scenes {
		n := 0
		for _, st := range scene {
			cols = max(cols, demoWidth("$ "+st.cmd))
			for _, l := range st.out {
				cols = max(cols, demoWidth(l))
			}
			n += 1 + len(st.out)
		}
		rows = max(rows, n)
	}
	width := max(demoMinWidth, 2*demoPad+float64(cols)*demoCol)
	height := demoBar + 2*demoPad + float64(rows)*demoLine

	var body strings.Builder
	var anims []demoAnim
	typing := map[string][2]float64{} // the class of a cover: when typing starts and ends
	t := demoStart
	for i, scene := range scenes {
		sceneClass := fmt.Sprintf("s%d", i)
		sceneFrom := t
		fmt.Fprintf(&body, "<g class=\"%s\">\n", sceneClass)
		row := 0
		for j, st := range scene {
			id := fmt.Sprintf("%d-%d", i, j)
			y := demoBar + demoPad + float64(row)*demoLine
			typed := float64(demoWidth(st.cmd)) * demoTypeCol

			// The command line appears with its prompt; a cover over the
			// command shrinks to the right, column by column, as if typed.
			anims = append(anims, demoAnim{"c" + id, t, -1})
			fmt.Fprintf(&body, "<g class=\"c%s\">\n", id)
			body.WriteString(demoText(y, []demoSeg{{"$", "prompt"}, {" " + st.cmd, "cmd"}}))
			fmt.Fprintf(&body, "<rect class=\"k%s cover\" x=\"%.1f\" y=\"%.1f\" width=\"%.1f\" height=\"%.1f\"/>\n</g>\n",
				id, demoPad+2*demoCol, y, float64(demoWidth(st.cmd))*demoCol+2, demoLine)
			typing["k"+id] = [2]float64{t, t + typed}
			t += typed + demoAfterType
			row++

			if len(st.out) == 0 {
				continue
			}
			anims = append(anims, demoAnim{"o" + id, t, -1})
			fmt.Fprintf(&body, "<g class=\"o%s\">\n", id)
			for _, l := range st.out {
				body.WriteString(demoText(demoBar+demoPad+float64(row)*demoLine, demoColor(l)))
				row++
			}
			body.WriteString("</g>\n")
			t += max(demoReadMin, float64(len(st.out))*demoReadLine)
		}
		body.WriteString("</g>\n")
		anims = append(anims, demoAnim{sceneClass, sceneFrom, t + demoSceneEnd})
		t += demoSceneEnd
	}
	total := t + demoLoopEnd

	var css strings.Builder
	pct := func(s float64) string { return fmt.Sprintf("%.2f%%", s/total*100) }
	after := func(s float64) string { return fmt.Sprintf("%.2f%%", s/total*100+0.01) }
	for _, a := range anims {
		fmt.Fprintf(&css, ".%s{animation:%s %.2fs linear infinite}\n", a.class, a.class, total)
		if a.to >= 0 {
			fmt.Fprintf(&css, "@keyframes %s{0%%,%s{opacity:0}%s,%s{opacity:1}%s,100%%{opacity:0}}\n",
				a.class, pct(a.from), after(a.from), pct(a.to), after(a.to))
		} else {
			fmt.Fprintf(&css, "@keyframes %s{0%%,%s{opacity:0}%s,100%%{opacity:1}}\n", a.class, pct(a.from), after(a.from))
		}
	}
	for i, scene := range scenes {
		for j, st := range scene {
			k := fmt.Sprintf("k%d-%d", i, j)
			span := typing[k]
			n := max(1, demoWidth(st.cmd))
			fmt.Fprintf(&css, ".%s{animation:%s %.2fs infinite}\n", k, k, total)
			fmt.Fprintf(&css, "@keyframes %s{0%%,%s{transform:scaleX(1);animation-timing-function:steps(%d,end)}%s,100%%{transform:scaleX(0)}}\n",
				k, pct(span[0]), n, pct(span[1]))
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"%.0f\" height=\"%.0f\" viewBox=\"0 0 %.0f %.0f\" role=\"img\" aria-label=\"fnaze in a terminal: new, title, set, status, show and log\">\n",
		width, height, width, height)
	b.WriteString("<!-- Made by TestDemoSVG (e2e/demo_test.go) from what fnaze prints; run qsoku demo to make it again. -->\n")
	b.WriteString("<style>\n")
	b.WriteString("text{font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,\"DejaVu Sans Mono\",\"Noto Sans Mono CJK JP\",monospace;font-size:14px;fill:#c9d1d9;white-space:pre}\n")
	b.WriteString(".prompt{fill:#3fb950;font-weight:bold}.cmd{fill:#f0f6fc;font-weight:bold}.title{fill:#f0f6fc;font-weight:bold}")
	b.WriteString(".path{fill:#79c0ff}.detail{fill:#8b949e}.warn{fill:#d29922}.dim{fill:#8b949e}\n")
	b.WriteString(".cover{fill:#0d1117;transform-box:fill-box;transform-origin:right}\n")
	b.WriteString(css.String())
	b.WriteString("@media (prefers-reduced-motion:reduce){*{animation:none!important}}\n")
	b.WriteString("</style>\n")
	fmt.Fprintf(&b, "<rect x=\"0.5\" y=\"0.5\" width=\"%.0f\" height=\"%.0f\" rx=\"10\" fill=\"#0d1117\" stroke=\"#30363d\"/>\n", width-1, height-1)
	fmt.Fprintf(&b, "<path d=\"M0.5 %.1fH%.1f\" stroke=\"#30363d\"/>\n", demoBar, width-0.5)
	b.WriteString("<circle cx=\"20\" cy=\"18\" r=\"6\" fill=\"#ff5f56\"/><circle cx=\"40\" cy=\"18\" r=\"6\" fill=\"#ffbd2e\"/><circle cx=\"60\" cy=\"18\" r=\"6\" fill=\"#27c93f\"/>\n")
	fmt.Fprintf(&b, "<text x=\"%.1f\" y=\"23\" text-anchor=\"middle\" class=\"dim\" style=\"font-size:12px\">fnaze</text>\n", width/2)
	b.WriteString(body.String())
	b.WriteString("</svg>\n")
	return b.String()
}

// demoSeg is a run of text drawn in one class.
type demoSeg struct {
	text, class string
}

// demoColor splits a line of output into the runs to color: the title of
// show, the paths of show and new, the details, the kinds of status.
func demoColor(line string) []demoSeg {
	switch {
	case strings.HasPrefix(line, "# "):
		return []demoSeg{{line, "title"}}
	case strings.HasPrefix(line, ".fnaze/"):
		if file, rest, ok := strings.Cut(line, ": "); ok { // status: <file>:<line>: <kind>: <message>
			if kind, msg, ok := strings.Cut(rest, ": "); ok {
				return []demoSeg{{file + ": ", "dim"}, {kind, "warn"}, {": " + msg, ""}}
			}
		}
		return []demoSeg{{line, "path"}}
	}
	trimmed := strings.TrimLeft(line, " ")
	lead := line[:len(line)-len(trimmed)]
	if strings.HasPrefix(trimmed, "| ") || trimmed == "|" {
		return []demoSeg{{line, "detail"}}
	}
	// An entry of show: the path, then the desc after two spaces or more.
	if lead != "" || strings.HasSuffix(strings.Fields(line + " x")[0], "/") || strings.Contains(trimmed, "  ") {
		path, desc, _ := strings.Cut(trimmed, "  ")
		return []demoSeg{{lead + path, "path"}, {"  " + desc, ""}}
	}
	return []demoSeg{{line, ""}}
}

// demoText draws one line at the baseline of the row whose top is y. Spaces
// are not drawn: every other character gets its own x, from its column.
func demoText(y float64, segs []demoSeg) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<text y=\"%.1f\">", y+demoFont+1)
	col := 0
	for _, s := range segs {
		var xs []string
		var chars strings.Builder
		for _, r := range s.text {
			if r != ' ' {
				xs = append(xs, fmt.Sprintf("%.1f", demoPad+float64(col)*demoCol))
				chars.WriteString(demoEscape(r))
			}
			col += demoRuneWidth(r)
		}
		if len(xs) == 0 {
			continue
		}
		class := ""
		if s.class != "" {
			class = fmt.Sprintf(" class=\"%s\"", s.class)
		}
		fmt.Fprintf(&b, "<tspan x=\"%s\"%s>%s</tspan>", strings.Join(xs, " "), class, chars.String())
	}
	b.WriteString("</text>\n")
	return b.String()
}

func demoEscape(r rune) string {
	switch r {
	case '&':
		return "&amp;"
	case '<':
		return "&lt;"
	case '>':
		return "&gt;"
	case '"':
		return "&quot;"
	}
	return string(r)
}
