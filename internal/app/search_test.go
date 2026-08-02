package app

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/sskohei/noto/internal/config"
	"github.com/sskohei/noto/internal/index"
	"github.com/sskohei/noto/internal/storage"
)

func TestSearchNotes(t *testing.T) {
	cfg := config.Config{NotesDir: t.TempDir()}

	now := time.Now()
	notes := []storage.Note{
		{ID: "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", Title: "買い物リスト", CreatedAt: now, UpdatedAt: now},
		{ID: "018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb", Title: "会議メモ", CreatedAt: now, UpdatedAt: now},
	}
	for _, n := range notes {
		path, err := storage.GeneratePath(cfg.NotesDir, n)
		if err != nil {
			t.Fatalf("storage.GeneratePath() returned error: %v", err)
		}
		if err := storage.Write(path, n); err != nil {
			t.Fatalf("storage.Write() returned error: %v", err)
		}
	}

	idx, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open() returned error: %v", err)
	}
	t.Cleanup(func() { idx.Close() })
	if err := idx.Sync(cfg.NotesDir); err != nil {
		t.Fatalf("Sync() returned error: %v", err)
	}

	got, err := SearchNotes(idx, "会議")
	if err != nil {
		t.Fatalf("SearchNotes() returned error: %v", err)
	}
	if len(got) != 1 || got[0].Title != "会議メモ" {
		t.Errorf("SearchNotes(%q) = %v, want exactly [会議メモ]", "会議", got)
	}

	got, err = SearchNotes(idx, "")
	if err != nil {
		t.Fatalf("SearchNotes(\"\") returned error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("SearchNotes(\"\") = %v, want 2 notes", got)
	}
}
