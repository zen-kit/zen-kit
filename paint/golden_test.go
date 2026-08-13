package paint_test

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zen-kit/zen-kit/paint"
	"github.com/zen-kit/zen-kit/syntax"
	"github.com/zen-kit/zen-kit/theme"
)

var update = flag.Bool("update", false, "regenerate the golden files")

// Painted rows are compared against testdata rather than asserted inline,
// because a line of escape sequences written out in a test is unreadable. `cat`
// a golden file in a terminal to see what it holds.
func golden(t *testing.T, name, got string) {
	t.Helper()

	path := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("creating testdata: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v (run: make golden)", path, err)
	}
	if got != string(want) {
		t.Errorf("painted output changed.\n got %q\nwant %q", got, want)
	}
}

// Tokens are hand-built rather than taken from Chroma, so a golden file records
// what the painter did and not what a lexer version thought of a line.
func tokens() []syntax.Token {
	return []syntax.Token{
		{Text: "const ", Color: theme.RosePineMoon.Accent},
		{Text: "n", Color: theme.RosePineMoon.Text},
		{Text: " = "},
		{Text: "4", Color: theme.RosePineMoon.Warning},
	}
}

func painter() paint.Painter {
	return paint.Painter{Theme: theme.RosePineMoon}
}

func TestGoldenLines(t *testing.T) {
	tests := []struct {
		name  string
		line  paint.Line
		width int
	}{
		{"line_added", paint.Line{Kind: paint.Added, New: 12, Tokens: tokens()}, 40},
		{"line_removed", paint.Line{Kind: paint.Removed, Old: 11, Tokens: tokens()}, 40},
		{"line_context", paint.Line{Kind: paint.Context, Old: 11, New: 12, Tokens: tokens()}, 40},
		{
			"tabs",
			paint.Line{Kind: paint.Context, Old: 11, New: 12, Tokens: []syntax.Token{
				{Text: "\t"},
				{Text: "return", Color: theme.RosePineMoon.Accent},
				{Text: "\tnil"},
			}},
			40,
		},
		{
			"clipped",
			paint.Line{Kind: paint.Added, New: 12, Tokens: []syntax.Token{
				{Text: "if err != nil { return fmt.Errorf(\"painting: %w\", err) }", Color: theme.RosePineMoon.Text},
			}},
			24,
		},
		// An odd remainder against two-cell runes is where the cut used to come
		// back a column short of the pane.
		{
			"clipped_wide",
			paint.Line{Kind: paint.Added, New: 12, Tokens: []syntax.Token{
				{Text: "// 日本語のコメント", Color: theme.RosePineMoon.Subtle},
			}},
			21,
		},
		{
			"fill_override",
			paint.Line{Kind: paint.Added, New: 12, Tokens: tokens(), Fill: theme.RosePineMoon.SelectedBackground},
			40,
		},
		{"wide_gutter", paint.Line{Kind: paint.Context, Old: 1234, New: 1235, Tokens: tokens()}, 40},
	}

	p := painter()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gutter := paint.Gutter(max(tt.line.Old, tt.line.New))
			golden(t, tt.name, p.Line(tt.line, gutter, tt.width))
		})
	}
}

// A held-open column has to be exactly as wide as a filled one, or the marker
// moves between a line that has both numbers and a line that has one.
func TestGoldenOneSided(t *testing.T) {
	p := painter()
	gutter := paint.Gutter(120)

	rows := []string{
		p.Line(paint.Line{Kind: paint.Added, New: 120, Tokens: tokens()}, gutter, 40),
		p.Line(paint.Line{Kind: paint.Removed, Old: 119, Tokens: tokens()}, gutter, 40),
		p.Line(paint.Line{Kind: paint.Context, Old: 119, New: 120, Tokens: tokens()}, gutter, 40),
	}
	golden(t, "one_sided", strings.Join(rows, "\n"))
}

func TestGoldenHunkHeader(t *testing.T) {
	golden(t, "hunk_header", painter().HunkHeader(paint.Header{Text: "@@ -11,4 +12,6 @@ func Paint()"}, paint.Gutter(1235), 40))
}

// The heading a cursor is on: filled to the edge, with the mark in the column
// the change marks under it use.
func TestGoldenHunkHeaderMarked(t *testing.T) {
	golden(t, "hunk_header_marked", painter().HunkHeader(paint.Header{
		Text:   "@@ -11,4 +12,6 @@ func Paint()",
		Marker: "▸",
		Fill:   theme.RosePineMoon.SelectedBackground,
	}, paint.Gutter(1235), 40))
}
