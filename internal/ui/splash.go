package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// splashDuration is how long the splash screen stays up before
// auto-dismissing if the user hasn't already pressed a key.
const splashDuration = 1500 * time.Millisecond

// splashTimeoutMsg fires splashDuration after startup.
type splashTimeoutMsg struct{}

const splashArt = `███╗   ██╗ ██████╗ ████████╗ ██████╗ 
████╗  ██║██╔═══██╗╚══██╔══╝██╔═══██╗
██╔██╗ ██║██║   ██║   ██║   ██║   ██║
██║╚██╗██║██║   ██║   ██║   ██║   ██║
██║ ╚████║╚██████╔╝   ██║   ╚██████╔╝
╚═╝  ╚═══╝ ╚═════╝    ╚═╝    ╚═════╝ `

const splashTagline = "書いて、すぐ見つける"
const splashHint = "何かキーを押すと始まります..."

// updateSplash handles input for modeSplash: any keypress dismisses it.
func (m Model) updateSplash(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.KeyMsg); ok {
		m.mode = modeMain
	}
	return m, nil
}

// handleSplashTimeout dismisses the splash after splashDuration, unless the
// user already dismissed it (in which case this is a stale, harmless tick).
func (m Model) handleSplashTimeout(msg splashTimeoutMsg) (tea.Model, tea.Cmd) {
	if m.mode != modeSplash {
		return m, nil
	}
	m.mode = modeMain
	return m, nil
}

func (m Model) viewSplash() string {
	body := splashArt + "\n\n" + splashTagline + "\n" + splashHint

	artWidth, artHeight := lipgloss.Width(splashArt), lipgloss.Height(body)
	if m.width < artWidth || m.height < artHeight+2 {
		// Too small to safely center/wrap; print the art unwrapped rather
		// than risk lipgloss corrupting it mid-glyph.
		return body
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(body)
}
