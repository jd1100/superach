package achio

import (
	"fmt"

	"github.com/moov-io/ach"
)

// AddBatch validates the header, builds a Batcher, and appends it.
func AddBatch(f *ach.File, bh *ach.BatchHeader) (ach.Batcher, error) {
	if f == nil || bh == nil {
		return nil, fmt.Errorf("nil file or header")
	}
	if bh.BatchNumber == 0 {
		bh.BatchNumber = nextBatchNumber(f)
	}
	b, err := ach.NewBatch(bh)
	if err != nil {
		return nil, err
	}
	f.AddBatch(b)
	return b, nil
}

// RemoveBatchAt deletes the batch at the given index in f.Batches.
func RemoveBatchAt(f *ach.File, idx int) error {
	if f == nil {
		return fmt.Errorf("nil file")
	}
	if idx < 0 || idx >= len(f.Batches) {
		return fmt.Errorf("batch index out of range")
	}
	f.Batches = append(f.Batches[:idx], f.Batches[idx+1:]...)
	return nil
}

// RemoveIATBatchAt deletes the IAT batch at the given index in f.IATBatches.
func RemoveIATBatchAt(f *ach.File, idx int) error {
	if f == nil {
		return fmt.Errorf("nil file")
	}
	if idx < 0 || idx >= len(f.IATBatches) {
		return fmt.Errorf("iat batch index out of range")
	}
	f.IATBatches = append(f.IATBatches[:idx], f.IATBatches[idx+1:]...)
	return nil
}

// AddEntry appends a new EntryDetail to the batch and assigns a default
// trace number derived from the batch's ODFI.
func AddEntry(b ach.Batcher, e *ach.EntryDetail) error {
	if b == nil {
		return fmt.Errorf("nil batch")
	}
	if e == nil {
		e = ach.NewEntryDetail()
	}
	if e.TraceNumber == "" {
		e.TraceNumber = nextTraceNumber(b)
	}
	if e.Category == "" {
		e.Category = ach.CategoryForward
	}
	b.AddEntry(e)
	return nil
}

// RemoveEntryAt deletes the entry at idx from the batch.
func RemoveEntryAt(b ach.Batcher, idx int) error {
	if b == nil {
		return fmt.Errorf("nil batch")
	}
	entries := b.GetEntries()
	if idx < 0 || idx >= len(entries) {
		return fmt.Errorf("entry index out of range")
	}
	target := entries[idx]
	b.DeleteEntries(func(e *ach.EntryDetail) bool { return e == target })
	return nil
}

// AddIATEntry appends a new IAT entry to the IAT batch.
func AddIATEntry(b *ach.IATBatch, e *ach.IATEntryDetail) error {
	if b == nil {
		return fmt.Errorf("nil iat batch")
	}
	if e == nil {
		e = ach.NewIATEntryDetail()
	}
	if e.TraceNumber == "" && b.Header != nil {
		odfi := b.Header.ODFIIdentification
		if len(odfi) > 8 {
			odfi = odfi[:8]
		}
		e.TraceNumber = fmt.Sprintf("%s%07d", odfi, len(b.Entries)+1)
	}
	if e.Category == "" {
		e.Category = ach.CategoryForward
	}
	b.AddEntry(e)
	return nil
}

// RemoveIATEntryAt deletes the IAT entry at idx.
func RemoveIATEntryAt(b *ach.IATBatch, idx int) error {
	if b == nil {
		return fmt.Errorf("nil iat batch")
	}
	if idx < 0 || idx >= len(b.Entries) {
		return fmt.Errorf("iat entry index out of range")
	}
	target := b.Entries[idx]
	b.DeleteEntries(func(e *ach.IATEntryDetail) bool { return e == target })
	return nil
}

func nextTraceNumber(b ach.Batcher) string {
	h := b.GetHeader()
	if h == nil {
		return ""
	}
	odfi := h.ODFIIdentification
	if len(odfi) > 8 {
		odfi = odfi[:8]
	}
	seq := len(b.GetEntries()) + 1
	return fmt.Sprintf("%s%07d", odfi, seq)
}
