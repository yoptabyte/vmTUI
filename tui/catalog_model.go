package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"vmctl/vm"
)

type catalogState int

const (
	catalogPicking catalogState = iota
	catalogCheckingVersion
	catalogVersionWarning
	catalogDownloading
	catalogDone
	catalogInstruction
)

type catalogDownloadDoneMsg struct {
	path string
}

type catalogDownloadErrMsg struct {
	err string
}

type catalogDownloadProgressMsg struct {
	percent int
	detail  string
}

type catalogVersionCheckMsg struct {
	warning string
}

type CatalogModel struct {
	entries         []vm.CatalogEntry
	entryIdx        int
	variantIdx      int
	choosingVariant bool
	promptingDir    bool
	dirInput        string
	state           catalogState
	status          string
	progressPercent int
	progressDetail  string
	err             string
	ISOReady        bool
	ISOPath         string
	downloadCh      chan tea.Msg
	cancelCh        chan struct{}
	versionWarning  string
}

func NewCatalogModel() CatalogModel {
	return CatalogModel{
		entries: vm.Catalog(),
		state:   catalogPicking,
	}
}

func (m CatalogModel) Init() tea.Cmd {
	return nil
}

func (m CatalogModel) ChoosingVariant() bool {
	return m.choosingVariant
}

func (m CatalogModel) PromptingDir() bool {
	return m.promptingDir
}

func (m CatalogModel) Update(msg tea.Msg) (CatalogModel, tea.Cmd) {
	switch msg := msg.(type) {
	case catalogDownloadDoneMsg:
		m.state = catalogDone
		m.status = "ISO ready"
		if !isISOPath(msg.path) {
			m.status = "Download ready"
		}
		m.progressPercent = 100
		m.progressDetail = "download complete"
		m.err = ""
		m.ISOReady = true
		m.ISOPath = msg.path
		m.downloadCh = nil
		m.cancelCh = nil
		return m, nil

	case catalogDownloadErrMsg:
		m.state = catalogPicking
		m.status = ""
		m.progressPercent = 0
		m.progressDetail = ""
		m.err = msg.err
		m.ISOReady = false
		m.ISOPath = ""
		m.downloadCh = nil
		m.cancelCh = nil
		return m, nil

	case catalogDownloadProgressMsg:
		m.progressPercent = msg.percent
		m.progressDetail = msg.detail
		if m.downloadCh != nil {
			return m, waitCatalogDownload(m.downloadCh)
		}
		return m, nil

	case catalogVersionCheckMsg:
		if msg.warning != "" {
			m.versionWarning = msg.warning
			m.state = catalogVersionWarning
		} else {
			m.state = catalogPicking
			m.promptingDir = true
			m.dirInput = vm.ISODir()
			m.err = ""
		}
		return m, nil

	case tea.KeyMsg:
		if m.state == catalogDownloading {
			switch msg.String() {
			case "esc":
				if m.cancelCh != nil {
					close(m.cancelCh)
					m.cancelCh = nil
				}
				return m, nil
			}
			return m, nil
		}

		if m.state == catalogVersionWarning {
			switch msg.String() {
			case "enter", "y":
				m.state = catalogPicking
				m.promptingDir = true
				m.dirInput = vm.ISODir()
				m.err = ""
				return m, nil
			case "esc", "n", "q":
				m.state = catalogPicking
				m.versionWarning = ""
				return m, nil
			}
			return m, nil
		}

		if m.state == catalogInstruction {
			switch msg.String() {
			case "enter", "y", "esc", "q":
				m.state = catalogPicking
				return m, nil
			}
			return m, nil
		}

		if m.promptingDir {
			switch msg.String() {
			case "esc", "left", "h":
				m.promptingDir = false
				return m, nil
			case "backspace":
				if len(m.dirInput) > 0 {
					m.dirInput = m.dirInput[:len(m.dirInput)-1]
				}
				m.err = ""
				return m, nil
			case "enter":
				dir := vm.ExpandPath(m.dirInput)
				if dir == "" {
					dir = vm.ISODir()
				}
				m.promptingDir = false
				m.state = catalogDownloading
				m.status = fmt.Sprintf("Downloading %s via aria2c -> %s", m.currentVariant().Name, dir)
				m.progressPercent = 0
				m.progressDetail = "starting aria2c"
				m.err = ""
				m.ISOReady = false
				m.ISOPath = ""
				m.downloadCh = make(chan tea.Msg, 32)
				m.cancelCh = make(chan struct{})
				return m, tea.Batch(m.cmdDownload(dir, m.downloadCh, m.cancelCh), waitCatalogDownload(m.downloadCh))
			default:
				if len(msg.String()) == 1 {
					m.dirInput += msg.String()
					m.err = ""
				}
				return m, nil
			}
		}

		switch msg.String() {
		case "up", "k":
			if m.choosingVariant {
				if m.variantIdx > 0 {
					m.variantIdx--
				}
			} else if m.entryIdx > 0 {
				m.entryIdx--
				m.variantIdx = 0
			}
		case "down", "j":
			if m.choosingVariant {
				if m.variantIdx < len(m.currentEntry().Variants)-1 {
					m.variantIdx++
				}
			} else if m.entryIdx < len(m.entries)-1 {
				m.entryIdx++
				m.variantIdx = 0
			}
		case "right", "l":
			if !m.choosingVariant && len(m.currentEntry().Variants) > 1 {
				m.choosingVariant = true
				m.variantIdx = 0
			}
		case "left", "h", "esc":
			if m.choosingVariant {
				m.choosingVariant = false
				return m, nil
			}
		case "enter":
			entry := m.currentEntry()
			if !m.choosingVariant && len(entry.Variants) > 1 {
				m.choosingVariant = true
				m.variantIdx = 0
				return m, nil
			}

			variant := m.currentVariant()
			if vm.IsInstructionOnly(variant) {
				m.state = catalogInstruction
				return m, nil
			}

			m.state = catalogCheckingVersion
			return m, func() tea.Msg {
				warning := vm.CheckVersion(entry, variant)
				return catalogVersionCheckMsg{warning: warning}
			}
		}
	}

	return m, nil
}

func (m CatalogModel) View() string {
	var sb strings.Builder

	sb.WriteString(styleTitle.Render("ISO Catalog"))
	sb.WriteString("\n")
	sb.WriteString(styleSubtitle.Render("Download installers with aria2c"))
	sb.WriteString("\n\n")

	for i, entry := range m.entries {
		line := fmt.Sprintf(" %s", entry.Distro)
		if i == m.entryIdx && !m.choosingVariant {
			sb.WriteString(styleSelected.Render(line))
		} else {
			sb.WriteString(styleNormal.Render(line))
		}
		sb.WriteString("\n")
		sb.WriteString(styleDim.Render("  " + entry.Desc))
		sb.WriteString("\n")
	}

	entry := m.currentEntry()
	sb.WriteString("\n")
	sb.WriteString(styleSection.Render("Variants"))
	sb.WriteString("\n")

	for i, variant := range entry.Variants {
		cached := ""
		if variant.URL != "" {
			if _, err := os.Stat(vm.LocalISOPath(entry, variant)); err == nil {
				cached = "  cached"
			}
		}
		line := fmt.Sprintf(" %s  ·  %d MiB%s", variant.Name, variant.SizeMiB, cached)
		if i == m.variantIdx && m.choosingVariant {
			sb.WriteString(styleSelected.Render(line))
		} else {
			sb.WriteString(styleNormal.Render(line))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	switch {
	case m.state == catalogInstruction:
		variant := m.currentVariant()
		sb.WriteString(styleTitle.Render("Download Instructions"))
		sb.WriteString("\n\n")
		sb.WriteString(styleNormal.Render(variant.Name))
		sb.WriteString("\n\n")
		sb.WriteString(styleDim.Render("Microsoft does not provide direct ISO download links."))
		sb.WriteString("\n")
		sb.WriteString(styleDim.Render("Visit this URL to download the ISO:"))
		sb.WriteString("\n\n")
		sb.WriteString(styleSelected.Render(variant.InstructionURL))
		sb.WriteString("\n\n")
		sb.WriteString(styleDim.Render("Alternatively, use the Media Creation Tool from the same page."))
		sb.WriteString("\n")
		sb.WriteString(styleDim.Render("Once downloaded, place the ISO in: " + vm.ISODir()))
		sb.WriteString("\n\n")
		sb.WriteString(styleDim.Render("Press any key to return."))
		sb.WriteString("\n\n")

	case m.promptingDir:
		sb.WriteString(styleSection.Render("Download directory"))
		sb.WriteString("\n")
		sb.WriteString(styleSelected.Render(m.dirInput + "▌"))
		sb.WriteString("\n")
		sb.WriteString(styleDim.Render("Enter a folder path; the ISO file name is chosen automatically."))
		sb.WriteString("\n\n")

	case m.state == catalogCheckingVersion:
		sb.WriteString(styleSubtitle.Render("Checking for newer version..."))
		sb.WriteString("\n\n")

	case m.state == catalogVersionWarning:
		sb.WriteString(styleWarning.Render("Version check warning"))
		sb.WriteString("\n")
		sb.WriteString(styleDim.Render(m.versionWarning))
		sb.WriteString("\n\n")
		sb.WriteString(styleNormal.Render("Press Enter or Y to proceed with the download anyway."))
		sb.WriteString("\n")
		sb.WriteString(styleDim.Render("Press Esc or N to cancel."))
		sb.WriteString("\n\n")

	case m.err != "":
		sb.WriteString(styleError.Render("✗ " + m.err))
		sb.WriteString("\n\n")

	case m.state == catalogDownloading:
		sb.WriteString(styleSubtitle.Render(m.status))
		sb.WriteString("\n")
		sb.WriteString(styleNormal.Render(progressBar(m.progressPercent, 36)))
		sb.WriteString("\n")
		if m.progressDetail != "" {
			sb.WriteString(styleDim.Render(m.progressDetail))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")

	case m.ISOReady:
		sb.WriteString(styleSuccess.Render("✓ " + m.ISOPath))
		sb.WriteString("\n")
		if isISOPath(m.ISOPath) {
			sb.WriteString(styleDim.Render("Press enter or y to use this ISO for the selected VM."))
			sb.WriteString("\n")
			sb.WriteString(styleDim.Render("Press Esc to return to the catalog."))
		} else {
			sb.WriteString(styleDim.Render("This file is not an ISO; return to the VM list and handle it manually."))
			sb.WriteString("\n")
			sb.WriteString(styleDim.Render("Press Esc to return to the catalog."))
		}
		sb.WriteString("\n\n")
	}

	help := m.buildHelp()
	sb.WriteString(styleStatusBar.Render(strings.Join(help, "   ")))
	return sb.String()
}

func (m CatalogModel) buildHelp() []string {
	switch m.state {
	case catalogDownloading:
		return []string{
			"esc cancel download",
		}
	case catalogDone:
		if !isISOPath(m.ISOPath) {
			return []string{
				"esc return to catalog",
			}
		}
		return []string{
			"enter/y use ISO",
			"esc return to catalog",
		}
	case catalogVersionWarning:
		return []string{
			"enter/y proceed",
			"esc/n cancel",
		}
	case catalogInstruction:
		return []string{
			"any key return",
		}
	default:
		return []string{
			"↑↓/jk navigate",
			"enter choose/download",
			"←/esc back",
			"y use ISO",
		}
	}
}

func (m CatalogModel) currentEntry() vm.CatalogEntry {
	if len(m.entries) == 0 {
		return vm.CatalogEntry{}
	}
	return m.entries[m.entryIdx]
}

func (m CatalogModel) currentVariant() vm.ImageVariant {
	entry := m.currentEntry()
	if len(entry.Variants) == 0 {
		return vm.ImageVariant{}
	}
	if m.variantIdx >= len(entry.Variants) {
		return entry.Variants[0]
	}
	return entry.Variants[m.variantIdx]
}

func (m CatalogModel) cmdDownload(dir string, ch chan tea.Msg, cancelCh chan struct{}) tea.Cmd {
	entry := m.currentEntry()
	variant := m.currentVariant()
	return func() tea.Msg {
		go func() {
			path, err := vm.DownloadISOWithCancel(entry, variant, dir, func(p vm.DownloadProgress) {
				ch <- catalogDownloadProgressMsg{percent: p.Percent, detail: p.Detail}
			}, cancelCh)
			if err != nil {
				ch <- catalogDownloadErrMsg{err: err.Error()}
				return
			}
			ch <- catalogDownloadDoneMsg{path: path}
		}()
		return nil
	}
}

func waitCatalogDownload(ch chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

func progressBar(percent int, width int) string {
	if width < 10 {
		width = 10
	}
	percent = min(max(percent, 0), 100)
	filled := (percent * width) / 100
	return fmt.Sprintf("[%s%s] %3d%%",
		strings.Repeat("=", filled),
		strings.Repeat("-", width-filled),
		percent,
	)
}

func isISOPath(path string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(path)), ".iso")
}
