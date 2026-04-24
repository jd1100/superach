// Package dialogs wraps modal Fyne workflows that don't fit into the
// master-detail form pane (new batch / new return / new NOC wizards,
// CSV import, about).
package dialogs

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// ShowAbout pops the standard about dialog.
func ShowAbout(parent fyne.Window) {
	body := container.NewVBox(
		widget.NewLabelWithStyle("SuperACH", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Cross-platform NACHA ACH viewer & editor."),
		widget.NewLabel("Built with Fyne and moov-io/ach."),
	)
	dialog.ShowCustom("About SuperACH", "Close", body, parent)
}
