package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// updateList handles key input for modeList.
func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	case "n":
		m.mode = modeTitleInput
		m.err = nil
		m.input.Reset()
		return m, m.input.Focus()
	case "q", "ctrl+c":
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) viewList() string {
	var b strings.Builder
	b.WriteString("noto\n\n")

	if len(m.notes) == 0 {
		b.WriteString("(メモがありません。n で新規作成)\n")
	}
	for i, n := range m.notes {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		title := n.Title
		if title == "" {
			title = "(無題)"
		}

		tags := ""
		if len(n.Tags) > 0 {
			tags = " [" + strings.Join(n.Tags, ", ") + "]"
		}

		fmt.Fprintf(&b, "%s%s%s  %s\n", cursor, title, tags, n.UpdatedAt.Format("2006-01-02 15:04"))
	}

	b.WriteString("\nn: 新規メモ  q: 終了\n")
	if m.err != nil {
		b.WriteString("\nerror: " + m.err.Error() + "\n")
	}
	return b.String()
}
