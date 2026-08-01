package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"noto/internal/app"
	"noto/internal/index"
)

// updateList handles key input for modeList.
func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	key := keyMsg.String()
	if m.pendingDelete && key != "d" {
		m.pendingDelete = false
	}

	switch key {
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
	case "n":
		m.mode = modeTitleInput
		m.err = nil
		m.input.Reset()
		return m, m.input.Focus()
	case "enter", "e":
		return m.startEditingExisting()
	case "/":
		m.mode = modeSearch
		m.err = nil
		return m, m.search.Focus()
	case "t":
		tags, err := app.ListTags(m.idx)
		if err != nil {
			m.err = err
			return m, nil
		}
		m.allTags = tags
		m.tagCursor = 0
		m.mode = modeTagFilter
		m.err = nil
		return m, nil
	case "d":
		if m.pendingDelete {
			m.pendingDelete = false
			if len(m.notes) == 0 {
				return m, nil
			}
			m.mode = modeConfirmDelete
			m.err = nil
			return m, nil
		}
		m.pendingDelete = true
		return m, nil
	case "?":
		m.mode = modeHelp
		m.err = nil
		return m, nil
	case "q", "ctrl+c":
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) viewList() string {
	var b strings.Builder
	b.WriteString("noto\n\n")

	if query := m.search.Value(); query != "" {
		fmt.Fprintf(&b, "絞り込み中: %q (/ で編集, Esc で解除)\n\n", query)
	}

	if len(m.selectedTags) > 0 {
		fmt.Fprintf(&b, "タグ絞り込み: %s (t で変更)\n\n", strings.Join(m.selectedTags, ", "))
	}

	if m.pendingDelete {
		b.WriteString("d をもう一度押すと削除\n\n")
	}

	b.WriteString(renderNoteRows(m.notes, m.cursor))

	b.WriteString("\nn: 新規メモ  /: 検索  t: タグ  dd: 削除  ?: ヘルプ  q: 終了\n")
	if m.err != nil {
		b.WriteString("\nerror: " + m.err.Error() + "\n")
	}
	return b.String()
}

// renderNoteRows renders notes as a cursor-marked list, one note per line.
// Shared between the list and search screens.
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
