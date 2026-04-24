package achio

import (
	"fmt"
	"strings"

	"github.com/moov-io/ach"
)

// NOCRequest describes a Notification of Change to be created from an
// existing forward entry. The non-empty Corrected* fields drive the
// CorrectedData payload format expected for the chosen ChangeCode.
type NOCRequest struct {
	OriginalBatchIndex int
	OriginalEntryIndex int
	ChangeCode         string
	Corrected          ach.CorrectedData
}

// BuildNOC creates a COR batch (or appends to an existing one) with an
// Addenda98 carrying the corrected data.
func BuildNOC(f *ach.File, req NOCRequest) (int, error) {
	if f == nil {
		return 0, fmt.Errorf("nil file")
	}
	if req.OriginalBatchIndex < 0 || req.OriginalBatchIndex >= len(f.Batches) {
		return 0, fmt.Errorf("original batch index out of range")
	}
	srcBatch := f.Batches[req.OriginalBatchIndex]
	srcHeader := srcBatch.GetHeader()
	if srcHeader == nil {
		return 0, fmt.Errorf("source batch missing header")
	}
	srcEntries := srcBatch.GetEntries()
	if req.OriginalEntryIndex < 0 || req.OriginalEntryIndex >= len(srcEntries) {
		return 0, fmt.Errorf("original entry index out of range")
	}
	if !ValidChangeCode(req.ChangeCode) {
		return 0, fmt.Errorf("unknown change code %q", req.ChangeCode)
	}
	srcEntry := srcEntries[req.OriginalEntryIndex]

	corBatch, corIdx := findOrCreateCORBatch(f, srcHeader)

	ne := ach.NewEntryDetail()
	ne.TransactionCode = nocTransactionCode(srcEntry.TransactionCode)
	ne.RDFIIdentification = srcEntry.RDFIIdentification
	ne.CheckDigit = srcEntry.CheckDigit
	ne.DFIAccountNumber = srcEntry.DFIAccountNumber
	ne.Amount = 0 // COR entries carry zero amount
	ne.IdentificationNumber = srcEntry.IdentificationNumber
	ne.IndividualName = srcEntry.IndividualName
	ne.SetTraceNumber(corBatch.GetHeader().ODFIIdentification, len(corBatch.GetEntries())+1)
	ne.AddendaRecordIndicator = 1
	ne.Category = ach.CategoryNOC

	a98 := ach.NewAddenda98()
	a98.ChangeCode = strings.ToUpper(req.ChangeCode)
	a98.OriginalTrace = srcEntry.TraceNumber
	odfi := srcHeader.ODFIIdentification
	if len(odfi) > 8 {
		odfi = odfi[:8]
	}
	a98.OriginalDFI = odfi
	a98.CorrectedData = ach.WriteCorrectionData(strings.ToUpper(req.ChangeCode), &req.Corrected)
	ne.Addenda98 = a98

	corBatch.AddEntry(ne)
	if err := corBatch.Create(); err != nil {
		return 0, fmt.Errorf("create COR batch: %w", err)
	}
	if err := f.Create(); err != nil {
		return 0, fmt.Errorf("rebuild file: %w", err)
	}
	return corIdx, nil
}

func findOrCreateCORBatch(f *ach.File, srcHeader *ach.BatchHeader) (ach.Batcher, int) {
	for i, b := range f.Batches {
		if h := b.GetHeader(); h != nil && h.StandardEntryClassCode == ach.COR {
			return b, i
		}
	}
	bh := ach.NewBatchHeader()
	bh.ServiceClassCode = ach.MixedDebitsAndCredits
	bh.StandardEntryClassCode = ach.COR
	bh.CompanyName = srcHeader.CompanyName
	bh.CompanyIdentification = srcHeader.CompanyIdentification
	bh.CompanyEntryDescription = "NOC"
	bh.EffectiveEntryDate = srcHeader.EffectiveEntryDate
	bh.ODFIIdentification = srcHeader.ODFIIdentification
	bh.OriginatorStatusCode = srcHeader.OriginatorStatusCode
	bh.BatchNumber = nextBatchNumber(f)

	b, err := ach.NewBatch(bh)
	if err != nil {
		return nil, -1
	}
	f.AddBatch(b)
	return b, len(f.Batches) - 1
}

// ValidChangeCode reports whether code is a known NACHA change code.
func ValidChangeCode(code string) bool {
	return ach.LookupChangeCode(strings.ToUpper(code)) != nil
}

// ChangeCodeOption represents one row in the change-code dropdown.
type ChangeCodeOption struct {
	Code        string
	Reason      string
	Description string
}

// nocTransactionCode maps a forward TransactionCode to the matching
// Return/NOC code expected inside a COR batch.
func nocTransactionCode(forward int) int {
	switch forward {
	case ach.CheckingCredit, ach.CheckingPrenoteCredit, ach.CheckingZeroDollarRemittanceCredit:
		return ach.CheckingReturnNOCCredit
	case ach.CheckingDebit, ach.CheckingPrenoteDebit, ach.CheckingZeroDollarRemittanceDebit:
		return ach.CheckingReturnNOCDebit
	case ach.SavingsCredit, ach.SavingsPrenoteCredit, ach.SavingsZeroDollarRemittanceCredit:
		return ach.SavingsReturnNOCCredit
	case ach.SavingsDebit, ach.SavingsPrenoteDebit, ach.SavingsZeroDollarRemittanceDebit:
		return ach.SavingsReturnNOCDebit
	case ach.GLCredit, ach.GLPrenoteCredit, ach.GLZeroDollarRemittanceCredit:
		return ach.GLReturnNOCCredit
	case ach.GLDebit, ach.GLPrenoteDebit, ach.GLZeroDollarRemittanceDebit:
		return ach.GLReturnNOCDebit
	case ach.LoanCredit, ach.LoanPrenoteCredit, ach.LoanZeroDollarRemittanceCredit:
		return ach.LoanReturnNOCCredit
	case ach.LoanDebit:
		return ach.LoanReturnNOCDebit
	}
	return forward
}

// AllChangeCodes returns C01..C14 with descriptions from moov-io/ach.
func AllChangeCodes() []ChangeCodeOption {
	out := make([]ChangeCodeOption, 0, 14)
	for n := 1; n <= 14; n++ {
		code := fmt.Sprintf("C%02d", n)
		opt := ChangeCodeOption{Code: code}
		if cc := ach.LookupChangeCode(code); cc != nil {
			opt.Reason = cc.Reason
			opt.Description = cc.Description
		}
		out = append(out, opt)
	}
	return out
}
