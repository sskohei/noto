package index

// schema is the SQLite/FTS5 index schema described in docs/data-model.md,
// as a sequence of individually-executed statements. It is applied
// idempotently every time the index database is opened, so it can always
// be rebuilt from scratch from the note files alone.
var schema = []string{
	`CREATE TABLE IF NOT EXISTS notes (
		id          TEXT PRIMARY KEY,
		path        TEXT NOT NULL UNIQUE,
		title       TEXT NOT NULL,
		created_at  TEXT NOT NULL,
		updated_at  TEXT NOT NULL,
		mtime       INTEGER NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS note_tags (
		note_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
		tag     TEXT NOT NULL,
		PRIMARY KEY (note_id, tag)
	)`,
	`CREATE VIRTUAL TABLE IF NOT EXISTS notes_fts USING fts5(
		id UNINDEXED,
		title,
		body,
		tags,
		tokenize = 'unicode61 remove_diacritics 2'
	)`,
}
