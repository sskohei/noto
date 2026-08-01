package index

import (
	"fmt"
	"time"
)

// NoteSummary is the subset of an indexed note's data needed to render it
// in the note list.
type NoteSummary struct {
	ID        string
	Path      string
	Title     string
	Tags      []string
	UpdatedAt time.Time
}

// List returns all indexed notes, ordered by most recently updated first.
func (db *DB) List() ([]NoteSummary, error) {
	rows, err := db.conn.Query(`SELECT id, path, title, updated_at FROM notes ORDER BY updated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("index: list notes: %w", err)
	}
	defer rows.Close()

	var summaries []NoteSummary
	for rows.Next() {
		var id, path, title, updatedAt string
		if err := rows.Scan(&id, &path, &title, &updatedAt); err != nil {
			return nil, fmt.Errorf("index: list notes: %w", err)
		}

		t, err := time.Parse(time.RFC3339, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("index: parse updated_at for %s: %w", id, err)
		}

		summaries = append(summaries, NoteSummary{ID: id, Path: path, Title: title, UpdatedAt: t})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("index: list notes: %w", err)
	}

	tags, err := db.tagsByNoteID()
	if err != nil {
		return nil, err
	}
	for i := range summaries {
		summaries[i].Tags = tags[summaries[i].ID]
	}

	return summaries, nil
}

// tagsByNoteID returns every note's tags, keyed by note id.
func (db *DB) tagsByNoteID() (map[string][]string, error) {
	rows, err := db.conn.Query(`SELECT note_id, tag FROM note_tags ORDER BY note_id, tag`)
	if err != nil {
		return nil, fmt.Errorf("index: list tags: %w", err)
	}
	defer rows.Close()

	tags := make(map[string][]string)
	for rows.Next() {
		var noteID, tag string
		if err := rows.Scan(&noteID, &tag); err != nil {
			return nil, fmt.Errorf("index: list tags: %w", err)
		}
		tags[noteID] = append(tags[noteID], tag)
	}
	return tags, rows.Err()
}
