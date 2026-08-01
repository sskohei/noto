package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// updateList handles key input for modeList. The list itself is a
// placeholder: populating it from the search index is out of scope for
// this flow (see the note list screen work).
func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
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
	b.WriteString("n: 新規メモ  q: 終了\n")
	if m.err != nil {
		b.WriteString("\nerror: " + m.err.Error() + "\n")
	}
	return b.String()
}
