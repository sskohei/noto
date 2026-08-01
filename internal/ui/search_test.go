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

// seedSearchNotes writes three notes with distinct, greppable titles/bodies
// so search filtering is easy to assert on.
func seedSearchNotes(t *testing.T, notesDir string, idx *index.DB) {
	t.Helper()

	fixtures := []struct {
		id, title, body string
		updatedAt       string
	}{
		{"018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "買い物リスト", "牛乳を買う", "2026-01-03T00:00:00Z"},
		{"018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "会議メモ", "来週の会議", "2026-01-02T00:00:00Z"},
		{"018f2e4a-cccc-cccc-cccc-cccccccccccc", "冷蔵庫の掃除", "掃除する予定", "2026-01-01T00:00:00Z"},
	}

	for _, f := range fixtures {
		ts, err := time.Parse(time.RFC3339, f.updatedAt)
		if err != nil {
			t.Fatalf("failed to parse fixture time %q: %v", f.updatedAt, err)
		}
		n := storage.Note{ID: f.id, Title: f.title, Body: f.body, CreatedAt: ts, UpdatedAt: ts}
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

func newSearchTestModel(t *testing.T) (Model, string, *index.DB) {
	t.Helper()
	notesDir := t.TempDir()
	cfg := config.Config{NotesDir: notesDir}
	idx, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open() returned error: %v", err)
	}
	t.Cleanup(func() { idx.Close() })

	seedSearchNotes(t, notesDir, idx)

	return New(cfg, idx), notesDir, idx
}

func TestSearchFlow_FiltersListAfterDebounce(t *testing.T) {
	m, _, _ := newSearchTestModel(t)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	teatest.WaitFor(t, out, containsBytes("買い物リスト"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	typeRunes(tm, "会議")

	// Give the debounce (200ms) time to fire, then confirm the list is
	// filtered down to just the matching note.
	teatest.WaitFor(t, out, func(bts []byte) bool {
		return containsBytes("会議メモ")(bts) && !containsBytes("買い物リスト")(bts)
	}, teatest.WithDuration(2*time.Second))

	if err := tm.Quit(); err != nil {
		t.Fatalf("Quit() returned error: %v", err)
	}
	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(Model)
	if len(final.notes) != 1 || final.notes[0].Title != "会議メモ" {
		t.Errorf("final.notes = %+v, want exactly [会議メモ]", final.notes)
	}
}

func TestSearchFlow_EscClearsAndRestoresFullList(t *testing.T) {
	m, _, _ := newSearchTestModel(t)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	teatest.WaitFor(t, out, containsBytes("買い物リスト"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	typeRunes(tm, "会議")
	teatest.WaitFor(t, out, containsBytes("会議メモ"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	teatest.WaitFor(t, out, func(bts []byte) bool {
		return containsBytes("買い物リスト")(bts) &&
			containsBytes("会議メモ")(bts) &&
			containsBytes("冷蔵庫の掃除")(bts)
	}, teatest.WithDuration(2*time.Second))

	if err := tm.Quit(); err != nil {
		t.Fatalf("Quit() returned error: %v", err)
	}
	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(Model)
	if final.mode != modeList {
		t.Errorf("final mode = %v, want modeList", final.mode)
	}
	if final.search.Value() != "" {
		t.Errorf("final search query = %q, want empty", final.search.Value())
	}
	if len(final.notes) != 3 {
		t.Errorf("final.notes = %+v, want all 3 notes restored", final.notes)
	}
}

func TestSearchFlow_EnterKeepsQueryAndReturnsFocusToList(t *testing.T) {
	m, _, _ := newSearchTestModel(t)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	teatest.WaitFor(t, out, containsBytes("買い物リスト"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	typeRunes(tm, "会議")
	teatest.WaitFor(t, out, containsBytes("会議メモ"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	// Give any (unwanted) further filtering a moment to happen, then check
	// the filtered state is unchanged.
	time.Sleep(searchDebounceDelay + 100*time.Millisecond)

	if err := tm.Quit(); err != nil {
		t.Fatalf("Quit() returned error: %v", err)
	}
	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(Model)
	if final.mode != modeList {
		t.Errorf("final mode = %v, want modeList", final.mode)
	}
	if final.search.Value() != "会議" {
		t.Errorf("final search query = %q, want %q", final.search.Value(), "会議")
	}
	if len(final.notes) != 1 || final.notes[0].Title != "会議メモ" {
		t.Errorf("final.notes = %+v, want exactly [会議メモ]", final.notes)
	}
}

func TestHandleSearchDebounce_IgnoresStaleGeneration(t *testing.T) {
	m, _, _ := newSearchTestModel(t)
	m.mode = modeSearch
	m.searchGeneration = 5

	got, _ := m.Update(searchDebounceMsg{query: "会議", generation: 3})
	final := got.(Model)

	// A stale (superseded) tick must not touch notes.
	if len(final.notes) != 3 {
		t.Errorf("notes after stale tick = %+v, want unchanged (3 notes)", final.notes)
	}
}

func TestHandleSearchDebounce_AppliesCurrentGeneration(t *testing.T) {
	m, _, _ := newSearchTestModel(t)
	m.mode = modeSearch
	m.searchGeneration = 3

	got, _ := m.Update(searchDebounceMsg{query: "会議", generation: 3})
	final := got.(Model)

	if len(final.notes) != 1 || final.notes[0].Title != "会議メモ" {
		t.Errorf("notes after current-generation tick = %+v, want exactly [会議メモ]", final.notes)
	}
}

// TestEsc_InvalidatesInFlightDebounceTick guards against a regression where
// a debounce tick scheduled just before Esc was pressed could arrive after
// Esc had already reset the query, re-applying the stale filter and
// clobbering the just-restored full list.
func TestEsc_InvalidatesInFlightDebounceTick(t *testing.T) {
	m, _, _ := newSearchTestModel(t)
	m.mode = modeSearch
	m.searchGeneration = 3
	m.search.SetValue("会議")

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = got.(Model)

	if len(m.notes) != 3 {
		t.Fatalf("notes immediately after Esc = %+v, want all 3 restored", m.notes)
	}

	// A debounce tick from before Esc, carrying the pre-Esc generation,
	// must now be treated as stale and ignored.
	got, _ = m.Update(searchDebounceMsg{query: "会議", generation: 3})
	final := got.(Model)

	if len(final.notes) != 3 {
		t.Errorf("notes after stale post-Esc tick = %+v, want unchanged (3 notes)", final.notes)
	}
}
