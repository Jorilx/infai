package tui

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const stopGraceTimeout = 5 * time.Second

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripAnsi(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

// Tea messages for server I/O.
type logLineMsg string
type serverExitMsg struct{ err error }
type stopTimeoutMsg struct{}

func listenForLog(ch <-chan string, exitCh <-chan error) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			err := <-exitCh
			return serverExitMsg{err: err}
		}
		return logLineMsg(line)
	}
}

const maxLogLines = 10000

// ServerModel is screen 5 — shows live llama-server output.
type ServerModel struct {
	cmd         *exec.Cmd
	logCh       chan string
	exitCh      chan error
	logs        []string
	vp          viewport.Model
	profileName string
	modelName   string
	port        int
	stopped     bool
	stopping    bool
	forceKilled bool
	exitErr     error
	width       int
	height      int
	initialized bool
}

// NewServerModel starts the server process and returns the model + initial listen cmd.
func NewServerModel(args []string, profileName, modelName string, port, w, h int) (ServerModel, tea.Cmd, error) {
	cmd := exec.Command(args[0], args[1:]...)

	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return ServerModel{}, nil, err
	}

	logCh := make(chan string, 256)
	exitCh := make(chan error, 1)

	// goroutine: read lines → channel
	go func() {
		sc := bufio.NewScanner(pr)
		for sc.Scan() {
			logCh <- stripAnsi(sc.Text())
		}
		close(logCh)
	}()

	// goroutine: wait for exit → close pipe, capture err
	go func() {
		err := cmd.Wait()
		pw.Close()
		exitCh <- err
	}()

	vpH := max(h-6, 5)
	vp := viewport.New(w-4, vpH)
	vp.Style = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary)

	m := ServerModel{
		cmd:         cmd,
		logCh:       logCh,
		exitCh:      exitCh,
		vp:          vp,
		profileName: profileName,
		modelName:   modelName,
		port:        port,
		width:       w,
		height:      h,
		initialized: true,
	}
	return m, listenForLog(logCh, exitCh), nil
}

func (s ServerModel) HandleLogLine(line string) (ServerModel, tea.Cmd) {
	s.logs = append(s.logs, line)
	if len(s.logs) > maxLogLines {
		s.logs = s.logs[len(s.logs)-maxLogLines:]
	}
	atBottom := s.vp.AtBottom()
	s.vp.SetContent(strings.Join(s.logs, "\n"))
	if atBottom {
		s.vp.GotoBottom()
	}
	return s, listenForLog(s.logCh, s.exitCh)
}

func (s ServerModel) SetExited(err error) ServerModel {
	s.stopped = true
	s.stopping = false
	s.exitErr = err
	return s
}

func (s ServerModel) Stop() (ServerModel, tea.Cmd) {
	if s.cmd == nil || s.cmd.Process == nil || s.stopped || s.stopping {
		return s, nil
	}
	s.stopping = true
	syscall.Kill(-s.cmd.Process.Pid, syscall.SIGTERM)
	cmd := tea.Tick(stopGraceTimeout, func(time.Time) tea.Msg { return stopTimeoutMsg{} })
	return s, cmd
}

func (s ServerModel) ForceKill() ServerModel {
	if s.cmd != nil && s.cmd.Process != nil && !s.stopped {
		s.forceKilled = true
		syscall.Kill(-s.cmd.Process.Pid, syscall.SIGKILL)
	}
	return s
}

func (s ServerModel) SetSize(w, h int) ServerModel {
	if !s.initialized {
		return s
	}
	s.width = w
	s.height = h
	vpH := max(h-6, 5)
	s.vp.Width = w - 4
	s.vp.Height = vpH
	return s
}

func (s ServerModel) Update(msg tea.Msg) (ServerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "c":
			s.logs = nil
			s.vp.SetContent("")
			return s, nil
		}
	}
	var cmd tea.Cmd
	s.vp, cmd = s.vp.Update(msg)
	return s, cmd
}

func (s ServerModel) View() string {
	t := ActiveTheme

	// Header
	status := lipgloss.NewStyle().Foreground(t.Success).Bold(true).Render("● running")
	if s.stopping {
		status = lipgloss.NewStyle().Foreground(t.Error).Bold(true).Render("◌ shutting down (SIGTERM)…")
	} else if s.stopped {
		label := "■ stopped"
		if s.forceKilled {
			label = "■ force-killed (SIGKILL)"
		}
		status = lipgloss.NewStyle().Foreground(t.Muted).Render(label)
	}
	pid := ""
	if s.cmd != nil && s.cmd.Process != nil {
		pid = styleMuted.Render(fmt.Sprintf("  pid:%d", s.cmd.Process.Pid))
	}
	portStr := styleKey.Render(fmt.Sprintf("  port:%d", s.port))
	header := styleTitle.Render(s.profileName) + "  " + status + pid + portStr +
		"\n" + styleMuted.Render("  model: "+s.modelName)

	// Log viewport
	logView := s.vp.View()

	// Footer
	footer := ""
	if s.stopped {
		exitStatus := styleSuccess.Render("exited cleanly")
		if s.exitErr != nil {
			exitStatus = styleError.Render("error: " + s.exitErr.Error())
		}
		footer = "\n" + exitStatus
	}

	return header + "\n\n" + logView + footer
}
