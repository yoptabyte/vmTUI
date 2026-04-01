package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"vmctl/vm"
)

type FormScreen struct {
}

func (s FormScreen) Update(m RootModel, msg tea.Msg) (RootModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)

	if m.form.Aborted() {
		m.editingVMID = ""
		m.screen = ListScreen{}
		return m, nil
	}

	if m.form.Done() {
		cfg := m.form.Result()
		if m.editingVMID != "" {
			cfg.ID = m.editingVMID
			existing := findUserVM(m, m.editingVMID)
			if existing != nil {
				if existing.DiskSizeGiB > 0 && cfg.DiskSizeGiB > existing.DiskSizeGiB {
					diskPath := vm.ResolvePath(cfg.DiskFile)
					if _, err := os.Stat(diskPath); err == nil {
						if err := vm.ResizeDisk(diskPath, cfg.DiskSizeGiB); err != nil {
							m.list, _ = m.list.Update(statusMsg{
								text:  fmt.Sprintf("Failed to resize disk: %v", err),
								isErr: true,
							})
							m.editingVMID = ""
							m.screen = ListScreen{}
							return m, nil
						}
					}
				}
			}
			if err := m.store.Update(cfg); err != nil {
				m.list, _ = m.list.Update(statusMsg{
					text:  fmt.Sprintf("Failed to update VM: %v", err),
					isErr: true,
				})
			} else {
				m.list = m.list.Reload(m.store, m.pids)
				m.list, _ = m.list.Update(statusMsg{
					text: fmt.Sprintf("✓ VM %q updated", cfg.Name),
				})
			}
			m.editingVMID = ""
			m.screen = ListScreen{}
			return m, nil
		}

		savedCfg, err := m.store.Add(cfg)
		if err != nil {
			m.list, _ = m.list.Update(statusMsg{
				text:  fmt.Sprintf("Failed to save VM: %v", err),
				isErr: true,
			})
			m.screen = ListScreen{}
			return m, nil
		}

		m.list = m.list.Reload(m.store, m.pids)
		m.editingVMID = ""

		if savedCfg.ISOPath != "" {
			m = beginInstall(m, savedCfg, savedCfg.ISOPath)
			m.screen = InstallScreen{}
			return m, nil
		}

		m = beginInstall(m, savedCfg, "")
		m.screen = InstallScreen{}
		return m, nil
	}

	return m, cmd
}

func (s FormScreen) View(m RootModel) string {
	return renderScreen(m.width, m.height, m.form.View())
}
