package forms

import (
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

func EntryDetailForm(e *ach.EntryDetail, save func()) fyne.CanvasObject {
	if e == nil {
		return widget.NewLabel("(no entry)")
	}
	txn := selectField(labelForTxn(e.TransactionCode), txnCodeOptions, func(s string) {
		e.TransactionCode = intFromLabel(s, e.TransactionCode)
	})
	form := widget.NewForm(
		widget.NewFormItem("Transaction Code", txn),
		widget.NewFormItem("RDFI Routing (8)", stringEntry(e.RDFIIdentification, func(v string) {
			e.RDFIIdentification = v
			if len(v) == 8 {
				if cd := ach.CalculateCheckDigit(v + "0"); cd >= 0 {
					e.CheckDigit = itoa(cd)
				}
			}
		})),
		widget.NewFormItem("Check Digit", stringEntry(e.CheckDigit, func(v string) { e.CheckDigit = v })),
		widget.NewFormItem("DFI Account #", stringEntry(e.DFIAccountNumber, func(v string) { e.DFIAccountNumber = v })),
		widget.NewFormItem("Amount ($)", amountEntry(e.Amount, func(v int) { e.Amount = v })),
		widget.NewFormItem("Identification #", stringEntry(e.IdentificationNumber, func(v string) { e.IdentificationNumber = v })),
		widget.NewFormItem("Individual Name", stringEntry(e.IndividualName, func(v string) { e.IndividualName = v })),
		widget.NewFormItem("Discretionary Data", stringEntry(e.DiscretionaryData, func(v string) { e.DiscretionaryData = v })),
		widget.NewFormItem("Trace Number", stringEntry(e.TraceNumber, func(v string) { e.TraceNumber = v })),
		widget.NewFormItem("Category", stringEntry(e.Category, func(v string) { e.Category = v })),
	)
	form.OnSubmit = save
	form.SubmitText = "Save & Recalculate"
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

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	buf := [12]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
