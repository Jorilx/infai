package tui

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type systemMetricsMsg struct {
	Metrics string
}

type tickMetricsMsg time.Time

func fetchSystemMetrics() string {
	var metrics []string

	if loadData, err := os.ReadFile("/proc/loadavg"); err == nil {
		parts := strings.Fields(string(loadData))
		if len(parts) >= 3 {
			metrics = append(metrics, fmt.Sprintf("cpu %s/%s/%s", parts[0], parts[1], parts[2]))
		}
	}

	if memData, err := os.ReadFile("/proc/meminfo"); err == nil {
		lines := strings.Split(string(memData), "\n")
		var memTotalKB, memAvailableKB int64
		for _, line := range lines {
			if strings.HasPrefix(line, "MemTotal:") {
				fmt.Sscanf(line, "MemTotal: %d kB", &memTotalKB)
			} else if strings.HasPrefix(line, "MemAvailable:") {
				fmt.Sscanf(line, "MemAvailable: %d kB", &memAvailableKB)
			}
		}
		if memTotalKB > 0 {
			usedKB := memTotalKB - memAvailableKB
			usedGiB := float64(usedKB) / 1024.0 / 1024.0
			totalGiB := float64(memTotalKB) / 1024.0 / 1024.0
			metrics = append(metrics, fmt.Sprintf("ram %.1f/%.1fGiB", usedGiB, totalGiB))
		}
	}

	cmd := exec.Command("nvidia-smi", "--query-gpu=utilization.gpu,memory.used,memory.total", "--format=csv,noheader")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err == nil {
		lines := strings.Split(strings.TrimSpace(out.String()), "\n")
		for i, line := range lines {
			parts := strings.Split(line, ",")
			if len(parts) >= 3 {
				gpuUtil := normalizePercent(parts[0])
				memUsedGiB, okUsed := mibToGiB(parts[1])
				memTotalGiB, okTotal := mibToGiB(parts[2])
				if okUsed && okTotal {
					metrics = append(metrics, fmt.Sprintf("gpu%d %s %.1f/%.1fGiB", i, gpuUtil, memUsedGiB, memTotalGiB))
				} else {
					metrics = append(metrics, fmt.Sprintf("gpu%d %s %s/%s", i, gpuUtil, strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2])))
				}
			}
		}
	}

	if len(metrics) == 0 {
		return "Metrics unavailable"
	}
	return strings.Join(metrics, "  |  ")
}

func normalizePercent(s string) string {
	t := strings.TrimSpace(s)
	t = strings.TrimSuffix(t, "%")
	t = strings.TrimSpace(t)
	if t == "" {
		return "0%"
	}
	return t + "%"
}

func mibToGiB(s string) (float64, bool) {
	t := strings.TrimSpace(s)
	fields := strings.Fields(t)
	if len(fields) == 0 {
		return 0, false
	}
	v, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, false
	}
	return v / 1024.0, true
}

func tickMetrics() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return tickMetricsMsg(t)
	})
}

func getMetricsCmd() tea.Cmd {
	return func() tea.Msg {
		return systemMetricsMsg{Metrics: fetchSystemMetrics()}
	}
}
