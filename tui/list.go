package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"vmctl/vm"
)

// ListModel is the main screen: shows all VMs with live status.
type ListModel struct {
	vms    []vm.VMConfig
	status map[string]vm.VMStatus // vm.ID -> status
	cursor int
	msg    string
	msgErr bool
}

func NewListModel(store *vm.Store, pids *vm.PIDStore) ListModel {
	vms := store.AllVMs()
	ids := vmIDs(vms)
	return ListModel{
		vms:    vms,
		status: pids.StatusAll(ids),
	}
}

// Reload rebuilds the VM list from the store (after adding a new VM).
func (m ListModel) Reload(store *vm.Store, pids *vm.PIDStore) ListModel {
	m.vms = store.AllVMs()
	if m.cursor >= len(m.vms) {
		m.cursor = max(0, len(m.vms)-1)
	}
	m.status = pids.StatusAll(vmIDs(m.vms))
	return m
}

// RefreshStatus re-checks running state from the PID store.
// Called every 2 seconds by the tick loop in model.go.
func (m ListModel) RefreshStatus(pids *vm.PIDStore) ListModel {
	m.status = pids.StatusAll(vmIDs(m.vms))
	return m
}

// Selected returns a pointer to the currently highlighted VM, or nil.
func (m ListModel) Selected() *vm.VMConfig {
	if len(m.vms) == 0 {
		return nil
	}
	v := m.vms[m.cursor]
	return &v
}

func (m ListModel) Update(msg tea.Msg) (ListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.vms)-1 {
				m.cursor++
			}
		}
	case statusMsg:
		m.msg = msg.text
		m.msgErr = msg.isErr
	}
	return m, nil
}

func (m ListModel) View(width, height int) string {
	contentWidth, contentHeight := panelContentSize(width, height)
	headerLeft := lipgloss.JoinHorizontal(
		lipgloss.Center,
		styleSelected.Render("vmTUI"),
		styleSubtitle.Render("QEMU Virtual Machines"),
	)
	headerBlock := lipgloss.JoinVertical(
		lipgloss.Left,
		headerLeft,
		styleDim.Render(fmt.Sprintf("%d configured machine(s)", len(m.vms))),
	)

	var messageBlock string
	if m.msg != "" {
		if m.msgErr {
			messageBlock = styleError.Render("✗ " + m.msg)
		} else {
			messageBlock = styleSuccess.Render("✓ " + m.msg)
		}
	}

	help := []string{
		"↑↓/jk  navigate",
		"enter  launch",
		"s  stop",
		"i  boot installer",
		"d  download ISO",
		"a  assets",
		"n  new VM",
		"e  edit VM",
		"l  view log",
		"x  delete VM",
		"q  quit",
	}
	footer := styleStatusBar.Width(contentWidth - styleStatusBar.GetHorizontalFrameSize()).Render(strings.Join(help, "   "))

	mainWidth := max(24, contentWidth)
	listWidth := min(38, max(26, mainWidth/3))
	detailWidth := max(28, mainWidth-listWidth-2)

	listBlock := m.renderVMList(listWidth)
	detailBlock := m.renderSelectedVM(detailWidth)
	mainBlock := lipgloss.JoinHorizontal(lipgloss.Top, listBlock, "  ", detailBlock)

	top := lipgloss.JoinVertical(
		lipgloss.Left,
		headerBlock,
		"",
		mainBlock,
	)
	if messageBlock != "" {
		top = lipgloss.JoinVertical(lipgloss.Left, top, "", messageBlock)
	}

	spacerHeight := max(0, contentHeight-lipgloss.Height(top)-lipgloss.Height(footer))
	body := lipgloss.JoinVertical(
		lipgloss.Left,
		top,
		lipgloss.NewStyle().Height(spacerHeight).Render(""),
		footer,
	)

	return renderScreen(width, height, body)
}

func (m ListModel) renderVMList(width int) string {
	title := styleSection.Render("Machines")
	if len(m.vms) == 0 {
		cardBody := lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			"",
			styleDim.Render("No virtual machines yet."),
			styleDim.Render("Press n to create one or d to fetch an ISO."),
		)
		return styleCard.Width(max(16, width-styleCard.GetHorizontalFrameSize())).Render(cardBody)
	}

	rowInnerWidth := max(16, width-styleCard.GetHorizontalFrameSize()-styleRow.GetHorizontalFrameSize())
	rows := []string{title}
	for i, v := range m.vms {
		st := m.status[v.ID]

		statusBadge := styleBadgeStopped.Render("stopped")
		if st.Running {
			statusBadge = styleBadgeRunning.Render("running")
		}

		nameStyle := styleNormal.Copy().Bold(true)
		rowStyle := styleRow.Copy().Width(rowInnerWidth)
		if i == m.cursor {
			nameStyle = nameStyle.Foreground(c(colors.AccentSoft))
			rowStyle = styleRowSelected.Copy().Width(rowInnerWidth)
		}

		meta := truncateMiddle(
			fmt.Sprintf("%d MiB  •  %d vCPU  •  ::%d", v.MemMiB, v.VCPUs, v.SSHPort),
			rowInnerWidth,
		)
		row := lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.JoinHorizontal(lipgloss.Left, nameStyle.Render(v.Name), "  ", statusBadge),
			styleDim.Width(rowInnerWidth).Render(meta),
		)
		rows = append(rows, rowStyle.Render(row))
	}

	return styleCard.Width(max(16, width-styleCard.GetHorizontalFrameSize())).Render(
		lipgloss.JoinVertical(lipgloss.Left, rows...),
	)
}

func (m ListModel) renderSelectedVM(width int) string {
	title := styleSection.Render("Selected VM")
	if len(m.vms) == 0 {
		body := lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			"",
			styleDim.Render("Select or create a virtual machine to see details here."),
		)
		return styleCard.Width(max(20, width-styleCard.GetHorizontalFrameSize())).Render(body)
	}

	sel := m.vms[m.cursor]
	st := m.status[sel.ID]
	statusText := styleBadgeStopped.Render("stopped")
	if st.Running {
		statusText = styleBadgeRunning.Render(fmt.Sprintf("running  pid %d", st.PID))
	}

	bodyWidth := max(20, width-styleCard.GetHorizontalFrameSize())
	details := []string{
		title,
		"",
		lipgloss.JoinHorizontal(lipgloss.Left, styleTitle.Render(sel.Name), "  ", statusText),
		"",
		renderKV(bodyWidth, "ID", sel.ID),
		renderKV(bodyWidth, "Memory", fmt.Sprintf("%d MiB", sel.MemMiB)),
		renderKV(bodyWidth, "vCPUs", fmt.Sprintf("%d", sel.VCPUs)),
		renderKV(bodyWidth, "SSH", fmt.Sprintf("127.0.0.1:%d", sel.SSHPort)),
		renderKV(bodyWidth, "Disk size", fmt.Sprintf("%d GiB", sel.DiskSizeGiB)),
		renderKV(bodyWidth, "Disk", sel.DiskFile),
	}
	if strings.TrimSpace(sel.ISOPath) != "" {
		details = append(details, renderKV(bodyWidth, "ISO", sel.ISOPath))
	} else {
		details = append(details, renderKV(bodyWidth, "ISO", "not assigned"))
	}

	actions := []string{
		"",
		styleSection.Render("Actions"),
		styleDim.Render("enter launch  •  s stop  •  i installer"),
		styleDim.Render("e edit  •  l log  •  x delete"),
	}
	details = append(details, actions...)

	return styleCard.Width(bodyWidth).Render(lipgloss.JoinVertical(lipgloss.Left, details...))
}

// vmIDs extracts the ID slice from a VM list.
func vmIDs(vms []vm.VMConfig) []string {
	ids := make([]string, len(vms))
	for i, v := range vms {
		ids[i] = v.ID
	}
	return ids
}

func truncateMiddle(s string, width int) string {
	if width <= 0 || lipgloss.Width(s) <= width {
		return s
	}
	if width <= 3 {
		return strings.Repeat(".", width)
	}

	runes := []rune(s)
	if len(runes) <= width {
		return s
	}

	left := (width - 1) / 2
	right := width - left - 1
	return string(runes[:left]) + "…" + string(runes[len(runes)-right:])
}

func renderKV(width int, key string, value string) string {
	labelWidth := min(12, max(8, width/5))
	valueWidth := max(8, width-labelWidth-3)
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		styleDim.Width(labelWidth).Render(key),
		" : ",
		styleNormal.Width(valueWidth).Render(truncateMiddle(value, valueWidth)),
	)
}
