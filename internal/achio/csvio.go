package achio

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/moov-io/ach"
)

var csvHeader = []string{
	"BatchNumber",
	"SECCode",
	"CompanyName",
	"CompanyID",
	"EffectiveEntryDate",
	"TransactionCode",
	"RDFIIdentification",
	"CheckDigit",
	"DFIAccountNumber",
	"AmountCents",
	"IdentificationNumber",
	"IndividualName",
	"DiscretionaryData",
	"TraceNumber",
	"AddendaPaymentInformation",
}

// EntriesToCSV flattens every forward batch's entries into a CSV stream.
// IAT entries are not included (use Export JSON for those).
func EntriesToCSV(w io.Writer, f *ach.File) error {
	if f == nil {
		return fmt.Errorf("nil file")
	}
	cw := csv.NewWriter(w)
	if err := cw.Write(csvHeader); err != nil {
		return err
	}
	for _, b := range f.Batches {
		bh := b.GetHeader()
		if bh == nil {
			continue
		}
		for _, e := range b.GetEntries() {
			row := []string{
				strconv.Itoa(bh.BatchNumber),
				bh.StandardEntryClassCode,
				bh.CompanyName,
				bh.CompanyIdentification,
				bh.EffectiveEntryDate,
				strconv.Itoa(e.TransactionCode),
				e.RDFIIdentification,
				e.CheckDigit,
				strings.TrimSpace(e.DFIAccountNumber),
				strconv.Itoa(e.Amount),
				e.IdentificationNumber,
				e.IndividualName,
				e.DiscretionaryData,
				e.TraceNumber,
				addenda05Payload(e),
			}
			if err := cw.Write(row); err != nil {
				return err
			}
		}
	}
	cw.Flush()
	return cw.Error()
}

func addenda05Payload(e *ach.EntryDetail) string {
	if len(e.Addenda05) == 0 {
		return ""
	}
	parts := make([]string, 0, len(e.Addenda05))
	for _, a := range e.Addenda05 {
		parts = append(parts, strings.TrimSpace(a.PaymentRelatedInformation))
	}
	return strings.Join(parts, " | ")
}

// ExportCSVFile writes a CSV to the given path.
func ExportCSVFile(path string, f *ach.File) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	return EntriesToCSV(out, f)
}

// ImportCSV builds a new ACH file by grouping rows by (SEC, CompanyID).
// Missing/extra columns trigger an error. Existing batches in `into` are
// preserved; new batches are appended.
func ImportCSV(r io.Reader, into *ach.File, originRouting, originName, destRouting, destName string) (*ach.File, error) {
	if into == nil {
		into = ach.NewFile()
		into.Header = ach.NewFileHeader()
		into.Header.ImmediateOrigin = originRouting
		into.Header.ImmediateOriginName = originName
		into.Header.ImmediateDestination = destRouting
		into.Header.ImmediateDestinationName = destName
		into.Header.FileIDModifier = "A"
		now := time.Now()
		into.Header.FileCreationDate = now.Format("060102")
		into.Header.FileCreationTime = now.Format("1504")
	}

	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	rows, err := cr.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return into, fmt.Errorf("csv has no data rows")
	}

	idx, err := mapHeader(rows[0])
	if err != nil {
		return nil, err
	}

	type key struct {
		sec, companyID, companyName, effective string
	}
	// rowWithLine keeps the original CSV line number (1-based, as Excel shows
	// it) so error messages can point the user at the exact row to fix.
	type rowWithLine struct {
		m    map[string]string
		line int
	}
	groups := map[key][]rowWithLine{}
	order := []key{}
	for i, row := range rows[1:] {
		m := map[string]string{}
		for col, ci := range idx {
			if ci >= 0 && ci < len(row) {
				m[col] = row[ci]
			}
		}
		k := key{
			sec:         strings.ToUpper(m["SECCode"]),
			companyID:   m["CompanyID"],
			companyName: m["CompanyName"],
			effective:   m["EffectiveEntryDate"],
		}
		if _, ok := groups[k]; !ok {
			order = append(order, k)
		}
		groups[k] = append(groups[k], rowWithLine{m: m, line: i + 2})
	}

	nextBatchNum := nextBatchNumber(into)
	for _, k := range order {
		bh := ach.NewBatchHeader()
		bh.ServiceClassCode = ach.MixedDebitsAndCredits
		bh.StandardEntryClassCode = k.sec
		bh.CompanyName = k.companyName
		bh.CompanyIdentification = k.companyID
		bh.CompanyEntryDescription = "PAYMENT"
		bh.EffectiveEntryDate = k.effective
		bh.ODFIIdentification = first8(originRouting)
		bh.BatchNumber = nextBatchNum
		nextBatchNum++

		batch, err := ach.NewBatch(bh)
		if err != nil {
			return nil, fmt.Errorf("new %s batch: %w", k.sec, err)
		}
		for _, r := range groups[k] {
			row := r.m
			e := ach.NewEntryDetail()
			tc, err := parseIntStrict(row["TransactionCode"])
			if err != nil {
				return nil, fmt.Errorf("row %d: TransactionCode %q: %w", r.line, row["TransactionCode"], err)
			}
			e.TransactionCode = tc
			e.RDFIIdentification = row["RDFIIdentification"]
			if row["CheckDigit"] == "" {
				cd := ach.CalculateCheckDigit(row["RDFIIdentification"])
				if cd >= 0 {
					e.CheckDigit = strconv.Itoa(cd)
				}
			} else {
				e.CheckDigit = row["CheckDigit"]
			}
			e.DFIAccountNumber = row["DFIAccountNumber"]
			amt, err := parseIntStrict(row["AmountCents"])
			if err != nil {
				return nil, fmt.Errorf("row %d: AmountCents %q: %w", r.line, row["AmountCents"], err)
			}
			e.Amount = amt
			e.IdentificationNumber = row["IdentificationNumber"]
			e.IndividualName = row["IndividualName"]
			e.DiscretionaryData = row["DiscretionaryData"]
			e.TraceNumber = row["TraceNumber"]
			e.Category = ach.CategoryForward
			batch.AddEntry(e)

			if pay := strings.TrimSpace(row["AddendaPaymentInformation"]); pay != "" {
				a := ach.NewAddenda05()
				a.PaymentRelatedInformation = pay
				e.AddAddenda05(a)
				e.AddendaRecordIndicator = 1
			}
		}
		if err := batch.Create(); err != nil {
			return nil, fmt.Errorf("build batch %d: %w", bh.BatchNumber, err)
		}
		into.AddBatch(batch)
	}
	if err := into.Create(); err != nil {
		return nil, fmt.Errorf("rebuild file: %w", err)
	}
	return into, nil
}

func first8(s string) string {
	if len(s) >= 8 {
		return s[:8]
	}
	return s
}

func nextBatchNumber(f *ach.File) int {
	max := 0
	for _, b := range f.Batches {
		if h := b.GetHeader(); h != nil && h.BatchNumber > max {
			max = h.BatchNumber
		}
	}
	for _, b := range f.IATBatches {
		if h := b.GetHeader(); h != nil && h.BatchNumber > max {
			max = h.BatchNumber
		}
	}
	return max + 1
}

func mapHeader(row []string) (map[string]int, error) {
	want := make(map[string]bool, len(csvHeader))
	for _, h := range csvHeader {
		want[h] = true
	}
	idx := map[string]int{}
	for i, col := range row {
		col = strings.TrimSpace(col)
		idx[col] = i
	}
	missing := []string{}
	for _, h := range csvHeader {
		if _, ok := idx[h]; !ok {
			missing = append(missing, h)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing CSV columns: %s", strings.Join(missing, ", "))
	}
	return idx, nil
}

// CSVHeader returns the column order written/expected by the CSV codec.
func CSVHeader() []string {
	out := make([]string, len(csvHeader))
	copy(out, csvHeader)
	return out
}

// parseIntStrict parses an integer exactly; empty or non-numeric input is
// a hard error. The predecessor used strconv.Atoi with the error dropped,
// so typos like "100X" silently became zero.
func parseIntStrict(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("missing value")
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("not an integer")
	}
	return n, nil
}
