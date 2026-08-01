package storage

import (
	"testing"
	"time"
)

func TestParseMarshalRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		note Note
	}{
		{
			name: "typical note",
			note: Note{
				ID:        "018f2e4a-6b3a-7c9e-9a1a-2f7e1a9c4b21",
				Title:     "買い物リスト",
				Tags:      []string{"life", "shopping"},
				CreatedAt: mustParseTime(t, "2026-08-01T09:12:00+09:00"),
				UpdatedAt: mustParseTime(t, "2026-08-01T09:20:31+09:00"),
				Body:      "- 牛乳\n- コーヒー豆\n",
			},
		},
		{
			name: "no tags, empty body",
			note: Note{
				ID:        "018f2e4a-0000-0000-0000-000000000000",
				Title:     "",
				Tags:      nil,
				CreatedAt: mustParseTime(t, "2026-01-01T00:00:00Z"),
				UpdatedAt: mustParseTime(t, "2026-01-01T00:00:00Z"),
				Body:      "",
			},
		},
		{
			name: "multiline body",
			note: Note{
				ID:        "018f2e4a-1111-1111-1111-111111111111",
				Title:     "meeting notes",
				Tags:      []string{"work"},
				CreatedAt: mustParseTime(t, "2026-03-15T14:30:00+09:00"),
				UpdatedAt: mustParseTime(t, "2026-03-15T15:00:00+09:00"),
				Body:      "line one\n\nline two\n- bullet\n",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.note.Marshal()
			if err != nil {
				t.Fatalf("Marshal() returned error: %v", err)
			}

			got, err := Parse(data)
			if err != nil {
				t.Fatalf("Parse() returned error: %v", err)
			}

			if got.ID != tt.note.ID {
				t.Errorf("ID = %q, want %q", got.ID, tt.note.ID)
			}
			if got.Title != tt.note.Title {
				t.Errorf("Title = %q, want %q", got.Title, tt.note.Title)
			}
			if !stringSlicesEqual(got.Tags, tt.note.Tags) {
				t.Errorf("Tags = %v, want %v", got.Tags, tt.note.Tags)
			}
			if !got.CreatedAt.Equal(tt.note.CreatedAt) {
				t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, tt.note.CreatedAt)
			}
			if !got.UpdatedAt.Equal(tt.note.UpdatedAt) {
				t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, tt.note.UpdatedAt)
			}
			if got.Body != tt.note.Body {
				t.Errorf("Body = %q, want %q", got.Body, tt.note.Body)
			}
		})
	}
}

func TestParseInvalidFrontmatter(t *testing.T) {
	_, err := Parse([]byte("no frontmatter here\n"))
	if err == nil {
		t.Fatal("Parse() with no frontmatter should return an error")
	}
}

func TestFileName(t *testing.T) {
	createdAt := mustParseTime(t, "2026-08-01T09:12:00+09:00")

	tests := []struct {
		name string
		note Note
		want string
	}{
		{
			name: "typical title",
			note: Note{ID: "018f2e4a-6b3a-7c9e-9a1a-2f7e1a9c4b21", Title: "買い物リスト", CreatedAt: createdAt},
			want: "20260801T091200-買い物リスト.md",
		},
		{
			name: "empty title falls back to id prefix",
			note: Note{ID: "018f2e4a-6b3a-7c9e-9a1a-2f7e1a9c4b21", Title: "", CreatedAt: createdAt},
			want: "20260801T091200-018f2e4a.md",
		},
		{
			name: "whitespace-only title falls back to id prefix",
			note: Note{ID: "018f2e4a-6b3a-7c9e-9a1a-2f7e1a9c4b21", Title: "   ", CreatedAt: createdAt},
			want: "20260801T091200-018f2e4a.md",
		},
		{
			name: "title made only of unsafe characters falls back to id prefix",
			note: Note{ID: "018f2e4a-6b3a-7c9e-9a1a-2f7e1a9c4b21", Title: "/\\", CreatedAt: createdAt},
			want: "20260801T091200-018f2e4a.md",
		},
		{
			name: "short id is used as-is when title is empty",
			note: Note{ID: "abc", Title: "", CreatedAt: createdAt},
			want: "20260801T091200-abc.md",
		},
		{
			name: "unsafe characters are stripped but rest of title is kept",
			note: Note{ID: "018f2e4a-6b3a-7c9e-9a1a-2f7e1a9c4b21", Title: "a/b\\c", CreatedAt: createdAt},
			want: "20260801T091200-abc.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FileName(tt.note)
			if got != tt.want {
				t.Errorf("FileName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("failed to parse test time %q: %v", s, err)
	}
	return tm
}

func stringSlicesEqual(a, b []string) bool {
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
