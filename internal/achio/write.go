package achio

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/moov-io/ach"
)

// WriteFile rebuilds control records (via File.Create) and writes the file
// to disk. .json paths emit JSON; anything else emits NACHA fixed-width.
func WriteFile(path string, f *ach.File) error {
	if f == nil {
		return fmt.Errorf("nil file")
	}
	if err := safeCreate(f); err != nil {
		return err
	}
	if strings.EqualFold(filepath.Ext(path), ".json") {
		bs, err := ToJSON(f)
		if err != nil {
			return err
		}
		return os.WriteFile(path, bs, 0o644)
	}
	var buf bytes.Buffer
	w := ach.NewWriter(&buf)
	if err := w.Write(f); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := w.Flush(); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// WriteBytes serialises to NACHA fixed-width bytes (for clipboard/preview).
func WriteBytes(f *ach.File) ([]byte, error) {
	if f == nil {
		return nil, fmt.Errorf("nil file")
	}
	if err := safeCreate(f); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	w := ach.NewWriter(&buf)
	if err := w.Write(f); err != nil {
		return nil, err
	}
	if err := w.Flush(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
