package dialogs

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/moov-io/ach"

	"github.com/jd1100/superach/internal/achio"
)

// CorrectedInput mirrors ach.CorrectedData but is decoupled so the dialogs
// package doesn't leak moov-io types to callers who want to stub it.
type CorrectedInput struct {
	AccountNumber   string
	RoutingNumber   string
	Name            string
	TransactionCode int
	Identification  string
}

// ToAch converts to the moov-io struct.
func (c CorrectedInput) ToAch() ach.CorrectedData {
	return ach.CorrectedData{
		AccountNumber:   c.AccountNumber,
		RoutingNumber:   c.RoutingNumber,
		Name:            c.Name,
		TransactionCode: c.TransactionCode,
		Identification:  c.Identification,
	}
}

// NewNOC shows the "New NOC" wizard. The form shows only the fields
// appropriate for the selected change code.
func NewNOC(parent fyne.Window, codes []achio.ChangeCodeOption, onCreate func(code string, corrected CorrectedInput)) {
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

	account := widget.NewEntry()
	routing := widget.NewEntry()
	name := widget.NewEntry()
	txnCode := widget.NewEntry()
	identification := widget.NewEntry()

	accountItem := widget.NewFormItem("New Account Number", account)
	routingItem := widget.NewFormItem("New Routing Number", routing)
	nameItem := widget.NewFormItem("New Individual Name", name)
	txnItem := widget.NewFormItem("New Transaction Code", txnCode)
	idItem := widget.NewFormItem("New Identification", identification)

	form := widget.NewForm(
		widget.NewFormItem("Change Code", sel),
		widget.NewFormItem("Description", description),
	)

	show := func(items ...*widget.FormItem) {
		form.Items = []*widget.FormItem{
			widget.NewFormItem("Change Code", sel),
			widget.NewFormItem("Description", description),
		}
		form.Items = append(form.Items, items...)
		form.Refresh()
	}

	apply := func(code string) {
		for _, c := range codes {
			if c.Code == code {
				description.SetText(c.Description)
			}
		}
		switch code {
		case "C01":
			show(accountItem)
		case "C02":
			show(routingItem)
		case "C03":
			show(routingItem, accountItem)
		case "C04":
			show(nameItem)
		case "C05":
			show(txnItem)
		case "C06":
			show(accountItem, txnItem)
		case "C07":
			show(routingItem, accountItem, txnItem)
		case "C09":
			show(idItem)
		default:
			show(accountItem, routingItem, nameItem, txnItem, idItem)
		}
	}
	sel.OnChanged = func(s string) { apply(byLabel[s]) }
	apply(byLabel[sel.Selected])

	d := dialog.NewCustomConfirm("New Notification of Change", "Create", "Cancel",
		container.NewPadded(form),
		func(confirm bool) {
			if !confirm {
				return
			}
			txn, _ := strconv.Atoi(txnCode.Text)
			onCreate(byLabel[sel.Selected], CorrectedInput{
				AccountNumber:   account.Text,
				RoutingNumber:   routing.Text,
				Name:            name.Text,
				TransactionCode: txn,
				Identification:  identification.Text,
			})
		}, parent)
	d.Resize(fyne.NewSize(520, 440))
	d.Show()
}
