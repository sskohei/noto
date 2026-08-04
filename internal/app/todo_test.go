package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sskohei/noto/internal/config"
	"github.com/sskohei/noto/internal/index"
	"github.com/sskohei/noto/internal/storage"
)

func TestTodosDir(t *testing.T) {
	cfg := config.Config{NotesDir: "/home/user/.local/share/noto/notes"}
	got := TodosDir(cfg)
	want := "/home/user/.local/share/noto/todos"
	if got != want {
		t.Errorf("TodosDir() = %q, want %q", got, want)
	}
}

func TestPrepareNewTodo(t *testing.T) {
	cfg := config.Config{NotesDir: filepath.Join(t.TempDir(), "notes")}

	todo, path, err := PrepareNewTodo(cfg, "経費精算を提出する")
	if err != nil {
		t.Fatalf("PrepareNewTodo() returned error: %v", err)
	}

	if todo.ID == "" {
		t.Error("PrepareNewTodo() todo ID is empty")
	}
	if todo.Title != "経費精算を提出する" {
		t.Errorf("todo.Title = %q, want %q", todo.Title, "経費精算を提出する")
	}
	if todo.Done {
		t.Error("todo.Done = true, want false for a new todo")
	}
	if todo.CreatedAt.IsZero() || todo.UpdatedAt.IsZero() {
		t.Error("todo.CreatedAt/UpdatedAt should be set")
	}

	got, err := storage.ReadTodo(path)
	if err != nil {
		t.Fatalf("storage.ReadTodo(%s) returned error: %v", path, err)
	}
	if got.ID != todo.ID || got.Title != todo.Title {
		t.Errorf("ReadTodo() = %+v, want %+v", got, todo)
	}
	if filepath.Dir(path) != TodosDir(cfg) {
		t.Errorf("path dir = %q, want %q", filepath.Dir(path), TodosDir(cfg))
	}
}

func TestFinalizeNewTodo(t *testing.T) {
	todosDir := t.TempDir()

	now := time.Now()
	todo := storage.Todo{
		ID:        "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Title:     "saved by editor",
		CreatedAt: now,
		UpdatedAt: now,
		Body:      "詳細メモ\n",
	}
	path, err := storage.GenerateTodoPath(todosDir, todo)
	if err != nil {
		t.Fatalf("storage.GenerateTodoPath() returned error: %v", err)
	}
	if err := storage.WriteTodo(path, todo); err != nil {
		t.Fatalf("storage.WriteTodo() returned error: %v", err)
	}

	idx, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open() returned error: %v", err)
	}
	t.Cleanup(func() { idx.Close() })

	got, err := FinalizeNewTodo(idx, todosDir, path)
	if err != nil {
		t.Fatalf("FinalizeNewTodo() returned error: %v", err)
	}
	if got.ID != todo.ID || got.Title != todo.Title || got.Body != todo.Body {
		t.Errorf("FinalizeNewTodo() = %+v, want %+v", got, todo)
	}

	todos, err := idx.ListTodos()
	if err != nil {
		t.Fatalf("ListTodos() returned error: %v", err)
	}
	if len(todos) != 1 {
		t.Fatalf("ListTodos() = %+v, want 1 todo", todos)
	}
}

func TestFinalizeTodoEdit(t *testing.T) {
	todosDir := t.TempDir()

	original := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	todo := storage.Todo{
		ID:        "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Title:     "edited by user",
		CreatedAt: original,
		UpdatedAt: original,
		Body:      "edited body\n",
	}
	path, err := storage.GenerateTodoPath(todosDir, todo)
	if err != nil {
		t.Fatalf("storage.GenerateTodoPath() returned error: %v", err)
	}
	if err := storage.WriteTodo(path, todo); err != nil {
		t.Fatalf("storage.WriteTodo() returned error: %v", err)
	}

	idx, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open() returned error: %v", err)
	}
	t.Cleanup(func() { idx.Close() })

	before := time.Now()
	got, err := FinalizeTodoEdit(idx, todosDir, path)
	if err != nil {
		t.Fatalf("FinalizeTodoEdit() returned error: %v", err)
	}

	if got.ID != todo.ID || got.Title != todo.Title || got.Body != todo.Body {
		t.Errorf("FinalizeTodoEdit() = %+v, want ID/Title/Body matching %+v", got, todo)
	}
	if !got.UpdatedAt.After(original) {
		t.Errorf("UpdatedAt = %v, want after original %v", got.UpdatedAt, original)
	}
	if got.UpdatedAt.Before(before) {
		t.Errorf("UpdatedAt = %v, want at or after %v", got.UpdatedAt, before)
	}

	onDisk, err := storage.ReadTodo(path)
	if err != nil {
		t.Fatalf("storage.ReadTodo() returned error: %v", err)
	}
	if !onDisk.UpdatedAt.Equal(got.UpdatedAt) {
		t.Errorf("on-disk UpdatedAt = %v, want %v", onDisk.UpdatedAt, got.UpdatedAt)
	}
}

func TestDeleteTodo(t *testing.T) {
	todosDir := t.TempDir()

	now := time.Now()
	todo := storage.Todo{
		ID:        "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Title:     "to be deleted",
		CreatedAt: now,
		UpdatedAt: now,
	}
	path, err := storage.GenerateTodoPath(todosDir, todo)
	if err != nil {
		t.Fatalf("storage.GenerateTodoPath() returned error: %v", err)
	}
	if err := storage.WriteTodo(path, todo); err != nil {
		t.Fatalf("storage.WriteTodo() returned error: %v", err)
	}

	idx, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open() returned error: %v", err)
	}
	t.Cleanup(func() { idx.Close() })
	if err := idx.SyncTodos(todosDir); err != nil {
		t.Fatalf("SyncTodos() returned error: %v", err)
	}

	if err := DeleteTodo(idx, todosDir, path); err != nil {
		t.Fatalf("DeleteTodo() returned error: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected file to be removed at %s, stat err = %v", path, err)
	}

	todos, err := idx.ListTodos()
	if err != nil {
		t.Fatalf("ListTodos() returned error: %v", err)
	}
	if len(todos) != 0 {
		t.Errorf("ListTodos() = %+v, want empty after DeleteTodo", todos)
	}
}

func TestDeleteTodoMissingFile(t *testing.T) {
	todosDir := t.TempDir()

	idx, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open() returned error: %v", err)
	}
	t.Cleanup(func() { idx.Close() })

	err = DeleteTodo(idx, todosDir, filepath.Join(todosDir, "missing.md"))
	if err == nil {
		t.Fatal("DeleteTodo() of a missing file should return an error")
	}
}

func TestListTodos(t *testing.T) {
	todosDir := t.TempDir()

	now := time.Now()
	todo := storage.Todo{
		ID:        "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Title:     "shopping",
		CreatedAt: now,
		UpdatedAt: now,
	}
	path, err := storage.GenerateTodoPath(todosDir, todo)
	if err != nil {
		t.Fatalf("storage.GenerateTodoPath() returned error: %v", err)
	}
	if err := storage.WriteTodo(path, todo); err != nil {
		t.Fatalf("storage.WriteTodo() returned error: %v", err)
	}

	idx, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open() returned error: %v", err)
	}
	t.Cleanup(func() { idx.Close() })
	if err := idx.SyncTodos(todosDir); err != nil {
		t.Fatalf("SyncTodos() returned error: %v", err)
	}

	got, err := ListTodos(idx)
	if err != nil {
		t.Fatalf("ListTodos() returned error: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("ListTodos() = %v, want 1 todo", got)
	}
	if got[0].ID != todo.ID || got[0].Title != todo.Title {
		t.Errorf("ListTodos()[0] = %+v, want ID=%q Title=%q", got[0], todo.ID, todo.Title)
	}
}

func TestToggleTodoDone(t *testing.T) {
	todosDir := t.TempDir()

	original := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	todo := storage.Todo{
		ID:        "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Title:     "toggle me",
		Done:      false,
		CreatedAt: original,
		UpdatedAt: original,
	}
	path, err := storage.GenerateTodoPath(todosDir, todo)
	if err != nil {
		t.Fatalf("storage.GenerateTodoPath() returned error: %v", err)
	}
	if err := storage.WriteTodo(path, todo); err != nil {
		t.Fatalf("storage.WriteTodo() returned error: %v", err)
	}

	idx, err := index.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatalf("index.Open() returned error: %v", err)
	}
	t.Cleanup(func() { idx.Close() })

	got, err := ToggleTodoDone(idx, todosDir, path)
	if err != nil {
		t.Fatalf("ToggleTodoDone() returned error: %v", err)
	}
	if !got.Done {
		t.Error("Done = false after first toggle, want true")
	}
	if !got.UpdatedAt.After(original) {
		t.Errorf("UpdatedAt = %v, want after original %v", got.UpdatedAt, original)
	}

	got, err = ToggleTodoDone(idx, todosDir, path)
	if err != nil {
		t.Fatalf("second ToggleTodoDone() returned error: %v", err)
	}
	if got.Done {
		t.Error("Done = true after second toggle, want false")
	}

	onDisk, err := storage.ReadTodo(path)
	if err != nil {
		t.Fatalf("storage.ReadTodo() returned error: %v", err)
	}
	if onDisk.Done != got.Done {
		t.Errorf("on-disk Done = %v, want %v", onDisk.Done, got.Done)
	}
}
