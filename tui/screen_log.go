package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"vmctl/vm"
)

type LogScreen struct{}

func (s LogScreen) Update(m RootModel, msg tea.Msg) (RootModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc", "left", "h", "l":
			m.screen = ListScreen{}
			return m, nil
		}
	}
	return m, nil
}

func (s LogScreen) View(m RootModel) string {
	var sb strings.Builder
	sb.WriteString(styleTitle.Render(m.logTitle))
	sb.WriteString("\n\n")
	if m.logErr != "" {
		sb.WriteString(styleError.Render("✗ " + m.logErr))
		sb.WriteString("\n\n")
	} else {
		sb.WriteString(styleNormal.Render(m.logContent))
		sb.WriteString("\n\n")
	}
	sb.WriteString(styleStatusBar.Render("esc back  •  q quit"))
	return renderScreen(m.width, m.height, sb.String())
}

func openLog(m RootModel, cfg vm.VMConfig) RootModel {
	m.logTitle = fmt.Sprintf("Log → %s", cfg.Name)
	m.logContent = ""
	m.logErr = ""

	path := vm.LogPath(cfg.ID)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		m.logErr = fmt.Sprintf("Log not found: %s", path)
	} else if err != nil {
		m.logErr = err.Error()
	} else {
		lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
		if len(lines) > 80 {
			lines = lines[len(lines)-80:]
		}
		m.logContent = "  " + strings.Join(lines, "\n  ")
	}
	m.screen = LogScreen{}
	return m
}
