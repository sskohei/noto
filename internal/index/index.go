package index

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/sskohei/noto/internal/storage"
)

// DB is a handle to noto's SQLite/FTS5 search index.
type DB struct {
	conn *sql.DB
}

// Open opens (creating the file and parent directory if necessary) the
// index database at path and ensures its schema exists.
func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if _, err := conn.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		conn.Close()
		return nil, err
	}

	for _, stmt := range schema {
		if _, err := conn.Exec(stmt); err != nil {
			conn.Close()
			return nil, fmt.Errorf("index: create schema: %w", err)
		}
	}

	return &DB{conn: conn}, nil
}

// Close closes the underlying database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// indexedNote is the subset of an indexed note's row needed to diff it
// against the notes on disk.
type indexedNote struct {
	id    string
	mtime int64
}

// Sync reconciles the index with the note files in notesDir: it compares
// each file's mtime against the indexed value and reindexes only notes
// that were added, changed, or removed since the last sync.
func (db *DB) Sync(notesDir string) error {
	paths, err := storage.List(notesDir)
	if err != nil {
		return fmt.Errorf("index: list notes: %w", err)
	}

	indexed, err := db.indexedByPath()
	if err != nil {
		return fmt.Errorf("index: read existing index: %w", err)
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

		note, err := storage.Read(path)
		if err != nil {
			return fmt.Errorf("index: read %s: %w", path, err)
		}

		if ok {
			if err := deleteNote(tx, existing.id); err != nil {
				return fmt.Errorf("index: delete stale row for %s: %w", path, err)
			}
		}
		if err := insertNote(tx, path, mtime, note); err != nil {
			return fmt.Errorf("index: insert %s: %w", path, err)
		}
	}

	for path, existing := range indexed {
		if onDisk[path] {
			continue
		}
		if err := deleteNote(tx, existing.id); err != nil {
			return fmt.Errorf("index: delete removed note %s: %w", path, err)
		}
	}

	return tx.Commit()
}

// indexedByPath returns the currently indexed notes, keyed by their file
// path.
func (db *DB) indexedByPath() (map[string]indexedNote, error) {
	rows, err := db.conn.Query(`SELECT id, path, mtime FROM notes`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indexed := make(map[string]indexedNote)
	for rows.Next() {
		var (
			id, path string
			mtime    int64
		)
		if err := rows.Scan(&id, &path, &mtime); err != nil {
			return nil, err
		}
		indexed[path] = indexedNote{id: id, mtime: mtime}
	}
	return indexed, rows.Err()
}

// deleteNote removes id's rows from notes_fts and notes. note_tags rows
// are removed automatically via ON DELETE CASCADE.
func deleteNote(tx *sql.Tx, id string) error {
	if _, err := tx.Exec(`DELETE FROM notes_fts WHERE id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM notes WHERE id = ?`, id); err != nil {
		return err
	}
	return nil
}

// insertNote inserts n (found at path, with the given mtime) into notes,
// note_tags, and notes_fts.
func insertNote(tx *sql.Tx, path string, mtime int64, n storage.Note) error {
	_, err := tx.Exec(
		`INSERT INTO notes (id, path, title, created_at, updated_at, mtime) VALUES (?, ?, ?, ?, ?, ?)`,
		n.ID, path, n.Title, n.CreatedAt.Format(time.RFC3339), n.UpdatedAt.Format(time.RFC3339), mtime,
	)
	if err != nil {
		return err
	}

	for _, tag := range n.Tags {
		if _, err := tx.Exec(`INSERT INTO note_tags (note_id, tag) VALUES (?, ?)`, n.ID, tag); err != nil {
			return err
		}
	}

	_, err = tx.Exec(
		`INSERT INTO notes_fts (id, title, body, tags) VALUES (?, ?, ?, ?)`,
		n.ID, n.Title, n.Body, strings.Join(n.Tags, " "),
	)
	return err
}
