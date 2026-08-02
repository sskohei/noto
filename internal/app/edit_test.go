package app

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/sskohei/noto/internal/index"
	"github.com/sskohei/noto/internal/storage"
)

func TestFinalizeEdit(t *testing.T) {
	notesDir := t.TempDir()

	original := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	n := storage.Note{
		ID:        "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Title:     "edited by user",
		Tags:      []string{"a"},
		CreatedAt: original,
		UpdatedAt: original,
		Body:      "edited body\n",
	}
	path, err := storage.GeneratePath(notesDir, n)
	if err != nil {
		t.Fatalf("storage.GeneratePath() returned error: %v", err)
	}
	// storage.Write simulates "the editor session already saved this
	// content"; updated_at in the file is still the stale original value.
	if err := storage.Write(path, n); err != nil {
		t.Fatalf("storage.Write() returned error: %v", err)
	}

	idx, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open() returned error: %v", err)
	}
	t.Cleanup(func() { idx.Close() })

	before := time.Now()
	got, err := FinalizeEdit(idx, notesDir, path)
	if err != nil {
		t.Fatalf("FinalizeEdit() returned error: %v", err)
	}

	if got.ID != n.ID || got.Title != n.Title || got.Body != n.Body {
		t.Errorf("FinalizeEdit() = %+v, want ID/Title/Body matching %+v", got, n)
	}
	if !got.UpdatedAt.After(original) {
		t.Errorf("UpdatedAt = %v, want after original %v", got.UpdatedAt, original)
	}
	if got.UpdatedAt.Before(before) {
		t.Errorf("UpdatedAt = %v, want at or after %v", got.UpdatedAt, before)
	}

	onDisk, err := storage.Read(path)
	if err != nil {
		t.Fatalf("storage.Read() returned error: %v", err)
	}
	if !onDisk.UpdatedAt.Equal(got.UpdatedAt) {
		t.Errorf("on-disk UpdatedAt = %v, want %v", onDisk.UpdatedAt, got.UpdatedAt)
	}
	if !onDisk.CreatedAt.Equal(original) {
		t.Errorf("on-disk CreatedAt = %v, want unchanged %v", onDisk.CreatedAt, original)
	}
}
