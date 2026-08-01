package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"noto/internal/app"
)

// editorFinishedMsg is sent once the $EDITOR process launched via
// tea.ExecProcess exits.
type editorFinishedMsg struct {
	path string
	err  error
}

func (m Model) updateTitleInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc":
			m.mode = modeMain
			m.input.Blur()
			return m, nil
		case "enter":
			return m.startEditing()
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// startEditing writes the new note's template file to disk and hands off
// to $EDITOR via tea.ExecProcess, the only place this may be called from
// (see docs/architecture.md and the plan for issue #5).
func (m Model) startEditing() (tea.Model, tea.Cmd) {
	title := m.input.Value()

	_, path, err := app.PrepareNewNote(m.cfg, title)
	if err != nil {
		m.err = err
		m.mode = modeMain
		return m, nil
	}

	m.mode = modeEditing
	m.err = nil

	cmd := app.EditorCommand(m.cfg, path)
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editorFinishedMsg{path: path, err: err}
	})
}

func (m Model) handleEditorFinished(msg editorFinishedMsg) (tea.Model, tea.Cmd) {
	m.mode = modeMain

	if msg.err != nil {
		m.err = msg.err
		return m, nil
	}

	if _, err := app.FinalizeNote(m.idx, m.cfg.NotesDir, msg.path); err != nil {
		m.err = err
		return m, nil
	}

	notes, err := m.refreshNotes()
	if err != nil {
		m.err = err
		return m, nil
	}
	m.notes = notes
	m.cursor = 0
	m.err = nil
	return m, nil
}

func (m Model) viewTitleInput() string {
	return "新規メモのタイトル:\n\n" + m.input.View() + "\n\n(Enter で確定 / Esc でキャンセル)\n"
}
