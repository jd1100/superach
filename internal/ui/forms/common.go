// Package forms contains the Fyne widgets that bind to moov-io/ach record
// structs. Each exported constructor returns a fyne.CanvasObject and wires
// OnChanged callbacks directly to the record pointer, so edits are visible
// to the tree/validator as soon as the user types. The save callback is
// invoked on blur/submit so the parent can trigger recalc + dirty.
package forms

import (
	"strconv"

	"fyne.io/fyne/v2/widget"
)

func stringEntry(cur string, set func(string)) *widget.Entry {
	e := widget.NewEntry()
	e.SetText(cur)
	e.OnChanged = set
	return e
}

func intEntry(cur int, set func(int)) *widget.Entry {
	e := widget.NewEntry()
	e.SetText(strconv.Itoa(cur))
	e.OnChanged = func(s string) {
		if s == "" {
			set(0)
			return
		}
		if n, err := strconv.Atoi(s); err == nil {
			set(n)
		}
	}
	return e
}

func amountEntry(cur int, set func(int)) *widget.Entry {
	e := widget.NewEntry()
	e.SetText(centsToDollars(cur))
	e.OnChanged = func(s string) {
		set(dollarsToCents(s))
	}
	return e
}

func centsToDollars(c int) string {
	neg := ""
	if c < 0 {
		neg = "-"
		c = -c
	}
	return neg + strconv.Itoa(c/100) + "." + pad2(c%100)
}

func pad2(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

func dollarsToCents(s string) int {
	neg := false
	if len(s) > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
	}
	dot := -1
	for i, c := range s {
		if c == '.' {
			dot = i
			break
		}
	}
	var dollars, cents int
	if dot < 0 {
		dollars, _ = strconv.Atoi(s)
	} else {
		dollars, _ = strconv.Atoi(s[:dot])
		frac := s[dot+1:]
		if len(frac) >= 2 {
			cents, _ = strconv.Atoi(frac[:2])
		} else if len(frac) == 1 {
			cents, _ = strconv.Atoi(frac + "0")
		}
	}
	total := dollars*100 + cents
	if neg {
		total = -total
	}
	return total
}

func selectField(cur string, options []string, set func(string)) *widget.Select {
	sel := widget.NewSelect(options, set)
	sel.SetSelected(cur)
	return sel
}
