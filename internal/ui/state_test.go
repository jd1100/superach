package ui

import (
	"testing"

	"github.com/moov-io/ach"
	"github.com/stretchr/testify/require"

	"github.com/jd1100/superach/internal/achio"
)

const samplePPD = "../../testdata/ppd-debit.ach"

func loadTestState(t *testing.T) *AppState {
	t.Helper()
	f, err := achio.ReadFile(samplePPD)
	require.NoError(t, err)
	s := NewState()
	s.LoadFile(f, samplePPD)
	s.SetReadOnly(false)
	return s
}

func TestDirty_RoundTripsWithUndo(t *testing.T) {
	s := loadTestState(t)

	require.False(t, s.Dirty(), "fresh state should not be dirty")

	err := s.Mutate(func(f *ach.File) error {
		f.Header.ImmediateOriginName = "CHANGED"
		return nil
	})
	require.NoError(t, err)
	require.True(t, s.Dirty(), "after mutation, state must be dirty")

	require.True(t, s.Undo(), "undo must succeed")
	require.False(t, s.Dirty(), "after undo back to save-point, state must be clean")
}

func TestDirty_MultipleUndoThenSave(t *testing.T) {
	s := loadTestState(t)

	for i := 0; i < 3; i++ {
		require.NoError(t, s.Mutate(func(f *ach.File) error { return nil }))
	}
	require.True(t, s.Dirty())

	s.MarkSaved("/tmp/fake.ach")
	require.False(t, s.Dirty(), "after MarkSaved the file matches disk")

	require.NoError(t, s.Mutate(func(f *ach.File) error { return nil }))
	require.True(t, s.Dirty())
	require.True(t, s.Undo())
	require.False(t, s.Dirty(), "undoing the post-save mutation returns to save-point")
}

func TestMutate_RollbackOnError(t *testing.T) {
	s := loadTestState(t)
	origName := s.File().Header.ImmediateOriginName

	err := s.Mutate(func(f *ach.File) error {
		f.Header.ImmediateOriginName = "SHOULD_ROLL_BACK"
		return errRollback
	})
	require.Error(t, err)
	require.Equal(t, origName, s.File().Header.ImmediateOriginName, "file pointer must roll back")
	require.False(t, s.Dirty(), "failed mutation must not mark dirty")
}

func TestUndoAtCap_Signals(t *testing.T) {
	s := loadTestState(t)
	for i := 0; i < maxUndo-1; i++ {
		require.NoError(t, s.Mutate(func(f *ach.File) error { return nil }))
	}
	require.False(t, s.UndoAtCap())
	require.NoError(t, s.Mutate(func(f *ach.File) error { return nil }))
	require.True(t, s.UndoAtCap())
}

// TestDirty_RecoversAfterCapOverflowSave asserts that once the undo stack
// overflows (savePointReachable becomes false) a subsequent save re-anchors
// the save-point so the file can legitimately report clean again. Previously
// savedCount was clobbered to -1 permanently, so Dirty stayed true forever.
func TestDirty_RecoversAfterCapOverflowSave(t *testing.T) {
	s := loadTestState(t)
	// Overflow the stack.
	for i := 0; i < maxUndo+1; i++ {
		require.NoError(t, s.Mutate(func(f *ach.File) error { return nil }))
	}
	require.True(t, s.Dirty(), "post-overflow must be dirty")

	s.MarkSaved("/tmp/fake.ach")
	require.False(t, s.Dirty(), "save after overflow re-anchors the save-point")

	// A second overflow cycle should also recover after a save.
	for i := 0; i < maxUndo+1; i++ {
		require.NoError(t, s.Mutate(func(f *ach.File) error { return nil }))
	}
	require.True(t, s.Dirty())
	s.MarkSaved("/tmp/fake.ach")
	require.False(t, s.Dirty(), "second overflow+save must also report clean")
}

func TestRecentFiles_MRU(t *testing.T) {
	s := NewState()
	for i := 0; i < 10; i++ {
		s.rememberRecent(intPath(i))
	}
	got := s.RecentFiles()
	require.LessOrEqual(t, len(got), 8)
	require.Equal(t, intPath(9), got[0], "most recent first")

	s.rememberRecent(intPath(5))
	got = s.RecentFiles()
	require.Equal(t, intPath(5), got[0])
	seen := 0
	for _, p := range got {
		if p == intPath(5) {
			seen++
		}
	}
	require.Equal(t, 1, seen)
}

func intPath(i int) string { return "/tmp/test" + string(rune('0'+i)) + ".ach" }

type rollbackErr struct{}

func (rollbackErr) Error() string { return "rollback" }

var errRollback = rollbackErr{}
