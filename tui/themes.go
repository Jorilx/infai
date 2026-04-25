package tui

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Name      string
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Success   lipgloss.Color
	Error     lipgloss.Color
	Muted     lipgloss.Color
	Text      lipgloss.Color
	Bg        lipgloss.Color
}

var ThemeList = []Theme{
	{
		Name:      "tokyonight",
		Primary:   "#7DCFFF",
		Secondary: "#BB9AF7",
		Success:   "#9ECE6A",
		Error:     "#F7768E",
		Muted:     "#565F89",
		Text:      "#C0CAF5",
		Bg:        "#1A1B26",
	},
	{
		Name:      "everforest",
		Primary:   "#83C092",
		Secondary: "#D699B6",
		Success:   "#A7C080",
		Error:     "#E67E80",
		Muted:     "#7A8478",
		Text:      "#D3C6AA",
		Bg:        "#2D353B",
	},
	{
		Name:      "onedark",
		Primary:   "#61AFEF",
		Secondary: "#C678DD",
		Success:   "#98C379",
		Error:     "#E06C75",
		Muted:     "#5C6370",
		Text:      "#ABB2BF",
		Bg:        "#282C34",
	},
	{
		Name:      "rosepine",
		Primary:   "#9CCFD8",
		Secondary: "#C4A7E7",
		Success:   "#31748F",
		Error:     "#EB6F92",
		Muted:     "#6E6A86",
		Text:      "#E0DEF4",
		Bg:        "#191724",
	},
	{
		Name:      "gruvbox",
		Primary:   "#83A598",
		Secondary: "#D3869B",
		Success:   "#B8BB26",
		Error:     "#FB4934",
		Muted:     "#928374",
		Text:      "#EBDBB2",
		Bg:        "#282828",
	},
}

var themeMap = func() map[string]Theme {
	m := make(map[string]Theme)
	for _, t := range ThemeList {
		m[t.Name] = t
	}
	return m
}()

var (
	ActiveTheme = ThemeList[0]
	themeIdx    = 0
)

func SetTheme(name string) {
	if t, ok := themeMap[name]; ok {
		ActiveTheme = t
		for i, th := range ThemeList {
			if th.Name == name {
				themeIdx = i
				break
			}
		}
		rebuildStyles()
	}
}

// CycleTheme advances to the next theme and returns its name.
func CycleTheme() string {
	themeIdx = (themeIdx + 1) % len(ThemeList)
	ActiveTheme = ThemeList[themeIdx]
	rebuildStyles()
	return ActiveTheme.Name
}
