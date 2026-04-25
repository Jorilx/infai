package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Color vars — updated by rebuildStyles on theme change.
	colorPrimary   lipgloss.Color
	colorSecondary lipgloss.Color
	colorSuccess   lipgloss.Color
	colorError     lipgloss.Color
	colorMuted     lipgloss.Color
	colorText      lipgloss.Color

	styleTitle    lipgloss.Style
	styleMuted    lipgloss.Style
	styleSuccess  lipgloss.Style
	styleError    lipgloss.Style
	styleBadge    lipgloss.Style
	styleBox      lipgloss.Style
	styleSelected lipgloss.Style
	styleLabel    lipgloss.Style
	styleHelp     lipgloss.Style
	styleKey      lipgloss.Style
)

func init() {
	rebuildStyles()
}

func rebuildStyles() {
	t := ActiveTheme
	colorPrimary = t.Primary
	colorSecondary = t.Secondary
	colorSuccess = t.Success
	colorError = t.Error
	colorMuted = t.Muted
	colorText = t.Text
	styleTitle = lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Padding(0, 1)
	styleMuted = lipgloss.NewStyle().Foreground(t.Muted)
	styleSuccess = lipgloss.NewStyle().Foreground(t.Success)
	styleError = lipgloss.NewStyle().Foreground(t.Error)
	styleBadge = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Bg)).
		Background(t.Secondary).
		Padding(0, 1)
	styleBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Padding(1, 2)
	styleSelected = lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	styleLabel = lipgloss.NewStyle().Foreground(t.Muted).Width(20).Align(lipgloss.Right)
	styleHelp = lipgloss.NewStyle().Foreground(t.Muted).Italic(true)
	styleKey = lipgloss.NewStyle().Foreground(t.Secondary).Bold(true)
}
