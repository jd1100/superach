package ui

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/moov-io/ach"

	"github.com/jd1100/superach/internal/achio"
)

// NodeKind enumerates the addressable record kinds in the tree.
type NodeKind int

const (
	NodeFile NodeKind = iota
	NodeBatch
	NodeEntry
	NodeAddenda05
	NodeAddenda98
	NodeAddenda99
	NodeIATBatch
	NodeIATEntry
	NodeIATAddenda
)

// Node is the typed result of decoding a tree path into a record pointer.
type Node struct {
	Kind        NodeKind
	BatchIndex  int
	EntryIndex  int
	AddIndex    int    // index for slice-based addenda (05, 17, 18)
	AddTypeCode string // "10".."18", "98", "99"
}

const (
	maxUndo = 20
)

// AppState carries the open file plus selection, dirty/undo bookkeeping.
// All mutations should go through methods so observers fire.
type AppState struct {
	mu        sync.RWMutex
	file      *ach.File
	path      string
	dirty     bool
	readOnly  bool
	selection string
	undoStack []*ach.File

	listeners []func()
}

// NewState returns an empty AppState with a fresh file loaded.
// The app starts in read-only mode so non-technical users exploring a file
// cannot accidentally mutate it by clicking around. Toggle via View menu.
func NewState() *AppState {
	s := &AppState{readOnly: true}
	s.file = ach.NewFile()
	s.file.Header = ach.NewFileHeader()
	return s
}

// ReadOnly reports whether edits are currently blocked.
func (s *AppState) ReadOnly() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.readOnly
}

// SetReadOnly toggles the read-only flag and notifies observers.
func (s *AppState) SetReadOnly(v bool) {
	s.mu.Lock()
	s.readOnly = v
	s.mu.Unlock()
	s.emit()
}

// Subscribe registers a callback fired on every state change. Returns a
// function that unsubscribes.
func (s *AppState) Subscribe(fn func()) func() {
	s.mu.Lock()
	id := len(s.listeners)
	s.listeners = append(s.listeners, fn)
	s.mu.Unlock()
	return func() {
		s.mu.Lock()
		s.listeners[id] = nil
		s.mu.Unlock()
	}
}

func (s *AppState) emit() {
	s.mu.RLock()
	cbs := make([]func(), 0, len(s.listeners))
	for _, fn := range s.listeners {
		if fn != nil {
			cbs = append(cbs, fn)
		}
	}
	s.mu.RUnlock()
	for _, fn := range cbs {
		fn()
	}
}

// File returns the current file.
func (s *AppState) File() *ach.File {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.file
}

// Path returns the on-disk path of the open file (empty if unsaved).
func (s *AppState) Path() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.path
}

// Dirty reports whether unsaved changes exist.
func (s *AppState) Dirty() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dirty
}

// Selection returns the currently-selected tree path.
func (s *AppState) Selection() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.selection
}

// SetSelection updates the current tree selection (no snapshot, no dirty).
func (s *AppState) SetSelection(p string) {
	s.mu.Lock()
	s.selection = p
	s.mu.Unlock()
	s.emit()
}

// LoadFile replaces the file (e.g. on Open). Resets dirty and undo, and
// re-locks the file in read-only mode so a freshly-loaded ACH can be
// browsed safely before the user opts into editing.
func (s *AppState) LoadFile(f *ach.File, path string) {
	s.mu.Lock()
	s.file = f
	s.path = path
	s.dirty = false
	s.readOnly = true
	s.undoStack = nil
	s.selection = ""
	s.mu.Unlock()
	s.emit()
}

// MarkSaved records that the current file is now persisted to path.
func (s *AppState) MarkSaved(path string) {
	s.mu.Lock()
	s.path = path
	s.dirty = false
	s.mu.Unlock()
	s.emit()
}

// ErrReadOnly is returned by Mutate when the app is in read-only mode.
var ErrReadOnly = fmt.Errorf("file is locked — enable editing from the View menu")

// Mutate snapshots the current file, runs fn against it, then on success
// recalculates control records, marks dirty, and emits to observers.
func (s *AppState) Mutate(fn func(*ach.File) error) error {
	s.mu.Lock()
	if s.file == nil {
		s.mu.Unlock()
		return fmt.Errorf("no file loaded")
	}
	if s.readOnly {
		s.mu.Unlock()
		return ErrReadOnly
	}
	snap, err := achio.Clone(s.file)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	s.mu.Unlock()

	if err := fn(s.file); err != nil {
		return err
	}

	s.mu.Lock()
	s.undoStack = append(s.undoStack, snap)
	if len(s.undoStack) > maxUndo {
		s.undoStack = s.undoStack[1:]
	}
	s.dirty = true
	s.mu.Unlock()
	_ = achio.Recalculate(s.file)
	s.emit()
	return nil
}

// Undo restores the most recent snapshot.
func (s *AppState) Undo() bool {
	s.mu.Lock()
	if len(s.undoStack) == 0 {
		s.mu.Unlock()
		return false
	}
	prev := s.undoStack[len(s.undoStack)-1]
	s.undoStack = s.undoStack[:len(s.undoStack)-1]
	s.file = prev
	s.dirty = true
	s.mu.Unlock()
	s.emit()
	return true
}

// Resolve decodes a tree path into a typed Node.
func Resolve(path string) (Node, bool) {
	if path == "" || path == "file" {
		return Node{Kind: NodeFile}, true
	}
	parts := strings.Split(path, "/")
	switch parts[0] {
	case "b":
		return resolveStandard(parts)
	case "i":
		return resolveIAT(parts)
	}
	return Node{}, false
}

func resolveStandard(parts []string) (Node, bool) {
	if len(parts) < 2 {
		return Node{}, false
	}
	bi, err := strconv.Atoi(parts[1])
	if err != nil {
		return Node{}, false
	}
	if len(parts) == 2 {
		return Node{Kind: NodeBatch, BatchIndex: bi}, true
	}
	if parts[2] != "e" || len(parts) < 4 {
		return Node{}, false
	}
	ei, err := strconv.Atoi(parts[3])
	if err != nil {
		return Node{}, false
	}
	if len(parts) == 4 {
		return Node{Kind: NodeEntry, BatchIndex: bi, EntryIndex: ei}, true
	}
	if parts[4] != "a" {
		// a98 / a99
		switch parts[4] {
		case "a98":
			return Node{Kind: NodeAddenda98, BatchIndex: bi, EntryIndex: ei, AddTypeCode: "98"}, true
		case "a99":
			return Node{Kind: NodeAddenda99, BatchIndex: bi, EntryIndex: ei, AddTypeCode: "99"}, true
		}
		return Node{}, false
	}
	if len(parts) < 6 {
		return Node{}, false
	}
	ai, err := strconv.Atoi(parts[5])
	if err != nil {
		return Node{}, false
	}
	return Node{Kind: NodeAddenda05, BatchIndex: bi, EntryIndex: ei, AddIndex: ai, AddTypeCode: "05"}, true
}

func resolveIAT(parts []string) (Node, bool) {
	if len(parts) < 2 {
		return Node{}, false
	}
	bi, err := strconv.Atoi(parts[1])
	if err != nil {
		return Node{}, false
	}
	if len(parts) == 2 {
		return Node{Kind: NodeIATBatch, BatchIndex: bi}, true
	}
	if parts[2] != "e" || len(parts) < 4 {
		return Node{}, false
	}
	ei, err := strconv.Atoi(parts[3])
	if err != nil {
		return Node{}, false
	}
	if len(parts) == 4 {
		return Node{Kind: NodeIATEntry, BatchIndex: bi, EntryIndex: ei}, true
	}
	// addenda10..16: "a10".. "a16"
	// addenda17/18:  "a17/{k}" or "a18/{k}"
	tag := parts[4]
	switch tag {
	case "a10", "a11", "a12", "a13", "a14", "a15", "a16", "a98", "a99":
		return Node{Kind: NodeIATAddenda, BatchIndex: bi, EntryIndex: ei, AddTypeCode: tag[1:]}, true
	case "a17", "a18":
		if len(parts) < 6 {
			return Node{}, false
		}
		ai, err := strconv.Atoi(parts[5])
		if err != nil {
			return Node{}, false
		}
		return Node{Kind: NodeIATAddenda, BatchIndex: bi, EntryIndex: ei, AddTypeCode: tag[1:], AddIndex: ai}, true
	}
	return Node{}, false
}

// Encode is the inverse of Resolve, used by tree.go.
func Encode(n Node) string {
	switch n.Kind {
	case NodeFile:
		return ""
	case NodeBatch:
		return fmt.Sprintf("b/%d", n.BatchIndex)
	case NodeEntry:
		return fmt.Sprintf("b/%d/e/%d", n.BatchIndex, n.EntryIndex)
	case NodeAddenda05:
		return fmt.Sprintf("b/%d/e/%d/a/%d", n.BatchIndex, n.EntryIndex, n.AddIndex)
	case NodeAddenda98:
		return fmt.Sprintf("b/%d/e/%d/a98", n.BatchIndex, n.EntryIndex)
	case NodeAddenda99:
		return fmt.Sprintf("b/%d/e/%d/a99", n.BatchIndex, n.EntryIndex)
	case NodeIATBatch:
		return fmt.Sprintf("i/%d", n.BatchIndex)
	case NodeIATEntry:
		return fmt.Sprintf("i/%d/e/%d", n.BatchIndex, n.EntryIndex)
	case NodeIATAddenda:
		switch n.AddTypeCode {
		case "17", "18":
			return fmt.Sprintf("i/%d/e/%d/a%s/%d", n.BatchIndex, n.EntryIndex, n.AddTypeCode, n.AddIndex)
		default:
			return fmt.Sprintf("i/%d/e/%d/a%s", n.BatchIndex, n.EntryIndex, n.AddTypeCode)
		}
	}
	return ""
}
