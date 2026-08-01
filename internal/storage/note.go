package storage

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"gopkg.in/yaml.v3"
)

// Note is the in-memory representation of a single note: the YAML
// frontmatter fields plus the Markdown body.
type Note struct {
	ID        string
	Title     string
	Tags      []string
	CreatedAt time.Time
	UpdatedAt time.Time
	Body      string
}

// frontmatter mirrors the YAML block described in docs/data-model.md.
type frontmatter struct {
	ID        string    `yaml:"id"`
	Title     string    `yaml:"title"`
	Tags      []string  `yaml:"tags,flow"`
	CreatedAt time.Time `yaml:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at"`
}

// frontmatterPattern splits a note file into its YAML frontmatter block and
// the Markdown body that follows it.
var frontmatterPattern = regexp.MustCompile(`(?s)\A---\n(.*?)\n---\n\n?(.*)\z`)

// Parse decodes a note file's raw contents into a Note.
func Parse(data []byte) (Note, error) {
	m := frontmatterPattern.FindSubmatch(data)
	if m == nil {
		return Note{}, fmt.Errorf("storage: no frontmatter found")
	}

	var fm frontmatter
	if err := yaml.Unmarshal(m[1], &fm); err != nil {
		return Note{}, fmt.Errorf("storage: parse frontmatter: %w", err)
	}

	return Note{
		ID:        fm.ID,
		Title:     fm.Title,
		Tags:      fm.Tags,
		CreatedAt: fm.CreatedAt,
		UpdatedAt: fm.UpdatedAt,
		Body:      string(m[2]),
	}, nil
}

// Marshal serializes a Note back into a note file's raw contents.
func (n Note) Marshal() ([]byte, error) {
	fm := frontmatter{
		ID:        n.ID,
		Title:     n.Title,
		Tags:      n.Tags,
		CreatedAt: n.CreatedAt,
		UpdatedAt: n.UpdatedAt,
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")

	enc := yaml.NewEncoder(&buf)
	if err := enc.Encode(fm); err != nil {
		return nil, fmt.Errorf("storage: marshal frontmatter: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("storage: marshal frontmatter: %w", err)
	}

	buf.WriteString("---\n\n")
	buf.WriteString(n.Body)

	return buf.Bytes(), nil
}

// FileName generates the on-disk file name for n, following the
// "<created_at>-<slug>.md" rule in docs/data-model.md. If the title is
// empty, or slugifies to an empty string, the note's id (first 8 chars) is
// used instead.
func FileName(n Note) string {
	slug := slugify(n.Title)
	if slug == "" {
		slug = shortID(n.ID)
	}
	return n.CreatedAt.Format("20060102T150405") + "-" + slug + ".md"
}

// shortID returns the first 8 characters of id, or id itself if shorter.
func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

// slugify strips characters that are unsafe in file names (path
// separators and control characters) while preserving everything else,
// including non-ASCII text.
func slugify(title string) string {
	title = strings.TrimSpace(title)

	var b strings.Builder
	for _, r := range title {
		if r == '/' || r == '\\' || unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}

	return strings.TrimSpace(b.String())
}
