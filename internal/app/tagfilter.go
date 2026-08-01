package app

import "noto/internal/index"

// ListTags returns every distinct tag present in the index, sorted
// alphabetically.
func ListTags(idx *index.DB) ([]string, error) {
	return idx.Tags()
}

// FilterNotes returns the notes in the index matching query (see
// SearchNotes) and, if tags is non-empty, having every tag in tags (AND),
// most recently updated first. An empty query and no tags return every
// note.
func FilterNotes(idx *index.DB, query string, tags []string) ([]index.NoteSummary, error) {
	return idx.FilterNotes(query, tags)
}
