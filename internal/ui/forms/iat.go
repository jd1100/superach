package forms

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/moov-io/ach"
)

var fxIndicatorOptions = []string{
	"FF — Fixed-to-Fixed",
	"FV — Fixed-to-Variable",
	"VF — Variable-to-Fixed",
}

func IATBatchHeaderForm(h *ach.IATBatchHeader, save func()) fyne.CanvasObject {
	if h == nil {
		return widget.NewLabel("(no iat batch header)")
	}
	svc := selectField(labelForSvc(h.ServiceClassCode), serviceClassOptions, func(s string) {
		h.ServiceClassCode = svcFromLabel(s)
	})
	fx := selectField(labelForFX(h.ForeignExchangeIndicator), fxIndicatorOptions, func(s string) {
		if len(s) >= 2 {
			h.ForeignExchangeIndicator = s[:2]
		}
	})
	form := widget.NewForm(
		widget.NewFormItem("Batch #", intEntry(h.BatchNumber, func(v int) { h.BatchNumber = v })),
		widget.NewFormItem("Service Class", svc),
		widget.NewFormItem("IAT Indicator", stringEntry(h.IATIndicator, func(v string) { h.IATIndicator = v })),
		widget.NewFormItem("FX Indicator", fx),
		widget.NewFormItem("FX Reference Qualifier", intEntry(h.ForeignExchangeReferenceIndicator, func(v int) { h.ForeignExchangeReferenceIndicator = v })),
		widget.NewFormItem("FX Reference", stringEntry(h.ForeignExchangeReference, func(v string) { h.ForeignExchangeReference = v })),
		widget.NewFormItem("Destination Country (ISO)", stringEntry(h.ISODestinationCountryCode, func(v string) { h.ISODestinationCountryCode = v })),
		widget.NewFormItem("Originator ID", stringEntry(h.OriginatorIdentification, func(v string) { h.OriginatorIdentification = v })),
		widget.NewFormItem("SEC Code", stringEntry(h.StandardEntryClassCode, func(v string) { h.StandardEntryClassCode = v })),
		widget.NewFormItem("Company Entry Description", stringEntry(h.CompanyEntryDescription, func(v string) { h.CompanyEntryDescription = v })),
		widget.NewFormItem("Originating Currency (ISO)", stringEntry(h.ISOOriginatingCurrencyCode, func(v string) { h.ISOOriginatingCurrencyCode = v })),
		widget.NewFormItem("Destination Currency (ISO)", stringEntry(h.ISODestinationCurrencyCode, func(v string) { h.ISODestinationCurrencyCode = v })),
		widget.NewFormItem("Effective Entry Date (YYMMDD)", stringEntry(h.EffectiveEntryDate, func(v string) { h.EffectiveEntryDate = v })),
		widget.NewFormItem("ODFI Identification", stringEntry(h.ODFIIdentification, func(v string) { h.ODFIIdentification = v })),
	)
	form.OnSubmit = save
	form.SubmitText = "Save & Recalculate"
	return form
}

func labelForFX(code string) string {
	for _, o := range fxIndicatorOptions {
		if len(o) >= 2 && o[:2] == code {
			return o
		}
	}
	return ""
}

func IATEntryDetailForm(e *ach.IATEntryDetail, save func()) fyne.CanvasObject {
	if e == nil {
		return widget.NewLabel("(no iat entry)")
	}
	form := widget.NewForm(
		widget.NewFormItem("Transaction Code", intEntry(e.TransactionCode, func(v int) { e.TransactionCode = v })),
		widget.NewFormItem("RDFI Routing (8)", stringEntry(e.RDFIIdentification, func(v string) { e.RDFIIdentification = v })),
		widget.NewFormItem("Check Digit", stringEntry(e.CheckDigit, func(v string) { e.CheckDigit = v })),
		widget.NewFormItem("Addenda Records", intEntry(e.AddendaRecords, func(v int) { e.AddendaRecords = v })),
		widget.NewFormItem("Amount ($)", amountEntry(e.Amount, func(v int) { e.Amount = v })),
		widget.NewFormItem("DFI Account #", stringEntry(e.DFIAccountNumber, func(v string) { e.DFIAccountNumber = v })),
		widget.NewFormItem("OFAC Indicator", stringEntry(e.OFACScreeningIndicator, func(v string) { e.OFACScreeningIndicator = v })),
		widget.NewFormItem("Secondary OFAC Indicator", stringEntry(e.SecondaryOFACScreeningIndicator, func(v string) { e.SecondaryOFACScreeningIndicator = v })),
		widget.NewFormItem("Trace Number", stringEntry(e.TraceNumber, func(v string) { e.TraceNumber = v })),
		widget.NewFormItem("Category", stringEntry(e.Category, func(v string) { e.Category = v })),
	)
	form.OnSubmit = save
	form.SubmitText = "Save & Recalculate"
	return form
}

// IATAddendaForm dispatches to the right addenda-specific form based on type.
func IATAddendaForm(e *ach.IATEntryDetail, typeCode string, idx int, save func()) fyne.CanvasObject {
	switch typeCode {
	case "10":
		return addenda10Form(e.Addenda10, save)
	case "11":
		return addenda11Form(e.Addenda11, save)
	case "12":
		return addenda12Form(e.Addenda12, save)
	case "13":
		return addenda13Form(e.Addenda13, save)
	case "14":
		return addenda14Form(e.Addenda14, save)
	case "15":
		return addenda15Form(e.Addenda15, save)
	case "16":
		return addenda16Form(e.Addenda16, save)
	case "17":
		if idx < len(e.Addenda17) {
			return addenda17Form(e.Addenda17[idx], save)
		}
	case "18":
		if idx < len(e.Addenda18) {
			return addenda18Form(e.Addenda18[idx], save)
		}
	case "98":
		return Addenda98Form(e.Addenda98, save)
	case "99":
		return Addenda99Form(e.Addenda99, save)
	}
	return widget.NewLabel(fmt.Sprintf("(unknown addenda %s)", typeCode))
}

func addenda10Form(a *ach.Addenda10, save func()) fyne.CanvasObject {
	if a == nil {
		return widget.NewLabel("(no addenda10)")
	}
	form := widget.NewForm(
		widget.NewFormItem("Transaction Type Code", stringEntry(a.TransactionTypeCode, func(v string) { a.TransactionTypeCode = v })),
		widget.NewFormItem("Foreign Payment Amount", intEntry(a.ForeignPaymentAmount, func(v int) { a.ForeignPaymentAmount = v })),
		widget.NewFormItem("Foreign Trace Number", stringEntry(a.ForeignTraceNumber, func(v string) { a.ForeignTraceNumber = v })),
		widget.NewFormItem("Name", stringEntry(a.Name, func(v string) { a.Name = v })),
	)
	form.OnSubmit = save
	form.SubmitText = "Save & Recalculate"
	return form
}

func addenda11Form(a *ach.Addenda11, save func()) fyne.CanvasObject {
	if a == nil {
		return widget.NewLabel("(no addenda11)")
	}
	form := widget.NewForm(
		widget.NewFormItem("Originator Name", stringEntry(a.OriginatorName, func(v string) { a.OriginatorName = v })),
		widget.NewFormItem("Originator Street", stringEntry(a.OriginatorStreetAddress, func(v string) { a.OriginatorStreetAddress = v })),
	)
	form.OnSubmit = save
	form.SubmitText = "Save & Recalculate"
	return form
}

func addenda12Form(a *ach.Addenda12, save func()) fyne.CanvasObject {
	if a == nil {
		return widget.NewLabel("(no addenda12)")
	}
	form := widget.NewForm(
		widget.NewFormItem("Originator City/State/Province", stringEntry(a.OriginatorCityStateProvince, func(v string) { a.OriginatorCityStateProvince = v })),
		widget.NewFormItem("Originator Country/Postal", stringEntry(a.OriginatorCountryPostalCode, func(v string) { a.OriginatorCountryPostalCode = v })),
	)
	form.OnSubmit = save
	form.SubmitText = "Save & Recalculate"
	return form
}

func addenda13Form(a *ach.Addenda13, save func()) fyne.CanvasObject {
	if a == nil {
		return widget.NewLabel("(no addenda13)")
	}
	form := widget.NewForm(
		widget.NewFormItem("ODFI Name", stringEntry(a.ODFIName, func(v string) { a.ODFIName = v })),
		widget.NewFormItem("ODFI ID Qualifier", stringEntry(a.ODFIIDNumberQualifier, func(v string) { a.ODFIIDNumberQualifier = v })),
		widget.NewFormItem("ODFI Identification", stringEntry(a.ODFIIdentification, func(v string) { a.ODFIIdentification = v })),
		widget.NewFormItem("ODFI Branch Country", stringEntry(a.ODFIBranchCountryCode, func(v string) { a.ODFIBranchCountryCode = v })),
	)
	form.OnSubmit = save
	form.SubmitText = "Save & Recalculate"
	return form
}

func addenda14Form(a *ach.Addenda14, save func()) fyne.CanvasObject {
	if a == nil {
		return widget.NewLabel("(no addenda14)")
	}
	form := widget.NewForm(
		widget.NewFormItem("RDFI Name", stringEntry(a.RDFIName, func(v string) { a.RDFIName = v })),
		widget.NewFormItem("RDFI ID Qualifier", stringEntry(a.RDFIIDNumberQualifier, func(v string) { a.RDFIIDNumberQualifier = v })),
		widget.NewFormItem("RDFI Identification", stringEntry(a.RDFIIdentification, func(v string) { a.RDFIIdentification = v })),
		widget.NewFormItem("RDFI Branch Country", stringEntry(a.RDFIBranchCountryCode, func(v string) { a.RDFIBranchCountryCode = v })),
	)
	form.OnSubmit = save
	form.SubmitText = "Save & Recalculate"
	return form
}

func addenda15Form(a *ach.Addenda15, save func()) fyne.CanvasObject {
	if a == nil {
		return widget.NewLabel("(no addenda15)")
	}
	form := widget.NewForm(
		widget.NewFormItem("Receiver ID #", stringEntry(a.ReceiverIDNumber, func(v string) { a.ReceiverIDNumber = v })),
		widget.NewFormItem("Receiver Street", stringEntry(a.ReceiverStreetAddress, func(v string) { a.ReceiverStreetAddress = v })),
	)
	form.OnSubmit = save
	form.SubmitText = "Save & Recalculate"
	return form
}

func addenda16Form(a *ach.Addenda16, save func()) fyne.CanvasObject {
	if a == nil {
		return widget.NewLabel("(no addenda16)")
	}
	form := widget.NewForm(
		widget.NewFormItem("Receiver City/State/Province", stringEntry(a.ReceiverCityStateProvince, func(v string) { a.ReceiverCityStateProvince = v })),
		widget.NewFormItem("Receiver Country/Postal", stringEntry(a.ReceiverCountryPostalCode, func(v string) { a.ReceiverCountryPostalCode = v })),
	)
	form.OnSubmit = save
	form.SubmitText = "Save & Recalculate"
	return form
}

func addenda17Form(a *ach.Addenda17, save func()) fyne.CanvasObject {
	if a == nil {
		return widget.NewLabel("(no addenda17)")
	}
	form := widget.NewForm(
		widget.NewFormItem("Payment Related Info", stringEntry(a.PaymentRelatedInformation, func(v string) { a.PaymentRelatedInformation = v })),
		widget.NewFormItem("Sequence #", intEntry(a.SequenceNumber, func(v int) { a.SequenceNumber = v })),
	)
	form.OnSubmit = save
	form.SubmitText = "Save & Recalculate"
	return form
}

func addenda18Form(a *ach.Addenda18, save func()) fyne.CanvasObject {
	if a == nil {
		return widget.NewLabel("(no addenda18)")
	}
	form := widget.NewForm(
		widget.NewFormItem("Foreign Corresp. Bank Name", stringEntry(a.ForeignCorrespondentBankName, func(v string) { a.ForeignCorrespondentBankName = v })),
		widget.NewFormItem("Bank ID Qualifier", stringEntry(a.ForeignCorrespondentBankIDNumberQualifier, func(v string) { a.ForeignCorrespondentBankIDNumberQualifier = v })),
		widget.NewFormItem("Bank ID Number", stringEntry(a.ForeignCorrespondentBankIDNumber, func(v string) { a.ForeignCorrespondentBankIDNumber = v })),
		widget.NewFormItem("Bank Branch Country", stringEntry(a.ForeignCorrespondentBankBranchCountryCode, func(v string) { a.ForeignCorrespondentBankBranchCountryCode = v })),
		widget.NewFormItem("Sequence #", intEntry(a.SequenceNumber, func(v int) { a.SequenceNumber = v })),
	)
	form.OnSubmit = save
	form.SubmitText = "Save & Recalculate"
	return form
}
