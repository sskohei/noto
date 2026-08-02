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
func panelStyle(focused bool, contentWidth int) lipgloss.Style {
	color := blurredBorderColor
	if focused {
		color = focusedBorderColor
	}
	return lipgloss.NewStyle().
		Border(panelBorderVariant).
		BorderForeground(color).
		Padding(0, 1).
		Width(contentWidth)
}

// panelHorizontalFrameSize is the border+padding width panelStyle adds on
// top of its content width, regardless of focus/width (border+padding are
// the same for every panel).
func panelHorizontalFrameSize() int {
	return panelFrameSizeProbe.GetHorizontalFrameSize()
}

// renderPanel wraps content in a titled, bordered box.
func renderPanel(title, content string, contentWidth int, focused bool) string {
	marker := "  "
	titleStyle := lipgloss.NewStyle()
	if focused {
		marker = "▶ "
		titleStyle = titleStyle.Bold(true).Foreground(focusedBorderColor)
	}
	titleLine := titleStyle.Render(marker + title)

	box := panelStyle(focused, contentWidth).Render(strings.TrimRight(content, "\n"))
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
		case "n":
			m.mode = modeTitleInput
			m.err = nil
			m.input.Reset()
			return m, m.input.Focus()
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
		case "q":
			return m, tea.Quit
		}
	}

	switch m.focusedPanel {
	case focusSearch:
		return m.updateSearchPanel(msg)
	case focusTags:
		return m.updateTagsPanel(msg)
	default:
		return m.updateListPanel(msg)
	}
}

// viewMain renders the always-visible 3-panel layout: a search+tags
// sidebar on the left, the notes list as the main panel on the right, and
// a footer below.
func (m Model) viewMain() string {
	frameSize := panelHorizontalFrameSize()
	sidebarTotalWidth := sidebarContentWidth + frameSize

	listContentWidth := m.width - sidebarTotalWidth - frameSize
	if listContentWidth < 1 {
		listContentWidth = 1
	}

	// Recomputed on every render, not cached, so real terminal resizes
	// keep reflowing correctly.
	m.search.Width = sidebarContentWidth

	searchBox := renderPanel("1:検索", m.searchPanelContent(), sidebarContentWidth, m.focusedPanel == focusSearch)
	tagsBox := renderPanel("2:タグ", m.tagsPanelContent(), sidebarContentWidth, m.focusedPanel == focusTags)
	sidebar := lipgloss.JoinVertical(lipgloss.Left, searchBox, tagsBox)

	listBox := renderPanel(m.listPanelTitle(), m.listPanelContent(), listContentWidth, m.focusedPanel == focusList)

	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, listBox)

	footer := "n: 新規メモ  dd: 削除  ?: ヘルプ  q: 終了  |  1/2/3, /, t: パネル移動"
	if m.err != nil {
		footer += "\nerror: " + m.err.Error()
	}

	return body + "\n" + footer
}
