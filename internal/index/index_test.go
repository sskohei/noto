package index

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/sskohei/noto/internal/storage"
)

func TestSyncInitialBuild(t *testing.T) {
	notesDir := t.TempDir()
	writeNote(t, notesDir, "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "買い物リスト", []string{"life", "shopping"}, "- 牛乳\n- コーヒー豆\n")
	writeNote(t, notesDir, "018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "work notes", []string{"work"}, "todo\n")

	db := openTestDB(t)
	if err := db.Sync(notesDir); err != nil {
		t.Fatalf("Sync() returned error: %v", err)
	}

	assertNoteCount(t, db, 2)
	assertNoteRow(t, db, "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "買い物リスト", []string{"life", "shopping"})
	assertNoteRow(t, db, "018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "work notes", []string{"work"})
	assertFTSRow(t, db, "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "買い物リスト")
}

func TestSyncAddsNewNote(t *testing.T) {
	notesDir := t.TempDir()
	writeNote(t, notesDir, "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "first", nil, "body\n")

	db := openTestDB(t)
	if err := db.Sync(notesDir); err != nil {
		t.Fatalf("Sync() returned error: %v", err)
	}
	assertNoteCount(t, db, 1)

	writeNote(t, notesDir, "018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "second", nil, "body\n")
	if err := db.Sync(notesDir); err != nil {
		t.Fatalf("second Sync() returned error: %v", err)
	}
	assertNoteCount(t, db, 2)
	assertNoteRow(t, db, "018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "second", nil)
}

func TestSyncReindexesChangedNote(t *testing.T) {
	notesDir := t.TempDir()
	path := writeNote(t, notesDir, "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "original title", []string{"a"}, "body\n")

	db := openTestDB(t)
	if err := db.Sync(notesDir); err != nil {
		t.Fatalf("Sync() returned error: %v", err)
	}
	assertNoteRow(t, db, "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "original title", []string{"a"})

	n, err := storage.Read(path)
	if err != nil {
		t.Fatalf("storage.Read() returned error: %v", err)
	}
	n.Title = "updated title"
	n.Tags = []string{"b", "c"}
	if err := storage.Write(path, n); err != nil {
		t.Fatalf("storage.Write() returned error: %v", err)
	}
	// Force a distinct mtime regardless of filesystem timestamp resolution.
	future := time.Now().Add(time.Hour)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatalf("os.Chtimes() returned error: %v", err)
	}

	if err := db.Sync(notesDir); err != nil {
		t.Fatalf("second Sync() returned error: %v", err)
	}

	assertNoteCount(t, db, 1)
	assertNoteRow(t, db, "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "updated title", []string{"b", "c"})
	assertFTSRow(t, db, "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "updated title")
}

func TestSyncUnchangedNoteIsSkipped(t *testing.T) {
	notesDir := t.TempDir()
	writeNote(t, notesDir, "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "title", nil, "body\n")

	db := openTestDB(t)
	if err := db.Sync(notesDir); err != nil {
		t.Fatalf("Sync() returned error: %v", err)
	}
	if err := db.Sync(notesDir); err != nil {
		t.Fatalf("second Sync() returned error: %v", err)
	}

	assertNoteCount(t, db, 1)
}

func TestSyncRemovesDeletedNote(t *testing.T) {
	notesDir := t.TempDir()
	keepPath := writeNote(t, notesDir, "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "keep", []string{"x"}, "body\n")
	removePath := writeNote(t, notesDir, "018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "remove", []string{"y"}, "body\n")
	_ = keepPath

	db := openTestDB(t)
	if err := db.Sync(notesDir); err != nil {
		t.Fatalf("Sync() returned error: %v", err)
	}
	assertNoteCount(t, db, 2)

	if err := os.Remove(removePath); err != nil {
		t.Fatalf("os.Remove() returned error: %v", err)
	}
	if err := db.Sync(notesDir); err != nil {
		t.Fatalf("second Sync() returned error: %v", err)
	}

	assertNoteCount(t, db, 1)

	var tagCount int
	if err := db.conn.QueryRow(`SELECT COUNT(*) FROM note_tags WHERE note_id = ?`, "018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb").Scan(&tagCount); err != nil {
		t.Fatalf("query note_tags returned error: %v", err)
	}
	if tagCount != 0 {
		t.Errorf("note_tags for removed note = %d rows, want 0", tagCount)
	}

	var ftsCount int
	if err := db.conn.QueryRow(`SELECT COUNT(*) FROM notes_fts WHERE id = ?`, "018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb").Scan(&ftsCount); err != nil {
		t.Fatalf("query notes_fts returned error: %v", err)
	}
	if ftsCount != 0 {
		t.Errorf("notes_fts for removed note = %d rows, want 0", ftsCount)
	}
}

func TestSyncFullRebuildAfterIndexDeletion(t *testing.T) {
	notesDir := t.TempDir()
	writeNote(t, notesDir, "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "買い物リスト", []string{"life", "shopping"}, "- 牛乳\n")
	writeNote(t, notesDir, "018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "work notes", []string{"work"}, "todo\n")

	dbPath := filepath.Join(t.TempDir(), "index.db")
	db1, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() returned error: %v", err)
	}
	if err := db1.Sync(notesDir); err != nil {
		t.Fatalf("Sync() returned error: %v", err)
	}
	if err := db1.Close(); err != nil {
		t.Fatalf("Close() returned error: %v", err)
	}

	if err := os.Remove(dbPath); err != nil {
		t.Fatalf("os.Remove(dbPath) returned error: %v", err)
	}

	db2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("re-Open() returned error: %v", err)
	}
	t.Cleanup(func() { db2.Close() })
	if err := db2.Sync(notesDir); err != nil {
		t.Fatalf("rebuild Sync() returned error: %v", err)
	}

	assertNoteCount(t, db2, 2)
	assertNoteRow(t, db2, "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "買い物リスト", []string{"life", "shopping"})
	assertNoteRow(t, db2, "018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "work notes", []string{"work"})
}

func openTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("Open() returned error: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func writeNote(t *testing.T, dir, id, title string, tags []string, body string) string {
	t.Helper()
	now := time.Now()
	n := storage.Note{
		ID:        id,
		Title:     title,
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
		Body:      body,
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

func assertNoteCount(t *testing.T, db *DB, want int) {
	t.Helper()
	var got int
	if err := db.conn.QueryRow(`SELECT COUNT(*) FROM notes`).Scan(&got); err != nil {
		t.Fatalf("query notes count returned error: %v", err)
	}
	if got != want {
		t.Errorf("notes count = %d, want %d", got, want)
	}
}

func assertNoteRow(t *testing.T, db *DB, id, wantTitle string, wantTags []string) {
	t.Helper()

	var title string
	if err := db.conn.QueryRow(`SELECT title FROM notes WHERE id = ?`, id).Scan(&title); err != nil {
		t.Fatalf("query notes row for %s returned error: %v", id, err)
	}
	if title != wantTitle {
		t.Errorf("notes.title for %s = %q, want %q", id, title, wantTitle)
	}

	rows, err := db.conn.Query(`SELECT tag FROM note_tags WHERE note_id = ?`, id)
	if err != nil {
		t.Fatalf("query note_tags for %s returned error: %v", id, err)
	}
	defer rows.Close()

	var gotTags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			t.Fatalf("scan note_tags row returned error: %v", err)
		}
		gotTags = append(gotTags, tag)
	}
	sort.Strings(gotTags)
	wantSorted := append([]string(nil), wantTags...)
	sort.Strings(wantSorted)

	if len(gotTags) != len(wantSorted) {
		t.Errorf("note_tags for %s = %v, want %v", id, gotTags, wantSorted)
		return
	}
	for i := range wantSorted {
		if gotTags[i] != wantSorted[i] {
			t.Errorf("note_tags for %s = %v, want %v", id, gotTags, wantSorted)
			return
		}
	}
}

func assertFTSRow(t *testing.T, db *DB, id, wantTitle string) {
	t.Helper()
	var title string
	if err := db.conn.QueryRow(`SELECT title FROM notes_fts WHERE id = ?`, id).Scan(&title); err != nil {
		t.Fatalf("query notes_fts row for %s returned error: %v", id, err)
	}
	if title != wantTitle {
		t.Errorf("notes_fts.title for %s = %q, want %q", id, title, wantTitle)
	}
}
