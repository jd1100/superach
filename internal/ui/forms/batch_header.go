package forms

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/moov-io/ach"
)

var serviceClassOptions = []string{
	strconv.Itoa(ach.MixedDebitsAndCredits) + " — Mixed",
	strconv.Itoa(ach.CreditsOnly) + " — Credits Only",
	strconv.Itoa(ach.DebitsOnly) + " — Debits Only",
	strconv.Itoa(ach.AutomatedAccountingAdvices) + " — ADV",
}

var secOptions = []string{
	ach.ACK, ach.ADV, ach.ARC, ach.ATX, ach.BOC, ach.CCD, ach.CIE, ach.COR,
	ach.CTX, ach.DNE, ach.ENR, ach.IAT, ach.MTE, ach.POP, ach.POS, ach.PPD,
	ach.RCK, ach.SHR, ach.TEL, ach.TRC, ach.TRX, ach.WEB, ach.XCK,
}

// BatchHeaderForm binds a *BatchHeader to a Fyne form.
func BatchHeaderForm(h *ach.BatchHeader, save func()) fyne.CanvasObject {
	if h == nil {
		return widget.NewLabel("(no batch header)")
	}
	svc := selectField(labelForSvc(h.ServiceClassCode), serviceClassOptions, func(s string) {
		h.ServiceClassCode = svcFromLabel(s)
	})
	sec := selectField(h.StandardEntryClassCode, secOptions, func(s string) {
		h.StandardEntryClassCode = s
	})
	form := widget.NewForm(
		widget.NewFormItem("Batch #", intEntry(h.BatchNumber, func(v int) { h.BatchNumber = v })),
		widget.NewFormItem("Service Class", svc),
		widget.NewFormItem("SEC Code", sec),
		widget.NewFormItem("Company Name", stringEntry(h.CompanyName, func(v string) { h.CompanyName = v })),
		widget.NewFormItem("Company ID", stringEntry(h.CompanyIdentification, func(v string) { h.CompanyIdentification = v })),
		widget.NewFormItem("Discretionary Data", stringEntry(h.CompanyDiscretionaryData, func(v string) { h.CompanyDiscretionaryData = v })),
		widget.NewFormItem("Entry Description", stringEntry(h.CompanyEntryDescription, func(v string) { h.CompanyEntryDescription = v })),
		widget.NewFormItem("Descriptive Date", stringEntry(h.CompanyDescriptiveDate, func(v string) { h.CompanyDescriptiveDate = v })),
		widget.NewFormItem("Effective Entry Date (YYMMDD)", stringEntry(h.EffectiveEntryDate, func(v string) { h.EffectiveEntryDate = v })),
		widget.NewFormItem("ODFI Identification", stringEntry(h.ODFIIdentification, func(v string) { h.ODFIIdentification = v })),
	)
	form.OnSubmit = save
	form.SubmitText = "Save & Recalculate"
	return form
}

func labelForSvc(code int) string {
	switch code {
	case ach.CreditsOnly:
		return serviceClassOptions[1]
	case ach.DebitsOnly:
		return serviceClassOptions[2]
	case ach.AutomatedAccountingAdvices:
		return serviceClassOptions[3]
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
