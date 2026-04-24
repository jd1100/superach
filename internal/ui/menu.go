package ui

import (
	"bytes"
	"fmt"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"github.com/moov-io/ach"

	"github.com/jd1100/superach/internal/achio"
	"github.com/jd1100/superach/internal/ui/dialogs"
)

func (a *App) buildMenu() *fyne.MainMenu {
	fileMenu := fyne.NewMenu("File",
		fyne.NewMenuItem("Open…", a.openDialog),
		fyne.NewMenuItem("Save", a.saveCurrent),
		fyne.NewMenuItem("Save As…", a.saveAsDialog),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Export JSON…", a.exportJSONDialog),
		fyne.NewMenuItem("Export CSV…", a.exportCSVDialog),
		fyne.NewMenuItem("Import CSV…", a.importCSVDialog),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Quit", func() { a.confirmDiscard(func() { a.fyneApp.Quit() }) }),
	)

	editMenu := fyne.NewMenu("Edit",
		fyne.NewMenuItem("Undo", func() {
			if !a.state.Undo() {
				a.statusLabel.SetText("Nothing to undo.")
			}
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("New Batch…", a.newBatchDialog),
		fyne.NewMenuItem("New Entry in Selected Batch", a.newEntryInSelection),
		fyne.NewMenuItem("Remove Selected", a.removeSelected),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("New Return from Selected Entry…", a.newReturnDialog),
		fyne.NewMenuItem("New NOC from Selected Entry…", a.newNOCDialog),
	)

	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("About", func() { dialogs.ShowAbout(a.window) }),
	)

	return fyne.NewMainMenu(fileMenu, editMenu, helpMenu)
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
	f := a.state.File()
	if f == nil {
		return
	}
	d := dialog.NewFileSave(func(u fyne.URIWriteCloser, err error) {
		if err != nil || u == nil {
			return
		}
		defer u.Close()
		bs, jerr := achio.ToJSON(f)
		if jerr != nil {
			dialog.ShowError(jerr, a.window)
			return
		}
		if _, werr := u.Write(bs); werr != nil {
			dialog.ShowError(werr, a.window)
			return
		}
		a.statusLabel.SetText("Exported JSON: " + u.URI().Name())
	}, a.window)
	d.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
	d.SetFileName("export.json")
	d.Show()
}

func (a *App) exportCSVDialog() {
	f := a.state.File()
	if f == nil {
		return
	}
	d := dialog.NewFileSave(func(u fyne.URIWriteCloser, err error) {
		if err != nil || u == nil {
			return
		}
		defer u.Close()
		var buf bytes.Buffer
		if cerr := achio.EntriesToCSV(&buf, f); cerr != nil {
			dialog.ShowError(cerr, a.window)
			return
		}
		if _, werr := u.Write(buf.Bytes()); werr != nil {
			dialog.ShowError(werr, a.window)
			return
		}
		a.statusLabel.SetText("Exported CSV: " + u.URI().Name())
	}, a.window)
	d.SetFilter(csvFilter())
	d.SetFileName("entries.csv")
	d.Show()
}

func (a *App) importCSVDialog() {
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
	n, ok := Resolve(a.state.Selection())
	if !ok || n.Kind == NodeFile {
		dialog.ShowInformation("Nothing to remove", "Select a batch, entry, or addenda first.", a.window)
		return
	}
	dialog.ShowConfirm("Remove record?", "This cannot be undone without Edit → Undo.", func(confirm bool) {
		if !confirm {
			return
		}
		if err := a.state.Mutate(func(file *ach.File) error {
			return removeAtNode(file, n)
		}); err != nil {
			dialog.ShowError(err, a.window)
			return
		}
		a.statusLabel.SetText("Removed record")
	}, a.window)
}

func (a *App) newReturnDialog() {
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
