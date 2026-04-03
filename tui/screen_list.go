package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"vmtui/vm"
)

type ListScreen struct{}

func (s ListScreen) Update(m RootModel, msg tea.Msg) (RootModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "enter":
			sel := m.list.Selected()
			if sel != nil {
				return m, m.cmdLaunch(*sel)
			}

		case "n":
			m.editingVMID = ""
			m.form = NewFormModel()
			m.screen = FormScreen{}
			return m, m.form.Init()

		case "i":
			sel := m.list.Selected()
			if sel == nil {
				return m, nil
			}
			m = beginInstall(m, *sel, sel.ISOPath)
			m.screen = InstallScreen{}
			return m, nil

		case "d":
			m.catalog = NewCatalogModel()
			m.screen = CatalogScreen{}
			return m, m.catalog.Init()

		case "a":
			m = openAssets(m)
			m.screen = AssetsScreen{}
			return m, nil

		case "s":
			sel := m.list.Selected()
			if sel == nil {
				return m, nil
			}
			if !m.pids.IsRunning(sel.ID) {
				m.list, _ = m.list.Update(statusMsg{
					text:  fmt.Sprintf("%s is not running", sel.Name),
					isErr: true,
				})
				return m, nil
			}
			pid := m.pids.PID(sel.ID)
			if err := vm.StopPID(pid); err != nil {
				m.list, _ = m.list.Update(statusMsg{
					text:  fmt.Sprintf("Failed to stop %s: %v", sel.Name, err),
					isErr: true,
				})
				return m, nil
			}
			m.pids.Clear(sel.ID)
			m.list = m.list.RefreshStatus(m.pids)
			m.list, _ = m.list.Update(statusMsg{
				text: fmt.Sprintf("✓ Stopped %s (pid %d)", sel.Name, pid),
			})
			return m, nil

		case "e":
			sel := m.list.Selected()
			if sel == nil {
				return m, nil
			}
			if m.pids.IsRunning(sel.ID) {
				m.list, _ = m.list.Update(statusMsg{
					text:  fmt.Sprintf("Stop %s before editing it", sel.Name),
					isErr: true,
				})
				return m, nil
			}
			m.editingVMID = sel.ID
			m.form = NewEditFormModel(*sel)
			m.screen = FormScreen{}
			return m, m.form.Init()

		case "l":
			sel := m.list.Selected()
			if sel == nil {
				return m, nil
			}
			m = openLog(m, *sel)
			return m, nil

		case "x", "delete":
			sel := m.list.Selected()
			if sel == nil {
				return m, nil
			}
			if m.pids.IsRunning(sel.ID) {
				m.list, _ = m.list.Update(statusMsg{
					text:  fmt.Sprintf("Stop %s before deleting it", sel.Name),
					isErr: true,
				})
				return m, nil
			}
			target := *sel
			m.deleteTarget = &target
			m.screen = DeleteScreen{}
			return m, nil
		}

	case vmStartedMsg:
		m.pids.Set(msg.id, msg.pid)
		m.list = m.list.RefreshStatus(m.pids)
		m.list, _ = m.list.Update(statusMsg{
			text: fmt.Sprintf("✓ %s launched (pid %d)", msg.id, msg.pid),
		})
		return m, nil

	case vmErrorMsg:
		m.list, _ = m.list.Update(statusMsg{
			text:  fmt.Sprintf("✗ %s: %s", msg.id, msg.err),
			isErr: true,
		})
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (s ListScreen) View(m RootModel) string {
	return m.list.View(m.width, m.height)
}

func (m RootModel) cmdLaunch(cfg vm.VMConfig) tea.Cmd {
	return func() tea.Msg {
		proc, err := vm.Launch(cfg, vm.RunOptions{})
		if err != nil {
			return vmErrorMsg{id: cfg.ID, err: err.Error()}
		}
		return vmStartedMsg{id: cfg.ID, pid: proc.Process.Pid}
	}
}
