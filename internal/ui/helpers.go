package ui

import (
	"fmt"

	"github.com/moov-io/ach"

	"github.com/jd1100/superach/internal/achio"
)

// newBatchHeader builds a BatchHeader from the dialog inputs.
func newBatchHeader(sec, company, companyID, effective, odfi string, serviceClass int) *ach.BatchHeader {
	bh := ach.NewBatchHeader()
	bh.ServiceClassCode = serviceClass
	bh.StandardEntryClassCode = sec
	bh.CompanyName = company
	bh.CompanyIdentification = companyID
	bh.CompanyEntryDescription = "PAYMENT"
	bh.EffectiveEntryDate = effective
	bh.ODFIIdentification = odfi
	return bh
}

// removeAtNode deletes the record at n from f, matching on kind+index.
func removeAtNode(f *ach.File, n Node) error {
	switch n.Kind {
	case NodeBatch:
		return achio.RemoveBatchAt(f, n.BatchIndex)
	case NodeIATBatch:
		return achio.RemoveIATBatchAt(f, n.BatchIndex)
	case NodeEntry:
		if n.BatchIndex >= len(f.Batches) {
			return fmt.Errorf("invalid batch")
		}
		return achio.RemoveEntryAt(f.Batches[n.BatchIndex], n.EntryIndex)
	case NodeIATEntry:
		if n.BatchIndex >= len(f.IATBatches) {
			return fmt.Errorf("invalid iat batch")
		}
		return achio.RemoveIATEntryAt(&f.IATBatches[n.BatchIndex], n.EntryIndex)
	case NodeAddenda05:
		e, ok := entryAt(f, n.BatchIndex, n.EntryIndex)
		if !ok {
			return fmt.Errorf("entry not found")
		}
		if n.AddIndex >= len(e.Addenda05) {
			return fmt.Errorf("addenda not found")
		}
		e.Addenda05 = append(e.Addenda05[:n.AddIndex], e.Addenda05[n.AddIndex+1:]...)
		if len(e.Addenda05) == 0 && e.Addenda98 == nil && e.Addenda99 == nil {
			e.AddendaRecordIndicator = 0
		}
		return nil
	case NodeAddenda98:
		e, ok := entryAt(f, n.BatchIndex, n.EntryIndex)
		if !ok {
			return fmt.Errorf("entry not found")
		}
		e.Addenda98 = nil
		return nil
	case NodeAddenda99:
		e, ok := entryAt(f, n.BatchIndex, n.EntryIndex)
		if !ok {
			return fmt.Errorf("entry not found")
		}
		e.Addenda99 = nil
		return nil
	}
	return fmt.Errorf("cannot remove this record type")
}

func entryAt(f *ach.File, bi, ei int) (*ach.EntryDetail, bool) {
	if bi >= len(f.Batches) {
		return nil, false
	}
	entries := f.Batches[bi].GetEntries()
	if ei >= len(entries) {
		return nil, false
	}
	return entries[ei], true
}
