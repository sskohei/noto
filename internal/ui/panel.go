package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sskohei/noto/internal/app"
)

// sidebarContentWidth is the fixed content width (inside border+padding)
// shared by the search and tags panels, so their boxes line up.
const sidebarContentWidth = 20

var (
	focusedBorderColor  = lipgloss.Color("62")
	blurredBorderColor  = lipgloss.Color("240")
	panelBorderVariant  = lipgloss.RoundedBorder()
	panelFrameSizeProbe = lipgloss.NewStyle().Border(panelBorderVariant).Padding(0, 1)
)

// panelStyle returns the bordered box style for a panel, content-width
// sized. focused panels get a distinct border color; a non-color marker is
// also used in the title (see renderPanel) so focus stays legible even
// when colors don't render (e.g. under go test, or on dumb terminals).
// contentHeight <= 0 leaves the box's height to size naturally from its
// content (used by the search/tags sidebar panels); contentHeight > 0
// pads/aligns the content to that many lines (used by the notes-list/todo
// panels so the focused one fills the remaining terminal height).
func panelStyle(focused bool, contentWidth, contentHeight int) lipgloss.Style {
	color := blurredBorderColor
	if focused {
		color = focusedBorderColor
	}
	style := lipgloss.NewStyle().
		Border(panelBorderVariant).
		BorderForeground(color).
		Padding(0, 1).
		Width(contentWidth)
	if contentHeight > 0 {
		style = style.Height(contentHeight)
	}
	return style
}

// panelHorizontalFrameSize is the border+padding width panelStyle adds on
// top of its content width, regardless of focus/width (border+padding are
// the same for every panel).
func panelHorizontalFrameSize() int {
	return panelFrameSizeProbe.GetHorizontalFrameSize()
}

// panelVerticalFrameSize is the border+padding height panelStyle adds on
// top of its content height (border top+bottom; padding is horizontal
// only, so this is currently just the border's vertical size).
func panelVerticalFrameSize() int {
	return panelFrameSizeProbe.GetVerticalFrameSize()
}

// renderPanel wraps content in a titled, bordered box. See panelStyle for
// contentHeight's meaning.
func renderPanel(title, content string, contentWidth, contentHeight int, focused bool) string {
	marker := "  "
	titleStyle := lipgloss.NewStyle()
	if focused {
		marker = "▶ "
		titleStyle = titleStyle.Bold(true).Foreground(focusedBorderColor)
	}
	titleLine := titleStyle.Render(marker + title)

	box := panelStyle(focused, contentWidth, contentHeight).Render(strings.TrimRight(content, "\n"))
	return lipgloss.JoinVertical(lipgloss.Left, titleLine, box)
}

// switchFocus moves keyboard focus to panel p, syncing the search
// textinput's focus state and, when moving onto the tags panel from
// somewhere else, refreshing the tag list (mirroring the historical `t`
// keybinding's behavior).
func (m Model) switchFocus(p focusPanel) (Model, tea.Cmd) {
	prev := m.focusedPanel
	m.focusedPanel = p
	m.err = nil

	if p == focusSearch {
		return m, m.search.Focus()
	}
	m.search.Blur()

	if p == focusTags && prev != focusTags {
		tags, err := app.ListTags(m.idx)
		if err != nil {
			m.err = err
			return m, nil
		}
		m.allTags = tags
		if m.tagCursor >= len(m.allTags) {
			m.tagCursor = 0
		}
	}

	return m, nil
}

// updateMain handles input for modeMain. Global keys (panel-jump digits,
// /, t, n, the dd delete chord, ?, q) are handled here first, unless the
// search panel is focused, in which case every key must reach its
// textinput untouched. Everything else is dispatched to the focused
// panel's own handler.
func (m Model) updateMain(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && m.focusedPanel != focusSearch {
		key := keyMsg.String()
		if m.pendingDelete && key != "d" {
			m.pendingDelete = false
		}

		switch key {
		case "1", "/":
			next, cmd := m.switchFocus(focusSearch)
			return next, cmd
		case "2", "t":
			next, cmd := m.switchFocus(focusTags)
			return next, cmd
		case "3":
			next, cmd := m.switchFocus(focusList)
			return next, cmd
		case "4":
			next, cmd := m.switchFocus(focusTodos)
			return next, cmd
		case "n":
			m.mode = modeTitleInput
			m.err = nil
			m.input.Reset()
			return m, m.input.Focus()
		case "d":
			if m.pendingDelete {
				m.pendingDelete = false
				if m.focusedPanel == focusTodos {
					if len(m.todos) == 0 {
						return m, nil
					}
				} else if len(m.notes) == 0 {
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
		case "q":
			return m, tea.Quit
		}
	}

	switch m.focusedPanel {
	case focusSearch:
		return m.updateSearchPanel(msg)
	case focusTags:
		return m.updateTagsPanel(msg)
	case focusTodos:
		return m.updateTodoPanel(msg)
	default:
		return m.updateListPanel(msg)
	}
}

// rightColumnContents returns the notes-list and todo-list panel bodies.
// Whichever of the two doesn't have focus collapses (title + border only,
// plus a one-line todo count summary in the todo panel's case) so the
// focused one gets the space; the notes list wins by default when neither
// has focus (search/tags focused).
func (m Model) rightColumnContents() (noteContent, todoContent string) {
	if m.focusedPanel == focusTodos {
		return "", m.todoPanelContent()
	}
	return m.listPanelContent(), m.todoSummaryLine()
}

// viewMain renders the always-visible 4-panel layout: a search+tags
// sidebar on the left, the notes list/todo list column on the right, and
// a footer below. The notes-list and todo panels split the terminal's
// remaining height (after the footer) between them, giving nearly all of
// it to whichever has focus, so that panel fills the screen regardless of
// how little content it has and the footer stays pinned to the bottom.
func (m Model) viewMain() string {
	frameSizeH := panelHorizontalFrameSize()
	frameSizeV := panelVerticalFrameSize()
	sidebarTotalWidth := sidebarContentWidth + frameSizeH

	listContentWidth := m.width - sidebarTotalWidth - frameSizeH
	if listContentWidth < 1 {
		listContentWidth = 1
	}

	// Recomputed on every render, not cached, so real terminal resizes
	// keep reflowing correctly.
	m.search.Width = sidebarContentWidth

	searchBox := renderPanel("1:検索", m.searchPanelContent(), sidebarContentWidth, 0, m.focusedPanel == focusSearch)
	tagsBox := renderPanel("2:タグ", m.tagsPanelContent(), sidebarContentWidth, 0, m.focusedPanel == focusTags)
	sidebar := lipgloss.JoinVertical(lipgloss.Left, searchBox, tagsBox)

	footer := m.footerText()
	if m.err != nil {
		footer += "\nerror: " + m.err.Error()
	}
	footerLines := strings.Count(footer, "\n") + 1

	bodyHeight := m.height - footerLines
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	const collapsedContentHeight = 1
	collapsedTotalHeight := 1 /* title */ + frameSizeV + collapsedContentHeight

	expandedContentHeight := bodyHeight - collapsedTotalHeight - 1 /* title */ - frameSizeV
	if expandedContentHeight < 1 {
		expandedContentHeight = 1
	}

	noteContent, todoContent := m.rightColumnContents()
	noteHeight, todoHeight := collapsedContentHeight, expandedContentHeight
	if m.focusedPanel != focusTodos {
		noteHeight, todoHeight = expandedContentHeight, collapsedContentHeight
	}

	noteBox := renderPanel(m.listPanelTitle(), noteContent, listContentWidth, noteHeight, m.focusedPanel == focusList)
	todoBox := renderPanel(m.todoPanelTitle(), todoContent, listContentWidth, todoHeight, m.focusedPanel == focusTodos)
	rightColumn := lipgloss.JoinVertical(lipgloss.Left, noteBox, todoBox)

	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, rightColumn)

	return body + "\n" + footer
}

// footerText returns the footer hint line for the currently focused panel.
// The search panel shows only its own keys: per updateMain, it consumes
// every key except Enter/Esc while focused, so the global shortcuts below
// don't actually work there.
func (m Model) footerText() string {
	const globalKeys = "?: ヘルプ  q: 終了  "

	switch m.focusedPanel {
	case focusSearch:
		return "文字入力: 絞り込み  Enter: 一覧へ戻る  Esc: クリアして一覧へ戻る"
	case focusTags:
		return "Enter/Space: 選択  Esc: 一覧へ戻る  n: 新規メモ  dd: 削除  " + globalKeys
	case focusTodos:
		return "Enter/e: 編集  Space: 完了切替  n: 新規todo  dd: 削除  D: 完了済みタスク一斉削除  " + globalKeys
	default: // focusList
		return "Enter/e: 編集  n: 新規メモ  dd: 削除  " + globalKeys
	}
}
