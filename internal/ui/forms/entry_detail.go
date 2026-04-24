package forms

import (
	"errors"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/moov-io/ach"
)

var txnCodeOptions = []string{
	"22 — Checking Credit",
	"27 — Checking Debit",
	"23 — Checking Prenote Credit",
	"28 — Checking Prenote Debit",
	"32 — Savings Credit",
	"37 — Savings Debit",
	"33 — Savings Prenote Credit",
	"38 — Savings Prenote Debit",
}

// routingEntry wires the RDFI routing field with strict 8-digit validation.
// A separate compute button (see withCopyAndCompute) fills the paired check
// digit explicitly, so a freshly-typed 8-digit routing never clobbers a
// user-typed check digit automatically.
func routingEntry(cur string, set func(string)) *widget.Entry {
	e := widget.NewEntry()
	e.SetText(cur)
	if readOnly.Load() {
		e.Disable()
		return e
	}
	e.Validator = func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" {
			return errors.New("routing required")
		}
		if !allDigits(s) {
			return errors.New("digits only")
		}
		if len(s) != 8 {
			return errors.New("must be 8 digits")
		}
		return nil
	}
	e.OnChanged = set
	return e
}

// checkDigitEntry flags an ABA checksum mismatch between the user's routing
// number and the check digit. Validation is silent until both fields are
// filled with a plausible shape — we don't want red underlines while the
// user is mid-typing.
func checkDigitEntry(cur string, getRouting func() string, set func(string)) *widget.Entry {
	e := widget.NewEntry()
	e.SetText(cur)
	if readOnly.Load() {
		e.Disable()
		return e
	}
	e.Validator = func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" || !allDigits(s) || len(s) != 1 {
			return errors.New("one digit")
		}
		r := strings.TrimSpace(getRouting())
		if len(r) != 8 || !allDigits(r) {
			return nil
		}
		if err := ach.CheckRoutingNumber(r + s); err != nil {
			return errors.New("ABA checksum mismatch")
		}
		return nil
	}
	e.OnChanged = set
	return e
}

func EntryDetailForm(e *ach.EntryDetail, save func()) fyne.CanvasObject {
	if e == nil {
		return widget.NewLabel("(no entry)")
	}
	txn := selectField(labelForTxn(e.TransactionCode), txnCodeOptions, func(s string) {
		e.TransactionCode = intFromLabel(s, e.TransactionCode)
	})
	routing := routingEntry(e.RDFIIdentification, func(v string) { e.RDFIIdentification = v })
	checkDigit := checkDigitEntry(e.CheckDigit, func() string { return e.RDFIIdentification }, func(v string) { e.CheckDigit = v })
	form := widget.NewForm(
		widget.NewFormItem("Transaction Code", txn),
		widget.NewFormItem("RDFI Routing (8)", withCopyAndCompute(routing, checkDigit)),
		widget.NewFormItem("Check Digit", checkDigit),
		widget.NewFormItem("DFI Account #", withCopy(stringEntry(e.DFIAccountNumber, func(v string) { e.DFIAccountNumber = v }))),
		widget.NewFormItem("Amount ($)", amountEntry(e.Amount, func(v int) { e.Amount = v })),
		widget.NewFormItem("Identification #", stringEntry(e.IdentificationNumber, func(v string) { e.IdentificationNumber = v })),
		widget.NewFormItem("Individual Name", stringEntry(e.IndividualName, func(v string) { e.IndividualName = v })),
		widget.NewFormItem("Discretionary Data", stringEntry(e.DiscretionaryData, func(v string) { e.DiscretionaryData = v })),
		widget.NewFormItem("Trace Number", withCopy(stringEntry(e.TraceNumber, func(v string) { e.TraceNumber = v }))),
		widget.NewFormItem("Category", stringEntry(e.Category, func(v string) { e.Category = v })),
	)
	attachSubmit(form, save)
	return form
}

func labelForTxn(code int) string {
	for _, o := range txnCodeOptions {
		if o[0] == byte('0'+code/10) && o[1] == byte('0'+code%10) {
			return o
		}
	}
	return ""
}

func intFromLabel(s string, fallback int) int {
	if len(s) < 2 {
		return fallback
	}
	d1 := int(s[0] - '0')
	d2 := int(s[1] - '0')
	if d1 < 0 || d1 > 9 || d2 < 0 || d2 > 9 {
		return fallback
	}
	return d1*10 + d2
}
