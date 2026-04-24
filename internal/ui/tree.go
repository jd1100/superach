package ui

import (
	"fmt"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/moov-io/ach"
)

// treeFilter caches the active filter plus the set of node IDs that match
// it. Without the cache, Fyne's per-branch visibleChildIDs calls would
// rewalk the whole tree on every keystroke.
type treeFilter struct {
	mu      sync.Mutex
	query   string
	visible map[string]bool // nil when query == ""; otherwise the matching-plus-ancestor set
}

var filter treeFilter

func setFilter(q string) {
	q = strings.TrimSpace(strings.ToLower(q))
	filter.mu.Lock()
	filter.query = q
	filter.visible = nil
	filter.mu.Unlock()
}

func invalidateFilter() {
	filter.mu.Lock()
	filter.visible = nil
	filter.mu.Unlock()
}

func visibleSet(f *ach.File) map[string]bool {
	filter.mu.Lock()
	defer filter.mu.Unlock()
	if filter.query == "" {
		return nil
	}
	if filter.visible != nil {
		return filter.visible
	}
	vis := map[string]bool{}
	var walk func(id string) bool
	walk = func(id string) bool {
		hit := false
		if id != "" && strings.Contains(strings.ToLower(labelFor(f, id)), filter.query) {
			hit = true
		}
		for _, c := range childIDs(f, id) {
			if walk(c) {
				hit = true
			}
		}
		if hit && id != "" {
			vis[id] = true
		}
		return hit
	}
	walk("")
	filter.visible = vis
	return vis
}

// BuildTree constructs the left-pane tree over the current state's file.
// onSelect is invoked with the Node that was selected (empty string path = file root).
func BuildTree(s *AppState, onSelect func(Node)) *widget.Tree {
	tree := widget.NewTree(
		func(id widget.TreeNodeID) []widget.TreeNodeID { return visibleChildIDs(s.File(), id) },
		func(id widget.TreeNodeID) bool { return isBranch(s.File(), id) },
		func(branch bool) fyne.CanvasObject {
			return widget.NewLabel("template template template template")
		},
		func(id widget.TreeNodeID, _ bool, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(labelFor(s.File(), id))
		},
	)
	tree.Root = ""
	tree.OnSelected = func(id widget.TreeNodeID) {
		n, ok := Resolve(id)
		if !ok {
			return
		}
		s.SetSelection(id)
		onSelect(n)
	}
	// Any state change may have touched the file, invalidating the cache.
	s.Subscribe(func() {
		invalidateFilter()
		tree.Refresh()
	})
	return tree
}

func visibleChildIDs(f *ach.File, id widget.TreeNodeID) []string {
	all := childIDs(f, id)
	vis := visibleSet(f)
	if vis == nil {
		return all
	}
	out := make([]string, 0, len(all))
	for _, c := range all {
		if vis[c] {
			out = append(out, c)
		}
	}
	return out
}

func isBranch(f *ach.File, id string) bool {
	if f == nil {
		return false
	}
	if id == "" {
		return true
	}
	n, ok := Resolve(id)
	if !ok {
		return false
	}
	switch n.Kind {
	case NodeBatch:
		return n.BatchIndex < len(f.Batches) && len(f.Batches[n.BatchIndex].GetEntries()) > 0
	case NodeEntry:
		if n.BatchIndex >= len(f.Batches) {
			return false
		}
		e := f.Batches[n.BatchIndex].GetEntries()
		if n.EntryIndex >= len(e) {
			return false
		}
		ed := e[n.EntryIndex]
		return len(ed.Addenda05) > 0 || ed.Addenda98 != nil || ed.Addenda99 != nil
	case NodeIATBatch:
		return n.BatchIndex < len(f.IATBatches) && len(f.IATBatches[n.BatchIndex].Entries) > 0
	case NodeIATEntry:
		return true // always has mandatory addenda10-16
	}
	return false
}

func childIDs(f *ach.File, id string) []string {
	if f == nil {
		return nil
	}
	if id == "" {
		out := []string{}
		for i := range f.Batches {
			out = append(out, fmt.Sprintf("b/%d", i))
		}
		for i := range f.IATBatches {
			out = append(out, fmt.Sprintf("i/%d", i))
		}
		return out
	}
	n, ok := Resolve(id)
	if !ok {
		return nil
	}
	switch n.Kind {
	case NodeBatch:
		if n.BatchIndex >= len(f.Batches) {
			return nil
		}
		out := []string{}
		for i := range f.Batches[n.BatchIndex].GetEntries() {
			out = append(out, fmt.Sprintf("b/%d/e/%d", n.BatchIndex, i))
		}
		return out
	case NodeEntry:
		if n.BatchIndex >= len(f.Batches) {
			return nil
		}
		entries := f.Batches[n.BatchIndex].GetEntries()
		if n.EntryIndex >= len(entries) {
			return nil
		}
		ed := entries[n.EntryIndex]
		out := []string{}
		for i := range ed.Addenda05 {
			out = append(out, fmt.Sprintf("b/%d/e/%d/a/%d", n.BatchIndex, n.EntryIndex, i))
		}
		if ed.Addenda98 != nil {
			out = append(out, fmt.Sprintf("b/%d/e/%d/a98", n.BatchIndex, n.EntryIndex))
		}
		if ed.Addenda99 != nil {
			out = append(out, fmt.Sprintf("b/%d/e/%d/a99", n.BatchIndex, n.EntryIndex))
		}
		return out
	case NodeIATBatch:
		if n.BatchIndex >= len(f.IATBatches) {
			return nil
		}
		out := []string{}
		for i := range f.IATBatches[n.BatchIndex].Entries {
			out = append(out, fmt.Sprintf("i/%d/e/%d", n.BatchIndex, i))
		}
		return out
	case NodeIATEntry:
		if n.BatchIndex >= len(f.IATBatches) {
			return nil
		}
		entries := f.IATBatches[n.BatchIndex].Entries
		if n.EntryIndex >= len(entries) {
			return nil
		}
		e := entries[n.EntryIndex]
		out := []string{}
		for _, code := range []string{"10", "11", "12", "13", "14", "15", "16"} {
			if iatAddendaPresent(e, code) {
				out = append(out, fmt.Sprintf("i/%d/e/%d/a%s", n.BatchIndex, n.EntryIndex, code))
			}
		}
		for i := range e.Addenda17 {
			out = append(out, fmt.Sprintf("i/%d/e/%d/a17/%d", n.BatchIndex, n.EntryIndex, i))
		}
		for i := range e.Addenda18 {
			out = append(out, fmt.Sprintf("i/%d/e/%d/a18/%d", n.BatchIndex, n.EntryIndex, i))
		}
		if e.Addenda98 != nil {
			out = append(out, fmt.Sprintf("i/%d/e/%d/a98", n.BatchIndex, n.EntryIndex))
		}
		if e.Addenda99 != nil {
			out = append(out, fmt.Sprintf("i/%d/e/%d/a99", n.BatchIndex, n.EntryIndex))
		}
		return out
	}
	return nil
}

func iatAddendaPresent(e *ach.IATEntryDetail, code string) bool {
	switch code {
	case "10":
		return e.Addenda10 != nil
	case "11":
		return e.Addenda11 != nil
	case "12":
		return e.Addenda12 != nil
	case "13":
		return e.Addenda13 != nil
	case "14":
		return e.Addenda14 != nil
	case "15":
		return e.Addenda15 != nil
	case "16":
		return e.Addenda16 != nil
	}
	return false
}

func labelFor(f *ach.File, id string) string {
	if id == "" {
		if f == nil || f.Header.ImmediateOrigin == "" {
			return "File"
		}
		return fmt.Sprintf("File — %s → %s",
			strings.TrimSpace(f.Header.ImmediateOriginName),
			strings.TrimSpace(f.Header.ImmediateDestinationName))
	}
	n, ok := Resolve(id)
	if !ok || f == nil {
		return id
	}
	switch n.Kind {
	case NodeBatch:
		if n.BatchIndex >= len(f.Batches) {
			return "(invalid batch)"
		}
		h := f.Batches[n.BatchIndex].GetHeader()
		cat := f.Batches[n.BatchIndex].Category()
		tag := ""
		switch cat {
		case ach.CategoryReturn, ach.CategoryDishonoredReturn, ach.CategoryDishonoredReturnContested:
			tag = " [RET]"
		case ach.CategoryNOC:
			tag = " [NOC]"
		}
		return fmt.Sprintf("Batch %d — %s — %s%s", h.BatchNumber, h.StandardEntryClassCode, strings.TrimSpace(h.CompanyName), tag)
	case NodeEntry:
		if n.BatchIndex >= len(f.Batches) {
			return "(invalid)"
		}
		entries := f.Batches[n.BatchIndex].GetEntries()
		if n.EntryIndex >= len(entries) {
			return "(invalid)"
		}
		e := entries[n.EntryIndex]
		return fmt.Sprintf("%s — %s — $%s", strings.TrimSpace(e.TraceNumber), strings.TrimSpace(e.IndividualName), formatAmount(e.Amount))
	case NodeAddenda05:
		return fmt.Sprintf("Addenda05 #%d", n.AddIndex+1)
	case NodeAddenda98:
		return "Addenda98 (NOC)"
	case NodeAddenda99:
		return "Addenda99 (Return)"
	case NodeIATBatch:
		if n.BatchIndex >= len(f.IATBatches) {
			return "(invalid iat batch)"
		}
		h := f.IATBatches[n.BatchIndex].GetHeader()
		return fmt.Sprintf("IAT Batch %d — %s → %s", h.BatchNumber, h.ISOOriginatingCurrencyCode, h.ISODestinationCurrencyCode)
	case NodeIATEntry:
		if n.BatchIndex >= len(f.IATBatches) {
			return "(invalid)"
		}
		entries := f.IATBatches[n.BatchIndex].Entries
		if n.EntryIndex >= len(entries) {
			return "(invalid)"
		}
		e := entries[n.EntryIndex]
		return fmt.Sprintf("IAT %s — $%s", strings.TrimSpace(e.TraceNumber), formatAmount(e.Amount))
	case NodeIATAddenda:
		if n.AddTypeCode == "17" || n.AddTypeCode == "18" {
			return fmt.Sprintf("Addenda%s #%d", n.AddTypeCode, n.AddIndex+1)
		}
		return fmt.Sprintf("Addenda%s", n.AddTypeCode)
	}
	return id
}

func formatAmount(cents int) string {
	neg := ""
	if cents < 0 {
		neg = "-"
		cents = -cents
	}
	dollars := cents / 100
	rem := cents % 100
	return fmt.Sprintf("%s%d.%02d", neg, dollars, rem)
}

// wrapTree packs the tree into a scrollable, titled container with a filter
// search bar at the top. The returned entry is stored on the App so Ctrl+F
// can focus it.
func wrapTree(t *widget.Tree, search *widget.Entry) fyne.CanvasObject {
	header := container.NewVBox(
		widget.NewLabelWithStyle("Structure", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		search,
	)
	return container.NewBorder(header, nil, nil, nil, t)
}
