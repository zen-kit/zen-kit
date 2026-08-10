package paint

import (
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
)

// gutterMin is the narrowest a line-number column gets. A file under ten lines
// still reads better with the two columns lined up against its neighbours.
const gutterMin = 2

// Gutter is the line-number column width for a file whose highest line number is
// widest. Callers hand the result to both Line and HunkHeader so the two agree.
func Gutter(widest int) int {
	return max(gutterMin, len(strconv.Itoa(widest)))
}

// Clip truncates to width, marking the cut. It always marks: a caller that
// wants the content left alone when it fits checks the width first.
//
// A single column has room for the mark and nothing else, and MaxWidth(0) means
// no limit rather than no room.
//
// The mark carries its own style, because the content is already rendered by the
// time it is cut: a caller that restyles the result afterwards passes a plain
// one, and a caller clipping a finished row passes the row's, or its background
// stops one cell short of the edge.
func Clip(content string, width int, mark lipgloss.Style) string {
	switch {
	case width <= 0:
		return ""
	case width == 1:
		return mark.Render("…")
	}
	return lipgloss.NewStyle().MaxWidth(width-1).Render(content) + mark.Render("…")
}

// clipTo is Clip for a row that may already fit, since Clip marks either way.
func clipTo(row string, width int, mark lipgloss.Style) string {
	if lipgloss.Width(row) <= width {
		return row
	}
	return Clip(row, width, mark)
}

// number right-aligns a line number, or holds the column open on the side a line
// does not belong to.
func number(n, width int) string {
	if n == 0 {
		return strings.Repeat(" ", width)
	}
	s := strconv.Itoa(n)
	return strings.Repeat(" ", max(0, width-len(s))) + s
}
