package achio

import (
	"encoding/json"
	"fmt"

	"github.com/moov-io/ach"
)

// ToJSON renders the file as pretty-printed JSON.
func ToJSON(f *ach.File) ([]byte, error) {
	if f == nil {
		return nil, fmt.Errorf("nil file")
	}
	return json.MarshalIndent(f, "", "  ")
}

// FromJSON parses JSON bytes into an *ach.File. moov-io/ach calls Create()
// and Validate() during parsing, so the result is round-trippable.
func FromJSON(data []byte) (*ach.File, error) {
	return ach.FileFromJSON(data)
}

// Clone deep-copies a file via JSON round-trip. Used by the undo stack.
func Clone(f *ach.File) (*ach.File, error) {
	if f == nil {
		return nil, nil
	}
	bs, err := json.Marshal(f)
	if err != nil {
		return nil, err
	}
	out, err := ach.FileFromJSON(bs)
	if err != nil {
		// Fall back to a non-validating unmarshal — clones during editing
		// can be transiently invalid.
		out = &ach.File{}
		if err := json.Unmarshal(bs, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}
