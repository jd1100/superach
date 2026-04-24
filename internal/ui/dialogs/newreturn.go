package dialogs

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/jd1100/superach/internal/achio"
)

// NewReturn shows the "New Return" wizard. onCreate receives the picked
// return code, the free-form reason text, and the dishonored/contested flags.
func NewReturn(parent fyne.Window, codes []achio.ReturnCodeOption, onCreate func(code, reason string, dishonored, contested bool)) {
	labels := make([]string, 0, len(codes))
	byLabel := make(map[string]string, len(codes))
	for _, c := range codes {
		l := c.Code
		if c.Reason != "" {
			l = fmt.Sprintf("%s — %s", c.Code, c.Reason)
		}
		labels = append(labels, l)
		byLabel[l] = c.Code
	}
	sel := widget.NewSelect(labels, nil)
	sel.SetSelected(labels[0])

	description := widget.NewLabel("")
	description.Wrapping = fyne.TextWrapWord
	lookup := func(code string) string {
		for _, c := range codes {
			if c.Code == code {
				return c.Description
			}
		}
		return ""
	}
	sel.OnChanged = func(s string) { description.SetText(lookup(byLabel[s])) }
	description.SetText(lookup(byLabel[sel.Selected]))

	reason := widget.NewMultiLineEntry()
	reason.SetPlaceHolder("Optional addenda information")
	reason.Wrapping = fyne.TextWrapWord

	dishonored := widget.NewCheck("Dishonored return", nil)
	contested := widget.NewCheck("Contested dishonored return", nil)

	form := widget.NewForm(
		widget.NewFormItem("Return Code", sel),
		widget.NewFormItem("Description", description),
		widget.NewFormItem("Reason / Addenda Info", reason),
		widget.NewFormItem("Dishonored", dishonored),
		widget.NewFormItem("Contested", contested),
	)

	d := dialog.NewCustomConfirm("New Return", "Create", "Cancel",
		container.NewPadded(form),
		func(confirm bool) {
			if !confirm {
				return
			}
			onCreate(byLabel[sel.Selected], reason.Text, dishonored.Checked, contested.Checked)
		}, parent)
	d.Resize(fyne.NewSize(520, 440))
	d.Show()
}
