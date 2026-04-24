package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// readableTheme wraps another Fyne theme and raises the contrast of disabled
// widgets. Read-only mode renders every ACH field as a disabled Entry/Select,
// and Fyne's default disabled foreground is a low-contrast gray that's hard
// to read — especially on the dark variant. We keep every other color from
// the wrapped theme untouched so user light/dark/system preferences still
// work.
type readableTheme struct{ base fyne.Theme }

func (t *readableTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameDisabled:
		// Match the normal foreground so read-only field text has the
		// same contrast as editable text.
		return t.base.Color(theme.ColorNameForeground, variant)
	case theme.ColorNameDisabledButton:
		// The disabled-entry background uses this color. Fall back to the
		// normal input background so a read-only Entry looks like a flat
		// label instead of a muddy gray rectangle.
		return t.base.Color(theme.ColorNameInputBackground, variant)
	}
	return t.base.Color(name, variant)
}

func (t *readableTheme) Font(style fyne.TextStyle) fyne.Resource { return t.base.Font(style) }
func (t *readableTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.base.Icon(name)
}
func (t *readableTheme) Size(name fyne.ThemeSizeName) float32 { return t.base.Size(name) }
