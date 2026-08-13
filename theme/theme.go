// Package theme holds the color palettes the UI styles from. Nothing in the
// TUI hardcodes a color: a color that isn't here means this struct needs a
// field.
package theme

import (
	"image/color"
	"sort"

	"charm.land/lipgloss/v2"
)

// Theme is one palette. Optional fields are nil-able and have accessors that
// fall back, so adding a field doesn't force every theme to be rewritten.
type Theme struct {
	Name string

	// Syntax names the Chroma style code is highlighted with. Chroma ships its
	// own palettes and a diff needs far more token colors than the chrome has
	// fields, so a theme points at the one that matches rather than restating
	// it. Empty falls back to Chroma's own default.
	Syntax string

	// Text, named for weight rather than rank, so a style says how loud it is
	// instead of where it came in a list. Subtle is text still meant to be
	// read; Muted is text there to be looked past.
	//
	// Text, Subtle and Muted are Rosé Pine's own roles at its own values.
	// Accent is this UI's name for the one color it highlights with, which in
	// that palette is iris.
	Text     color.Color
	Accent   color.Color
	Subtle   color.Color
	Muted    color.Color
	Inverted color.Color

	// Semantic
	Success color.Color
	Warning color.Color
	Error   color.Color
	Actor   color.Color

	// Surfaces. A nil Background means "leave the terminal's own background
	// alone", which is what keeps transparency working.
	Background         color.Color
	SelectedBackground color.Color

	// Diff surfaces. A changed line is read as a block, not a character at a
	// time, and a marker column alone does not carry that. They are tints of
	// Success and Error over the base rather than the colors themselves: a
	// filled row at full strength buries the code sitting on it.
	AddedBackground   color.Color
	RemovedBackground color.Color

	// Borders
	Border       color.Color
	BorderSubtle color.Color
	BorderMuted  color.Color
}

// InvertedOrText is the text color to use on top of a filled surface.
func (t Theme) InvertedOrText() color.Color {
	if t.Inverted != nil {
		return t.Inverted
	}
	return t.Text
}

// BorderSubtleOrBorder falls back for themes that define one border color.
func (t Theme) BorderSubtleOrBorder() color.Color {
	if t.BorderSubtle != nil {
		return t.BorderSubtle
	}
	return t.Border
}

// BorderMutedOrSubtle falls back through the border ladder.
func (t Theme) BorderMutedOrSubtle() color.Color {
	if t.BorderMuted != nil {
		return t.BorderMuted
	}
	return t.BorderSubtleOrBorder()
}

// RosePineMoon is the default. The text, semantic and border colors are Rosé
// Pine Moon's own values. The two diff tints are not: the palette has no role
// for a changed row, so they are mixed here.
var RosePineMoon = Theme{
	Name:               "rose-pine-moon",
	Syntax:             "rose-pine-moon",
	Text:               lipgloss.Color("#e0def4"),
	Accent:             lipgloss.Color("#c4a7e7"),
	Subtle:             lipgloss.Color("#908caa"),
	Muted:              lipgloss.Color("#6e6a86"),
	Inverted:           lipgloss.Color("#232136"),
	Success:            lipgloss.Color("#9ccfd8"),
	Warning:            lipgloss.Color("#f6c177"),
	Error:              lipgloss.Color("#eb6f92"),
	Actor:              lipgloss.Color("#ea9a97"),
	Background:         nil,
	SelectedBackground: lipgloss.Color("#2a283e"),
	AddedBackground:    lipgloss.Color("#26383c"),
	RemovedBackground:  lipgloss.Color("#3c2635"),
	Border:             lipgloss.Color("#56526e"),
	BorderSubtle:       lipgloss.Color("#44415a"),
	BorderMuted:        lipgloss.Color("#393552"),
}

// Default names the theme used when config asks for one that doesn't exist.
const Default = "rose-pine-moon"

var registry = map[string]Theme{
	RosePineMoon.Name: RosePineMoon,
}

// Get returns the named theme. An unknown name yields the default and false,
// so a typo in config degrades to a working UI instead of a crash.
func Get(name string) (Theme, bool) {
	t, ok := registry[name]
	if !ok {
		return registry[Default], false
	}
	return t, true
}

// Names lists the registered themes in a stable order.
func Names() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
