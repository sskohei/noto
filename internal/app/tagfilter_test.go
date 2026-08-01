package app

import (
	"path/filepath"
	"testing"
	"time"

	"noto/internal/config"
	"noto/internal/index"
	"noto/internal/storage"
)

func TestFilterNotesAndListTags(t *testing.T) {
	cfg := config.Config{NotesDir: t.TempDir()}

	now := time.Now()
	notes := []storage.Note{
		{ID: "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", Title: "買い物リスト", Tags: []string{"life", "shopping"}, CreatedAt: now, UpdatedAt: now},
		{ID: "018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb", Title: "会議メモ", Tags: []string{"work"}, CreatedAt: now, UpdatedAt: now},
		{ID: "018f2e4a-cccc-cccc-cccc-cccccccccccc", Title: "作業ログ", Tags: []string{"work", "life"}, CreatedAt: now, UpdatedAt: now},
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

	gotTags, err := ListTags(idx)
	if err != nil {
		t.Fatalf("ListTags() returned error: %v", err)
	}
	wantTags := []string{"life", "shopping", "work"}
	if len(gotTags) != len(wantTags) {
		t.Fatalf("ListTags() = %v, want %v", gotTags, wantTags)
	}
	for i, tag := range wantTags {
		if gotTags[i] != tag {
			t.Errorf("ListTags()[%d] = %q, want %q", i, gotTags[i], tag)
		}
	}

	got, err := FilterNotes(idx, "", []string{"work", "life"})
	if err != nil {
		t.Fatalf("FilterNotes() returned error: %v", err)
	}
	if len(got) != 1 || got[0].Title != "作業ログ" {
		t.Errorf("FilterNotes(\"\", [work life]) = %v, want exactly [作業ログ]", got)
	}

	got, err = FilterNotes(idx, "", nil)
	if err != nil {
		t.Fatalf("FilterNotes(\"\", nil) returned error: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("FilterNotes(\"\", nil) = %v, want 3 notes", got)
	}
}
