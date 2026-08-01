package ui

import (
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"noto/internal/config"
	"noto/internal/index"
	"noto/internal/storage"
)

// seedNotes writes fixed-timestamp notes to notesDir and syncs idx against
// it, so the resulting list output is fully deterministic.
func seedNotes(t *testing.T, notesDir string, idx *index.DB) {
	t.Helper()

	fixtures := []struct {
		id, title string
		tags      []string
		updatedAt string
	}{
		{"018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "一番新しいメモ", []string{"work"}, "2026-03-01T09:00:00Z"},
		{"018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "二番目のメモ", nil, "2026-02-01T09:00:00Z"},
		{"018f2e4a-cccc-cccc-cccc-cccccccccccc", "一番古いメモ", nil, "2026-01-01T09:00:00Z"},
	}

	for _, f := range fixtures {
		ts, err := time.Parse(time.RFC3339, f.updatedAt)
		if err != nil {
			t.Fatalf("failed to parse fixture time %q: %v", f.updatedAt, err)
		}
		n := storage.Note{ID: f.id, Title: f.title, Tags: f.tags, CreatedAt: ts, UpdatedAt: ts}
		path, err := storage.GeneratePath(notesDir, n)
		if err != nil {
			t.Fatalf("storage.GeneratePath() returned error: %v", err)
		}
		if err := storage.Write(path, n); err != nil {
			t.Fatalf("storage.Write() returned error: %v", err)
		}
	}

	if err := idx.Sync(notesDir); err != nil {
		t.Fatalf("Sync() returned error: %v", err)
	}
}

func TestListScreen_ShowsNotesAtStartup(t *testing.T) {
	notesDir := t.TempDir()
	cfg := config.Config{NotesDir: notesDir}
	idx, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open() returned error: %v", err)
	}
	t.Cleanup(func() { idx.Close() })

	seedNotes(t, notesDir, idx)

	m := New(cfg, idx)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	// All three titles should be visible, and the most recently updated
	// note (first in the list) should carry the cursor.
	teatest.WaitFor(t, out, func(bts []byte) bool {
		return containsBytes("> 一番新しいメモ")(bts) &&
			containsBytes("二番目のメモ")(bts) &&
			containsBytes("一番古いメモ")(bts)
	}, teatest.WithDuration(2*time.Second))

	if err := tm.Quit(); err != nil {
		t.Fatalf("Quit() returned error: %v", err)
	}
	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(Model)
	if final.cursor != 0 {
		t.Errorf("cursor = %d, want 0", final.cursor)
	}
	if len(final.notes) != 3 {
		t.Errorf("len(notes) = %d, want 3", len(final.notes))
	}
}

func TestListScreen_CursorMovesWithJ(t *testing.T) {
	notesDir := t.TempDir()
	cfg := config.Config{NotesDir: notesDir}
	idx, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open() returned error: %v", err)
	}
	t.Cleanup(func() { idx.Close() })

	seedNotes(t, notesDir, idx)

	m := New(cfg, idx)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	teatest.WaitFor(t, out, containsBytes("> 一番新しいメモ"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	teatest.WaitFor(t, out, containsBytes("> 二番目のメモ"), teatest.WithDuration(2*time.Second))

	if err := tm.Quit(); err != nil {
		t.Fatalf("Quit() returned error: %v", err)
	}
	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(Model)
	if final.cursor != 1 {
		t.Errorf("cursor = %d, want 1", final.cursor)
	}
}

func TestUpdateList_CursorClampsAtBounds(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.notes = []index.NoteSummary{{ID: "a"}, {ID: "b"}, {ID: "c"}}

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = got.(Model)
	if m.cursor != 0 {
		t.Fatalf("cursor after k at top = %d, want 0", m.cursor)
	}

	for range m.notes {
		got, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		m = got.(Model)
	}
	if m.cursor != len(m.notes)-1 {
		t.Fatalf("cursor after repeated j = %d, want %d", m.cursor, len(m.notes)-1)
	}

	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = got.(Model)
	if m.cursor != len(m.notes)-1 {
		t.Errorf("cursor after j past bottom = %d, want %d", m.cursor, len(m.notes)-1)
	}
}
