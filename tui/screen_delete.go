package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type DeleteScreen struct{}

func (s DeleteScreen) Update(m RootModel, msg tea.Msg) (RootModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc", "n":
			m.deleteTarget = nil
			m.screen = ListScreen{}
			return m, nil
		case "y":
			if m.deleteTarget == nil {
				m.screen = ListScreen{}
				return m, nil
			}
			target := *m.deleteTarget
			m.deleteTarget = nil
			if err := deleteVM(m, target); err != nil {
				m.list, _ = m.list.Update(statusMsg{
					text:  fmt.Sprintf("Failed to delete VM: %v", err),
					isErr: true,
				})
			} else {
				m.list = m.list.Reload(m.store, m.pids)
				m.list, _ = m.list.Update(statusMsg{
					text: fmt.Sprintf("✓ Deleted VM %q", target.Name),
				})
			}
			m.screen = ListScreen{}
			return m, nil
		}
	}
	return m, nil
}

func (s DeleteScreen) View(m RootModel) string {
	var sb strings.Builder
	sb.WriteString(styleTitle.Render("Confirm Delete"))
	sb.WriteString("\n\n")
	if m.deleteTarget != nil {
		sb.WriteString(styleError.Render(fmt.Sprintf("Delete VM %q?", m.deleteTarget.Name)))
		sb.WriteString("\n\n")
		sb.WriteString(styleDim.Render("This will remove the VM entry and its qcow2 disk file:"))
		sb.WriteString("\n")
		sb.WriteString(styleNormal.Render(m.deleteTarget.DiskFile))
		sb.WriteString("\n\n")
	}
	sb.WriteString(styleStatusBar.Render("y confirm  •  n/esc cancel  •  q quit"))
	return renderScreen(m.width, m.height, sb.String())
}
