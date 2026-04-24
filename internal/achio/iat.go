package achio

import (
	"fmt"

	"github.com/moov-io/ach"
)

// IATOriginator carries the address fields needed to populate the
// originator-related IAT addenda (10, 11, 12).
type IATOriginator struct {
	Name              string
	StreetAddress     string
	CityStateProvince string
	CountryPostalCode string
	Identification    string
}

// IATReceiver carries the address/identification fields needed for the
// receiver-related IAT addenda (15, 16).
type IATReceiver struct {
	IDNumber          string
	StreetAddress     string
	CityStateProvince string
	CountryPostalCode string
}

// IATBank carries the bank-side fields needed for ODFI/RDFI addenda (13, 14).
type IATBank struct {
	Name                string
	IDNumberQualifier   string
	Identification      string
	BranchCountryCode   string
}

// NewIATBatch creates a configured IATBatch and appends it to the file.
func NewIATBatch(f *ach.File, bh *ach.IATBatchHeader) (*ach.IATBatch, error) {
	if f == nil || bh == nil {
		return nil, fmt.Errorf("nil file or header")
	}
	if bh.BatchNumber == 0 {
		bh.BatchNumber = nextBatchNumber(f)
	}
	b := ach.NewIATBatch(bh)
	f.AddIATBatch(b)
	return &f.IATBatches[len(f.IATBatches)-1], nil
}

// FillMandatoryAddenda pre-populates Addenda10–16 on an IAT entry from a
// small set of structured inputs. Callers can then tweak fields in the form.
func FillMandatoryAddenda(e *ach.IATEntryDetail, txnTypeCode string, originator IATOriginator, odfi, rdfi IATBank, receiver IATReceiver) {
	if e == nil {
		return
	}
	a10 := ach.NewAddenda10()
	a10.TransactionTypeCode = txnTypeCode
	a10.ForeignPaymentAmount = e.Amount
	a10.Name = originator.Name
	e.Addenda10 = a10

	a11 := ach.NewAddenda11()
	a11.OriginatorName = originator.Name
	a11.OriginatorStreetAddress = originator.StreetAddress
	e.Addenda11 = a11

	a12 := ach.NewAddenda12()
	a12.OriginatorCityStateProvince = originator.CityStateProvince
	a12.OriginatorCountryPostalCode = originator.CountryPostalCode
	e.Addenda12 = a12

	a13 := ach.NewAddenda13()
	a13.ODFIName = odfi.Name
	a13.ODFIIDNumberQualifier = odfi.IDNumberQualifier
	a13.ODFIIdentification = odfi.Identification
	a13.ODFIBranchCountryCode = odfi.BranchCountryCode
	e.Addenda13 = a13

	a14 := ach.NewAddenda14()
	a14.RDFIName = rdfi.Name
	a14.RDFIIDNumberQualifier = rdfi.IDNumberQualifier
	a14.RDFIIdentification = rdfi.Identification
	a14.RDFIBranchCountryCode = rdfi.BranchCountryCode
	e.Addenda14 = a14

	a15 := ach.NewAddenda15()
	a15.ReceiverIDNumber = receiver.IDNumber
	a15.ReceiverStreetAddress = receiver.StreetAddress
	e.Addenda15 = a15

	a16 := ach.NewAddenda16()
	a16.ReceiverCityStateProvince = receiver.CityStateProvince
	a16.ReceiverCountryPostalCode = receiver.CountryPostalCode
	e.Addenda16 = a16

	e.AddendaRecords = 7
	e.AddendaRecordIndicator = 1
}
