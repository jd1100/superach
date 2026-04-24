package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/moov-io/ach"

	"github.com/jd1100/superach/internal/ui/forms"
)

// Detail renders the right-pane form for the currently-selected tree node.
type Detail struct {
	state     *AppState
	container *fyne.Container
	lastNode  Node
	// parent is the window used to surface dialog.ShowError when a form
	// save fails (e.g. recalculate rejects a half-valid edit). When nil,
	// save errors are silently dropped.
	parent fyne.Window
}

// NewDetail wires the detail pane up to the state so it re-renders on change.
func NewDetail(s *AppState) *Detail {
	d := &Detail{state: s, container: container.NewStack()}
	d.render(Node{Kind: NodeFile})
	return d
}

// AttachParent records the window used to report save/recalculate errors.
// Called from NewApp once the window exists.
func (d *Detail) AttachParent(w fyne.Window) { d.parent = w }

// CanvasObject returns the wrapped Fyne container.
func (d *Detail) CanvasObject() fyne.CanvasObject { return d.container }

// Render swaps the detail form to match node n.
func (d *Detail) Render(n Node) { d.render(n) }

// RerenderCurrent rebuilds the currently-visible form. Used after a mode
// toggle (read-only ↔ editing) so widgets reflect the new state.
func (d *Detail) RerenderCurrent() { d.render(d.lastNode) }

func (d *Detail) render(n Node) {
	d.lastNode = n
	f := d.state.File()
	if f == nil {
		d.swap(widget.NewLabel("No file loaded."))
		return
	}

	save := func() {
		if d.state.ReadOnly() {
			return
		}
		// Recalculate / clone / read-only errors bubble up here; surfacing
		// them is the whole point of plumbing Mutate's error through.
		if err := d.state.Mutate(func(_ *ach.File) error { return nil }); err != nil && d.parent != nil {
			dialog.ShowError(err, d.parent)
		}
	}

	var obj fyne.CanvasObject
	switch n.Kind {
	case NodeFile:
		obj = forms.FileHeaderForm(&f.Header, save)
	case NodeBatch:
		if n.BatchIndex >= len(f.Batches) {
			obj = widget.NewLabel("Batch not found.")
			break
		}
		obj = forms.BatchHeaderForm(f.Batches[n.BatchIndex].GetHeader(), save)
	case NodeEntry:
		if n.BatchIndex >= len(f.Batches) {
			obj = widget.NewLabel("Batch not found.")
			break
		}
		entries := f.Batches[n.BatchIndex].GetEntries()
		if n.EntryIndex >= len(entries) {
			obj = widget.NewLabel("Entry not found.")
			break
		}
		obj = forms.EntryDetailForm(entries[n.EntryIndex], save)
	case NodeAddenda05:
		e, ok := entryFor(f, n)
		if !ok || n.AddIndex >= len(e.Addenda05) {
			obj = widget.NewLabel("Addenda not found.")
			break
		}
		obj = forms.Addenda05Form(e.Addenda05[n.AddIndex], save)
	case NodeAddenda98:
		e, ok := entryFor(f, n)
		if !ok || e.Addenda98 == nil {
			obj = widget.NewLabel("Addenda98 not found.")
			break
		}
		obj = forms.Addenda98Form(e.Addenda98, save)
	case NodeAddenda99:
		e, ok := entryFor(f, n)
		if !ok || e.Addenda99 == nil {
			obj = widget.NewLabel("Addenda99 not found.")
			break
		}
		obj = forms.Addenda99Form(e.Addenda99, save)
	case NodeIATBatch:
		if n.BatchIndex >= len(f.IATBatches) {
			obj = widget.NewLabel("IAT batch not found.")
			break
		}
		obj = forms.IATBatchHeaderForm(f.IATBatches[n.BatchIndex].GetHeader(), save)
	case NodeIATEntry:
		if n.BatchIndex >= len(f.IATBatches) {
			obj = widget.NewLabel("IAT batch not found.")
			break
		}
		entries := f.IATBatches[n.BatchIndex].Entries
		if n.EntryIndex >= len(entries) {
			obj = widget.NewLabel("IAT entry not found.")
			break
		}
		obj = forms.IATEntryDetailForm(entries[n.EntryIndex], save)
	case NodeIATAddenda:
		if n.BatchIndex >= len(f.IATBatches) {
			obj = widget.NewLabel("IAT batch not found.")
			break
		}
		entries := f.IATBatches[n.BatchIndex].Entries
		if n.EntryIndex >= len(entries) {
			obj = widget.NewLabel("IAT entry not found.")
			break
		}
		e := entries[n.EntryIndex]
		obj = forms.IATAddendaForm(e, n.AddTypeCode, n.AddIndex, save)
	default:
		obj = widget.NewLabel("Select a record on the left.")
	}

	title := nodeTitle(f, n)
	header := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	d.swap(container.NewBorder(header, nil, nil, nil, container.NewVScroll(obj)))
}

func (d *Detail) swap(obj fyne.CanvasObject) {
	d.container.Objects = []fyne.CanvasObject{obj}
	d.container.Refresh()
}

func entryFor(f *ach.File, n Node) (*ach.EntryDetail, bool) {
	if n.BatchIndex >= len(f.Batches) {
		return nil, false
	}
	entries := f.Batches[n.BatchIndex].GetEntries()
	if n.EntryIndex >= len(entries) {
		return nil, false
	}
	return entries[n.EntryIndex], true
}

func nodeTitle(f *ach.File, n Node) string {
	switch n.Kind {
	case NodeFile:
		return "File Header"
	case NodeBatch:
		if n.BatchIndex < len(f.Batches) {
			return fmt.Sprintf("Batch %d — %s", f.Batches[n.BatchIndex].GetHeader().BatchNumber, strings.TrimSpace(f.Batches[n.BatchIndex].GetHeader().StandardEntryClassCode))
		}
	case NodeEntry:
		return "Entry Detail"
	case NodeAddenda05:
		return fmt.Sprintf("Addenda05 #%d", n.AddIndex+1)
	case NodeAddenda98:
		return "Addenda98 — Notification of Change"
	case NodeAddenda99:
		return "Addenda99 — Return"
	case NodeIATBatch:
		return "IAT Batch Header"
	case NodeIATEntry:
		return "IAT Entry Detail"
	case NodeIATAddenda:
		return fmt.Sprintf("IAT Addenda %s", n.AddTypeCode)
	}
	return ""
}
