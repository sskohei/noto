package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteReadRoundTrip(t *testing.T) {
	dir := t.TempDir()

	note := Note{
		ID:        "018f2e4a-6b3a-7c9e-9a1a-2f7e1a9c4b21",
		Title:     "買い物リスト",
		Tags:      []string{"life", "shopping"},
		CreatedAt: mustParseTime(t, "2026-08-01T09:12:00+09:00"),
		UpdatedAt: mustParseTime(t, "2026-08-01T09:20:31+09:00"),
		Body:      "- 牛乳\n- コーヒー豆\n",
	}

	path, err := GeneratePath(dir, note)
	if err != nil {
		t.Fatalf("GeneratePath() returned error: %v", err)
	}

	if err := Write(path, note); err != nil {
		t.Fatalf("Write() returned error: %v", err)
	}

	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read() returned error: %v", err)
	}

	if got.ID != note.ID || got.Title != note.Title || got.Body != note.Body {
		t.Errorf("Read() = %+v, want %+v", got, note)
	}
	if !stringSlicesEqual(got.Tags, note.Tags) {
		t.Errorf("Tags = %v, want %v", got.Tags, note.Tags)
	}
}

func TestWriteCreatesParentDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "notes")

	note := Note{
		ID:        "018f2e4a-0000-0000-0000-000000000000",
		Title:     "test",
		CreatedAt: mustParseTime(t, "2026-01-01T00:00:00Z"),
		UpdatedAt: mustParseTime(t, "2026-01-01T00:00:00Z"),
	}

	path := filepath.Join(dir, "note.md")
	if err := Write(path, note); err != nil {
		t.Fatalf("Write() returned error: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file to exist at %s: %v", path, err)
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()

	for _, name := range []string{"b.md", "a.md", "c.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
			t.Fatalf("failed to seed file %s: %v", name, err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "subdir.md"), 0o755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	got, err := List(dir)
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}

	want := []string{filepath.Join(dir, "a.md"), filepath.Join(dir, "b.md")}
	if len(got) != len(want) {
		t.Fatalf("List() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("List()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestGeneratePathAvoidsCollisions(t *testing.T) {
	dir := t.TempDir()
	createdAt := mustParseTime(t, "2026-08-01T09:12:00+09:00")

	first := Note{ID: "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", Title: "買い物リスト", CreatedAt: createdAt}
	second := Note{ID: "018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb", Title: "買い物リスト", CreatedAt: createdAt}
	third := Note{ID: "018f2e4a-cccc-cccc-cccc-cccccccccccc", Title: "買い物リスト", CreatedAt: createdAt}

	firstPath, err := GeneratePath(dir, first)
	if err != nil {
		t.Fatalf("GeneratePath() returned error: %v", err)
	}
	if err := Write(firstPath, first); err != nil {
		t.Fatalf("Write() returned error: %v", err)
	}

	secondPath, err := GeneratePath(dir, second)
	if err != nil {
		t.Fatalf("GeneratePath() returned error: %v", err)
	}
	if secondPath == firstPath {
		t.Fatalf("GeneratePath() returned colliding path %q for second note", secondPath)
	}
	if err := Write(secondPath, second); err != nil {
		t.Fatalf("Write() returned error: %v", err)
	}

	thirdPath, err := GeneratePath(dir, third)
	if err != nil {
		t.Fatalf("GeneratePath() returned error: %v", err)
	}
	if thirdPath == firstPath || thirdPath == secondPath {
		t.Fatalf("GeneratePath() returned colliding path %q for third note", thirdPath)
	}

	wantFirst := filepath.Join(dir, "20260801T091200-買い物リスト.md")
	wantSecond := filepath.Join(dir, "20260801T091200-買い物リスト-2.md")
	wantThird := filepath.Join(dir, "20260801T091200-買い物リスト-3.md")
	if firstPath != wantFirst {
		t.Errorf("firstPath = %q, want %q", firstPath, wantFirst)
	}
	if secondPath != wantSecond {
		t.Errorf("secondPath = %q, want %q", secondPath, wantSecond)
	}
	if thirdPath != wantThird {
		t.Errorf("thirdPath = %q, want %q", thirdPath, wantThird)
	}
}

func TestReadMissingFile(t *testing.T) {
	_, err := Read(filepath.Join(t.TempDir(), "missing.md"))
	if err == nil {
		t.Fatal("Read() of a missing file should return an error")
	}
}

func TestDeleteRemovesFile(t *testing.T) {
	dir := t.TempDir()
	note := Note{
		ID:        "018f2e4a-0000-0000-0000-000000000000",
		Title:     "test",
		CreatedAt: mustParseTime(t, "2026-01-01T00:00:00Z"),
		UpdatedAt: mustParseTime(t, "2026-01-01T00:00:00Z"),
	}

	path, err := GeneratePath(dir, note)
	if err != nil {
		t.Fatalf("GeneratePath() returned error: %v", err)
	}
	if err := Write(path, note); err != nil {
		t.Fatalf("Write() returned error: %v", err)
	}

	if err := Delete(path); err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected file to be removed at %s, stat err = %v", path, err)
	}
}

func TestDeleteMissingFile(t *testing.T) {
	err := Delete(filepath.Join(t.TempDir(), "missing.md"))
	if err == nil {
		t.Fatal("Delete() of a missing file should return an error")
	}
}
