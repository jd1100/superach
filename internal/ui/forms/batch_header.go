package forms

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/moov-io/ach"
)

// serviceClassOptions excludes ADV on purpose: switching a batch's service
// class to ADV in place doesn't convert the underlying batch type, and the
// moov-io/ach library will then nil-pointer-deref on Save because the batch
// never initialised its ADVControl record. ADV batches must be created as
// ADV from the start via "New Batch…".
var serviceClassOptions = []string{
	strconv.Itoa(ach.MixedDebitsAndCredits) + " — Mixed",
	strconv.Itoa(ach.CreditsOnly) + " — Credits Only",
	strconv.Itoa(ach.DebitsOnly) + " — Debits Only",
}

// advServiceClassLabel is what we show when the header is already in the ADV
// service class (from a loaded file). It's read-only because the user cannot
// edit their way out — they'd need to remove the batch and add a new one.
const advServiceClassLabel = "200 — Automated Accounting Advice (ADV)"

// BatchHeaderForm binds a *BatchHeader to a Fyne form.
//
// SEC Code is deliberately rendered as a read-only field. It defines the
// batch type; editing it in place does not swap the underlying BatchPPD /
// BatchCCD / etc. struct and leaves the file in a shape the NACHA writer
// cannot serialise. To use a different SEC code, delete the batch and add
// a new one via File → New Batch.
func BatchHeaderForm(h *ach.BatchHeader, save func()) fyne.CanvasObject {
	if h == nil {
		return widget.NewLabel("(no batch header)")
	}

	var svc fyne.CanvasObject
	if h.ServiceClassCode == ach.AutomatedAccountingAdvices {
		svc = readOnlyLabel(advServiceClassLabel)
	} else {
		svc = selectField(labelForSvc(h.ServiceClassCode), serviceClassOptions, func(s string) {
			h.ServiceClassCode = svcFromLabel(s)
		})
	}

	form := widget.NewForm(
		widget.NewFormItem("Batch #", intEntry(h.BatchNumber, func(v int) { h.BatchNumber = v })),
		widget.NewFormItem("Service Class", svc),
		widget.NewFormItem("SEC Code (fixed)", readOnlyLabel(h.StandardEntryClassCode)),
		widget.NewFormItem("Company Name", stringEntry(h.CompanyName, func(v string) { h.CompanyName = v })),
		widget.NewFormItem("Company ID", stringEntry(h.CompanyIdentification, func(v string) { h.CompanyIdentification = v })),
		widget.NewFormItem("Discretionary Data", stringEntry(h.CompanyDiscretionaryData, func(v string) { h.CompanyDiscretionaryData = v })),
		widget.NewFormItem("Entry Description", stringEntry(h.CompanyEntryDescription, func(v string) { h.CompanyEntryDescription = v })),
		widget.NewFormItem("Descriptive Date", stringEntry(h.CompanyDescriptiveDate, func(v string) { h.CompanyDescriptiveDate = v })),
		widget.NewFormItem("Effective Entry Date (YYMMDD)", stringEntry(h.EffectiveEntryDate, func(v string) { h.EffectiveEntryDate = v })),
		widget.NewFormItem("ODFI Identification", stringEntry(h.ODFIIdentification, func(v string) { h.ODFIIdentification = v })),
	)
	attachSubmit(form, save)
	return form
}

func labelForSvc(code int) string {
	switch code {
	case ach.CreditsOnly:
		return serviceClassOptions[1]
	case ach.DebitsOnly:
		return serviceClassOptions[2]
	case ach.AutomatedAccountingAdvices:
		return advServiceClassLabel
	}
	return serviceClassOptions[0]
}

func svcFromLabel(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			if i == 0 {
				continue
			}
			n, _ := strconv.Atoi(s[:i])
			return n
		}
	}
	return ach.MixedDebitsAndCredits
}
