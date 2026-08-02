package app

import "github.com/sskohei/noto/internal/index"

// SearchNotes returns the notes in the index whose title, body, or tags
// contain query, most recently updated first. An empty query returns
// every note.
func SearchNotes(idx *index.DB, query string) ([]index.NoteSummary, error) {
	return idx.Search(query)
}
