package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseMarshalTodoRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		todo Todo
	}{
		{
			name: "typical todo",
			todo: Todo{
				ID:        "018f2e4a-7a1e-7b3f-9c2a-4d8e6f1a0b33",
				Title:     "経費精算を提出する",
				Done:      false,
				CreatedAt: mustParseTime(t, "2026-08-01T09:12:00+09:00"),
				UpdatedAt: mustParseTime(t, "2026-08-01T09:12:00+09:00"),
				Body:      "領収書は経理フォルダにある\n",
			},
		},
		{
			name: "done todo, empty body",
			todo: Todo{
				ID:        "018f2e4a-0000-0000-0000-000000000000",
				Title:     "done task",
				Done:      true,
				CreatedAt: mustParseTime(t, "2026-01-01T00:00:00Z"),
				UpdatedAt: mustParseTime(t, "2026-01-02T00:00:00Z"),
				Body:      "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.todo.Marshal()
			if err != nil {
				t.Fatalf("Marshal() returned error: %v", err)
			}

			got, err := ParseTodo(data)
			if err != nil {
				t.Fatalf("ParseTodo() returned error: %v", err)
			}

			if got.ID != tt.todo.ID {
				t.Errorf("ID = %q, want %q", got.ID, tt.todo.ID)
			}
			if got.Title != tt.todo.Title {
				t.Errorf("Title = %q, want %q", got.Title, tt.todo.Title)
			}
			if got.Done != tt.todo.Done {
				t.Errorf("Done = %v, want %v", got.Done, tt.todo.Done)
			}
			if !got.CreatedAt.Equal(tt.todo.CreatedAt) {
				t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, tt.todo.CreatedAt)
			}
			if !got.UpdatedAt.Equal(tt.todo.UpdatedAt) {
				t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, tt.todo.UpdatedAt)
			}
			if got.Body != tt.todo.Body {
				t.Errorf("Body = %q, want %q", got.Body, tt.todo.Body)
			}
		})
	}
}

func TestParseTodoInvalidFrontmatter(t *testing.T) {
	_, err := ParseTodo([]byte("no frontmatter here\n"))
	if err == nil {
		t.Fatal("ParseTodo() with no frontmatter should return an error")
	}
}

func TestTodoFileName(t *testing.T) {
	createdAt := mustParseTime(t, "2026-08-01T09:12:00+09:00")

	tests := []struct {
		name string
		todo Todo
		want string
	}{
		{
			name: "typical title",
			todo: Todo{ID: "018f2e4a-7a1e-7b3f-9c2a-4d8e6f1a0b33", Title: "経費精算を提出する", CreatedAt: createdAt},
			want: "20260801T091200-経費精算を提出する.md",
		},
		{
			name: "empty title falls back to id prefix",
			todo: Todo{ID: "018f2e4a-7a1e-7b3f-9c2a-4d8e6f1a0b33", Title: "", CreatedAt: createdAt},
			want: "20260801T091200-018f2e4a.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TodoFileName(tt.todo)
			if got != tt.want {
				t.Errorf("TodoFileName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWriteReadTodoRoundTrip(t *testing.T) {
	dir := t.TempDir()

	todo := Todo{
		ID:        "018f2e4a-7a1e-7b3f-9c2a-4d8e6f1a0b33",
		Title:     "経費精算を提出する",
		Done:      false,
		CreatedAt: mustParseTime(t, "2026-08-01T09:12:00+09:00"),
		UpdatedAt: mustParseTime(t, "2026-08-01T09:20:31+09:00"),
		Body:      "領収書は経理フォルダにある\n",
	}

	path, err := GenerateTodoPath(dir, todo)
	if err != nil {
		t.Fatalf("GenerateTodoPath() returned error: %v", err)
	}

	if err := WriteTodo(path, todo); err != nil {
		t.Fatalf("WriteTodo() returned error: %v", err)
	}

	got, err := ReadTodo(path)
	if err != nil {
		t.Fatalf("ReadTodo() returned error: %v", err)
	}

	if got.ID != todo.ID || got.Title != todo.Title || got.Done != todo.Done || got.Body != todo.Body {
		t.Errorf("ReadTodo() = %+v, want %+v", got, todo)
	}
}

func TestListTodos(t *testing.T) {
	dir := t.TempDir()

	for _, name := range []string{"b.md", "a.md", "c.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
			t.Fatalf("failed to seed file %s: %v", name, err)
		}
	}

	got, err := ListTodos(dir)
	if err != nil {
		t.Fatalf("ListTodos() returned error: %v", err)
	}

	want := []string{filepath.Join(dir, "a.md"), filepath.Join(dir, "b.md")}
	if len(got) != len(want) {
		t.Fatalf("ListTodos() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ListTodos()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestGenerateTodoPathAvoidsCollisions(t *testing.T) {
	dir := t.TempDir()
	createdAt := mustParseTime(t, "2026-08-01T09:12:00+09:00")

	first := Todo{ID: "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", Title: "経費精算", CreatedAt: createdAt}
	second := Todo{ID: "018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb", Title: "経費精算", CreatedAt: createdAt}

	firstPath, err := GenerateTodoPath(dir, first)
	if err != nil {
		t.Fatalf("GenerateTodoPath() returned error: %v", err)
	}
	if err := WriteTodo(firstPath, first); err != nil {
		t.Fatalf("WriteTodo() returned error: %v", err)
	}

	secondPath, err := GenerateTodoPath(dir, second)
	if err != nil {
		t.Fatalf("GenerateTodoPath() returned error: %v", err)
	}
	if secondPath == firstPath {
		t.Fatalf("GenerateTodoPath() returned colliding path %q for second todo", secondPath)
	}
}

func TestDeleteTodoRemovesFile(t *testing.T) {
	dir := t.TempDir()
	todo := Todo{
		ID:        "018f2e4a-0000-0000-0000-000000000000",
		Title:     "test",
		CreatedAt: mustParseTime(t, "2026-01-01T00:00:00Z"),
		UpdatedAt: mustParseTime(t, "2026-01-01T00:00:00Z"),
	}

	path, err := GenerateTodoPath(dir, todo)
	if err != nil {
		t.Fatalf("GenerateTodoPath() returned error: %v", err)
	}
	if err := WriteTodo(path, todo); err != nil {
		t.Fatalf("WriteTodo() returned error: %v", err)
	}

	if err := DeleteTodo(path); err != nil {
		t.Fatalf("DeleteTodo() returned error: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected file to be removed at %s, stat err = %v", path, err)
	}
}
