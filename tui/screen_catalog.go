package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type CatalogScreen struct{}

func (s CatalogScreen) Update(m RootModel, msg tea.Msg) (RootModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.catalog.state == catalogPicking && !m.catalog.ChoosingVariant() && !m.catalog.PromptingDir() {
				m.screen = ListScreen{}
				return m, nil
			}
			if m.catalog.state == catalogDone {
				m.catalog.state = catalogPicking
				m.catalog.ISOReady = false
				m.catalog.ISOPath = ""
				m.catalog.versionWarning = ""
				return m, nil
			}
			if m.catalog.state == catalogVersionWarning || m.catalog.state == catalogInstruction {
				m.catalog.state = catalogPicking
				m.catalog.versionWarning = ""
				return m, nil
			}
		case "y", "enter":
			if m.catalog.state == catalogDone && m.catalog.ISOReady {
				if !strings.HasSuffix(strings.ToLower(m.catalog.ISOPath), ".iso") {
					m.list, _ = m.list.Update(statusMsg{
						text:  fmt.Sprintf("Downloaded file is not an ISO: %s", m.catalog.ISOPath),
						isErr: true,
					})
					m.screen = ListScreen{}
					return m, nil
				}
				sel := m.list.Selected()
				if sel != nil {
					m = beginInstall(m, *sel, m.catalog.ISOPath)
					m.screen = InstallScreen{}
					return m, nil
				}
				m.list, _ = m.list.Update(statusMsg{
					text: fmt.Sprintf("ISO ready: %s — select a VM and press 'i'", m.catalog.ISOPath),
				})
				m.screen = ListScreen{}
				return m, nil
			}
		case "n":
			if m.catalog.state == catalogDone || m.catalog.state == catalogVersionWarning || m.catalog.state == catalogInstruction {
				m.catalog.state = catalogPicking
				m.catalog.versionWarning = ""
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.catalog, cmd = m.catalog.Update(msg)
	return m, cmd
}

func (s CatalogScreen) View(m RootModel) string {
	return renderScreen(m.width, m.height, m.catalog.View())
}
