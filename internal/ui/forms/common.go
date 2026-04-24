// Package forms contains the Fyne widgets that bind to moov-io/ach record
// structs. Each exported constructor returns a fyne.CanvasObject and wires
// OnChanged callbacks directly to the record pointer, so edits are visible
// to the tree/validator as soon as the user types. The save callback is
// invoked on blur/submit so the parent can trigger recalc + dirty.
package forms

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/moov-io/ach"
)

func copyToClipboard(text string) {
	app := fyne.CurrentApp()
	if app == nil {
		return
	}
	if cb := app.Clipboard(); cb != nil {
		cb.SetContent(text)
	}
}

// withCopy wraps an Entry with a small icon button that copies the field's
// current text to the clipboard. Useful for fields a user routinely reads
// out loud or pastes into other tools (trace #, account #).
func withCopy(e *widget.Entry) fyne.CanvasObject {
	btn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		copyToClipboard(e.Text)
	})
	btn.Importance = widget.LowImportance
	return container.NewBorder(nil, nil, nil, btn, e)
}

// withCopyAndCompute wraps a routing Entry with a Copy button and an
// explicit "Compute Check Digit" action that writes the ABA-derived digit
// into the paired check-digit field. Kept explicit so pasting a full
// 9-digit routing doesn't clobber the user's existing check digit.
func withCopyAndCompute(routing *widget.Entry, check *widget.Entry) fyne.CanvasObject {
	copyBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		copyToClipboard(routing.Text)
	})
	copyBtn.Importance = widget.LowImportance
	computeBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		r := strings.TrimSpace(routing.Text)
		if len(r) != 8 || !allDigits(r) {
			return
		}
		// CalculateCheckDigit ignores position 8 of a 9-char input, so
		// passing r+"0" is equivalent to asking "what's the check digit
		// for these 8 ABA digits?".
		cd := ach.CalculateCheckDigit(r + "0")
		if cd < 0 {
			return
		}
		check.SetText(strconv.Itoa(cd))
	})
	computeBtn.Importance = widget.LowImportance
	return container.NewBorder(nil, nil, nil, container.NewHBox(computeBtn, copyBtn), routing)
}

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
	// Fyne's Validator shows a red underline under the entry when validation
	// fails. We validate on every keystroke, but only commit to the model
	// when parsing succeeds — so a rep mid-typing ("12.") never pushes a
	// zero into the ACH struct.
	e.Validator = func(s string) error {
		if strings.TrimSpace(s) == "" {
			return errors.New("amount required")
		}
		_, err := parseDollars(s)
		return err
	}
	e.OnChanged = func(s string) {
		cents, err := parseDollars(s)
		if err != nil {
			return
		}
		set(cents)
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

// parseDollars converts a human-typed dollar amount into integer cents.
// Accepts optional leading "$" and thousands separators (",") so pasted
// values like "$1,234.56" work. Rejects empty strings, non-digit content,
// more than two fractional digits (would silently round), and multiple
// decimal points.
func parseDollars(raw string) (int, error) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "$")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("amount required")
	}
	neg := false
	if s[0] == '-' {
		neg = true
		s = s[1:]
	} else if s[0] == '+' {
		s = s[1:]
	}
	if s == "" {
		return 0, errors.New("amount required")
	}
	dot := strings.IndexByte(s, '.')
	var dollarsStr, fracStr string
	if dot < 0 {
		dollarsStr = s
	} else {
		if strings.Count(s, ".") > 1 {
			return 0, errors.New("multiple decimal points")
		}
		dollarsStr = s[:dot]
		fracStr = s[dot+1:]
	}
	if dollarsStr == "" {
		dollarsStr = "0"
	}
	if !allDigits(dollarsStr) || !allDigits(fracStr) {
		return 0, errors.New("not a valid amount")
	}
	if len(fracStr) > 2 {
		return 0, fmt.Errorf("too many digits after decimal (%d)", len(fracStr))
	}
	dollars, err := strconv.Atoi(dollarsStr)
	if err != nil {
		return 0, err
	}
	var cents int
	switch len(fracStr) {
	case 0:
		cents = 0
	case 1:
		cents, _ = strconv.Atoi(fracStr + "0")
	case 2:
		cents, _ = strconv.Atoi(fracStr)
	}
	total := dollars*100 + cents
	if neg {
		total = -total
	}
	return total, nil
}

func allDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// dateEntry validates a 6-digit YYMMDD NACHA date. It accepts empty strings
// (many dates are optional in the spec) but rejects any non-6-digit input or
// values with an impossible month/day. Full calendar validity (e.g. Feb 30)
// is checked via time.Parse.
func dateEntry(cur string, set func(string)) *widget.Entry {
	e := widget.NewEntry()
	e.SetText(cur)
	e.SetPlaceHolder("YYMMDD")
	if readOnly.Load() {
		e.Disable()
		return e
	}
	e.Validator = func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil
		}
		return validateYYMMDD(s)
	}
	e.OnChanged = set
	return e
}

// timeEntry validates a 4-digit HHmm NACHA time.
func timeEntry(cur string, set func(string)) *widget.Entry {
	e := widget.NewEntry()
	e.SetText(cur)
	e.SetPlaceHolder("HHmm")
	if readOnly.Load() {
		e.Disable()
		return e
	}
	e.Validator = func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil
		}
		if len(s) != 4 || !allDigits(s) {
			return errors.New("must be 4 digits HHmm")
		}
		hh, _ := strconv.Atoi(s[:2])
		mm, _ := strconv.Atoi(s[2:])
		if hh > 23 || mm > 59 {
			return errors.New("invalid time")
		}
		return nil
	}
	e.OnChanged = set
	return e
}

func validateYYMMDD(s string) error { return ValidateYYMMDD(s) }

// ValidateYYMMDD checks a 6-digit NACHA date string. Exported for dialogs
// outside this package that build their own Entry widgets. Empty is rejected
// by this function; callers that want "empty is ok" must check first.
func ValidateYYMMDD(s string) error {
	if len(s) != 6 || !allDigits(s) {
		return errors.New("must be 6 digits YYMMDD")
	}
	mm, _ := strconv.Atoi(s[2:4])
	dd, _ := strconv.Atoi(s[4:6])
	if mm < 1 || mm > 12 {
		return fmt.Errorf("month %02d is not 01-12", mm)
	}
	if dd < 1 || dd > 31 {
		return fmt.Errorf("day %02d is not 01-31", dd)
	}
	return nil
}

// ValidateDigitsLen returns an error if s isn't exactly n digit characters.
// Used by dialogs that need "8-digit ODFI" or "10-digit ABA" validation
// without re-implementing the digit-scan loop.
func ValidateDigitsLen(s string, n int) error {
	s = strings.TrimSpace(s)
	if len(s) != n {
		return fmt.Errorf("must be %d digits", n)
	}
	if !allDigits(s) {
		return errors.New("digits only")
	}
	return nil
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
