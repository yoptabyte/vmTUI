package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"vmctl/vm"
)

type InstallScreen struct{}

func (s InstallScreen) Update(m RootModel, msg tea.Msg) (RootModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.screen = ListScreen{}
			return m, nil
		case "enter":
			iso := vm.ExpandPath(m.isoInput)
			if iso == "" && len(m.isoChoices) > 0 {
				if m.isoChoiceIdx < 0 {
					m.isoChoiceIdx = 0
				}
				if m.isoChoiceIdx >= len(m.isoChoices) {
					m.isoChoiceIdx = len(m.isoChoices) - 1
				}
				iso = m.isoChoices[m.isoChoiceIdx]
				m.isoInput = iso
			}
			if iso == "" {
				m.isoErr = "ISO path cannot be empty"
				return m, nil
			}
			info, err := os.Stat(iso)
			if err != nil {
				m.isoErr = fmt.Sprintf("File not found: %s", iso)
				return m, nil
			}
			if info.IsDir() {
				m.isoErr = fmt.Sprintf("Expected ISO file, got directory: %s", iso)
				return m, nil
			}
			sizeGiB := 40
			if m.installTarget.DiskSizeGiB > 0 {
				sizeGiB = m.installTarget.DiskSizeGiB
			}
			if m.installTarget.DiskFile != "" {
				diskPath := vm.ResolvePath(m.installTarget.DiskFile)
				if _, err := os.Stat(diskPath); os.IsNotExist(err) {
					if err2 := vm.CreateDisk(diskPath, sizeGiB); err2 != nil {
						m.isoErr = fmt.Sprintf("Failed to create disk: %v", err2)
						return m, nil
					}
				}
				m.installTarget.DiskFile = diskPath
			}
			if m.installTarget.ISOPath != iso {
				updated := *m.installTarget
				updated.ISOPath = iso
				if err := m.store.Update(updated); err != nil {
					m.isoErr = fmt.Sprintf("Failed to save ISO path: %v", err)
					return m, nil
				}
				m.installTarget = &updated
				m.list = m.list.Reload(m.store, m.pids)
			}
			return m, cmdInstall(*m.installTarget, iso)

		case "up", "k":
			if m.isoInput == "" && m.isoChoiceIdx > 0 {
				m.isoChoiceIdx--
			}
		case "down", "j":
			if m.isoInput == "" && m.isoChoiceIdx < len(m.isoChoices)-1 {
				m.isoChoiceIdx++
			}
		case "backspace":
			if len(m.isoInput) > 0 {
				m.isoInput = m.isoInput[:len(m.isoInput)-1]
				m.isoErr = ""
			}
		case "tab":
			if m.isoInput == "" && len(m.isoChoices) > 0 {
				m.isoInput = m.isoChoices[m.isoChoiceIdx]
			} else {
				matches := isoMatches(m.isoInput)
				if len(matches) == 1 {
					m.isoInput = matches[0]
				}
			}
		default:
			if len(msg.String()) == 1 {
				m.isoInput += msg.String()
				m.isoErr = ""
			}
		}

	case vmStartedMsg:
		m.pids.Set(msg.id, msg.pid)
		m.list = m.list.RefreshStatus(m.pids)
		m.list, _ = m.list.Update(statusMsg{
			text: fmt.Sprintf("✓ Installer for %s launched (pid %d)", msg.id, msg.pid),
		})
		m.screen = ListScreen{}

	case vmErrorMsg:
		m.isoErr = msg.err
	}
	return m, nil
}

func (s InstallScreen) View(m RootModel) string {
	var sb strings.Builder
	name := ""
	if m.installTarget != nil {
		name = m.installTarget.Name
	}
	sb.WriteString(styleTitle.Render(fmt.Sprintf("Boot Installer ISO -> %s", name)))
	sb.WriteString("\n\n")
	sb.WriteString(styleSection.Render("ISO path"))
	sb.WriteString("\n")
	sb.WriteString(styleSelected.Render(m.isoInput + "▌"))
	sb.WriteString("\n\n")

	if m.isoErr != "" {
		sb.WriteString(styleError.Render("✗ " + m.isoErr))
		sb.WriteString("\n\n")
	}

	if len(m.isoChoices) > 0 {
		sb.WriteString(styleSection.Render("Cached ISOs"))
		sb.WriteString("\n")
		for i, choice := range m.isoChoices {
			line := choice
			if m.isoInput == "" && i == m.isoChoiceIdx {
				sb.WriteString(styleSelected.Render(line))
			} else {
				sb.WriteString(styleDim.Render(line))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if matches := isoMatches(m.isoInput); len(matches) > 0 {
		sb.WriteString(styleSection.Render("Suggestions"))
		sb.WriteString("\n")
		for _, match := range matches {
			sb.WriteString(styleDim.Render("  " + match))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if m.installTarget != nil && m.installTarget.DiskFile != "" {
		if _, err := os.Stat(m.installTarget.DiskFile); os.IsNotExist(err) {
			sizeGiB := 40
			if m.installTarget.DiskSizeGiB > 0 {
				sizeGiB = m.installTarget.DiskSizeGiB
			}
			sb.WriteString(styleSubtitle.Render(
				fmt.Sprintf("Disk %s not found; it will be created automatically (%dG)",
					m.installTarget.DiskFile, sizeGiB)))
			sb.WriteString("\n\n")
		}
	}

	sb.WriteString(styleStatusBar.Render("↑↓ cached ISO  •  tab autocomplete  •  enter confirm  •  esc back  •  ctrl+c quit"))
	return renderScreen(m.width, m.height, sb.String())
}

func cmdInstall(cfg vm.VMConfig, isoPath string) tea.Cmd {
	return func() tea.Msg {
		proc, err := vm.InstallISO(cfg, isoPath)
		if err != nil {
			return vmErrorMsg{id: cfg.ID, err: err.Error()}
		}
		return vmStartedMsg{id: cfg.ID, pid: proc.Process.Pid}
	}
}

func isoMatches(partial string) []string {
	if partial == "" {
		return nil
	}
	partial = vm.ExpandPath(partial)
	dir := filepath.Dir(partial)
	base := filepath.Base(partial)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var matches []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, base) && strings.HasSuffix(name, ".iso") {
			matches = append(matches, filepath.Join(dir, name))
			if len(matches) >= 5 {
				break
			}
		}
	}
	return matches
}
