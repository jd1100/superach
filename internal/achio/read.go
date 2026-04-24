package achio

import (
	"fmt"
	"os"
	"strings"

	"github.com/moov-io/ach"
)

// ReadFile reads an ACH file from disk. The format is auto-detected by
// extension (.json → JSON, anything else → NACHA fixed-width).
func ReadFile(path string) (*ach.File, error) {
	if strings.EqualFold(strings.TrimPrefix(extension(path), "."), "json") {
		return ach.ReadJSONFile(path)
	}
	return ach.ReadFile(path)
}

// ReadBytes parses an in-memory ACH payload. Detects JSON by leading '{'.
func ReadBytes(data []byte) (*ach.File, error) {
	trimmed := trimLeadingSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		return ach.FileFromJSON(data)
	}
	r := ach.NewReader(strings.NewReader(string(data)))
	f, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("parse ach: %w", err)
	}
	return &f, nil
}

func extension(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			return path[i:]
		}
		if path[i] == '/' || path[i] == '\\' {
			break
		}
	}
	return ""
}

func trimLeadingSpace(b []byte) []byte {
	i := 0
	for i < len(b) {
		switch b[i] {
		case ' ', '\t', '\n', '\r':
			i++
			continue
		}
		break
	}
	return b[i:]
}

// ReadFileBytes is a convenience helper for tests that wraps os.ReadFile + ReadBytes.
func ReadFileBytes(path string) (*ach.File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ReadBytes(data)
}
