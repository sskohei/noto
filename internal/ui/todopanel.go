package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sskohei/noto/internal/app"
	"github.com/sskohei/noto/internal/index"
)

// updateTodoPanel handles key input local to the todo list panel
// (focusTodos). Global keys (n, dd, ?, q, 1/2/3/4, /, t) are handled by
// updateMain before dispatching here.
func (m Model) updateTodoPanel(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "j", "down":
		if m.todoCursor < len(m.todos)-1 {
			m.todoCursor++
		}
		return m, nil
	case "k", "up":
		if m.todoCursor > 0 {
			m.todoCursor--
		}
		return m, nil
	case "enter", "e":
		return m.startEditingExisting()
	case " ":
		return m.toggleSelectedTodo()
	case "D":
		return m.startDeleteCompletedTodos()
	}

	return m, nil
}

// startDeleteCompletedTodos opens the bulk-delete confirmation prompt for
// completed todos, or is a no-op if none are completed.
func (m Model) startDeleteCompletedTodos() (tea.Model, tea.Cmd) {
	if countCompletedTodos(m.todos) == 0 {
		return m, nil
	}
	m.mode = modeConfirmDeleteCompletedTodos
	m.err = nil
	return m, nil
}

// countCompletedTodos returns how many of todos are marked done.
func countCompletedTodos(todos []index.TodoSummary) int {
	n := 0
	for _, t := range todos {
		if t.Done {
			n++
		}
	}
	return n
}

// toggleSelectedTodo flips the done state of the todo under the cursor and
// refreshes the list. The cursor position (not the toggled item) is kept,
// so repeatedly pressing Space checks off items top-to-bottom.
func (m Model) toggleSelectedTodo() (tea.Model, tea.Cmd) {
	if len(m.todos) == 0 {
		return m, nil
	}

	path := m.todos[m.todoCursor].Path
	if _, err := app.ToggleTodoDone(m.idx, app.TodosDir(m.cfg), path); err != nil {
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

// todoPanelTitle returns the todo list panel's title.
func (m Model) todoPanelTitle() string {
	return "4:todoリスト"
}

// todoPanelContent renders the todo list panel's body.
func (m Model) todoPanelContent() string {
	var b strings.Builder

	if m.pendingDelete && m.focusedPanel == focusTodos {
		b.WriteString("d をもう一度押すと削除\n\n")
	}

	b.WriteString(renderTodoRows(m.todos, m.todoCursor))

	return b.String()
}

// todoSummaryLine returns a one-line "未完了/完了" count summary, shown in
// place of the row list when the todo panel doesn't have focus (collapsed).
func (m Model) todoSummaryLine() string {
	completed := countCompletedTodos(m.todos)
	pending := len(m.todos) - completed
	return fmt.Sprintf("未完了: %d件 / 完了: %d件", pending, completed)
}

// renderTodoRows renders todos as a cursor-marked list, one todo per line:
// a done/not-done checkbox followed by the title.
func renderTodoRows(todos []index.TodoSummary, cursor int) string {
	if len(todos) == 0 {
		return "(todoがありません。n で新規作成)\n"
	}

	var b strings.Builder
	for i, t := range todos {
		mark := "  "
		if i == cursor {
			mark = "> "
		}

		checkbox := "[ ]"
		if t.Done {
			checkbox = "[x]"
		}

		title := t.Title
		if title == "" {
			title = "(無題)"
		}

		b.WriteString(mark + checkbox + " " + title + "\n")
	}
	return b.String()
}

// indexOfTodo returns the index of the todo with the given id in todos, or
// 0 if not found (e.g. it was deleted concurrently).
func indexOfTodo(todos []index.TodoSummary, id string) int {
	for i, t := range todos {
		if t.ID == id {
			return i
		}
	}
	return 0
}
