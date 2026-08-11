package paint_test

import (
	"fmt"
	"image/color"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	xansi "github.com/charmbracelet/x/ansi"

	"github.com/zen-kit/zen-kit/paint"
	"github.com/zen-kit/zen-kit/syntax"
	"github.com/zen-kit/zen-kit/theme"
)

func TestGutterHoldsTwoColumnsUntilThereAreMoreDigits(t *testing.T) {
	tests := []struct {
		widest int
		want   int
	}{
		{0, 2},
		{1, 2},
		{9, 2},
		{10, 2},
		{99, 2},
		{100, 3},
		{999, 3},
		{1000, 4},
		{12345, 5},
	}

	for _, tt := range tests {
		if got := paint.Gutter(tt.widest); got != tt.want {
			t.Errorf("Gutter(%d) = %d, want %d", tt.widest, got, tt.want)
		}
	}
}

// The gutter and the numbers going into it are computed apart, and them
// disagreeing misaligns every row in a long file. Three digits proves nothing:
// the floor of two hides the arithmetic until the fourth.
func TestEveryRowIsTheSameWidthUpToTheGutterWhateverTheNumber(t *testing.T) {
	p := paint.Painter{Theme: theme.RosePineMoon}

	for _, widest := range []int{4, 42, 421, 4210, 42100} {
		gutter := paint.Gutter(widest)
		want := markerColumn(t, p, paint.Line{Kind: paint.Added, New: widest}, gutter)

		for _, n := range []int{1, 9, 10, widest} {
			got := markerColumn(t, p, paint.Line{Kind: paint.Added, New: n}, gutter)
			if got != want {
				t.Errorf("widest %d, line %d: marker at column %d, want %d", widest, n, got, want)
			}
		}
	}
}

// markerColumn is where the +/− lands once the escapes are stripped.
func markerColumn(t *testing.T, p paint.Painter, l paint.Line, gutter int) int {
	t.Helper()
	plain := xansi.Strip(p.Line(l, gutter, 40))
	i := strings.IndexAny(plain, "+−")
	if i < 0 {
		t.Fatalf("no marker in %q", plain)
	}
	return lipgloss.Width(plain[:i])
}

// Every styled run ends in a reset that clears the background with it, so a row
// carrying a tint has to be painted to the last cell. Short of the width, the
// block reads ragged down its right edge.
func TestARowWithATintIsPaintedToTheFullWidth(t *testing.T) {
	p := paint.Painter{Theme: theme.RosePineMoon}

	tests := []struct {
		name string
		line paint.Line
	}{
		{"added", paint.Line{Kind: paint.Added, New: 12}},
		{"removed", paint.Line{Kind: paint.Removed, Old: 11}},
		{"context under a fill", paint.Line{Kind: paint.Context, Old: 11, New: 12, Fill: theme.RosePineMoon.SelectedBackground}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.line.Tokens = []syntax.Token{{Text: "n = 4"}}
			if got := lipgloss.Width(p.Line(tt.line, 2, 40)); got != 40 {
				t.Errorf("row width = %d, want the full 40", got)
			}
		})
	}
}

// A context line has no background to run out, and padding it would hand the
// caller trailing cells it has to reason about.
func TestAContextRowIsLeftShort(t *testing.T) {
	p := paint.Painter{Theme: theme.RosePineMoon}
	row := p.Line(paint.Line{Kind: paint.Context, Old: 11, New: 12, Tokens: []syntax.Token{{Text: "n = 4"}}}, 2, 40)

	if got := lipgloss.Width(row); got >= 40 {
		t.Errorf("row width = %d, want it to stop at the code", got)
	}
}

func TestFillBeatsTheKindTint(t *testing.T) {
	p := paint.Painter{Theme: theme.RosePineMoon}
	line := paint.Line{Kind: paint.Added, New: 12, Tokens: []syntax.Token{{Text: "n = 4"}}, Fill: theme.RosePineMoon.SelectedBackground}
	row := p.Line(line, 2, 40)

	if !strings.Contains(row, bgSeq(theme.RosePineMoon.SelectedBackground)) {
		t.Error("the fill is not on the row")
	}
	if strings.Contains(row, bgSeq(theme.RosePineMoon.AddedBackground)) {
		t.Error("the added tint painted over the fill")
	}
}

// A theme leaving a surface nil means "leave the terminal's own showing", and
// handing that to Lipgloss is what breaks a transparent background.
func TestARowTakesNoBackgroundFromAThemeThatDefinesNone(t *testing.T) {
	bare := theme.Theme{Name: "bare", Primary: theme.RosePineMoon.Primary, Faint: theme.RosePineMoon.Faint}
	p := paint.Painter{Theme: bare}
	row := p.Line(paint.Line{Kind: paint.Added, New: 12, Tokens: []syntax.Token{{Text: "n = 4"}}}, 2, 40)

	if strings.Contains(row, "48;2;") {
		t.Errorf("row set a background the theme does not define: %q", row)
	}
}

func TestTabsExpandToTheTabWidth(t *testing.T) {
	row := func(p paint.Painter, code string) string {
		return xansi.Strip(p.Line(paint.Line{
			Kind: paint.Context, Old: 11, New: 12, Tokens: []syntax.Token{{Text: code}},
		}, 2, 60))
	}

	tests := []struct {
		name  string
		width int
		want  int
	}{
		{"unset means four", 0, 4},
		{"two", 2, 2},
		{"eight", 8, 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := paint.Painter{Theme: theme.RosePineMoon, TabWidth: tt.width}
			tabbed := row(p, "\tn")

			if strings.Contains(tabbed, "\t") {
				t.Fatalf("a raw tab survived: %q", tabbed)
			}
			if got := lipgloss.Width(tabbed) - lipgloss.Width(row(p, "n")); got != tt.want {
				t.Errorf("tab took %d cells, want %d (%q)", got, tt.want, tabbed)
			}
		})
	}
}

// A wrapped row of code puts its tail under the gutter and every row below it
// out of step, so overflow is cut instead.
func TestARowWiderThanThePaneIsClippedNotWrapped(t *testing.T) {
	p := paint.Painter{Theme: theme.RosePineMoon}
	long := strings.Repeat("n = 4; ", 20)
	row := p.Line(paint.Line{Kind: paint.Added, New: 12, Tokens: []syntax.Token{{Text: long}}}, 2, 24)

	if strings.Contains(row, "\n") {
		t.Error("the row wrapped")
	}
	if got := lipgloss.Width(row); got != 24 {
		t.Errorf("row width = %d, want 24", got)
	}
	if !strings.Contains(xansi.Strip(row), "…") {
		t.Error("the cut is not marked")
	}
}

// A clipped row is still a row with a background, and it has to reach the pane
// edge. CJK and emoji are where it stops doing that: the cut lands on a two-cell
// rune and comes back a column short. Every odd width is a separate case,
// because whether the remainder is one cell or two decides it.
func TestAClippedRowWithWideRunesStillFillsTheWidth(t *testing.T) {
	p := paint.Painter{Theme: theme.RosePineMoon}

	for _, code := range []string{"日本語のコメントです", "🌱 seedling 🌱 seedling"} {
		for width := 18; width <= 25; width++ {
			row := p.Line(paint.Line{Kind: paint.Added, New: 12, Tokens: []syntax.Token{{Text: code}}}, 2, width)
			if got := lipgloss.Width(row); got != width {
				t.Errorf("%q at width %d painted %d cells", code, width, got)
			}
		}
	}
}

// Clip is the primitive both tools truncate with, and it marks either way. A
// caller that wants short content left alone checks the width itself.
func TestClipAlwaysMarksTheCut(t *testing.T) {
	plain := lipgloss.NewStyle()

	tests := []struct {
		name    string
		content string
		width   int
		want    string
	}{
		{"no room at all", "hello", 0, ""},
		{"room for the mark alone", "hello", 1, "…"},
		{"cut", "hello", 3, "he…"},
		{"content that already fits", "hi", 5, "hi…"},
		// A two-cell rune cannot half-fill the last column. Unpadded, the result
		// comes back a column short of the width it was asked for, and a tinted
		// row stops before the pane edge.
		{"cut landing on a wide rune", "日本語", 4, "日 …"},
		{"wide runes cut on the boundary", "日本語", 5, "日本…"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := xansi.Strip(paint.Clip(tt.content, tt.width, plain)); got != tt.want {
				t.Errorf("Clip(%q, %d) = %q, want %q", tt.content, tt.width, got, tt.want)
			}
		})
	}
}

// The header sits over the source it introduces. Asserting only "past the
// marker" passes with the header parked in the marker's own gap, which is where
// it sat until this test was tightened.
func TestTheHunkHeaderStartsAtTheCodeColumn(t *testing.T) {
	p := paint.Painter{Theme: theme.RosePineMoon}

	for _, widest := range []int{9, 120, 4210} {
		gutter := paint.Gutter(widest)
		header := xansi.Strip(p.HunkHeader("@@ -1,2 +1,3 @@", gutter, 60))
		indent := lipgloss.Width(header[:strings.Index(header, "@@")])

		if want := codeColumn(t, p, gutter); indent != want {
			t.Errorf("gutter %d: header starts at column %d, want the code column %d", gutter, indent, want)
		}
	}
}

// codeColumn is where the source starts in a painted row, found by painting a
// token nothing else in the row can contain.
func codeColumn(t *testing.T, p paint.Painter, gutter int) int {
	t.Helper()

	plain := xansi.Strip(p.Line(paint.Line{
		Kind: paint.Added, New: 1, Tokens: []syntax.Token{{Text: "X"}},
	}, gutter, 60))

	i := strings.Index(plain, "X")
	if i < 0 {
		t.Fatalf("no code in %q", plain)
	}
	return lipgloss.Width(plain[:i])
}

func TestAHunkHeaderWiderThanThePaneIsClipped(t *testing.T) {
	p := paint.Painter{Theme: theme.RosePineMoon}
	header := p.HunkHeader("@@ -1,200 +1,240 @@ func AVeryLongEnclosingSymbolName()", 2, 24)

	if got := lipgloss.Width(header); got != 24 {
		t.Errorf("header width = %d, want 24", got)
	}
	if !strings.Contains(xansi.Strip(header), "…") {
		t.Error("the cut is not marked")
	}
}

// paint and syntax compose or neither is worth having: Chroma's colors have to
// arrive as foregrounds over the row's own background.
func TestRealChromaTokensPaintOverTheRowsBackground(t *testing.T) {
	s, ok := syntax.New(theme.RosePineMoon.Syntax)
	if !ok {
		t.Fatalf("Chroma does not know %q", theme.RosePineMoon.Syntax)
	}

	p := paint.Painter{Theme: theme.RosePineMoon}
	lines := s.Lines("a.go", "const n = 4")
	row := p.Line(paint.Line{Kind: paint.Added, New: 12, Tokens: lines[0]}, 2, 40)

	if got := strings.Count(row, bgSeq(theme.RosePineMoon.AddedBackground)); got < 3 {
		t.Errorf("the tint survives %d runs, want it under every token", got)
	}
	if got := xansi.Strip(row); !strings.Contains(got, "const n = 4") {
		t.Errorf("the code did not come through: %q", got)
	}
}

func bgSeq(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("48;2;%d;%d;%d", r>>8, g>>8, b>>8)
}
