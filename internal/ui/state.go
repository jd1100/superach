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
//
// Dirty tracking uses a monotonic mutation count compared against a save
// point. When the undo stack overflows the cap we can no longer reach the
// saved state via Undo, so savePointReachable flips false; once the user
// saves again it re-anchors on the current mutation count.
type AppState struct {
	mu                 sync.RWMutex
	file               *ach.File
	path               string
	mutCount           int
	savedCount         int
	savePointReachable bool
	readOnly           bool
	selection          string
	undoStack          []*ach.File
	recentFiles        []string

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
	if !s.savePointReachable {
		return true
	}
	return s.mutCount != s.savedCount
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
	s.mutCount = 0
	s.savedCount = 0
	s.savePointReachable = true
	s.readOnly = true
	s.undoStack = nil
	s.selection = ""
	s.mu.Unlock()
	s.rememberRecent(path)
	s.emit()
}

// MarkSaved records that the current file is now persisted to path.
func (s *AppState) MarkSaved(path string) {
	s.mu.Lock()
	s.path = path
	s.savedCount = s.mutCount
	s.savePointReachable = true
	s.mu.Unlock()
	s.rememberRecent(path)
	s.emit()
}

// rememberRecent pushes path onto the MRU list (most-recent first, deduped,
// capped at 8 entries). Empty paths are ignored; a path that's already at
// the head is a no-op so repeated saves don't thrash observers.
func (s *AppState) rememberRecent(path string) {
	if path == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.recentFiles) > 0 && s.recentFiles[0] == path {
		return
	}
	out := make([]string, 0, len(s.recentFiles)+1)
	out = append(out, path)
	for _, p := range s.recentFiles {
		if p == path {
			continue
		}
		out = append(out, p)
		if len(out) >= 8 {
			break
		}
	}
	s.recentFiles = out
}

// RecentFiles returns the MRU list (most-recent first).
func (s *AppState) RecentFiles() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, len(s.recentFiles))
	copy(out, s.recentFiles)
	return out
}

// SetRecentFiles seeds the MRU list (e.g. from Fyne preferences on startup).
func (s *AppState) SetRecentFiles(paths []string) {
	s.mu.Lock()
	s.recentFiles = append([]string(nil), paths...)
	s.mu.Unlock()
	s.emit()
}

// ErrReadOnly is returned by Mutate when the app is in read-only mode.
var ErrReadOnly = fmt.Errorf("file is locked — enable editing from the View menu")

// Mutate snapshots the current file, runs fn against it, then on success
// recalculates control records, marks dirty, and emits to observers.
//
// If the mutation function returns an error, the file is rolled back to the
// pre-mutation snapshot so the UI is never stuck in a half-applied state.
// If Recalculate() errors after a successful mutation (e.g. the moov-io
// library rejects the new shape), the error is returned to the caller so the
// UI can surface it — the mutation is still applied and the snapshot is kept
// on the undo stack so the user can Undo out.
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
		// Mutation failed before anything was committed. Roll the pointer
		// back to the snapshot so later reads see a clean file.
		s.mu.Lock()
		s.file = snap
		s.mu.Unlock()
		return err
	}

	s.mu.Lock()
	s.undoStack = append(s.undoStack, snap)
	if len(s.undoStack) > maxUndo {
		// Dropping the oldest snapshot means we can no longer reach the
		// saved state via Undo; the file stays dirty until the next save.
		s.undoStack = s.undoStack[1:]
		s.savePointReachable = false
	}
	s.mutCount++
	s.mu.Unlock()
	recalcErr := achio.Recalculate(s.file)
	s.emit()
	return recalcErr
}

// Undo restores the most recent snapshot. Returns true iff something was
// undone. Dirty() is recomputed from the mutation count, so undoing back to
// the saved state correctly reports clean.
func (s *AppState) Undo() bool {
	s.mu.Lock()
	if len(s.undoStack) == 0 {
		s.mu.Unlock()
		return false
	}
	prev := s.undoStack[len(s.undoStack)-1]
	s.undoStack = s.undoStack[:len(s.undoStack)-1]
	s.file = prev
	s.mutCount--
	s.mu.Unlock()
	s.emit()
	return true
}

// UndoAtCap reports whether the undo stack has reached maxUndo; used by the
// remove-confirm dialog to stop promising "Undo will save you" when the
// next mutation will drop an older snapshot.
func (s *AppState) UndoAtCap() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.undoStack) >= maxUndo
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
