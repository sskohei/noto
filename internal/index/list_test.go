package index

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/sskohei/noto/internal/storage"
)

func TestListOrderedByUpdatedAtDesc(t *testing.T) {
	notesDir := t.TempDir()

	writeSummaryNote(t, notesDir, "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "oldest", nil, "2026-01-01T00:00:00Z")
	writeSummaryNote(t, notesDir, "018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "newest", nil, "2026-03-01T00:00:00Z")
	writeSummaryNote(t, notesDir, "018f2e4a-cccc-cccc-cccc-cccccccccccc", "middle", nil, "2026-02-01T00:00:00Z")

	db := openListTestDB(t)
	if err := db.Sync(notesDir); err != nil {
		t.Fatalf("Sync() returned error: %v", err)
	}

	got, err := db.List()
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("List() returned %d notes, want 3", len(got))
	}

	wantOrder := []string{"newest", "middle", "oldest"}
	for i, want := range wantOrder {
		if got[i].Title != want {
			t.Errorf("List()[%d].Title = %q, want %q", i, got[i].Title, want)
		}
	}
}

func TestListIncludesTags(t *testing.T) {
	notesDir := t.TempDir()
	writeSummaryNote(t, notesDir, "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "tagged", []string{"b", "a"}, "2026-01-01T00:00:00Z")
	writeSummaryNote(t, notesDir, "018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "untagged", nil, "2026-01-02T00:00:00Z")

	db := openListTestDB(t)
	if err := db.Sync(notesDir); err != nil {
		t.Fatalf("Sync() returned error: %v", err)
	}

	got, err := db.List()
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}

	byTitle := make(map[string]NoteSummary)
	for _, n := range got {
		byTitle[n.Title] = n
	}

	tagged, ok := byTitle["tagged"]
	if !ok {
		t.Fatal(`"tagged" note not found in List() result`)
	}
	if len(tagged.Tags) != 2 || tagged.Tags[0] != "a" || tagged.Tags[1] != "b" {
		t.Errorf("tagged.Tags = %v, want [a b]", tagged.Tags)
	}

	untagged, ok := byTitle["untagged"]
	if !ok {
		t.Fatal(`"untagged" note not found in List() result`)
	}
	if len(untagged.Tags) != 0 {
		t.Errorf("untagged.Tags = %v, want empty", untagged.Tags)
	}
}

func TestListIncludesPath(t *testing.T) {
	notesDir := t.TempDir()
	wantPath := writeSummaryNote(t, notesDir, "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "note", nil, "2026-01-01T00:00:00Z")

	db := openListTestDB(t)
	if err := db.Sync(notesDir); err != nil {
		t.Fatalf("Sync() returned error: %v", err)
	}

	got, err := db.List()
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("List() returned %d notes, want 1", len(got))
	}
	if got[0].Path != wantPath {
		t.Errorf("List()[0].Path = %q, want %q", got[0].Path, wantPath)
	}
}

func TestListEmptyIndex(t *testing.T) {
	db := openListTestDB(t)

	got, err := db.List()
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("List() = %v, want empty", got)
	}
}

func openListTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("Open() returned error: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func writeSummaryNote(t *testing.T, dir, id, title string, tags []string, updatedAt string) string {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		t.Fatalf("failed to parse test time %q: %v", updatedAt, err)
	}

	n := storage.Note{
		ID:        id,
		Title:     title,
		Tags:      tags,
		CreatedAt: ts,
		UpdatedAt: ts,
	}
	path, err := storage.GeneratePath(dir, n)
	if err != nil {
		t.Fatalf("storage.GeneratePath() returned error: %v", err)
	}
	if err := storage.Write(path, n); err != nil {
		t.Fatalf("storage.Write() returned error: %v", err)
	}
	return path
}
