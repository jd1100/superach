package ui

import (
	"bytes"
	"fmt"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/storage"
	"github.com/moov-io/ach"

	"github.com/jd1100/superach/internal/achio"
	"github.com/jd1100/superach/internal/ui/dialogs"
)

// menuItem wires a labelled action with its shortcut on the menu *and* on
// the window canvas in one place, so the displayed accelerator can never
// drift from the binding that actually fires it. Pass a nil shortcut to
// skip the canvas binding (items exposed only in the menu).
func (a *App) menuItem(label string, do func(), shortcut fyne.Shortcut) *fyne.MenuItem {
	m := fyne.NewMenuItem(label, do)
	if shortcut != nil {
		m.Shortcut = shortcut
		if c := a.window.Canvas(); c != nil {
			c.AddShortcut(shortcut, func(fyne.Shortcut) { do() })
		}
	}
	return m
}

func ctrlShortcut(key fyne.KeyName) *desktop.CustomShortcut {
	return &desktop.CustomShortcut{KeyName: key, Modifier: fyne.KeyModifierControl}
}

func (a *App) buildMenu() *fyne.MainMenu {
	fileMenu := fyne.NewMenu("File",
		a.menuItem("Open…", a.openDialog, ctrlShortcut(fyne.KeyO)),
		a.buildRecentMenuItem(),
		a.menuItem("Save", a.saveCurrent, ctrlShortcut(fyne.KeyS)),
		a.menuItem("Save As…", a.saveAsDialog,
			&desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierControl | fyne.KeyModifierShift}),
		fyne.NewMenuItemSeparator(),
		a.menuItem("Export JSON…", a.exportJSONDialog, nil),
		a.menuItem("Export CSV…", a.exportCSVDialog, nil),
		a.menuItem("Import CSV…", a.importCSVDialog, nil),
		fyne.NewMenuItemSeparator(),
		a.menuItem("Quit", func() { a.confirmDiscard(func() { a.fyneApp.Quit() }) }, ctrlShortcut(fyne.KeyQ)),
	)

	viewMenu := fyne.NewMenu("View",
		a.menuItem("Toggle Read-only / Editing", a.toggleReadOnly, ctrlShortcut(fyne.KeyE)),
		a.menuItem("Focus Search", a.focusSearch, ctrlShortcut(fyne.KeyF)),
	)

	undo := func() {
		if a.state.ReadOnly() {
			a.requireEditable()
			return
		}
		if !a.state.Undo() {
			a.statusLabel.SetText("Nothing to undo.")
		}
	}

	editMenu := fyne.NewMenu("Edit",
		a.menuItem("Undo", undo, &fyne.ShortcutUndo{}),
		fyne.NewMenuItemSeparator(),
		a.menuItem("New Batch…", a.newBatchDialog, ctrlShortcut(fyne.KeyB)),
		a.menuItem("New Entry in Selected Batch", a.newEntryInSelection, ctrlShortcut(fyne.KeyN)),
		a.menuItem("Remove Selected", a.removeSelected,
			&desktop.CustomShortcut{KeyName: fyne.KeyDelete}),
		fyne.NewMenuItemSeparator(),
		a.menuItem("New Return from Selected Entry…", a.newReturnDialog, nil),
		a.menuItem("New NOC from Selected Entry…", a.newNOCDialog, nil),
	)

	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("About", func() { dialogs.ShowAbout(a.window) }),
	)

	return fyne.NewMainMenu(fileMenu, viewMenu, editMenu, helpMenu)
}

func (a *App) focusSearch() {
	if a.searchEntry != nil {
		a.window.Canvas().Focus(a.searchEntry)
	}
}

// buildRecentMenuItem produces the "Open Recent" submenu. Fyne reuses the
// MenuItem.ChildMenu field for submenus, so we rebuild it eagerly each time
// the menu is requested.
func (a *App) buildRecentMenuItem() *fyne.MenuItem {
	recents := a.state.RecentFiles()
	item := fyne.NewMenuItem("Open Recent", nil)
	if len(recents) == 0 {
		sub := fyne.NewMenu("", fyne.NewMenuItem("(none)", nil))
		sub.Items[0].Disabled = true
		item.ChildMenu = sub
		return item
	}
	items := make([]*fyne.MenuItem, 0, len(recents))
	for _, p := range recents {
		p := p
		items = append(items, fyne.NewMenuItem(p, func() {
			a.confirmDiscard(func() { a.loadPath(p) })
		}))
	}
	item.ChildMenu = fyne.NewMenu("", items...)
	return item
}

func (a *App) openDialog() {
	a.confirmDiscard(func() {
		d := dialog.NewFileOpen(func(u fyne.URIReadCloser, err error) {
			if err != nil || u == nil {
				return
			}
			path := uriPath(u.URI())
			_ = u.Close()
			a.loadPath(path)
		}, a.window)
		d.SetFilter(defaultFilter())
		d.Show()
	})
}

func (a *App) saveCurrent() {
	if p := a.state.Path(); p != "" {
		a.savePath(p)
		return
	}
	a.saveAsDialog()
}

func (a *App) saveAsDialog() {
	d := dialog.NewFileSave(func(u fyne.URIWriteCloser, err error) {
		if err != nil || u == nil {
			return
		}
		path := uriPath(u.URI())
		_ = u.Close()
		a.savePath(path)
	}, a.window)
	d.SetFilter(defaultFilter())
	d.SetFileName("untitled.ach")
	d.Show()
}

func (a *App) exportJSONDialog() {
	a.runExport("JSON", "export.json",
		storage.NewExtensionFileFilter([]string{".json"}),
		func(f *ach.File) ([]byte, error) { return achio.ToJSON(f) })
}

func (a *App) exportCSVDialog() {
	a.runExport("CSV", "entries.csv", csvFilter(), func(f *ach.File) ([]byte, error) {
		var buf bytes.Buffer
		if err := achio.EntriesToCSV(&buf, f); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	})
}

// runExport drives the export flow for both JSON and CSV: validate first,
// confirm if there are errors, then open a save dialog and write the bytes
// produced by encode.
func (a *App) runExport(kind, defaultName string, filter storage.FileFilter, encode func(*ach.File) ([]byte, error)) {
	f := a.state.File()
	if f == nil {
		return
	}
	a.confirmExportWhenInvalid(kind, func() {
		d := dialog.NewFileSave(func(u fyne.URIWriteCloser, err error) {
			if err != nil || u == nil {
				return
			}
			defer u.Close()
			bs, eerr := encode(f)
			if eerr != nil {
				dialog.ShowError(eerr, a.window)
				return
			}
			if _, werr := u.Write(bs); werr != nil {
				dialog.ShowError(werr, a.window)
				return
			}
			a.statusLabel.SetText("Exported " + kind + ": " + u.URI().Name())
		}, a.window)
		d.SetFilter(filter)
		d.SetFileName(defaultName)
		d.Show()
	})
}

// confirmExportWhenInvalid warns if the file has validation errors so a
// user doesn't silently ship broken data to a downstream system.
func (a *App) confirmExportWhenInvalid(kind string, onContinue func()) {
	f := a.state.File()
	if f == nil {
		onContinue()
		return
	}
	errs, _ := achio.ValidateFile(f)
	if len(errs) == 0 {
		onContinue()
		return
	}
	first := errs[0].Error()
	if len(errs) > 1 {
		first = fmt.Sprintf("%s (+%d more)", first, len(errs)-1)
	}
	msg := fmt.Sprintf("This file has %d validation error(s) — the %s export will include them.\n\nFirst error: %s\n\nExport anyway?", len(errs), kind, first)
	dialog.ShowConfirm("Export file with validation errors?", msg, func(ok bool) {
		if ok {
			onContinue()
		}
	}, a.window)
}

func (a *App) importCSVDialog() {
	if !a.requireEditable() {
		return
	}
	d := dialog.NewFileOpen(func(u fyne.URIReadCloser, err error) {
		if err != nil || u == nil {
			return
		}
		defer u.Close()
		data, rerr := os.ReadFile(uriPath(u.URI()))
		if rerr != nil {
			dialog.ShowError(rerr, a.window)
			return
		}
		if merr := a.state.Mutate(func(file *ach.File) error {
			_, err := achio.ImportCSV(bytes.NewReader(data), file, file.Header.ImmediateOrigin, file.Header.ImmediateOriginName, file.Header.ImmediateDestination, file.Header.ImmediateDestinationName)
			return err
		}); merr != nil {
			dialog.ShowError(merr, a.window)
			return
		}
		a.statusLabel.SetText("Imported " + u.URI().Name())
	}, a.window)
	d.SetFilter(csvFilter())
	d.Show()
}

func (a *App) newBatchDialog() {
	if !a.requireEditable() {
		return
	}
	f := a.state.File()
	if f == nil {
		return
	}
	dialogs.NewBatch(a.window, func(sec, company, companyID, effective, odfi string, svc int) {
		if err := a.state.Mutate(func(file *ach.File) error {
			bh := newBatchHeader(sec, company, companyID, effective, odfi, svc)
			_, err := achio.AddBatch(file, bh)
			return err
		}); err != nil {
			dialog.ShowError(err, a.window)
			return
		}
		a.statusLabel.SetText("Added batch: " + sec)
	})
}

func (a *App) newEntryInSelection() {
	if !a.requireEditable() {
		return
	}
	n, ok := Resolve(a.state.Selection())
	if !ok || (n.Kind != NodeBatch && n.Kind != NodeEntry) {
		dialog.ShowInformation("Select a batch or entry first", "Pick a batch or one of its entries in the tree, then try again.", a.window)
		return
	}
	if err := a.state.Mutate(func(file *ach.File) error {
		if n.BatchIndex >= len(file.Batches) {
			return fmt.Errorf("invalid batch index")
		}
		b := file.Batches[n.BatchIndex]
		return achio.AddEntry(b, nil)
	}); err != nil {
		dialog.ShowError(err, a.window)
		return
	}
	a.statusLabel.SetText("Added new entry")
}

func (a *App) removeSelected() {
	if !a.requireEditable() {
		return
	}
	n, ok := Resolve(a.state.Selection())
	if !ok || n.Kind == NodeFile {
		dialog.ShowInformation("Nothing to remove", "Select a batch, entry, or addenda first.", a.window)
		return
	}
	summary := labelFor(a.state.File(), Encode(n))
	undoHint := "You can reverse this with Edit → Undo (Ctrl+Z)."
	if a.state.UndoAtCap() {
		undoHint = "Undo history is at its cap — this removal may not be reversible. " +
			"Save a backup first if the record matters."
	}
	dialog.ShowConfirm("Remove this record?",
		fmt.Sprintf("Remove %s?\n\n%s", summary, undoHint),
		func(confirm bool) {
			if !confirm {
				return
			}
			if err := a.state.Mutate(func(file *ach.File) error {
				return removeAtNode(file, n)
			}); err != nil {
				dialog.ShowError(err, a.window)
				return
			}
			a.statusLabel.SetText("Removed " + summary)
		}, a.window)
}

func (a *App) newReturnDialog() {
	if !a.requireEditable() {
		return
	}
	n, ok := Resolve(a.state.Selection())
	if !ok || n.Kind != NodeEntry {
		dialog.ShowInformation("Select a forward entry first", "The return wizard needs an existing EntryDetail selected.", a.window)
		return
	}
	dialogs.NewReturn(a.window, achio.AllReturnCodes(), func(code, reason string, dishonored, contested bool) {
		if err := a.state.Mutate(func(file *ach.File) error {
			_, err := achio.BuildReturn(file, achio.ReturnRequest{
				OriginalBatchIndex: n.BatchIndex,
				OriginalEntryIndex: n.EntryIndex,
				ReturnCode:         code,
				Reason:             reason,
				Dishonored:         dishonored,
				Contested:          contested,
			})
			return err
		}); err != nil {
			dialog.ShowError(err, a.window)
			return
		}
		a.statusLabel.SetText("Return " + code + " created")
	})
}

func (a *App) newNOCDialog() {
	if !a.requireEditable() {
		return
	}
	n, ok := Resolve(a.state.Selection())
	if !ok || n.Kind != NodeEntry {
		dialog.ShowInformation("Select a forward entry first", "The NOC wizard needs an existing EntryDetail selected.", a.window)
		return
	}
	dialogs.NewNOC(a.window, achio.AllChangeCodes(), func(code string, corrected dialogs.CorrectedInput) {
		if err := a.state.Mutate(func(file *ach.File) error {
			_, err := achio.BuildNOC(file, achio.NOCRequest{
				OriginalBatchIndex: n.BatchIndex,
				OriginalEntryIndex: n.EntryIndex,
				ChangeCode:         code,
				Corrected:          corrected.ToAch(),
			})
			return err
		}); err != nil {
			dialog.ShowError(err, a.window)
			return
		}
		a.statusLabel.SetText("NOC " + code + " created")
	})
}
