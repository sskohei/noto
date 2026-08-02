package index

import (
	"testing"

	"github.com/sskohei/noto/internal/storage"
)

func TestSearch(t *testing.T) {
	notesDir := t.TempDir()

	writeSummaryNote(t, notesDir, "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "買い物リスト", []string{"life", "shopping"}, "2026-01-01T00:00:00Z")
	writeSummaryNoteWithBody(t, notesDir, "018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "会議メモ", nil, "2026-01-02T00:00:00Z", "来週の会議について議論する")
	writeSummaryNoteWithBody(t, notesDir, "018f2e4a-cccc-cccc-cccc-cccccccccccc", "冷蔵庫の掃除", nil, "2026-01-03T00:00:00Z", "冷蔵庫を掃除する予定")
	writeSummaryNoteWithBody(t, notesDir, "018f2e4a-dddd-dddd-dddd-dddddddddddd", "英語のメモ", []string{"work"}, "2026-01-04T00:00:00Z", "remember to buy shopping bags")

	db := openListTestDB(t)
	if err := db.Sync(notesDir); err != nil {
		t.Fatalf("Sync() returned error: %v", err)
	}

	tests := []struct {
		name  string
		query string
		want  []string // titles, any order unless noted
	}{
		{name: "japanese two-char title substring", query: "会議", want: []string{"会議メモ"}},
		{name: "japanese two-char body substring buried mid-field", query: "議論", want: []string{"会議メモ"}},
		{name: "japanese single-char query", query: "庫", want: []string{"冷蔵庫の掃除"}},
		{name: "japanese title substring not at start", query: "掃除", want: []string{"冷蔵庫の掃除"}},
		{name: "tag match", query: "shopping", want: []string{"買い物リスト", "英語のメモ"}},
		{name: "english body substring", query: "buy", want: []string{"英語のメモ"}},
		{name: "case-insensitive english", query: "SHOPPING", want: []string{"買い物リスト", "英語のメモ"}},
		{name: "no match", query: "存在しない", want: nil},
		{name: "empty query returns everything", query: "", want: []string{"英語のメモ", "冷蔵庫の掃除", "会議メモ", "買い物リスト"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := db.Search(tt.query)
			if err != nil {
				t.Fatalf("Search(%q) returned error: %v", tt.query, err)
			}

			gotTitles := make([]string, len(got))
			for i, n := range got {
				gotTitles[i] = n.Title
			}

			if tt.name == "empty query returns everything" {
				if !equalSlices(gotTitles, tt.want) {
					t.Errorf("Search(%q) titles = %v, want %v (in order)", tt.query, gotTitles, tt.want)
				}
				return
			}

			if !sameSet(gotTitles, tt.want) {
				t.Errorf("Search(%q) titles = %v, want %v (any order)", tt.query, gotTitles, tt.want)
			}
		})
	}
}

func TestSearchEscapesLikeWildcards(t *testing.T) {
	notesDir := t.TempDir()
	writeSummaryNote(t, notesDir, "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "100% done", nil, "2026-01-01T00:00:00Z")
	writeSummaryNote(t, notesDir, "018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "other note", nil, "2026-01-02T00:00:00Z")

	db := openListTestDB(t)
	if err := db.Sync(notesDir); err != nil {
		t.Fatalf("Sync() returned error: %v", err)
	}

	got, err := db.Search("100%")
	if err != nil {
		t.Fatalf("Search() returned error: %v", err)
	}
	if len(got) != 1 || got[0].Title != "100% done" {
		t.Errorf("Search(%q) = %v, want exactly [100%% done]", "100%", got)
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	count := make(map[string]int)
	for _, s := range a {
		count[s]++
	}
	for _, s := range b {
		count[s]--
	}
	for _, c := range count {
		if c != 0 {
			return false
		}
	}
	return true
}

func writeSummaryNoteWithBody(t *testing.T, dir, id, title string, tags []string, updatedAt, body string) string {
	t.Helper()
	path := writeSummaryNote(t, dir, id, title, tags, updatedAt)
	n, err := storage.Read(path)
	if err != nil {
		t.Fatalf("failed to read back note: %v", err)
	}
	n.Body = body
	if err := storage.Write(path, n); err != nil {
		t.Fatalf("failed to rewrite note body: %v", err)
	}
	return path
}
