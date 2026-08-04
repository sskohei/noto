package index

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sskohei/noto/internal/storage"
)

func TestSyncTodosInitialBuild(t *testing.T) {
	todosDir := t.TempDir()
	writeTodo(t, todosDir, "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "経費精算", false, "2026-01-01T00:00:00Z")
	writeTodo(t, todosDir, "018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "買い物", false, "2026-01-02T00:00:00Z")

	db := openTestDB(t)
	if err := db.SyncTodos(todosDir); err != nil {
		t.Fatalf("SyncTodos() returned error: %v", err)
	}

	assertTodoCount(t, db, 2)
}

func TestSyncTodosAddsNewTodo(t *testing.T) {
	todosDir := t.TempDir()
	writeTodo(t, todosDir, "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "first", false, "2026-01-01T00:00:00Z")

	db := openTestDB(t)
	if err := db.SyncTodos(todosDir); err != nil {
		t.Fatalf("SyncTodos() returned error: %v", err)
	}
	assertTodoCount(t, db, 1)

	writeTodo(t, todosDir, "018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "second", false, "2026-01-02T00:00:00Z")
	if err := db.SyncTodos(todosDir); err != nil {
		t.Fatalf("second SyncTodos() returned error: %v", err)
	}
	assertTodoCount(t, db, 2)
}

func TestSyncTodosReindexesChangedTodo(t *testing.T) {
	todosDir := t.TempDir()
	path := writeTodo(t, todosDir, "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "original title", false, "2026-01-01T00:00:00Z")

	db := openTestDB(t)
	if err := db.SyncTodos(todosDir); err != nil {
		t.Fatalf("SyncTodos() returned error: %v", err)
	}

	todo, err := storage.ReadTodo(path)
	if err != nil {
		t.Fatalf("storage.ReadTodo() returned error: %v", err)
	}
	todo.Title = "updated title"
	todo.Done = true
	if err := storage.WriteTodo(path, todo); err != nil {
		t.Fatalf("storage.WriteTodo() returned error: %v", err)
	}
	future := time.Now().Add(time.Hour)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatalf("os.Chtimes() returned error: %v", err)
	}

	if err := db.SyncTodos(todosDir); err != nil {
		t.Fatalf("second SyncTodos() returned error: %v", err)
	}

	assertTodoCount(t, db, 1)

	todos, err := db.ListTodos()
	if err != nil {
		t.Fatalf("ListTodos() returned error: %v", err)
	}
	if len(todos) != 1 || todos[0].Title != "updated title" || !todos[0].Done {
		t.Errorf("ListTodos() = %+v, want updated title with done=true", todos)
	}
}

func TestSyncTodosUnchangedTodoIsSkipped(t *testing.T) {
	todosDir := t.TempDir()
	writeTodo(t, todosDir, "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "title", false, "2026-01-01T00:00:00Z")

	db := openTestDB(t)
	if err := db.SyncTodos(todosDir); err != nil {
		t.Fatalf("SyncTodos() returned error: %v", err)
	}
	if err := db.SyncTodos(todosDir); err != nil {
		t.Fatalf("second SyncTodos() returned error: %v", err)
	}

	assertTodoCount(t, db, 1)
}

func TestSyncTodosRemovesDeletedTodo(t *testing.T) {
	todosDir := t.TempDir()
	writeTodo(t, todosDir, "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "keep", false, "2026-01-01T00:00:00Z")
	removePath := writeTodo(t, todosDir, "018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "remove", false, "2026-01-02T00:00:00Z")

	db := openTestDB(t)
	if err := db.SyncTodos(todosDir); err != nil {
		t.Fatalf("SyncTodos() returned error: %v", err)
	}
	assertTodoCount(t, db, 2)

	if err := os.Remove(removePath); err != nil {
		t.Fatalf("os.Remove() returned error: %v", err)
	}
	if err := db.SyncTodos(todosDir); err != nil {
		t.Fatalf("second SyncTodos() returned error: %v", err)
	}

	assertTodoCount(t, db, 1)
}

func TestSyncTodosFullRebuildAfterIndexDeletion(t *testing.T) {
	todosDir := t.TempDir()
	writeTodo(t, todosDir, "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "todo one", false, "2026-01-01T00:00:00Z")
	writeTodo(t, todosDir, "018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "todo two", false, "2026-01-02T00:00:00Z")

	dbPath := filepath.Join(t.TempDir(), "index.db")
	db1, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() returned error: %v", err)
	}
	if err := db1.SyncTodos(todosDir); err != nil {
		t.Fatalf("SyncTodos() returned error: %v", err)
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
	if err := db2.SyncTodos(todosDir); err != nil {
		t.Fatalf("rebuild SyncTodos() returned error: %v", err)
	}

	assertTodoCount(t, db2, 2)
}

func TestListTodosOrderedByDoneThenUpdatedAtDesc(t *testing.T) {
	todosDir := t.TempDir()
	writeTodo(t, todosDir, "018f2e4a-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "done old", true, "2026-01-01T00:00:00Z")
	writeTodo(t, todosDir, "018f2e4a-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "not done new", false, "2026-03-01T00:00:00Z")
	writeTodo(t, todosDir, "018f2e4a-cccc-cccc-cccc-cccccccccccc", "not done old", false, "2026-01-01T00:00:00Z")
	writeTodo(t, todosDir, "018f2e4a-dddd-dddd-dddd-dddddddddddd", "done new", true, "2026-03-01T00:00:00Z")

	db := openTestDB(t)
	if err := db.SyncTodos(todosDir); err != nil {
		t.Fatalf("SyncTodos() returned error: %v", err)
	}

	got, err := db.ListTodos()
	if err != nil {
		t.Fatalf("ListTodos() returned error: %v", err)
	}

	wantOrder := []string{"not done new", "not done old", "done new", "done old"}
	if len(got) != len(wantOrder) {
		t.Fatalf("ListTodos() returned %d todos, want %d", len(got), len(wantOrder))
	}
	for i, want := range wantOrder {
		if got[i].Title != want {
			t.Errorf("ListTodos()[%d].Title = %q, want %q", i, got[i].Title, want)
		}
	}
}

func TestListTodosEmptyIndex(t *testing.T) {
	db := openTestDB(t)

	got, err := db.ListTodos()
	if err != nil {
		t.Fatalf("ListTodos() returned error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("ListTodos() = %v, want empty", got)
	}
}

func writeTodo(t *testing.T, dir, id, title string, done bool, updatedAt string) string {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		t.Fatalf("failed to parse test time %q: %v", updatedAt, err)
	}

	todo := storage.Todo{
		ID:        id,
		Title:     title,
		Done:      done,
		CreatedAt: ts,
		UpdatedAt: ts,
	}
	path, err := storage.GenerateTodoPath(dir, todo)
	if err != nil {
		t.Fatalf("storage.GenerateTodoPath() returned error: %v", err)
	}
	if err := storage.WriteTodo(path, todo); err != nil {
		t.Fatalf("storage.WriteTodo() returned error: %v", err)
	}
	return path
}

func assertTodoCount(t *testing.T, db *DB, want int) {
	t.Helper()
	var got int
	if err := db.conn.QueryRow(`SELECT COUNT(*) FROM todos`).Scan(&got); err != nil {
		t.Fatalf("query todos count returned error: %v", err)
	}
	if got != want {
		t.Errorf("todos count = %d, want %d", got, want)
	}
}
