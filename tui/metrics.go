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
	System string
	Model  string
}

type tickMetricsMsg time.Time

func fetchSystemMetrics(pid int) (string, string) {
	return buildSystemUsage(), buildModelUsage(pid)
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

func buildSystemUsage() string {
	parts := make([]string, 0, 4)

	if cpuPercent, ok := cpuUsagePercent(200 * time.Millisecond); ok {
		parts = append(parts, fmt.Sprintf("cpu %.0f%%", cpuPercent))
	}

	if usedGiB, totalGiB, usedPercent, ok := readSystemMemory(); ok {
		parts = append(parts, fmt.Sprintf("ram %.1f/%.1fGiB %.0f%%", usedGiB, totalGiB, usedPercent))
	}

	if gpus := readSystemGPUUsage(); gpus != "" {
		parts = append(parts, gpus)
	}

	if len(parts) == 0 {
		return "n/a"
	}
	return strings.Join(parts, "  |  ")
}

func buildModelUsage(pid int) string {
	parts := make([]string, 0, 3)

	if cpuPercent, rssGiB, ok := readProcessCPUAndRAM(pid); ok {
		parts = append(parts, fmt.Sprintf("cpu %s%%", strings.TrimSpace(cpuPercent)))
		parts = append(parts, fmt.Sprintf("ram %.2fGiB", rssGiB))
	}

	if vramGiB, ok := readProcessGPUVRAM(pid); ok {
		parts = append(parts, fmt.Sprintf("vram %.2fGiB", vramGiB))
	}

	if len(parts) == 0 {
		return "n/a"
	}
	return strings.Join(parts, "  |  ")
}

func cpuUsagePercent(interval time.Duration) (float64, bool) {
	t1, i1, ok := readCPUSample()
	if !ok {
		return 0, false
	}
	time.Sleep(interval)
	t2, i2, ok := readCPUSample()
	if !ok || t2 <= t1 || i2 < i1 {
		return 0, false
	}
	total := float64(t2 - t1)
	idle := float64(i2 - i1)
	return (1 - idle/total) * 100, true
}

func readCPUSample() (uint64, uint64, bool) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0, false
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return 0, 0, false
	}
	fields := strings.Fields(lines[0])
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0, 0, false
	}
	var total uint64
	vals := make([]uint64, 0, len(fields)-1)
	for _, f := range fields[1:] {
		v, err := strconv.ParseUint(f, 10, 64)
		if err != nil {
			return 0, 0, false
		}
		vals = append(vals, v)
		total += v
	}
	idle := vals[3]
	if len(vals) > 4 {
		idle += vals[4]
	}
	return total, idle, true
}

func readSystemMemory() (float64, float64, float64, bool) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, 0, false
	}
	var totalKB, availableKB int64
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			fmt.Sscanf(line, "MemTotal: %d kB", &totalKB)
		} else if strings.HasPrefix(line, "MemAvailable:") {
			fmt.Sscanf(line, "MemAvailable: %d kB", &availableKB)
		}
	}
	if totalKB <= 0 {
		return 0, 0, 0, false
	}
	usedKB := totalKB - availableKB
	usedGiB := float64(usedKB) / 1024.0 / 1024.0
	totalGiB := float64(totalKB) / 1024.0 / 1024.0
	usedPercent := float64(usedKB) * 100 / float64(totalKB)
	return usedGiB, totalGiB, usedPercent, true
}

func readSystemGPUUsage() string {
	cmd := exec.Command("nvidia-smi", "--query-gpu=utilization.gpu,memory.used,memory.total", "--format=csv,noheader,nounits")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	parts := make([]string, 0, len(lines))
	for i, line := range lines {
		f := strings.Split(line, ",")
		if len(f) < 3 {
			continue
		}
		util := strings.TrimSpace(f[0])
		usedGiB, okUsed := mibToGiB(f[1])
		totalGiB, okTotal := mibToGiB(f[2])
		if !okUsed || !okTotal {
			continue
		}
		parts = append(parts, fmt.Sprintf("gpu%d %s%% %.1f/%.1fGiB", i, util, usedGiB, totalGiB))
	}
	return strings.Join(parts, "  |  ")
}

func readProcessCPUAndRAM(pid int) (string, float64, bool) {
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "%cpu=,rss=")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", 0, false
	}
	fields := strings.Fields(strings.TrimSpace(out.String()))
	if len(fields) < 2 {
		return "", 0, false
	}
	rssKB, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return "", 0, false
	}
	return fields[0], rssKB / 1024.0 / 1024.0, true
}

func readProcessGPUVRAM(pid int) (float64, bool) {
	cmd := exec.Command("nvidia-smi", "--query-compute-apps=pid,used_memory", "--format=csv,noheader,nounits")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return 0, false
	}
	pidStr := strconv.Itoa(pid)
	var totalMiB float64
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		f := strings.Split(line, ",")
		if len(f) < 2 {
			continue
		}
		if strings.TrimSpace(f[0]) != pidStr {
			continue
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(f[1]), 64)
		if err != nil {
			continue
		}
		totalMiB += v
	}
	if totalMiB <= 0 {
		return 0, false
	}
	return totalMiB / 1024.0, true
}

func tickMetrics() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return tickMetricsMsg(t)
	})
}

func getMetricsCmd(pid int) tea.Cmd {
	return func() tea.Msg {
		systemLine, modelLine := fetchSystemMetrics(pid)
		return systemMetricsMsg{System: systemLine, Model: modelLine}
	}
}
