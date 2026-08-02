package ui

import (
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/sskohei/noto/internal/config"
	"github.com/sskohei/noto/internal/index"
	"github.com/sskohei/noto/internal/storage"
)

// seedTagFilterNotes writes three notes with overlapping tags and distinct,
// greppable titles/bodies so tag AND-filtering (alone and combined with
// search) is easy to assert on.
func seedTagFilterNotes(t *testing.T, notesDir string, idx *index.DB) {
	t.Helper()

	fixtures := []struct {
		id, title, body string
		tags            []string
		updatedAt       string
	}{
		{"018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "会議メモ", "来週の会議", []string{"work"}, "2026-01-03T00:00:00Z"},
		{"018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "作業ログ", "今日の作業内容", []string{"work", "life"}, "2026-01-02T00:00:00Z"},
		{"018f2e4a-cccc-cccc-cccc-cccccccccccc", "買い物リスト", "牛乳を買う", []string{"life"}, "2026-01-01T00:00:00Z"},
	}

	for _, f := range fixtures {
		ts, err := time.Parse(time.RFC3339, f.updatedAt)
		if err != nil {
			t.Fatalf("failed to parse fixture time %q: %v", f.updatedAt, err)
		}
		n := storage.Note{ID: f.id, Title: f.title, Body: f.body, Tags: f.tags, CreatedAt: ts, UpdatedAt: ts}
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

func newTagFilterTestModel(t *testing.T) (Model, string, *index.DB) {
	t.Helper()
	notesDir := t.TempDir()
	cfg := config.Config{NotesDir: notesDir}
	idx, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open() returned error: %v", err)
	}
	t.Cleanup(func() { idx.Close() })

	seedTagFilterNotes(t, notesDir, idx)

	return skipSplash(New(cfg, idx)), notesDir, idx
}

func TestTagFilterFlow_SelectingTagFiltersList(t *testing.T) {
	m, _, _ := newTagFilterTestModel(t)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	teatest.WaitFor(t, out, containsBytes("会議メモ"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	teatest.WaitFor(t, out, containsBytes("▶ 2:タグ"), teatest.WithDuration(2*time.Second))

	// "life" sorts before "work" alphabetically, so the cursor starts on
	// "life"; move down once to reach "work".
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})

	// Selecting a tag refreshes the list synchronously (no debounce), so the
	// filtered indicator folded into the list panel's title is a reliable
	// checkpoint.
	teatest.WaitFor(t, out, containsBytes("タグ: work"), teatest.WithDuration(2*time.Second))

	if err := tm.Quit(); err != nil {
		t.Fatalf("Quit() returned error: %v", err)
	}
	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(Model)
	if final.mode != modeMain {
		t.Errorf("final mode = %v, want modeMain", final.mode)
	}
	if len(final.selectedTags) != 1 || final.selectedTags[0] != "work" {
		t.Errorf("final.selectedTags = %v, want [work]", final.selectedTags)
	}
	wantTitles := map[string]bool{"会議メモ": true, "作業ログ": true}
	if len(final.notes) != len(wantTitles) {
		t.Fatalf("final.notes = %+v, want exactly the 2 notes tagged work", final.notes)
	}
	for _, n := range final.notes {
		if !wantTitles[n.Title] {
			t.Errorf("final.notes contains unexpected title %q", n.Title)
		}
	}
}

func TestTagFilterFlow_CombinesWithSearch(t *testing.T) {
	m, _, _ := newTagFilterTestModel(t)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	teatest.WaitFor(t, out, containsBytes("会議メモ"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	teatest.WaitFor(t, out, containsBytes("▶ 2:タグ"), teatest.WithDuration(2*time.Second))
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // life -> work
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	teatest.WaitFor(t, out, containsBytes("タグ: work"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	typeRunes(tm, "作業")

	// Let the debounce settle, then inspect the resulting model directly
	// rather than racing on intermediate rendered frames.
	time.Sleep(searchDebounceDelay + 100*time.Millisecond)

	if err := tm.Quit(); err != nil {
		t.Fatalf("Quit() returned error: %v", err)
	}
	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(Model)
	if len(final.notes) != 1 || final.notes[0].Title != "作業ログ" {
		t.Errorf("final.notes = %+v, want exactly [作業ログ]", final.notes)
	}
}

func TestUpdateTagFilter_CursorClampsAtBounds(t *testing.T) {
	m, _, _ := newTagFilterTestModel(t)
	m.mode = modeMain
	m.focusedPanel = focusTags
	m.allTags = []string{"a", "b", "c"}

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = got.(Model)
	if m.tagCursor != 0 {
		t.Fatalf("tagCursor after k at top = %d, want 0", m.tagCursor)
	}

	for range m.allTags {
		got, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		m = got.(Model)
	}
	if m.tagCursor != len(m.allTags)-1 {
		t.Fatalf("tagCursor after repeated j = %d, want %d", m.tagCursor, len(m.allTags)-1)
	}

	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = got.(Model)
	if m.tagCursor != len(m.allTags)-1 {
		t.Errorf("tagCursor after j past bottom = %d, want %d", m.tagCursor, len(m.allTags)-1)
	}
}

func TestUpdateTagFilter_EnterTogglesSelection(t *testing.T) {
	m, _, _ := newTagFilterTestModel(t)
	m.mode = modeMain
	m.focusedPanel = focusTags
	m.allTags = []string{"life", "work"}
	m.tagCursor = 1 // "work"

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = got.(Model)
	if !hasTag(m.selectedTags, "work") {
		t.Fatalf("selectedTags = %v, want to contain %q after first toggle", m.selectedTags, "work")
	}

	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = got.(Model)
	if hasTag(m.selectedTags, "work") {
		t.Errorf("selectedTags = %v, want to no longer contain %q after second toggle", m.selectedTags, "work")
	}
}

func TestUpdateTagFilter_EscReturnsToListWithoutClearingSelection(t *testing.T) {
	m, _, _ := newTagFilterTestModel(t)
	m.mode = modeMain
	m.focusedPanel = focusTags
	m.allTags = []string{"work"}
	m.selectedTags = []string{"work"}

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	final := got.(Model)

	if final.mode != modeMain {
		t.Errorf("mode = %v, want modeMain", final.mode)
	}
	if len(final.selectedTags) != 1 || final.selectedTags[0] != "work" {
		t.Errorf("selectedTags = %v, want [work] preserved", final.selectedTags)
	}
}
