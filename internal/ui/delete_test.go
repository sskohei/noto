package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/sskohei/noto/internal/config"
	"github.com/sskohei/noto/internal/index"
	"github.com/sskohei/noto/internal/storage"
)

func newDeleteTestModel(t *testing.T) (Model, string, string) {
	t.Helper()

	notesDir := t.TempDir()
	cfg := config.Config{NotesDir: notesDir}
	idx, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open() returned error: %v", err)
	}
	t.Cleanup(func() { idx.Close() })

	now := time.Now()
	n := storage.Note{ID: "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", Title: "削除対象", CreatedAt: now, UpdatedAt: now}
	path, err := storage.GeneratePath(notesDir, n)
	if err != nil {
		t.Fatalf("storage.GeneratePath() returned error: %v", err)
	}
	if err := storage.Write(path, n); err != nil {
		t.Fatalf("storage.Write() returned error: %v", err)
	}
	if err := idx.Sync(notesDir); err != nil {
		t.Fatalf("Sync() returned error: %v", err)
	}

	return skipSplash(New(cfg, idx)), notesDir, path
}

func TestDeleteFlow_ConfirmRemovesNoteAndFile(t *testing.T) {
	m, notesDir, path := newDeleteTestModel(t)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	teatest.WaitFor(t, out, containsBytes("削除対象"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	teatest.WaitFor(t, out, func(bts []byte) bool {
		return containsBytes("削除対象")(bts) && containsBytes("削除しますか")(bts)
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	teatest.WaitFor(t, out, containsBytes("メモがありません"), teatest.WithDuration(2*time.Second))

	if err := tm.Quit(); err != nil {
		t.Fatalf("Quit() returned error: %v", err)
	}
	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(Model)
	if final.mode != modeMain {
		t.Errorf("final mode = %v, want modeMain", final.mode)
	}
	if len(final.notes) != 0 {
		t.Errorf("final.notes = %+v, want empty", final.notes)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected file to be removed at %s, stat err = %v", path, err)
	}
	paths, err := storage.List(notesDir)
	if err != nil {
		t.Fatalf("storage.List() returned error: %v", err)
	}
	if len(paths) != 0 {
		t.Errorf("storage.List() = %v, want no notes remaining", paths)
	}
}

func TestDeleteFlow_RemovesTagWhenLastTaggedNoteDeleted(t *testing.T) {
	notesDir := t.TempDir()
	cfg := config.Config{NotesDir: notesDir}
	idx, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open() returned error: %v", err)
	}
	t.Cleanup(func() { idx.Close() })

	now := time.Now()
	tagged := storage.Note{
		ID:        "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Title:     "タグ付きメモ",
		CreatedAt: now,
		UpdatedAt: now,
		Tags:      []string{"消えるタグ"},
	}
	taggedPath, err := storage.GeneratePath(notesDir, tagged)
	if err != nil {
		t.Fatalf("storage.GeneratePath() returned error: %v", err)
	}
	if err := storage.Write(taggedPath, tagged); err != nil {
		t.Fatalf("storage.Write() returned error: %v", err)
	}

	untagged := storage.Note{
		ID:        "018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		Title:     "タグなしメモ",
		CreatedAt: now.Add(time.Second),
		UpdatedAt: now.Add(time.Second),
	}
	untaggedPath, err := storage.GeneratePath(notesDir, untagged)
	if err != nil {
		t.Fatalf("storage.GeneratePath() returned error: %v", err)
	}
	if err := storage.Write(untaggedPath, untagged); err != nil {
		t.Fatalf("storage.Write() returned error: %v", err)
	}

	if err := idx.Sync(notesDir); err != nil {
		t.Fatalf("Sync() returned error: %v", err)
	}

	m := skipSplash(New(cfg, idx))
	if !containsTag(m.allTags, "消えるタグ") {
		t.Fatalf("allTags before delete = %v, want it to contain 消えるタグ", m.allTags)
	}

	// Focus the tags panel before deleting: switchFocus only refreshes
	// allTags on a transition *into* focusTags, so if confirmDeleteNote
	// didn't refresh it itself, staying focused on the tags panel throughout
	// would never notice the tag disappearing.
	m.focusedPanel = focusTags

	// Notes are sorted newest-first, so the tagged note (older) is at
	// cursor 1.
	m.cursor = 1
	if m.notes[m.cursor].Path != taggedPath {
		t.Fatalf("m.notes[%d].Path = %q, want %q", m.cursor, m.notes[m.cursor].Path, taggedPath)
	}

	got, _ := m.confirmDeleteNote()
	final := got.(Model)

	if final.err != nil {
		t.Fatalf("err = %v, want nil", final.err)
	}
	if containsTag(final.allTags, "消えるタグ") {
		t.Errorf("allTags = %v, want it to no longer contain 消えるタグ", final.allTags)
	}
}

func TestDeleteFlow_KeepsTagWhenOtherNoteStillHasIt(t *testing.T) {
	notesDir := t.TempDir()
	cfg := config.Config{NotesDir: notesDir}
	idx, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open() returned error: %v", err)
	}
	t.Cleanup(func() { idx.Close() })

	now := time.Now()
	first := storage.Note{
		ID:        "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Title:     "1件目",
		CreatedAt: now,
		UpdatedAt: now,
		Tags:      []string{"残るタグ"},
	}
	firstPath, err := storage.GeneratePath(notesDir, first)
	if err != nil {
		t.Fatalf("storage.GeneratePath() returned error: %v", err)
	}
	if err := storage.Write(firstPath, first); err != nil {
		t.Fatalf("storage.Write() returned error: %v", err)
	}

	second := storage.Note{
		ID:        "018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		Title:     "2件目",
		CreatedAt: now.Add(time.Second),
		UpdatedAt: now.Add(time.Second),
		Tags:      []string{"残るタグ"},
	}
	secondPath, err := storage.GeneratePath(notesDir, second)
	if err != nil {
		t.Fatalf("storage.GeneratePath() returned error: %v", err)
	}
	if err := storage.Write(secondPath, second); err != nil {
		t.Fatalf("storage.Write() returned error: %v", err)
	}

	if err := idx.Sync(notesDir); err != nil {
		t.Fatalf("Sync() returned error: %v", err)
	}

	m := skipSplash(New(cfg, idx))
	// Notes are sorted newest-first, so "2件目" is at cursor 0.
	m.cursor = 0
	if m.notes[m.cursor].Path != secondPath {
		t.Fatalf("m.notes[%d].Path = %q, want %q", m.cursor, m.notes[m.cursor].Path, secondPath)
	}

	got, _ := m.confirmDeleteNote()
	final := got.(Model)

	if final.err != nil {
		t.Fatalf("err = %v, want nil", final.err)
	}
	if !containsTag(final.allTags, "残るタグ") {
		t.Errorf("allTags = %v, want it to still contain 残るタグ", final.allTags)
	}
}

func TestDeleteFlow_NCancelsWithoutDeleting(t *testing.T) {
	m, _, path := newDeleteTestModel(t)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	teatest.WaitFor(t, out, containsBytes("削除対象"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	teatest.WaitFor(t, out, containsBytes("削除しますか"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	teatest.WaitFor(t, out, containsBytes("dd: 削除"), teatest.WithDuration(2*time.Second))

	if err := tm.Quit(); err != nil {
		t.Fatalf("Quit() returned error: %v", err)
	}
	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(Model)
	if final.mode != modeMain {
		t.Errorf("final mode = %v, want modeMain", final.mode)
	}
	if len(final.notes) != 1 {
		t.Errorf("final.notes = %+v, want the note to remain", final.notes)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file to remain at %s: %v", path, err)
	}
}

func TestDeleteFlow_EscCancelsWithoutDeleting(t *testing.T) {
	m, _, path := newDeleteTestModel(t)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	teatest.WaitFor(t, out, containsBytes("削除対象"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	teatest.WaitFor(t, out, containsBytes("削除しますか"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	teatest.WaitFor(t, out, containsBytes("dd: 削除"), teatest.WithDuration(2*time.Second))

	if err := tm.Quit(); err != nil {
		t.Fatalf("Quit() returned error: %v", err)
	}
	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(Model)
	if final.mode != modeMain {
		t.Errorf("final mode = %v, want modeMain", final.mode)
	}
	if len(final.notes) != 1 {
		t.Errorf("final.notes = %+v, want the note to remain", final.notes)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file to remain at %s: %v", path, err)
	}
}

func TestUpdateList_FirstDArmsPendingDelete(t *testing.T) {
	m, _, _ := newTestModel(t)

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	final := got.(Model)

	if !final.pendingDelete {
		t.Error("pendingDelete = false after first d, want true")
	}
	if final.mode != modeMain {
		t.Errorf("mode after first d = %v, want modeMain", final.mode)
	}
}

func TestUpdateList_SecondDEntersConfirmDelete(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.notes = []index.NoteSummary{{ID: "a", Path: "/tmp/a.md", Title: "a"}}
	m.pendingDelete = true

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	final := got.(Model)

	if final.mode != modeConfirmDelete {
		t.Errorf("mode after second d = %v, want modeConfirmDelete", final.mode)
	}
	if final.pendingDelete {
		t.Error("pendingDelete = true after second d, want false")
	}
}

func TestUpdateList_DDNoOpWhenListEmpty(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.pendingDelete = true

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	final := got.(Model)

	if final.mode != modeMain {
		t.Errorf("mode = %v, want modeMain (dd on empty list should be a no-op)", final.mode)
	}
}

func TestUpdateList_NonDKeyResetsPendingDeleteButStillProcessesKey(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.notes = []index.NoteSummary{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	m.pendingDelete = true

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	final := got.(Model)

	if final.pendingDelete {
		t.Error("pendingDelete = true after non-d key, want false")
	}
	if final.cursor != 1 {
		t.Errorf("cursor = %d, want 1 (j should still move the cursor)", final.cursor)
	}
}

func TestUpdateConfirmDelete_NAndEscCancelWithoutDeleting(t *testing.T) {
	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune("n")},
		{Type: tea.KeyEsc},
	} {
		m, _, _ := newTestModel(t)
		m.notes = []index.NoteSummary{{ID: "a", Path: "/does/not/matter.md", Title: "a"}}
		m.mode = modeConfirmDelete

		got, _ := m.Update(key)
		final := got.(Model)

		if final.mode != modeMain {
			t.Errorf("[%v] mode = %v, want modeMain", key, final.mode)
		}
		if len(final.notes) != 1 {
			t.Errorf("[%v] notes = %+v, want unchanged", key, final.notes)
		}
	}
}
