package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sskohei/noto/internal/index"
)

// updateListPanel handles key input local to the notes list panel
// (focusList). Global keys (n, dd, ?, q, 1/2/3, /, t) are handled by
// updateMain before dispatching here.
func (m Model) updateListPanel(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "j", "down":
		if m.cursor < len(m.notes)-1 {
			m.cursor++
		}
		return m, nil
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case "enter", "e":
		return m.startEditingExisting()
	}

	return m, nil
}

// listPanelTitle returns the notes list panel's title, folding in a hint
// about any active search/tag filter so that context isn't lost now that
// the filter panels no longer take over the whole screen.
func (m Model) listPanelTitle() string {
	title := "3:メモ一覧"

	var filters []string
	if query := m.search.Value(); query != "" {
		filters = append(filters, fmt.Sprintf("検索: %q", query))
	}
	if len(m.selectedTags) > 0 {
		filters = append(filters, "タグ: "+strings.Join(m.selectedTags, ", "))
	}
	if len(filters) > 0 {
		title += fmt.Sprintf(" (%s)", strings.Join(filters, " / "))
	}

	return title
}

// listPanelContent renders the notes list panel's body.
func (m Model) listPanelContent() string {
	var b strings.Builder

	if m.pendingDelete {
		b.WriteString("d をもう一度押すと削除\n\n")
	}

	b.WriteString(renderNoteRows(m.notes, m.cursor))

	return b.String()
}

// renderNoteRows renders notes as a cursor-marked list, one note per line.
func renderNoteRows(notes []index.NoteSummary, cursor int) string {
	if len(notes) == 0 {
		return "(メモがありません。n で新規作成)\n"
	}

	var b strings.Builder
	for i, n := range notes {
		mark := "  "
		if i == cursor {
			mark = "> "
		}

		title := n.Title
		if title == "" {
			title = "(無題)"
		}

		tags := ""
		if len(n.Tags) > 0 {
			tags = " [" + strings.Join(n.Tags, ", ") + "]"
		}

		fmt.Fprintf(&b, "%s%s%s  %s\n", mark, title, tags, n.UpdatedAt.Format("2006-01-02 15:04"))
	}
	return b.String()
}
