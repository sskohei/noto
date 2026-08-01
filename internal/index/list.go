package index

import (
	"fmt"
	"strings"
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
	return db.query(`SELECT id, path, title, updated_at FROM notes ORDER BY updated_at DESC`)
}

// Search returns indexed notes whose title, body, or tags contain q
// (case-insensitive substring match), ordered by most recently updated
// first. An empty q behaves like List.
//
// This deliberately uses LIKE against notes_fts rather than an FTS5 MATCH
// query: with the unicode61 tokenizer, a Japanese field with no whitespace
// becomes a single token, so MATCH's prefix search only ever matches
// whole-field prefixes, not substrings buried mid-field. LIKE scans the
// stored column text directly regardless of tokenizer, giving correct
// substring matches (including single-character queries) in both Japanese
// and English. At the scale of a personal note collection, a full scan via
// LIKE is fast enough that FTS5 ranking isn't needed.
func (db *DB) Search(q string) ([]NoteSummary, error) {
	return db.FilterNotes(q, nil)
}

// Tags returns every distinct tag present in the index, sorted
// alphabetically.
func (db *DB) Tags() ([]string, error) {
	rows, err := db.conn.Query(`SELECT DISTINCT tag FROM note_tags ORDER BY tag`)
	if err != nil {
		return nil, fmt.Errorf("index: list distinct tags: %w", err)
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, fmt.Errorf("index: list distinct tags: %w", err)
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

// FilterNotes returns indexed notes matching q (see Search) and, if tags is
// non-empty, having every tag in tags (AND), ordered by most recently
// updated first. An empty q and no tags behave like List.
func (db *DB) FilterNotes(q string, tags []string) ([]NoteSummary, error) {
	if q == "" && len(tags) == 0 {
		return db.List()
	}

	var (
		conditions []string
		args       []any
	)

	if q != "" {
		like := "%" + escapeLike(q) + "%"
		conditions = append(conditions, `(f.title LIKE ? ESCAPE '\' OR f.body LIKE ? ESCAPE '\' OR f.tags LIKE ? ESCAPE '\')`)
		args = append(args, like, like, like)
	}

	if len(tags) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(tags)), ",")
		conditions = append(conditions, fmt.Sprintf(
			`n.id IN (SELECT note_id FROM note_tags WHERE tag IN (%s) GROUP BY note_id HAVING COUNT(DISTINCT tag) = ?)`,
			placeholders))
		for _, tag := range tags {
			args = append(args, tag)
		}
		args = append(args, len(tags))
	}

	sqlQuery := `
		SELECT n.id, n.path, n.title, n.updated_at
		FROM notes n
		JOIN notes_fts f ON f.id = n.id
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY n.updated_at DESC`
	return db.query(sqlQuery, args...)
}

// escapeLike escapes \, %, and _ so q is matched literally when used with
// a LIKE ... ESCAPE '\' pattern.
func escapeLike(q string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(q)
}

// query runs a SELECT (id, path, title, updated_at) statement and merges
// in each note's tags.
func (db *DB) query(sql string, args ...any) ([]NoteSummary, error) {
	rows, err := db.conn.Query(sql, args...)
	if err != nil {
		return nil, fmt.Errorf("index: query notes: %w", err)
	}
	defer rows.Close()

	var summaries []NoteSummary
	for rows.Next() {
		var id, path, title, updatedAt string
		if err := rows.Scan(&id, &path, &title, &updatedAt); err != nil {
			return nil, fmt.Errorf("index: query notes: %w", err)
		}

		t, err := time.Parse(time.RFC3339, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("index: parse updated_at for %s: %w", id, err)
		}

		summaries = append(summaries, NoteSummary{ID: id, Path: path, Title: title, UpdatedAt: t})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("index: query notes: %w", err)
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
