package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"noto/internal/index"
	"noto/internal/storage"
)

func TestDeleteNote(t *testing.T) {
	notesDir := t.TempDir()

	now := time.Now()
	n := storage.Note{
		ID:        "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Title:     "to be deleted",
		Tags:      []string{"a"},
		CreatedAt: now,
		UpdatedAt: now,
	}
	path, err := storage.GeneratePath(notesDir, n)
	if err != nil {
		t.Fatalf("storage.GeneratePath() returned error: %v", err)
	}
	if err := storage.Write(path, n); err != nil {
		t.Fatalf("storage.Write() returned error: %v", err)
	}

	idx, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open() returned error: %v", err)
	}
	t.Cleanup(func() { idx.Close() })
	if err := idx.Sync(notesDir); err != nil {
		t.Fatalf("Sync() returned error: %v", err)
	}

	if err := DeleteNote(idx, notesDir, path); err != nil {
		t.Fatalf("DeleteNote() returned error: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected file to be removed at %s, stat err = %v", path, err)
	}

	notes, err := idx.List()
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}
	if len(notes) != 0 {
		t.Errorf("List() = %+v, want empty after DeleteNote", notes)
	}
}

func TestDeleteNoteMissingFile(t *testing.T) {
	notesDir := t.TempDir()

	idx, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open() returned error: %v", err)
	}
	t.Cleanup(func() { idx.Close() })

	err = DeleteNote(idx, notesDir, filepath.Join(notesDir, "missing.md"))
	if err == nil {
		t.Fatal("DeleteNote() of a missing file should return an error")
	}
}
