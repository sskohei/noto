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
	if final.mode != modeList {
		t.Errorf("final mode = %v, want modeList", final.mode)
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
	if final.mode != modeList {
		t.Errorf("final mode = %v, want modeList", final.mode)
	}
}

func TestStartEditingExisting_NoOpWhenListEmpty(t *testing.T) {
	m, _, _ := newTestModel(t)

	got, cmd := m.startEditingExisting()
	final := got.(Model)

	if cmd != nil {
		t.Error("startEditingExisting() with empty list returned a non-nil Cmd, want nil")
	}
	if final.mode != modeList {
		t.Errorf("mode = %v, want modeList", final.mode)
	}
}

func TestHandleEditSessionFinished_PropagatesError(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.mode = modeEditing

	wantErr := errFake{}
	got, _ := m.Update(editSessionFinishedMsg{path: "/does/not/matter", err: wantErr})
	final := got.(Model)

	if final.mode != modeList {
		t.Errorf("mode = %v, want modeList", final.mode)
	}
	if final.err != wantErr {
		t.Errorf("err = %v, want %v", final.err, wantErr)
	}
}
