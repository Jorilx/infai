package tui

import (
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dipankardas011/infai/db"
)

type ExecutorSavedMsg struct{ Bin string }

type ExecutorModel struct {
	database *db.DB
	current  string
	detected string
	editing  bool
	input    textinput.Model
	errMsg   string
	width    int
	height   int
}

func NewExecutorModel(database *db.DB, current string, w, h int) ExecutorModel {
	detected := ""
	if path, err := exec.LookPath("llama-server"); err == nil {
		detected = path
	}
	ti := textinput.New()
	ti.Placeholder = "/path/to/llama-server"
	ti.CharLimit = 512
	ti.SetValue(current)
	return ExecutorModel{
		database: database,
		current:  current,
		detected: detected,
		input:    ti,
		width:    w,
		height:   h,
	}
}

func (m ExecutorModel) SetSize(w, h int) ExecutorModel {
	m.width, m.height = w, h
	return m
}

func (m ExecutorModel) Update(msg tea.Msg) (ExecutorModel, tea.Cmd) {
	if m.editing {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				raw := strings.TrimSpace(m.input.Value())
				m.editing = false
				if raw == "" {
					m.current = ""
					return m, nil
				}
				path, err := expandPath(raw)
				if err != nil {
					m.errMsg = "bad path: " + err.Error()
					return m, nil
				}
				m.current = path
				m.errMsg = ""
				return m, nil
			case "esc":
				m.editing = false
				m.input.SetValue(m.current)
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "e":
			m.editing = true
			m.errMsg = ""
			m.input.SetValue(m.current)
			m.input.Focus()
			return m, textinput.Blink
		case "d":
			if m.detected != "" {
				m.current = m.detected
				m.errMsg = ""
			} else {
				m.errMsg = "llama-server not found in PATH"
			}
		}
	}
	return m, nil
}

func (m ExecutorModel) SaveAndExit() (ExecutorModel, tea.Cmd) {
	_ = m.database.SetSetting("server_bin", m.current)
	return m, func() tea.Msg { return ExecutorSavedMsg{Bin: m.current} }
}

func (m ExecutorModel) View() string {
	t := ActiveTheme
	titleStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Padding(0, 1)
	mutedStyle := lipgloss.NewStyle().Foreground(t.Muted)
	valStyle := lipgloss.NewStyle().Foreground(t.Text).Bold(true)
	helpStyle := lipgloss.NewStyle().Foreground(t.Muted).Italic(true)
	errStyle := lipgloss.NewStyle().Foreground(t.Error)
	detStyle := lipgloss.NewStyle().Foreground(t.Secondary)

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("executor · llama.cpp binary") + "\n\n")

	cur := m.current
	if cur == "" {
		cur = mutedStyle.Render("(not set)")
	} else {
		cur = valStyle.Render(cur)
	}
	sb.WriteString(mutedStyle.Render("  current:  ") + cur + "\n\n")

	if m.detected != "" {
		sb.WriteString(mutedStyle.Render("  detected: ") + detStyle.Render(m.detected) + "\n\n")
	} else {
		sb.WriteString(mutedStyle.Render("  detected: ") + mutedStyle.Render("not found in PATH") + "\n\n")
	}

	if m.editing {
		sb.WriteString(lipgloss.NewStyle().Foreground(t.Secondary).Render("  path: "))
		sb.WriteString(m.input.View() + "\n")
		sb.WriteString(helpStyle.Render("  enter: confirm  esc: cancel") + "\n")
	} else {
		if m.errMsg != "" {
			sb.WriteString(errStyle.Render("  "+m.errMsg) + "\n")
		}
		sb.WriteString(helpStyle.Render("  e: edit path  d: use detected  esc: save & back"))
	}
	return sb.String()
}
