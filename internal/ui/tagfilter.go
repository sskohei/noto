package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// updateTagFilter handles key input for modeTagFilter.
func (m Model) updateTagFilter(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "j", "down":
		if m.tagCursor < len(m.allTags)-1 {
			m.tagCursor++
		}
		return m, nil
	case "k", "up":
		if m.tagCursor > 0 {
			m.tagCursor--
		}
		return m, nil
	case "enter", " ":
		if len(m.allTags) == 0 {
			return m, nil
		}
		m.selectedTags = toggleTag(m.selectedTags, m.allTags[m.tagCursor])

		notes, err := m.refreshNotes()
		if err != nil {
			m.err = err
			return m, nil
		}
		m.notes = notes
		m.cursor = 0
		m.err = nil
		return m, nil
	case "esc":
		m.mode = modeList
		return m, nil
	}

	return m, nil
}

func (m Model) viewTagFilter() string {
	var b strings.Builder
	b.WriteString("タグで絞り込み\n\n")

	if len(m.allTags) == 0 {
		b.WriteString("(タグがありません)\n")
	}

	for i, tag := range m.allTags {
		mark := "  "
		if i == m.tagCursor {
			mark = "> "
		}

		checkbox := "[ ]"
		if hasTag(m.selectedTags, tag) {
			checkbox = "[x]"
		}

		fmt.Fprintf(&b, "%s%s %s\n", mark, checkbox, tag)
	}

	b.WriteString("\nj/k: 移動  Enter/Space: 選択切替  Esc: 一覧に戻る\n")
	return b.String()
}

// toggleTag returns tags with tag added if absent, or removed if present.
func toggleTag(tags []string, tag string) []string {
	for i, t := range tags {
		if t == tag {
			return append(tags[:i], tags[i+1:]...)
		}
	}
	return append(tags, tag)
}

func hasTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}
