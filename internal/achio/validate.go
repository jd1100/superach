package achio

import (
	"errors"
	"fmt"
	"strings"

	"github.com/moov-io/ach"
	"github.com/moov-io/base"
)

// FieldError points at a specific record in the file. Path is dotted,
// e.g. "batches[0].entries[3].amount". Field is the offending column.
type FieldError struct {
	Path    string
	Field   string
	Message string
}

func (e FieldError) Error() string {
	if e.Field == "" {
		return fmt.Sprintf("%s: %s", e.Path, e.Message)
	}
	return fmt.Sprintf("%s.%s: %s", e.Path, e.Field, e.Message)
}

// ValidateFile runs ach.File.Validate and decomposes any base.ErrorList into
// a flat slice of FieldErrors. The error result is non-nil iff any field
// failed; the slice is always returned (possibly empty).
func ValidateFile(f *ach.File) ([]FieldError, error) {
	if f == nil {
		return nil, errors.New("nil file")
	}
	var out []FieldError
	if err := f.Validate(); err != nil {
		out = append(out, flatten("file", err)...)
	}
	for bi, b := range f.Batches {
		path := fmt.Sprintf("batches[%d]", bi)
		if err := b.Validate(); err != nil {
			out = append(out, flatten(path, err)...)
		}
		for ei, e := range b.GetEntries() {
			epath := fmt.Sprintf("%s.entries[%d]", path, ei)
			if err := e.Validate(); err != nil {
				out = append(out, flatten(epath, err)...)
			}
		}
	}
	for bi, b := range f.IATBatches {
		path := fmt.Sprintf("iatBatches[%d]", bi)
		if err := b.Validate(); err != nil {
			out = append(out, flatten(path, err)...)
		}
		for ei, e := range b.GetEntries() {
			epath := fmt.Sprintf("%s.entries[%d]", path, ei)
			if err := e.Validate(); err != nil {
				out = append(out, flatten(epath, err)...)
			}
		}
	}
	if len(out) == 0 {
		return out, nil
	}
	return out, errors.New(out[0].Error())
}

func flatten(path string, err error) []FieldError {
	if err == nil {
		return nil
	}
	var list base.ErrorList
	if errors.As(err, &list) {
		out := make([]FieldError, 0, len(list))
		for _, e := range list {
			if e == nil {
				continue
			}
			out = append(out, fieldFromError(path, e))
		}
		return out
	}
	return []FieldError{fieldFromError(path, err)}
}

func fieldFromError(path string, err error) FieldError {
	msg := err.Error()
	field := ""
	// moov-io error strings are usually "FieldError ... FieldName=Foo Msg=..."
	if i := strings.Index(msg, "FieldName="); i >= 0 {
		rest := msg[i+len("FieldName="):]
		if j := strings.IndexAny(rest, " \t"); j > 0 {
			field = rest[:j]
		} else {
			field = rest
		}
	}
	return FieldError{Path: path, Field: field, Message: msg}
}

// Recalculate reruns File.Create to refresh hashes, totals, and control
// records after a mutation. Panics inside the moov-io/ach library (e.g. nil
// ADVControl when a batch's SEC code has been edited to ADV in place) are
// converted to errors so the GUI stays alive.
func Recalculate(f *ach.File) error {
	if f == nil {
		return nil
	}
	for _, b := range f.Batches {
		if err := safeBatchCreate(b); err != nil {
			return err
		}
	}
	for i := range f.IATBatches {
		if err := safeIATBatchCreate(f.IATBatches[i]); err != nil {
			return err
		}
	}
	return safeCreate(f)
}

// safeCreate calls f.Create() but converts any panic inside the moov-io/ach
// library into a friendly error. The library occasionally dereferences nil
// control records when a file is in a half-valid state (for example, a batch
// whose SEC code says ADV but was never constructed as an ADV batch). We don't
// want those bugs to kill the GUI.
func safeCreate(f *ach.File) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("cannot save file in its current shape (internal library error: %v). "+
				"This usually means a batch's SEC Code or Service Class doesn't match its "+
				"underlying batch type. Try Edit → Undo, or remove and re-add the batch", r)
		}
	}()
	return f.Create()
}

func safeBatchCreate(b ach.Batcher) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("cannot recalculate batch (internal library error: %v)", r)
		}
	}()
	return b.Create()
}

func safeIATBatchCreate(b ach.IATBatch) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("cannot recalculate IAT batch (internal library error: %v)", r)
		}
	}()
	return b.Create()
}
