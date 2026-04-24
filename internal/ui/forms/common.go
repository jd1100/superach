// Package forms contains the Fyne widgets that bind to moov-io/ach record
// structs. Each exported constructor returns a fyne.CanvasObject and wires
// OnChanged callbacks directly to the record pointer, so edits are visible
// to the tree/validator as soon as the user types. The save callback is
// invoked on blur/submit so the parent can trigger recalc + dirty.
package forms

import (
	"strconv"
	"sync/atomic"

	"fyne.io/fyne/v2/widget"
)

// readOnly is a package-level flag toggled by the UI shell before rendering
// a form. When true, all entry/select widgets come up disabled and their
// OnChanged callbacks are skipped. This is the app's "safe view" mode.
var readOnly atomic.Bool

// SetReadOnly controls whether newly constructed form widgets accept edits.
// Call this *before* building a form; existing widgets are not retroactively
// toggled.
func SetReadOnly(v bool) { readOnly.Store(v) }

// IsReadOnly reports the current read-only state.
func IsReadOnly() bool { return readOnly.Load() }

func stringEntry(cur string, set func(string)) *widget.Entry {
	e := widget.NewEntry()
	e.SetText(cur)
	if readOnly.Load() {
		e.Disable()
		return e
	}
	e.OnChanged = set
	return e
}

func intEntry(cur int, set func(int)) *widget.Entry {
	e := widget.NewEntry()
	e.SetText(strconv.Itoa(cur))
	if readOnly.Load() {
		e.Disable()
		return e
	}
	e.OnChanged = func(s string) {
		if s == "" {
			set(0)
			return
		}
		if n, err := strconv.Atoi(s); err == nil {
			set(n)
		}
	}
	return e
}

func amountEntry(cur int, set func(int)) *widget.Entry {
	e := widget.NewEntry()
	e.SetText(centsToDollars(cur))
	if readOnly.Load() {
		e.Disable()
		return e
	}
	e.OnChanged = func(s string) {
		set(dollarsToCents(s))
	}
	return e
}

func centsToDollars(c int) string {
	neg := ""
	if c < 0 {
		neg = "-"
		c = -c
	}
	return neg + strconv.Itoa(c/100) + "." + pad2(c%100)
}

func pad2(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

func dollarsToCents(s string) int {
	neg := false
	if len(s) > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
	}
	dot := -1
	for i, c := range s {
		if c == '.' {
			dot = i
			break
		}
	}
	var dollars, cents int
	if dot < 0 {
		dollars, _ = strconv.Atoi(s)
	} else {
		dollars, _ = strconv.Atoi(s[:dot])
		frac := s[dot+1:]
		if len(frac) >= 2 {
			cents, _ = strconv.Atoi(frac[:2])
		} else if len(frac) == 1 {
			cents, _ = strconv.Atoi(frac + "0")
		}
	}
	total := dollars*100 + cents
	if neg {
		total = -total
	}
	return total
}

func selectField(cur string, options []string, set func(string)) *widget.Select {
	sel := widget.NewSelect(options, set)
	sel.SetSelected(cur)
	if readOnly.Load() {
		sel.OnChanged = nil
		sel.Disable()
	}
	return sel
}

// readOnlyLabel returns a disabled Entry prefilled with text. Use this for
// fields that are never editable regardless of mode (e.g. SEC Code, which
// defines the batch type and cannot be changed in place).
func readOnlyLabel(text string) *widget.Entry {
	e := widget.NewEntry()
	e.SetText(text)
	e.Disable()
	return e
}

// attachSubmit wires the "Save & Recalculate" button on a form, unless the
// form package is in read-only mode — in which case the submit button is
// suppressed entirely so the user cannot accidentally mark the file dirty.
func attachSubmit(form *widget.Form, save func()) {
	if readOnly.Load() {
		return
	}
	form.OnSubmit = save
	form.SubmitText = "Save & Recalculate"
}
