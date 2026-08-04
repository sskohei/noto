package app

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/sskohei/noto/internal/config"
	"github.com/sskohei/noto/internal/index"
	"github.com/sskohei/noto/internal/storage"
)

// TodosDir derives the todo storage directory from cfg.NotesDir, per
// docs/data-model.md (todos/ lives alongside the notes/ directory).
func TodosDir(cfg config.Config) string {
	return filepath.Join(filepath.Dir(cfg.NotesDir), "todos")
}

// PrepareNewTodo generates a new todo (id, timestamps) with the given
// title and an empty body, and writes its frontmatter-only template file
// into TodosDir(cfg). It returns the todo and the path it was written to.
func PrepareNewTodo(cfg config.Config, title string) (storage.Todo, string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return storage.Todo{}, "", fmt.Errorf("app: generate todo id: %w", err)
	}

	now := time.Now()
	t := storage.Todo{
		ID:        id.String(),
		Title:     title,
		CreatedAt: now,
		UpdatedAt: now,
	}

	path, err := storage.GenerateTodoPath(TodosDir(cfg), t)
	if err != nil {
		return storage.Todo{}, "", fmt.Errorf("app: generate todo path: %w", err)
	}

	if err := storage.WriteTodo(path, t); err != nil {
		return storage.Todo{}, "", fmt.Errorf("app: write todo template: %w", err)
	}

	return t, path, nil
}

// FinalizeNewTodo re-reads path (as left behind by an editor session) and
// reconciles the search index against todosDir.
func FinalizeNewTodo(idx *index.DB, todosDir, path string) (storage.Todo, error) {
	t, err := storage.ReadTodo(path)
	if err != nil {
		return storage.Todo{}, fmt.Errorf("app: read saved todo: %w", err)
	}

	if err := idx.SyncTodos(todosDir); err != nil {
		return storage.Todo{}, fmt.Errorf("app: sync todo index: %w", err)
	}

	return t, nil
}

// FinalizeTodoEdit re-reads path (as left behind by an editor session),
// bumps its updated_at to now, persists that change, and reconciles the
// search index against todosDir.
func FinalizeTodoEdit(idx *index.DB, todosDir, path string) (storage.Todo, error) {
	t, err := storage.ReadTodo(path)
	if err != nil {
		return storage.Todo{}, fmt.Errorf("app: read edited todo: %w", err)
	}

	t.UpdatedAt = time.Now()
	if err := storage.WriteTodo(path, t); err != nil {
		return storage.Todo{}, fmt.Errorf("app: write edited todo: %w", err)
	}

	if err := idx.SyncTodos(todosDir); err != nil {
		return storage.Todo{}, fmt.Errorf("app: sync todo index: %w", err)
	}

	return t, nil
}

// DeleteTodo deletes the todo file at path and reconciles the search index
// against todosDir.
func DeleteTodo(idx *index.DB, todosDir, path string) error {
	if err := storage.DeleteTodo(path); err != nil {
		return fmt.Errorf("app: delete todo: %w", err)
	}

	if err := idx.SyncTodos(todosDir); err != nil {
		return fmt.Errorf("app: sync todo index: %w", err)
	}

	return nil
}

// DeleteCompletedTodos deletes every completed (done) todo file and
// reconciles the search index against todosDir. It returns the number of
// todos deleted.
func DeleteCompletedTodos(idx *index.DB, todosDir string) (int, error) {
	todos, err := idx.ListTodos()
	if err != nil {
		return 0, fmt.Errorf("app: list todos: %w", err)
	}

	deleted := 0
	for _, t := range todos {
		if !t.Done {
			continue
		}
		if err := storage.DeleteTodo(t.Path); err != nil {
			return deleted, fmt.Errorf("app: delete completed todo: %w", err)
		}
		deleted++
	}

	if err := idx.SyncTodos(todosDir); err != nil {
		return deleted, fmt.Errorf("app: sync todo index: %w", err)
	}

	return deleted, nil
}

// ListTodos returns the todos currently in the index, incomplete first then
// completed, each group ordered by most recently updated first.
func ListTodos(idx *index.DB) ([]index.TodoSummary, error) {
	return idx.ListTodos()
}

// ToggleTodoDone flips the done state of the todo at path, bumps its
// updated_at, persists the change, reconciles the search index against
// todosDir, and returns the updated todo.
func ToggleTodoDone(idx *index.DB, todosDir, path string) (storage.Todo, error) {
	t, err := storage.ReadTodo(path)
	if err != nil {
		return storage.Todo{}, fmt.Errorf("app: read todo: %w", err)
	}

	t.Done = !t.Done
	t.UpdatedAt = time.Now()
	if err := storage.WriteTodo(path, t); err != nil {
		return storage.Todo{}, fmt.Errorf("app: write todo: %w", err)
	}

	if err := idx.SyncTodos(todosDir); err != nil {
		return storage.Todo{}, fmt.Errorf("app: sync todo index: %w", err)
	}

	return t, nil
}
