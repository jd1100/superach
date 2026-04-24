package achio_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/moov-io/ach"
	"github.com/stretchr/testify/require"

	"github.com/jd1100/superach/internal/achio"
)

const (
	samplePPD = "../../testdata/ppd-debit.ach"
	sampleIAT = "../../testdata/iat-credit.ach"
	sampleCOR = "../../testdata/cor-example.ach"
)

func TestReadWriteRoundTrip(t *testing.T) {
	for _, p := range []string{samplePPD, sampleIAT, sampleCOR} {
		t.Run(filepath.Base(p), func(t *testing.T) {
			f, err := achio.ReadFile(p)
			require.NoError(t, err)
			require.NotNil(t, f)

			out, err := achio.WriteBytes(f)
			require.NoError(t, err)
			require.NotEmpty(t, out)

			f2, err := achio.ReadBytes(out)
			require.NoError(t, err)
			require.Equal(t, len(f.Batches), len(f2.Batches))
			require.Equal(t, len(f.IATBatches), len(f2.IATBatches))
		})
	}
}

func TestJSONRoundTrip(t *testing.T) {
	f, err := achio.ReadFile(samplePPD)
	require.NoError(t, err)

	bs, err := achio.ToJSON(f)
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(bytes.TrimSpace(bs), []byte("{")))

	f2, err := achio.FromJSON(bs)
	require.NoError(t, err)
	require.Equal(t, f.Header.ImmediateOrigin, f2.Header.ImmediateOrigin)
	require.Equal(t, len(f.Batches), len(f2.Batches))
}

func TestCSVExportImportRoundTrip(t *testing.T) {
	src, err := achio.ReadFile(samplePPD)
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, achio.EntriesToCSV(&buf, src))
	require.Greater(t, buf.Len(), 0)

	dst, err := achio.ImportCSV(bytes.NewReader(buf.Bytes()), nil,
		src.Header.ImmediateOrigin, src.Header.ImmediateOriginName,
		src.Header.ImmediateDestination, src.Header.ImmediateDestinationName)
	require.NoError(t, err)
	require.Equal(t, len(src.Batches), len(dst.Batches))

	srcEntries := src.Batches[0].GetEntries()
	dstEntries := dst.Batches[0].GetEntries()
	require.Equal(t, len(srcEntries), len(dstEntries))
	require.Equal(t, srcEntries[0].Amount, dstEntries[0].Amount)
}

func TestValidateClean(t *testing.T) {
	f, err := achio.ReadFile(samplePPD)
	require.NoError(t, err)
	errs, err := achio.ValidateFile(f)
	require.NoError(t, err)
	require.Empty(t, errs)
}

func TestAddRemoveBatch(t *testing.T) {
	f, err := achio.ReadFile(samplePPD)
	require.NoError(t, err)
	originalCount := len(f.Batches)

	bh := ach.NewBatchHeader()
	bh.ServiceClassCode = ach.MixedDebitsAndCredits
	bh.StandardEntryClassCode = ach.PPD
	bh.CompanyName = "Test Co"
	bh.CompanyIdentification = "121042882"
	bh.CompanyEntryDescription = "TEST"
	bh.EffectiveEntryDate = "210601"
	bh.ODFIIdentification = "12104288"

	newBatch, err := achio.AddBatch(f, bh)
	require.NoError(t, err)
	require.NotNil(t, newBatch)
	require.Equal(t, originalCount+1, len(f.Batches))

	require.NoError(t, achio.RemoveBatchAt(f, len(f.Batches)-1))
	require.Equal(t, originalCount, len(f.Batches))
}

func TestBuildReturn(t *testing.T) {
	f, err := achio.ReadFile(samplePPD)
	require.NoError(t, err)
	originalBatches := len(f.Batches)
	require.NotEmpty(t, f.Batches)
	require.NotEmpty(t, f.Batches[0].GetEntries())

	idx, err := achio.BuildReturn(f, achio.ReturnRequest{
		OriginalBatchIndex: 0,
		OriginalEntryIndex: 0,
		ReturnCode:         "R01",
		Reason:             "INSUFFICIENT FUNDS",
	})
	require.NoError(t, err)
	require.Equal(t, originalBatches, idx)
	require.Equal(t, originalBatches+1, len(f.Batches))

	retEntry := f.Batches[idx].GetEntries()[0]
	require.NotNil(t, retEntry.Addenda99)
	require.Equal(t, "R01", retEntry.Addenda99.ReturnCode)
	require.Equal(t, ach.CategoryReturn, retEntry.Category)
}

func TestBuildNOC(t *testing.T) {
	f, err := achio.ReadFile(samplePPD)
	require.NoError(t, err)
	originalBatches := len(f.Batches)

	_, err = achio.BuildNOC(f, achio.NOCRequest{
		OriginalBatchIndex: 0,
		OriginalEntryIndex: 0,
		ChangeCode:         "C01",
		Corrected:          ach.CorrectedData{AccountNumber: "987654321"},
	})
	require.NoError(t, err)
	require.Equal(t, originalBatches+1, len(f.Batches))

	corBatch := f.Batches[originalBatches]
	require.Equal(t, ach.COR, corBatch.GetHeader().StandardEntryClassCode)
	corEntry := corBatch.GetEntries()[0]
	require.NotNil(t, corEntry.Addenda98)
	require.Equal(t, "C01", corEntry.Addenda98.ChangeCode)
	require.Equal(t, ach.CategoryNOC, corEntry.Category)
}

func TestWriteAndReReadFile(t *testing.T) {
	f, err := achio.ReadFile(samplePPD)
	require.NoError(t, err)

	dir := t.TempDir()
	out := filepath.Join(dir, "out.ach")
	require.NoError(t, achio.WriteFile(out, f))

	data, err := os.ReadFile(out)
	require.NoError(t, err)
	require.True(t, len(data) > 0)

	f2, err := achio.ReadFile(out)
	require.NoError(t, err)
	require.Equal(t, len(f.Batches), len(f2.Batches))
}

func TestClone(t *testing.T) {
	f, err := achio.ReadFile(samplePPD)
	require.NoError(t, err)
	c, err := achio.Clone(f)
	require.NoError(t, err)
	require.Equal(t, len(f.Batches), len(c.Batches))
	// Mutate clone, original must not change
	c.Header.ImmediateOriginName = "MUTATED"
	require.NotEqual(t, f.Header.ImmediateOriginName, c.Header.ImmediateOriginName)
}

// TestWriteFile_SECSwitchedToADV_NoPanic reproduces the nil-pointer crash the
// user hit: a PPD batch whose header's SEC code was changed to "ADV" after
// construction (via the Batch Header form) would make File.Create dereference
// a nil ADVControl. WriteFile must return an error instead of killing the
// process.
func TestWriteFile_SECSwitchedToADV_NoPanic(t *testing.T) {
	f, err := achio.ReadFile(samplePPD)
	require.NoError(t, err)
	require.NotEmpty(t, f.Batches)

	f.Batches[0].GetHeader().StandardEntryClassCode = ach.ADV

	dir := t.TempDir()
	out := filepath.Join(dir, "out.ach")

	werr := achio.WriteFile(out, f)
	require.Error(t, werr, "expected friendly error, not a panic")
	require.Contains(t, werr.Error(), "SEC Code")
}

// TestWriteBytes_SECSwitchedToADV_NoPanic covers the same crash via the
// NACHA-bytes serialiser used for clipboard/preview.
func TestWriteBytes_SECSwitchedToADV_NoPanic(t *testing.T) {
	f, err := achio.ReadFile(samplePPD)
	require.NoError(t, err)
	require.NotEmpty(t, f.Batches)

	f.Batches[0].GetHeader().StandardEntryClassCode = ach.ADV

	_, werr := achio.WriteBytes(f)
	require.Error(t, werr, "expected friendly error, not a panic")
}

// TestRecalculate_SECSwitchedToADV_NoPanic covers the post-mutation recalc
// path that runs after every form edit.
func TestRecalculate_SECSwitchedToADV_NoPanic(t *testing.T) {
	f, err := achio.ReadFile(samplePPD)
	require.NoError(t, err)
	require.NotEmpty(t, f.Batches)

	f.Batches[0].GetHeader().StandardEntryClassCode = ach.ADV

	require.Error(t, achio.Recalculate(f), "expected friendly error, not a panic")
}

func TestImportCSV_RejectsBadAmount(t *testing.T) {
	// A typo in AmountCents ("100X") used to silently convert to 0 cents,
	// shipping a $0 payment. The import must now fail loudly with row context.
	src, err := achio.ReadFile(samplePPD)
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, achio.EntriesToCSV(&buf, src))

	// Corrupt the first data row's AmountCents (header is csvHeader).
	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	require.Greater(t, len(lines), 2)
	header := string(lines[0])
	require.Contains(t, header, "AmountCents")
	// Locate AmountCents column index.
	cols := bytes.Split(lines[0], []byte(","))
	col := -1
	for i, c := range cols {
		if string(c) == "AmountCents" {
			col = i
			break
		}
	}
	require.GreaterOrEqual(t, col, 0)
	row1 := bytes.Split(lines[1], []byte(","))
	row1[col] = []byte("100X")
	lines[1] = bytes.Join(row1, []byte(","))

	_, ierr := achio.ImportCSV(bytes.NewReader(bytes.Join(lines, []byte("\n"))), nil,
		src.Header.ImmediateOrigin, src.Header.ImmediateOriginName,
		src.Header.ImmediateDestination, src.Header.ImmediateDestinationName)
	require.Error(t, ierr, "expected parse error on bad AmountCents")
	require.Contains(t, ierr.Error(), "AmountCents")
	require.Contains(t, ierr.Error(), "row 2") // 1-based CSV line number
}

func TestImportCSV_RejectsEmptyAmount(t *testing.T) {
	src, err := achio.ReadFile(samplePPD)
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, achio.EntriesToCSV(&buf, src))

	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	cols := bytes.Split(lines[0], []byte(","))
	col := -1
	for i, c := range cols {
		if string(c) == "AmountCents" {
			col = i
			break
		}
	}
	row1 := bytes.Split(lines[1], []byte(","))
	row1[col] = []byte("")
	lines[1] = bytes.Join(row1, []byte(","))

	_, ierr := achio.ImportCSV(bytes.NewReader(bytes.Join(lines, []byte("\n"))), nil,
		src.Header.ImmediateOrigin, src.Header.ImmediateOriginName,
		src.Header.ImmediateDestination, src.Header.ImmediateDestinationName)
	require.Error(t, ierr, "expected error on empty AmountCents")
}

func TestReturnCodesAndChangeCodes(t *testing.T) {
	rcs := achio.AllReturnCodes()
	require.Len(t, rcs, 85)
	require.Equal(t, "R01", rcs[0].Code)

	ccs := achio.AllChangeCodes()
	require.Len(t, ccs, 14)
	require.Equal(t, "C01", ccs[0].Code)
}
