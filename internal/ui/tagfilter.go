package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// updateTagsPanel handles key input local to the tags panel (focusTags).
func (m Model) updateTagsPanel(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		next, cmd := m.switchFocus(focusList)
		return next, cmd
	}

	return m, nil
}

// tagsPanelContent renders the tags panel's body.
func (m Model) tagsPanelContent() string {
	var b strings.Builder

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
