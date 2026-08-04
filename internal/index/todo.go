package index

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/sskohei/noto/internal/storage"
)

// TodoSummary is the subset of an indexed todo's data needed to render it
// in the todo list.
type TodoSummary struct {
	ID        string
	Path      string
	Title     string
	Done      bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// indexedTodo is the subset of an indexed todo's row needed to diff it
// against the todo files on disk.
type indexedTodo struct {
	id    string
	mtime int64
}

// SyncTodos reconciles the index with the todo files in todosDir: it
// compares each file's mtime against the indexed value and reindexes only
// todos that were added, changed, or removed since the last sync.
func (db *DB) SyncTodos(todosDir string) error {
	paths, err := storage.ListTodos(todosDir)
	if err != nil {
		return fmt.Errorf("index: list todos: %w", err)
	}

	indexed, err := db.indexedTodosByPath()
	if err != nil {
		return fmt.Errorf("index: read existing todo index: %w", err)
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	onDisk := make(map[string]bool, len(paths))
	for _, path := range paths {
		onDisk[path] = true

		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("index: stat %s: %w", path, err)
		}
		mtime := info.ModTime().UnixNano()

		existing, ok := indexed[path]
		if ok && existing.mtime == mtime {
			continue // unchanged since last sync
		}

		todo, err := storage.ReadTodo(path)
		if err != nil {
			return fmt.Errorf("index: read %s: %w", path, err)
		}

		if ok {
			if err := deleteTodoRow(tx, existing.id); err != nil {
				return fmt.Errorf("index: delete stale row for %s: %w", path, err)
			}
		}
		if err := insertTodoRow(tx, path, mtime, todo); err != nil {
			return fmt.Errorf("index: insert %s: %w", path, err)
		}
	}

	for path, existing := range indexed {
		if onDisk[path] {
			continue
		}
		if err := deleteTodoRow(tx, existing.id); err != nil {
			return fmt.Errorf("index: delete removed todo %s: %w", path, err)
		}
	}

	return tx.Commit()
}

// indexedTodosByPath returns the currently indexed todos, keyed by their
// file path.
func (db *DB) indexedTodosByPath() (map[string]indexedTodo, error) {
	rows, err := db.conn.Query(`SELECT id, path, mtime FROM todos`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indexed := make(map[string]indexedTodo)
	for rows.Next() {
		var (
			id, path string
			mtime    int64
		)
		if err := rows.Scan(&id, &path, &mtime); err != nil {
			return nil, err
		}
		indexed[path] = indexedTodo{id: id, mtime: mtime}
	}
	return indexed, rows.Err()
}

// deleteTodoRow removes id's row from todos.
func deleteTodoRow(tx *sql.Tx, id string) error {
	_, err := tx.Exec(`DELETE FROM todos WHERE id = ?`, id)
	return err
}

// insertTodoRow inserts t (found at path, with the given mtime) into todos.
func insertTodoRow(tx *sql.Tx, path string, mtime int64, t storage.Todo) error {
	_, err := tx.Exec(
		`INSERT INTO todos (id, path, title, done, created_at, updated_at, mtime) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		t.ID, path, t.Title, t.Done, t.CreatedAt.Format(time.RFC3339), t.UpdatedAt.Format(time.RFC3339), mtime,
	)
	return err
}

// ListTodos returns all indexed todos, incomplete first then completed,
// each group ordered by most recently updated first.
func (db *DB) ListTodos() ([]TodoSummary, error) {
	rows, err := db.conn.Query(`SELECT id, path, title, done, created_at, updated_at FROM todos ORDER BY done ASC, updated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("index: query todos: %w", err)
	}
	defer rows.Close()

	var summaries []TodoSummary
	for rows.Next() {
		var (
			id, path, title, createdAt, updatedAt string
			done                                  bool
		)
		if err := rows.Scan(&id, &path, &title, &done, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("index: query todos: %w", err)
		}

		ct, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("index: parse created_at for %s: %w", id, err)
		}
		ut, err := time.Parse(time.RFC3339, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("index: parse updated_at for %s: %w", id, err)
		}

		summaries = append(summaries, TodoSummary{
			ID: id, Path: path, Title: title, Done: done, CreatedAt: ct, UpdatedAt: ut,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("index: query todos: %w", err)
	}

	return summaries, nil
}
