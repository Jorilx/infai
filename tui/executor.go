package tui

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dipankardas011/infai/db"
)

type ExecutorSavedMsg struct{ Bin string }

var supportedExecutors = []string{"llamacpp"}

type ExecutorModel struct {
	database     *db.DB
	executors    []db.Executor
	cursor       int
	detected     string
	addingBrowse bool
	fileBrowser  FileBrowserModel
	typeIdx      int
	errMsg       string
	width        int
	height       int
}

func NewExecutorModel(database *db.DB, current string, w, h int) ExecutorModel {
	executors, _ := database.ListExecutors()

	detected := ""
	if path, err := exec.LookPath("llama-server"); err == nil {
		detected = path
	}

	curIdx := 0
	for i, e := range executors {
		if e.IsDefault {
			curIdx = i
			break
		}
	}

	return ExecutorModel{
		database:  database,
		executors: executors,
		cursor:    curIdx,
		detected:  detected,
		width:     w,
		height:    h,
	}
}

func (m ExecutorModel) SetSize(w, h int) ExecutorModel {
	m.width, m.height = w, h
	m.fileBrowser = m.fileBrowser.SetSize(w, h)
	return m
}

func (m ExecutorModel) AddingBrowse() bool { return m.addingBrowse }

func (m ExecutorModel) Update(msg tea.Msg) (ExecutorModel, tea.Cmd) {
	if m.addingBrowse {
		var cmd tea.Cmd
		m.fileBrowser, cmd = m.fileBrowser.Update(msg)
		if _, ok := msg.(tea.KeyMsg); ok {
			switch msg.(type) {
			case FileBrowserSavedMsg:
			default:
				return m, cmd
			}
		}
		if fm, ok := msg.(FileBrowserSavedMsg); ok {
			m.addingBrowse = false
			if fm.Path == "" {
				return m, nil
			}
			path := fm.Path

			id := supportedExecutors[m.typeIdx]
			absPath, err := expandPath(path)
			if err != nil {
				m.errMsg = "bad path: " + err.Error()
				return m, nil
			}

			isDefault := len(m.executors) == 0

			err = m.database.UpsertExecutor(db.Executor{
				ID:        id,
				Path:      absPath,
				IsDefault: isDefault,
			})
			if err != nil {
				m.errMsg = err.Error()
				return m, nil
			}

			m.executors, _ = m.database.ListExecutors()
			m.errMsg = styleSuccess.Render("✓ added " + id)
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.executors)-1 {
				m.cursor++
			}
		case "a":
			m.addingBrowse = true
			m.errMsg = ""
			m.fileBrowser = NewFileBrowserModel().SetSize(m.width, m.height).SetSelectFile(true)
			return m, nil
		case "enter":
			if len(m.executors) > 0 {
				id := m.executors[m.cursor].ID
				_ = m.database.SetDefaultExecutor(id)
				m.executors, _ = m.database.ListExecutors()
			}
		case "d":
			if m.detected != "" {
				// For 'd', we always use llamacpp and make it default
				_ = m.database.UpsertExecutor(db.Executor{
					ID:        "llamacpp",
					Path:      m.detected,
					IsDefault: true,
				})
				m.executors, _ = m.database.ListExecutors()
				m.errMsg = ""
			} else {
				m.errMsg = "llama-server not found in PATH"
			}
		}
	}
	return m, nil
}

func (m ExecutorModel) SaveAndExit() (ExecutorModel, tea.Cmd) {
	path, _ := m.database.GetDefaultExecutorPath()
	return m, func() tea.Msg { return ExecutorSavedMsg{Bin: path} }
}

func (m ExecutorModel) View() string {
	t := ActiveTheme
	titleStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Padding(0, 1)
	mutedStyle := lipgloss.NewStyle().Foreground(t.Muted)
	selStyle := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	errStyle := lipgloss.NewStyle().Foreground(t.Error)
	defStyle := lipgloss.NewStyle().Foreground(t.Success).Bold(true)

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("executors") + "\n\n")

	if len(m.executors) == 0 {
		sb.WriteString("  " + mutedStyle.Render("nothing") + "\n")
	} else {
		for i, e := range m.executors {
			prefix := "  "
			style := lipgloss.NewStyle()
			if i == m.cursor {
				prefix = selStyle.Render("▶ ")
				style = selStyle
			}

			def := ""
			if e.IsDefault {
				def = defStyle.Render(" (default)")
			}

			sb.WriteString(fmt.Sprintf("%s%s: %s%s\n", prefix, style.Render(e.ID), e.Path, def))
		}
	}

	sb.WriteString("\n")
	if m.addingBrowse {
		return m.fileBrowser.View()
	} else {
		if m.errMsg != "" {
			if strings.HasPrefix(m.errMsg, "\x1b[") || strings.HasPrefix(m.errMsg, "✓") {
				sb.WriteString(m.errMsg + "\n")
			} else {
				sb.WriteString(errStyle.Render("  "+m.errMsg) + "\n")
			}
		}
	}

	content := sb.String()
	boxWidth := 60
	if m.width < 60 {
		boxWidth = m.width - 4
	}
	if boxWidth < 0 {
		boxWidth = 0
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Muted).
		Padding(1, 2).
		Width(boxWidth)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, boxStyle.Render(content))
}
