package dialogs

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/moov-io/ach"
)

// NewBatch pops the "New Batch" dialog and calls onCreate when the user
// confirms. onCreate arguments: (sec, companyName, companyID, effective, odfi, serviceClass).
func NewBatch(parent fyne.Window, onCreate func(sec, company, companyID, effective, odfi string, serviceClass int)) {
	secOpts := []string{
		ach.PPD, ach.CCD, ach.CTX, ach.WEB, ach.TEL, ach.ARC, ach.BOC, ach.POP, ach.RCK, ach.COR,
	}
	secSel := widget.NewSelect(secOpts, nil)
	secSel.SetSelected(ach.PPD)

	svcOpts := []string{
		strconv.Itoa(ach.MixedDebitsAndCredits) + " — Mixed",
		strconv.Itoa(ach.CreditsOnly) + " — Credits Only",
		strconv.Itoa(ach.DebitsOnly) + " — Debits Only",
	}
	svcSel := widget.NewSelect(svcOpts, nil)
	svcSel.SetSelected(svcOpts[0])

	company := widget.NewEntry()
	companyID := widget.NewEntry()
	effective := widget.NewEntry()
	effective.SetPlaceHolder("YYMMDD")
	odfi := widget.NewEntry()
	odfi.SetPlaceHolder("8-digit ODFI ID")

	form := widget.NewForm(
		widget.NewFormItem("SEC Code", secSel),
		widget.NewFormItem("Service Class", svcSel),
		widget.NewFormItem("Company Name", company),
		widget.NewFormItem("Company ID (9)", companyID),
		widget.NewFormItem("Effective Date", effective),
		widget.NewFormItem("ODFI Identification", odfi),
	)

	d := dialog.NewCustomConfirm("New Batch", "Create", "Cancel",
		container.NewPadded(form),
		func(confirm bool) {
			if !confirm {
				return
			}
			onCreate(secSel.Selected, company.Text, companyID.Text, effective.Text, odfi.Text, svcFromLabel(svcSel.Selected))
		}, parent)
	d.Resize(fyne.NewSize(420, 320))
	d.Show()
}

func svcFromLabel(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			if i == 0 {
				continue
			}
			n, _ := strconv.Atoi(s[:i])
			return n
		}
	}
	return ach.MixedDebitsAndCredits
}
