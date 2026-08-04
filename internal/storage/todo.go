package storage

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Todo is the in-memory representation of a single todo item: the YAML
// frontmatter fields plus the Markdown body.
type Todo struct {
	ID        string
	Title     string
	Done      bool
	CreatedAt time.Time
	UpdatedAt time.Time
	Body      string
}

// todoFrontmatter mirrors the YAML block described in docs/data-model.md.
type todoFrontmatter struct {
	ID        string    `yaml:"id"`
	Title     string    `yaml:"title"`
	Done      bool      `yaml:"done"`
	CreatedAt time.Time `yaml:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at"`
}

// ParseTodo decodes a todo file's raw contents into a Todo.
func ParseTodo(data []byte) (Todo, error) {
	m := frontmatterPattern.FindSubmatch(data)
	if m == nil {
		return Todo{}, fmt.Errorf("storage: no frontmatter found")
	}

	var fm todoFrontmatter
	if err := yaml.Unmarshal(m[1], &fm); err != nil {
		return Todo{}, fmt.Errorf("storage: parse frontmatter: %w", err)
	}

	return Todo{
		ID:        fm.ID,
		Title:     fm.Title,
		Done:      fm.Done,
		CreatedAt: fm.CreatedAt,
		UpdatedAt: fm.UpdatedAt,
		Body:      string(m[2]),
	}, nil
}

// Marshal serializes a Todo back into a todo file's raw contents.
func (t Todo) Marshal() ([]byte, error) {
	fm := todoFrontmatter{
		ID:        t.ID,
		Title:     t.Title,
		Done:      t.Done,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
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
	buf.WriteString(t.Body)

	return buf.Bytes(), nil
}

// TodoFileName generates the on-disk file name for t, following the same
// "<created_at>-<slug>.md" rule as notes (see docs/data-model.md). If the
// title is empty, or slugifies to an empty string, the todo's id (first 8
// chars) is used instead.
func TodoFileName(t Todo) string {
	slug := slugify(t.Title)
	if slug == "" {
		slug = shortID(t.ID)
	}
	return t.CreatedAt.Format("20060102T150405") + "-" + slug + ".md"
}

// ReadTodo reads and parses the todo file at path.
func ReadTodo(path string) (Todo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Todo{}, err
	}

	t, err := ParseTodo(data)
	if err != nil {
		return Todo{}, fmt.Errorf("storage: read %s: %w", path, err)
	}
	return t, nil
}

// WriteTodo serializes t and writes it to path, creating the parent
// directory if necessary. It does not compute a file name; callers that are
// creating a new todo should obtain path via GenerateTodoPath first.
// Callers updating an existing todo can pass the path it was read from.
func WriteTodo(path string, t Todo) error {
	data, err := t.Marshal()
	if err != nil {
		return fmt.Errorf("storage: write %s: %w", path, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

// ListTodos returns the sorted paths of all todo files (*.md) directly
// inside dir.
func ListTodos(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var paths []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		paths = append(paths, filepath.Join(dir, e.Name()))
	}

	sort.Strings(paths)
	return paths, nil
}

// DeleteTodo removes the todo file at path.
func DeleteTodo(path string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("storage: delete %s: %w", path, err)
	}
	return nil
}

// GenerateTodoPath returns a path in dir for a new todo t, based on
// TodoFileName(t). If that name is already taken, a "-2", "-3", ... suffix
// is appended before the extension until a free path is found.
func GenerateTodoPath(dir string, t Todo) (string, error) {
	name := TodoFileName(t)
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)

	for i := 1; ; i++ {
		candidate := name
		if i > 1 {
			candidate = fmt.Sprintf("%s-%d%s", base, i, ext)
		}

		path := filepath.Join(dir, candidate)
		_, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			return path, nil
		}
		if err != nil {
			return "", err
		}
	}
}
