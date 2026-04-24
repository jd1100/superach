package achio

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/moov-io/ach"
)

// ReturnRequest describes a return entry to be created from an existing
// forward entry.
type ReturnRequest struct {
	OriginalBatchIndex int
	OriginalEntryIndex int
	ReturnCode         string // R01, R02, ...
	Reason             string // optional addenda info text
	Dishonored         bool
	Contested          bool
}

// BuildReturn creates a return entry inside a new (or merged) return batch
// in the same file. Returns the index of the new return batch in f.Batches.
func BuildReturn(f *ach.File, req ReturnRequest) (int, error) {
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
	if !ValidReturnCode(req.ReturnCode) {
		return 0, fmt.Errorf("unknown return code %q", req.ReturnCode)
	}
	srcEntry := srcEntries[req.OriginalEntryIndex]

	rh := ach.NewBatchHeader()
	*rh = *srcHeader
	rh.BatchNumber = nextBatchNumber(f)
	rh.CompanyEntryDescription = "RETURN"
	rh.ServiceClassCode = ach.MixedDebitsAndCredits

	rb, err := ach.NewBatch(rh)
	if err != nil {
		return 0, err
	}

	re := ach.NewEntryDetail()
	re.TransactionCode = srcEntry.TransactionCode
	re.RDFIIdentification = srcEntry.RDFIIdentification
	re.CheckDigit = srcEntry.CheckDigit
	re.DFIAccountNumber = srcEntry.DFIAccountNumber
	re.Amount = srcEntry.Amount
	re.IdentificationNumber = srcEntry.IdentificationNumber
	re.IndividualName = srcEntry.IndividualName
	re.DiscretionaryData = srcEntry.DiscretionaryData
	re.SetTraceNumber(rh.ODFIIdentification, len(rb.GetEntries())+1)
	re.AddendaRecordIndicator = 1
	re.Category = ach.CategoryReturn

	a99 := ach.NewAddenda99()
	a99.ReturnCode = strings.ToUpper(req.ReturnCode)
	a99.OriginalTrace = srcEntry.TraceNumber
	odfi := srcHeader.ODFIIdentification
	if len(odfi) > 8 {
		odfi = odfi[:8]
	}
	a99.OriginalDFI = odfi
	if req.Reason != "" {
		a99.AddendaInformation = req.Reason
	}
	re.Addenda99 = a99

	if req.Dishonored {
		re.Category = ach.CategoryDishonoredReturn
	}
	if req.Contested {
		re.Category = ach.CategoryDishonoredReturnContested
	}

	rb.AddEntry(re)
	if err := rb.Create(); err != nil {
		return 0, fmt.Errorf("create return batch: %w", err)
	}
	f.AddBatch(rb)
	if err := f.Create(); err != nil {
		return 0, fmt.Errorf("rebuild file: %w", err)
	}
	return len(f.Batches) - 1, nil
}

// ValidReturnCode reports whether code is a known NACHA return code.
func ValidReturnCode(code string) bool {
	if rc := ach.LookupReturnCode(strings.ToUpper(code)); rc != nil {
		return true
	}
	// Fallback: accept R01..R85 even if not in lookup table
	if len(code) == 3 && (code[0] == 'R' || code[0] == 'r') {
		if n, err := strconv.Atoi(code[1:]); err == nil && n >= 1 && n <= 99 {
			return true
		}
	}
	return false
}

// ReturnCodeOption represents one row in the return-code dropdown.
type ReturnCodeOption struct {
	Code        string
	Reason      string
	Description string
}

// AllReturnCodes returns the canonical set R01..R85 with looked-up
// descriptions where available.
func AllReturnCodes() []ReturnCodeOption {
	out := make([]ReturnCodeOption, 0, 85)
	for n := 1; n <= 85; n++ {
		code := fmt.Sprintf("R%02d", n)
		opt := ReturnCodeOption{Code: code}
		if rc := ach.LookupReturnCode(code); rc != nil {
			opt.Reason = rc.Reason
			opt.Description = rc.Description
		}
		out = append(out, opt)
	}
	return out
}
