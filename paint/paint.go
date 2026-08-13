// Package paint renders one row of a diff. Every exported function is pure:
// the same line at the same width gives the same string. Folding, scroll,
// side-by-side layout, hunk grouping and review state belong to the caller.
package paint

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/zen-kit/zen-kit/syntax"
	"github.com/zen-kit/zen-kit/theme"
)

// defaultTabWidth is what a tab expands to when a Painter names no width. A raw
// tab is a variable number of cells, and one anywhere in a line puts every
// column after it out of step with the line above.
const defaultTabWidth = 4

// Kind is the side of the change a line belongs to.
type Kind int

const (
	Context Kind = iota
	Added
	Removed
)

// Line is one row ready to paint. Old and New are line numbers; 0 means that
// side has none, and the column is still held open so the marker beside it
// does not move.
type Line struct {
	Kind     Kind
	Old, New int
	Tokens   []syntax.Token

	// Fill beats the kind's tint. nil uses it. A cursor row, an active range
	// selection and a reviewed-hunk tint all have to win against added and
	// removed, and each of them is the caller's state, not this package's.
	Fill color.Color
}

// Painter paints rows from one theme.
type Painter struct {
	Theme    theme.Theme
	TabWidth int // 0 means 4
}

// Line is one row of code: the two line numbers, the marker, and the
// highlighted source over a tint of the change it is part of.
//
// The tint is painted cell by cell and the row is padded out to the full width.
// Every styled run ends in a reset that clears the background with it, so a
// joined row wrapped in the background style afterwards would carry it only as
// far as the first token.
//
// Anything wider than the pane is clipped rather than wrapped: a wrapped row
// puts its tail under the gutter and every row below it out of step.
func (p Painter) Line(l Line, gutter, width int) string {
	marker, c := " ", p.Theme.Subtle
	var tint color.Color

	switch l.Kind {
	case Added:
		marker, c, tint = "+", p.Theme.Success, p.Theme.AddedBackground
	case Removed:
		marker, c, tint = "−", p.Theme.Error, p.Theme.RemovedBackground
	}
	if l.Fill != nil {
		tint = l.Fill
	}

	base := background(lipgloss.NewStyle(), tint)
	kind := base.Foreground(c)
	faint := base.Foreground(p.Theme.Subtle)

	oldNum, newNum := faint, faint
	switch l.Kind {
	case Added:
		newNum = kind
	case Removed:
		oldNum = kind
	}

	row := base.Render(" ") +
		oldNum.Render(number(l.Old, gutter)) + base.Render(" ") +
		newNum.Render(number(l.New, gutter)) + base.Render(" ") +
		kind.Render(marker) + base.Render(" ") + p.code(l.Tokens, base)

	if w := lipgloss.Width(row); w > width {
		return Clip(row, width, faint)
	} else if tint != nil {
		// Only a row with a background has one to run out. A context line with
		// no fill is left short, and the pane's own padding finishes it.
		row += base.Render(strings.Repeat(" ", width-w))
	}
	return row
}

// HunkHeader is the @@ line, indented to the code column so it sits over the
// source it introduces rather than in the marker's gap.
func (p Painter) HunkHeader(text string, gutter, width int) string {
	row := strings.Repeat(" ", codeColumn(gutter)) +
		lipgloss.NewStyle().Foreground(p.Theme.Accent).Render(text)
	return clipTo(row, width, lipgloss.NewStyle().Foreground(p.Theme.Subtle))
}

// code renders one row's tokens over the style the row is painted in. Every
// token takes only a foreground from it, so whatever is behind the row survives
// all the way across.
func (p Painter) code(tokens []syntax.Token, base lipgloss.Style) string {
	tab := strings.Repeat(" ", p.tabWidth())

	var b strings.Builder
	for _, t := range tokens {
		text := strings.ReplaceAll(t.Text, "\t", tab)
		if t.Color == nil {
			b.WriteString(base.Render(text))
			continue
		}
		b.WriteString(base.Foreground(t.Color).Render(text))
	}
	return b.String()
}

func (p Painter) tabWidth() int {
	if p.TabWidth <= 0 {
		return defaultTabWidth
	}
	return p.TabWidth
}

// background applies a color the theme may not define. A nil one leaves the
// terminal's own showing, which is what keeps a transparent one transparent.
func background(s lipgloss.Style, c color.Color) lipgloss.Style {
	if c == nil {
		return s
	}
	return s.Background(c)
}
