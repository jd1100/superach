package forms

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/moov-io/ach"

	"github.com/jd1100/superach/internal/achio"
)

func Addenda05Form(a *ach.Addenda05, save func()) fyne.CanvasObject {
	if a == nil {
		return widget.NewLabel("(no addenda)")
	}
	form := widget.NewForm(
		widget.NewFormItem("Payment Related Info", stringEntry(a.PaymentRelatedInformation, func(v string) { a.PaymentRelatedInformation = v })),
		widget.NewFormItem("Sequence #", intEntry(a.SequenceNumber, func(v int) { a.SequenceNumber = v })),
		widget.NewFormItem("Entry Detail Seq #", intEntry(a.EntryDetailSequenceNumber, func(v int) { a.EntryDetailSequenceNumber = v })),
	)
	form.OnSubmit = save
	form.SubmitText = "Save & Recalculate"
	return form
}

func Addenda98Form(a *ach.Addenda98, save func()) fyne.CanvasObject {
	if a == nil {
		return widget.NewLabel("(no addenda)")
	}
	codes := achio.AllChangeCodes()
	opts := make([]string, 0, len(codes))
	labels := map[string]string{}
	for _, c := range codes {
		label := c.Code
		if c.Reason != "" {
			label = fmt.Sprintf("%s — %s", c.Code, c.Reason)
		}
		opts = append(opts, label)
		labels[c.Code] = label
	}
	sel := widget.NewSelect(opts, func(s string) {
		if len(s) >= 3 {
			a.ChangeCode = s[:3]
		}
	})
	if l, ok := labels[a.ChangeCode]; ok {
		sel.SetSelected(l)
	}

	form := widget.NewForm(
		widget.NewFormItem("Change Code", sel),
		widget.NewFormItem("Original Trace #", stringEntry(a.OriginalTrace, func(v string) { a.OriginalTrace = v })),
		widget.NewFormItem("Original DFI", stringEntry(a.OriginalDFI, func(v string) { a.OriginalDFI = v })),
		widget.NewFormItem("Corrected Data (29)", stringEntry(a.CorrectedData, func(v string) { a.CorrectedData = v })),
		widget.NewFormItem("Trace Number", stringEntry(a.TraceNumber, func(v string) { a.TraceNumber = v })),
	)
	form.OnSubmit = save
	form.SubmitText = "Save & Recalculate"
	return form
}

func Addenda99Form(a *ach.Addenda99, save func()) fyne.CanvasObject {
	if a == nil {
		return widget.NewLabel("(no addenda)")
	}
	codes := achio.AllReturnCodes()
	opts := make([]string, 0, len(codes))
	labels := map[string]string{}
	for _, c := range codes {
		label := c.Code
		if c.Reason != "" {
			label = fmt.Sprintf("%s — %s", c.Code, c.Reason)
		}
		opts = append(opts, label)
		labels[c.Code] = label
	}
	sel := widget.NewSelect(opts, func(s string) {
		if len(s) >= 3 {
			a.ReturnCode = s[:3]
		}
	})
	if l, ok := labels[a.ReturnCode]; ok {
		sel.SetSelected(l)
	}

	form := widget.NewForm(
		widget.NewFormItem("Return Code", sel),
		widget.NewFormItem("Original Trace #", stringEntry(a.OriginalTrace, func(v string) { a.OriginalTrace = v })),
		widget.NewFormItem("Date of Death (YYMMDD)", stringEntry(a.DateOfDeath, func(v string) { a.DateOfDeath = v })),
		widget.NewFormItem("Original DFI", stringEntry(a.OriginalDFI, func(v string) { a.OriginalDFI = v })),
		widget.NewFormItem("Addenda Information", stringEntry(a.AddendaInformation, func(v string) { a.AddendaInformation = v })),
		widget.NewFormItem("Trace Number", stringEntry(a.TraceNumber, func(v string) { a.TraceNumber = v })),
	)
	form.OnSubmit = save
	form.SubmitText = "Save & Recalculate"
	return form
}
