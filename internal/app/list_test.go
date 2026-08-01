package app

import (
	"path/filepath"
	"testing"
	"time"

	"noto/internal/config"
	"noto/internal/index"
	"noto/internal/storage"
)

func TestListNotes(t *testing.T) {
	cfg := config.Config{NotesDir: t.TempDir()}

	now := time.Now()
	n := storage.Note{
		ID:        "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Title:     "shopping list",
		Tags:      []string{"life"},
		CreatedAt: now,
		UpdatedAt: now,
	}
	path, err := storage.GeneratePath(cfg.NotesDir, n)
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
	if err := idx.Sync(cfg.NotesDir); err != nil {
		t.Fatalf("Sync() returned error: %v", err)
	}

	got, err := ListNotes(idx)
	if err != nil {
		t.Fatalf("ListNotes() returned error: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("ListNotes() = %v, want 1 note", got)
	}
	if got[0].ID != n.ID || got[0].Title != n.Title {
		t.Errorf("ListNotes()[0] = %+v, want ID=%q Title=%q", got[0], n.ID, n.Title)
	}
}
