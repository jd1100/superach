package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"github.com/moov-io/ach"

	"github.com/jd1100/superach/internal/achio"
	"github.com/jd1100/superach/internal/ui/forms"
)

// App is the top-level Fyne application shell.
type App struct {
	fyneApp     fyne.App
	window      fyne.Window
	state       *AppState
	detail      *Detail
	tree        *widget.Tree
	searchEntry *widget.Entry
	statusLabel *widget.Label
	errorList   *widget.List
	errors      []achio.FieldError
}

// Run constructs and blocks on the Fyne event loop.
func Run() {
	a := NewApp()
	a.window.ShowAndRun()
}

// NewApp wires the window, menu, and layout but does not show the window.
func NewApp() *App {
	fyneApp := fyneapp.NewWithID("io.superach.app")
	state := NewState()
	win := fyneApp.NewWindow("SuperACH — ACH Viewer & Editor")

	a := &App{
		fyneApp:     fyneApp,
		window:      win,
		state:       state,
		statusLabel: widget.NewLabel("Ready."),
	}

	// Sync the forms package with the initial read-only flag before any
	// form is built so widgets come up disabled from the first render.
	forms.SetReadOnly(state.ReadOnly())

	// Restore MRU list from Fyne preferences so the recent-files menu
	// survives restarts.
	if paths := fyneApp.Preferences().StringList(prefRecentFiles); len(paths) > 0 {
		state.SetRecentFiles(paths)
	}

	a.detail = NewDetail(state)
	a.detail.AttachParent(win)
	a.tree = BuildTree(state, func(n Node) { a.detail.Render(n) })
	a.searchEntry = widget.NewEntry()
	a.searchEntry.SetPlaceHolder("Filter: trace #, name, amount…")
	a.searchEntry.OnChanged = func(q string) {
		setFilter(q) // invalidates the memoized visible set
		a.tree.Refresh()
	}
	a.errorList = widget.NewList(
		func() int { return len(a.errors) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			lbl := obj.(*widget.Label)
			if i < len(a.errors) {
				lbl.SetText(a.errors[i].Error())
			}
		},
	)

	// Persist the MRU list on change so Open Recent survives restarts, but
	// do NOT call win.SetMainMenu again — a second SetMainMenu on macOS in
	// Fyne v2.7.3 crashes with SIGSEGV in glfwPollEvents the next time the
	// user opens any menu (fyne-io/fyne#6264: separator items are autoreleased
	// but insertDarwinMenuItem releases them unconditionally, so resetDarwin
	// Menu later dereferences freed pointers). The Open Recent submenu
	// therefore reflects startup state plus whatever was persisted — the
	// trade-off is acceptable vs. a hard crash.
	lastRecent := a.state.RecentFiles()
	state.Subscribe(func() {
		forms.SetReadOnly(state.ReadOnly())
		a.refreshTitle()
		a.refreshErrors()
		cur := a.state.RecentFiles()
		if !stringSliceEqual(cur, lastRecent) {
			lastRecent = cur
			a.persistRecent()
		}
	})

	split := container.NewHSplit(wrapTree(a.tree, a.searchEntry), a.detail.CanvasObject())
	split.Offset = 0.32

	errPane := container.NewBorder(
		widget.NewLabelWithStyle("Validation", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil, nil, nil, a.errorList)
	body := container.NewVSplit(split, errPane)
	body.Offset = 0.78

	// Menu construction also registers keyboard shortcuts on the canvas, so
	// it has to come after the window exists.
	win.SetMainMenu(a.buildMenu())
	win.SetContent(container.NewBorder(nil, a.statusLabel, nil, nil, body))
	win.Resize(fyne.NewSize(1280, 820))
	win.SetCloseIntercept(func() {
		a.confirmDiscard(func() { win.Close() })
	})
	a.refreshTitle()
	return a
}

// persistRecent writes the MRU list to Fyne preferences so it survives restarts.
func (a *App) persistRecent() {
	a.fyneApp.Preferences().SetStringList(prefRecentFiles, a.state.RecentFiles())
}

const prefRecentFiles = "recentFiles"

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Window exposes the underlying Fyne window (useful for tests).
func (a *App) Window() fyne.Window { return a.window }

func (a *App) refreshTitle() {
	path := a.state.Path()
	name := "Untitled.ach"
	if path != "" {
		name = filepath.Base(path)
	}
	mark := ""
	if a.state.Dirty() {
		mark = " •"
	}
	mode := "🔒 Read-only"
	if !a.state.ReadOnly() {
		mode = "✏️ Editing"
	}
	a.window.SetTitle(fmt.Sprintf("SuperACH — %s%s  —  %s", name, mark, mode))
}

// toggleReadOnly flips the edit-lock. If the user is exiting edit mode with
// unsaved changes we keep the changes — we just stop accepting new ones.
func (a *App) toggleReadOnly() {
	next := !a.state.ReadOnly()
	a.state.SetReadOnly(next)
	// Force the currently-open detail pane to rebuild with the new mode.
	a.detail.RerenderCurrent()
	if next {
		a.statusLabel.SetText("Read-only mode — View → Enable Editing to make changes.")
	} else {
		a.statusLabel.SetText("Editing enabled. Be careful: every keystroke updates the file.")
	}
}

// requireEditable shows a dialog explaining read-only mode and returns false
// if edits are currently blocked.
func (a *App) requireEditable() bool {
	if !a.state.ReadOnly() {
		return true
	}
	dialog.ShowInformation("File is locked",
		"The file is in read-only mode so you can browse without accidentally changing anything.\n\n"+
			"Choose View → Enable Editing to make changes.",
		a.window)
	return false
}

func (a *App) refreshErrors() {
	f := a.state.File()
	if f == nil {
		a.errors = nil
		a.errorList.Refresh()
		a.statusLabel.SetText("No file.")
		return
	}
	errs, _ := achio.ValidateFile(f)
	a.errors = errs
	a.errorList.Refresh()
	if len(errs) == 0 {
		a.statusLabel.SetText("Valid — " + fileSummary(f))
	} else {
		a.statusLabel.SetText(fmt.Sprintf("%d validation error(s) — %s", len(errs), fileSummary(f)))
	}
}

func fileSummary(f *ach.File) string {
	if f == nil {
		return ""
	}
	return fmt.Sprintf("%d batch(es), %d IAT batch(es)", len(f.Batches), len(f.IATBatches))
}

// loadPath is called by the Open dialog.
func (a *App) loadPath(path string) {
	f, err := achio.ReadFile(path)
	if err != nil {
		dialog.ShowError(friendlyReadError(path, err), a.window)
		return
	}
	a.state.LoadFile(f, path)
	a.statusLabel.SetText("Loaded " + filepath.Base(path))
}

// friendlyReadError wraps moov-io/ach parse failures with a user-actionable
// hint. The underlying library returns messages like "invalid record type
// 'F'" which don't point at the fix.
func friendlyReadError(path string, err error) error {
	msg := err.Error()
	hint := ""
	switch {
	case strings.Contains(msg, "invalid record type"):
		hint = "— likely not a NACHA ACH file, or the file is truncated/corrupted. " +
			"If this is a JSON export, rename it to end in .json and retry."
	case strings.Contains(msg, "unexpected EOF"):
		hint = "— the file ended mid-record. Ask the sender to re-export."
	case strings.Contains(msg, "permission denied"):
		hint = "— no read permission. Check the file's owner/group."
	}
	if hint == "" {
		return fmt.Errorf("could not read %s: %w", filepath.Base(path), err)
	}
	return fmt.Errorf("could not read %s: %w %s", filepath.Base(path), err, hint)
}

func (a *App) savePath(path string) {
	f := a.state.File()
	if f == nil {
		return
	}
	if err := achio.WriteFile(path, f); err != nil {
		dialog.ShowError(fmt.Errorf("could not write %s: %w", filepath.Base(path), err), a.window)
		return
	}
	a.state.MarkSaved(path)
	a.statusLabel.SetText("Saved " + filepath.Base(path))
}

func (a *App) confirmDiscard(onContinue func()) {
	if !a.state.Dirty() {
		onContinue()
		return
	}
	dialog.ShowConfirm("Discard changes?", "You have unsaved changes. Continue anyway?", func(ok bool) {
		if ok {
			onContinue()
		}
	}, a.window)
}

func uriPath(u fyne.URI) string {
	if u == nil {
		return ""
	}
	s := u.Path()
	if s == "" {
		s = u.String()
	}
	if strings.HasPrefix(s, "file://") {
		s = s[len("file://"):]
	}
	return s
}

func defaultFilter() storage.FileFilter {
	return storage.NewExtensionFileFilter([]string{".ach", ".json"})
}

func csvFilter() storage.FileFilter {
	return storage.NewExtensionFileFilter([]string{".csv"})
}
