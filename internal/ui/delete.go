package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sskohei/noto/internal/app"
)

// updateConfirmDelete handles key input for modeConfirmDelete.
func (m Model) updateConfirmDelete(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "y":
		return m.confirmDelete()
	case "n", "esc":
		m.mode = modeMain
		return m, nil
	}

	return m, nil
}

// confirmDelete deletes the item under the cursor (a todo if the todo panel
// is focused, otherwise a note) and refreshes the relevant list.
func (m Model) confirmDelete() (tea.Model, tea.Cmd) {
	m.mode = modeMain

	if m.focusedPanel == focusTodos {
		return m.confirmDeleteTodo()
	}
	return m.confirmDeleteNote()
}

func (m Model) confirmDeleteNote() (tea.Model, tea.Cmd) {
	note := m.notes[m.cursor]

	if err := app.DeleteNote(m.idx, m.cfg.NotesDir, note.Path); err != nil {
		m.err = err
		return m, nil
	}

	notes, err := m.refreshNotes()
	if err != nil {
		m.err = err
		return m, nil
	}
	m.notes = notes
	if m.cursor >= len(m.notes) && m.cursor > 0 {
		m.cursor--
	}
	m.err = nil
	return m, nil
}

func (m Model) confirmDeleteTodo() (tea.Model, tea.Cmd) {
	todo := m.todos[m.todoCursor]

	if err := app.DeleteTodo(m.idx, app.TodosDir(m.cfg), todo.Path); err != nil {
		m.err = err
		return m, nil
	}

	todos, err := m.refreshTodos()
	if err != nil {
		m.err = err
		return m, nil
	}
	m.todos = todos
	if m.todoCursor >= len(m.todos) && m.todoCursor > 0 {
		m.todoCursor--
	}
	m.err = nil
	return m, nil
}

// updateConfirmDeleteCompletedTodos handles key input for
// modeConfirmDeleteCompletedTodos.
func (m Model) updateConfirmDeleteCompletedTodos(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "y":
		return m.confirmDeleteCompletedTodos()
	case "n", "esc":
		m.mode = modeMain
		return m, nil
	}

	return m, nil
}

// confirmDeleteCompletedTodos deletes every completed todo and refreshes
// the todo list.
func (m Model) confirmDeleteCompletedTodos() (tea.Model, tea.Cmd) {
	m.mode = modeMain

	if _, err := app.DeleteCompletedTodos(m.idx, app.TodosDir(m.cfg)); err != nil {
		m.err = err
		return m, nil
	}

	todos, err := m.refreshTodos()
	if err != nil {
		m.err = err
		return m, nil
	}
	m.todos = todos
	if m.todoCursor >= len(m.todos) && m.todoCursor > 0 {
		m.todoCursor--
	}
	m.err = nil
	return m, nil
}

func (m Model) viewConfirmDeleteCompletedTodos() string {
	return fmt.Sprintf("完了済みのtodoを%d件削除しますか? (y/n)\n", countCompletedTodos(m.todos))
}

func (m Model) viewConfirmDelete() string {
	title := "(無題)"
	if m.focusedPanel == focusTodos {
		if len(m.todos) > 0 {
			if t := m.todos[m.todoCursor].Title; t != "" {
				title = t
			}
		}
	} else if len(m.notes) > 0 {
		if t := m.notes[m.cursor].Title; t != "" {
			title = t
		}
	}
	return fmt.Sprintf("%q を削除しますか? (y/n)\n", title)
}
