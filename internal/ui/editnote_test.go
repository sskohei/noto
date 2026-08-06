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

func TestEditFlow_FullRoundTrip(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "true")

	notesDir := t.TempDir()
	cfg := config.Config{NotesDir: notesDir}
	idx, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open() returned error: %v", err)
	}
	t.Cleanup(func() { idx.Close() })

	original := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	n := storage.Note{
		ID:        "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Title:     "既存のメモ",
		CreatedAt: original,
		UpdatedAt: original,
		Body:      "original body\n",
	}
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

	m := skipSplash(New(cfg, idx))
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	teatest.WaitFor(t, out, containsBytes("既存のメモ"), teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// The updated_at display should no longer show the original fixed
	// date once FinalizeEdit has bumped it to "now" and the list refreshes.
	teatest.WaitFor(t, out, func(bts []byte) bool {
		return containsBytes("既存のメモ")(bts) && !containsBytes("2026-01-01")(bts)
	}, teatest.WithDuration(2*time.Second))

	if err := tm.Quit(); err != nil {
		t.Fatalf("Quit() returned error: %v", err)
	}
	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(Model)
	if final.mode != modeMain {
		t.Errorf("final mode = %v, want modeMain", final.mode)
	}
	if final.err != nil {
		t.Errorf("final err = %v, want nil", final.err)
	}
	if len(final.notes) != 1 || !final.notes[0].UpdatedAt.After(original) {
		t.Errorf("final.notes = %+v, want 1 note with UpdatedAt after %v", final.notes, original)
	}

	onDisk, err := storage.Read(path)
	if err != nil {
		t.Fatalf("storage.Read() returned error: %v", err)
	}
	if !onDisk.UpdatedAt.After(original) {
		t.Errorf("on-disk UpdatedAt = %v, want after %v", onDisk.UpdatedAt, original)
	}
}

func TestEditFlow_EKeyAlsoStartsEditing(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "true")

	notesDir := t.TempDir()
	cfg := config.Config{NotesDir: notesDir}
	idx, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open() returned error: %v", err)
	}
	t.Cleanup(func() { idx.Close() })

	now := time.Now()
	n := storage.Note{ID: "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", Title: "note", CreatedAt: now, UpdatedAt: now}
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

	m := skipSplash(New(cfg, idx))
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	teatest.WaitFor(t, out, containsBytes("note"), teatest.WithDuration(2*time.Second))
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})

	teatest.WaitFor(t, out, containsBytes("note"), teatest.WithDuration(2*time.Second))

	if err := tm.Quit(); err != nil {
		t.Fatalf("Quit() returned error: %v", err)
	}
	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(Model)
	if final.mode != modeMain {
		t.Errorf("final mode = %v, want modeMain", final.mode)
	}
}

func TestStartEditingExisting_NoOpWhenListEmpty(t *testing.T) {
	m, _, _ := newTestModel(t)

	got, cmd := m.startEditingExisting()
	final := got.(Model)

	if cmd != nil {
		t.Error("startEditingExisting() with empty list returned a non-nil Cmd, want nil")
	}
	if final.mode != modeMain {
		t.Errorf("mode = %v, want modeMain", final.mode)
	}
}

func TestHandleEditSessionFinished_RefreshesTags(t *testing.T) {
	notesDir := t.TempDir()
	cfg := config.Config{NotesDir: notesDir}
	idx, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open() returned error: %v", err)
	}
	t.Cleanup(func() { idx.Close() })

	now := time.Now()
	n := storage.Note{
		ID:        "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Title:     "既存のメモ",
		CreatedAt: now,
		UpdatedAt: now,
		Body:      "original body\n",
	}
	path, err := storage.GeneratePath(notesDir, n)
	if err != nil {
		t.Fatalf("storage.GeneratePath() returned error: %v", err)
	}
	if err := storage.Write(path, n); err != nil {
		t.Fatalf("storage.Write() returned error: %v", err)
	}
	// Deliberately not idx.Sync()'d yet at this point: FinalizeEdit's own
	// call to idx.Sync will index the note for the first time, so the fix
	// under test isn't confounded by the mtime-diffing Sync does on notes
	// it has already indexed (see index.DB.Sync).

	m := skipSplash(New(cfg, idx))
	if containsTag(m.allTags, "追加タグ") {
		t.Fatalf("allTags before edit = %v, want it to not yet contain 追加タグ", m.allTags)
	}

	// Focus the tags panel before "finishing" the edit session: switchFocus
	// only refreshes allTags on a transition *into* focusTags, so if
	// handleEditSessionFinished didn't refresh it itself, staying focused on
	// the tags panel throughout would never pick up the new tag.
	m.focusedPanel = focusTags

	// Simulate the editor session adding a new tag to the note.
	n.Tags = []string{"追加タグ"}
	if err := storage.Write(path, n); err != nil {
		t.Fatalf("storage.Write() returned error: %v", err)
	}

	m.mode = modeEditing
	got, _ := m.Update(editSessionFinishedMsg{path: path, err: nil})
	final := got.(Model)

	if final.err != nil {
		t.Fatalf("err = %v, want nil", final.err)
	}
	if !containsTag(final.allTags, "追加タグ") {
		t.Errorf("allTags = %v, want it to contain %q", final.allTags, "追加タグ")
	}
}

func TestHandleEditSessionFinished_PropagatesError(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.mode = modeEditing

	wantErr := errFake{}
	got, _ := m.Update(editSessionFinishedMsg{path: "/does/not/matter", err: wantErr})
	final := got.(Model)

	if final.mode != modeMain {
		t.Errorf("mode = %v, want modeMain", final.mode)
	}
	if final.err != wantErr {
		t.Errorf("err = %v, want %v", final.err, wantErr)
	}
}
