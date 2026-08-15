// Command kitdemo paints a canned diff to stdout and exits. Golden files hold
// the painter still; this is where a rendering change is judged.
package main

import (
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/zen-kit/zen-kit/paint"
	"github.com/zen-kit/zen-kit/syntax"
	"github.com/zen-kit/zen-kit/theme"
)

// width is a pane narrow enough that one row overflows it. A truncating change
// only shows at a width where something has to be cut.
const width = 76

// row is one line of the canned diff. Old and New are line numbers, 0 where the
// line is not on that side. Cursor marks the row a caller has selected, which is
// what Fill is for.
type row struct {
	Kind     paint.Kind
	Old, New int
	Text     string
	Cursor   bool
}

type hunk struct {
	Header string

	// Cursor marks the hunk a caller has landed on, which is what the header's
	// own Marker and Fill are for. Badged is the state beside it.
	Cursor bool
	Badged bool
	Rows   []row
}

// Two hunks, four-digit numbers in the second. One gutter serves a whole file,
// so a demo that never leaves two digits proves nothing about the alignment.
var hunks = []hunk{
	{
		Header: "@@ -41,8 +41,9 @@ func (p Painter) Line",
		Cursor: true,
		Rows: []row{
			{Kind: paint.Context, Old: 41, New: 41, Text: "// Line is one row: numbers, marker, source."},
			{Kind: paint.Removed, Old: 42, Text: "func (p Painter) Line(l Line, width int) string {"},
			{Kind: paint.Added, New: 42, Text: "func (p Painter) Line(l Line, gutter, width int) string {"},
			{Kind: paint.Removed, Old: 43, Text: "\tmarker := \" \""},
			{Kind: paint.Added, New: 43, Text: "\tmarker, tint := \" \", color.Color(nil)"},
			{Kind: paint.Context, Old: 44, New: 44, Text: "\tif l.Kind == Added {"},
			{Kind: paint.Removed, Old: 45, Text: "\t\tmarker = \"+\""},
			{Kind: paint.Added, New: 45, Text: "\t\tmarker, tint = \"+\", p.Theme.AddedBackground", Cursor: true},
			{Kind: paint.Context, Old: 46, New: 46, Text: "\t}"},
			{Kind: paint.Removed, Old: 47, Text: "\treturn marker + code(l.Tokens)"},
			{Kind: paint.Added, New: 47, Text: "\trow := background(lipgloss.NewStyle(), tint).Render(marker) + p.code(l.Tokens, base)"},
			{Kind: paint.Added, New: 48, Text: "\treturn clipTo(row, width, p.faint())"},
			{Kind: paint.Context, Old: 48, New: 49, Text: "}"},
		},
	},
	{
		Header: "@@ -1229,3 +1230,3 @@ func Gutter(widest int) int",
		Badged: true,
		Rows: []row{
			{Kind: paint.Context, Old: 1229, New: 1230, Text: "func Gutter(widest int) int {"},
			{Kind: paint.Removed, Old: 1230, Text: "\treturn len(strconv.Itoa(widest))"},
			{Kind: paint.Added, New: 1231, Text: "\treturn max(gutterMin, len(strconv.Itoa(widest)))"},
			{Kind: paint.Context, Old: 1231, New: 1232, Text: "}"},
		},
	},
}

func main() {
	t := theme.RosePineMoon

	s, ok := syntax.New(t.Syntax)
	if !ok {
		fmt.Fprintf(os.Stderr, "kitdemo: Chroma does not know %q, using its default style\n", t.Syntax)
	}

	p := paint.Painter{Theme: t}
	gutter := paint.Gutter(widest())

	// Each side is tokenised whole and the two are tokenised apart. A lexer
	// carries state across lines, so highlighting row by row comes apart on the
	// first multi-line string, and running the sides together hands it a file
	// holding both halves of every change.
	oldSide := s.Lines("paint.go", source(paint.Removed))
	newSide := s.Lines("paint.go", source(paint.Added))

	out := []string{lipgloss.NewStyle().Foreground(t.Subtle).
		Render(fmt.Sprintf("theme %s, pane %d columns, gutter %d", t.Name, width, gutter))}

	oldAt, newAt := 0, 0
	for _, h := range hunks {
		head := paint.Header{Text: h.Header}
		if h.Cursor {
			head.Marker, head.Fill = "▸", t.SelectedBackground
		}
		if h.Badged {
			head.Badge = "●"
		}
		out = append(out, p.HunkHeader(head, gutter, width))

		for _, r := range h.Rows {
			l := paint.Line{Kind: r.Kind, Old: r.Old, New: r.New}
			if r.Cursor {
				l.Fill = t.SelectedBackground
			}

			// A context line takes its color from the new side, and advances both.
			switch r.Kind {
			case paint.Removed:
				l.Tokens = nth(oldSide, oldAt)
				oldAt++
			case paint.Added:
				l.Tokens = nth(newSide, newAt)
				newAt++
			case paint.Context:
				l.Tokens = nth(newSide, newAt)
				oldAt++
				newAt++
			}

			out = append(out, p.Line(l, gutter, width))
		}
	}

	fmt.Println(strings.Join(out, "\n"))
}

// source is one side of the diff as a file, context lines included. A side
// missing its unchanged lines does not read as source to a lexer.
func source(kind paint.Kind) string {
	var lines []string
	for _, h := range hunks {
		for _, r := range h.Rows {
			if r.Kind == kind || r.Kind == paint.Context {
				lines = append(lines, r.Text)
			}
		}
	}
	return strings.Join(lines, "\n")
}

func nth(lines [][]syntax.Token, i int) []syntax.Token {
	if i >= len(lines) {
		return nil
	}
	return lines[i]
}

func widest() int {
	n := 0
	for _, h := range hunks {
		for _, r := range h.Rows {
			n = max(n, r.Old, r.New)
		}
	}
	return n
}
