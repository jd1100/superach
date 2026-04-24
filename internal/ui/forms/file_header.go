package forms

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/moov-io/ach"
)

// FileHeaderForm returns a form bound to h. save is invoked on explicit
// submit so the parent can trigger File.Create and refresh the tree.
func FileHeaderForm(h *ach.FileHeader, save func()) fyne.CanvasObject {
	form := widget.NewForm(
		widget.NewFormItem("Immediate Destination", stringEntry(h.ImmediateDestination, func(v string) { h.ImmediateDestination = v })),
		widget.NewFormItem("Destination Name", stringEntry(h.ImmediateDestinationName, func(v string) { h.ImmediateDestinationName = v })),
		widget.NewFormItem("Immediate Origin", stringEntry(h.ImmediateOrigin, func(v string) { h.ImmediateOrigin = v })),
		widget.NewFormItem("Origin Name", stringEntry(h.ImmediateOriginName, func(v string) { h.ImmediateOriginName = v })),
		widget.NewFormItem("File Creation Date (YYMMDD)", dateEntry(h.FileCreationDate, func(v string) { h.FileCreationDate = v })),
		widget.NewFormItem("File Creation Time (HHmm)", timeEntry(h.FileCreationTime, func(v string) { h.FileCreationTime = v })),
		widget.NewFormItem("File ID Modifier", stringEntry(h.FileIDModifier, func(v string) { h.FileIDModifier = v })),
		widget.NewFormItem("Reference Code", stringEntry(h.ReferenceCode, func(v string) { h.ReferenceCode = v })),
	)
	attachSubmit(form, save)
	return form
}
