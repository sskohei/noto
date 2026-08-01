package index

import "testing"

func TestFilterNotes(t *testing.T) {
	notesDir := t.TempDir()

	writeSummaryNote(t, notesDir, "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "買い物リスト", []string{"life", "shopping"}, "2026-01-01T00:00:00Z")
	writeSummaryNote(t, notesDir, "018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "会議メモ", []string{"work"}, "2026-01-02T00:00:00Z")
	writeSummaryNote(t, notesDir, "018f2e4a-cccc-cccc-cccc-cccccccccccc", "作業ログ", []string{"work", "life"}, "2026-01-03T00:00:00Z")
	writeSummaryNote(t, notesDir, "018f2e4a-dddd-dddd-dddd-dddddddddddd", "タグなし", nil, "2026-01-04T00:00:00Z")

	db := openListTestDB(t)
	if err := db.Sync(notesDir); err != nil {
		t.Fatalf("Sync() returned error: %v", err)
	}

	tests := []struct {
		name  string
		query string
		tags  []string
		want  []string // titles, any order
	}{
		{name: "single tag", query: "", tags: []string{"work"}, want: []string{"会議メモ", "作業ログ"}},
		{name: "AND of two tags", query: "", tags: []string{"work", "life"}, want: []string{"作業ログ"}},
		{name: "tag with no matches", query: "", tags: []string{"nonexistent"}, want: nil},
		{name: "tag combined with matching search term", query: "作業", tags: []string{"work"}, want: []string{"作業ログ"}},
		{name: "tag combined with non-matching search term", query: "買い物", tags: []string{"work"}, want: nil},
		{name: "search term only", query: "会議", tags: nil, want: []string{"会議メモ"}},
		{name: "no query and no tags returns everything", query: "", tags: nil, want: []string{"買い物リスト", "会議メモ", "作業ログ", "タグなし"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := db.FilterNotes(tt.query, tt.tags)
			if err != nil {
				t.Fatalf("FilterNotes(%q, %v) returned error: %v", tt.query, tt.tags, err)
			}

			gotTitles := make([]string, len(got))
			for i, n := range got {
				gotTitles[i] = n.Title
			}

			if !sameSet(gotTitles, tt.want) {
				t.Errorf("FilterNotes(%q, %v) titles = %v, want %v (any order)", tt.query, tt.tags, gotTitles, tt.want)
			}
		})
	}
}

func TestTags(t *testing.T) {
	notesDir := t.TempDir()

	writeSummaryNote(t, notesDir, "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "買い物リスト", []string{"life", "shopping"}, "2026-01-01T00:00:00Z")
	writeSummaryNote(t, notesDir, "018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "会議メモ", []string{"work", "life"}, "2026-01-02T00:00:00Z")
	writeSummaryNote(t, notesDir, "018f2e4a-cccc-cccc-cccc-cccccccccccc", "タグなし", nil, "2026-01-03T00:00:00Z")

	db := openListTestDB(t)
	if err := db.Sync(notesDir); err != nil {
		t.Fatalf("Sync() returned error: %v", err)
	}

	got, err := db.Tags()
	if err != nil {
		t.Fatalf("Tags() returned error: %v", err)
	}

	want := []string{"life", "shopping", "work"}
	if !equalSlices(got, want) {
		t.Errorf("Tags() = %v, want %v (sorted, deduplicated)", got, want)
	}
}

func TestTagsEmptyIndex(t *testing.T) {
	db := openListTestDB(t)

	got, err := db.Tags()
	if err != nil {
		t.Fatalf("Tags() returned error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Tags() = %v, want empty", got)
	}
}
