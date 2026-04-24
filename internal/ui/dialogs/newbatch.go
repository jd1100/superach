package dialogs

import (
	"errors"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/moov-io/ach"

	"github.com/jd1100/superach/internal/ui/forms"
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
	companyID.Validator = func(s string) error {
		if strings.TrimSpace(s) == "" {
			return errors.New("required")
		}
		if len(s) > 10 {
			return errors.New("max 10 chars")
		}
		return nil
	}
	effective := widget.NewEntry()
	effective.SetPlaceHolder("YYMMDD")
	effective.Validator = requiredDate
	odfi := widget.NewEntry()
	odfi.SetPlaceHolder("8-digit ODFI ID")
	odfi.Validator = func(s string) error { return forms.ValidateDigitsLen(s, 8) }

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
			// dialog.NewCustomConfirm doesn't block OK on invalid state, so
			// re-run each validator here and abort on first failure.
			for _, v := range []*widget.Entry{company, companyID, effective, odfi} {
				if v.Validator != nil {
					if err := v.Validator(v.Text); err != nil {
						dialog.ShowError(err, parent)
						return
					}
				}
			}
			onCreate(secSel.Selected, company.Text, companyID.Text, effective.Text, odfi.Text, svcFromLabel(svcSel.Selected))
		}, parent)
	d.Resize(fyne.NewSize(420, 340))
	d.Show()
	// Focus the first user-typed field so the keyboard lands in a usable
	// place — SEC / Service Class have sensible defaults already.
	if canvas := parent.Canvas(); canvas != nil {
		canvas.Focus(company)
	}
}

func requiredDate(s string) error {
	if strings.TrimSpace(s) == "" {
		return errors.New("required")
	}
	return forms.ValidateYYMMDD(s)
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
