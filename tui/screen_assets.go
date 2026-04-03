package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"vmtui/vm"
)

type AssetsScreen struct{}

func (s AssetsScreen) Update(m RootModel, msg tea.Msg) (RootModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() != "x" {
			m.assetPendingDeletePath = ""
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc", "left", "h", "a":
			m.screen = ListScreen{}
			return m, nil
		case "up", "k":
			if m.assetCursor > 0 {
				m.assetCursor--
			}
			return m, nil
		case "down", "j":
			if m.assetCursor < assetCount(m)-1 {
				m.assetCursor++
			}
			return m, nil
		case "l":
			entry := selectedAsset(m)
			if entry == nil {
				return m, nil
			}
			if len(entry.OwnerIDs) == 0 {
				m.assetErr = "Selected asset is not linked to any VM"
				return m, nil
			}
			if len(entry.OwnerIDs) > 1 {
				m.assetErr = "Selected asset is shared by multiple VMs; open the log from the VM list"
				return m, nil
			}
			if owner := findVMByID(m, entry.OwnerIDs[0]); owner != nil {
				m = openLog(m, *owner)
			}
			return m, nil
		case "x":
			entry := selectedAsset(m)
			if entry == nil {
				return m, nil
			}
			if entry.Kind != "iso" {
				m.assetErr = "Only cached ISO files can be deleted from Assets"
				return m, nil
			}
			if !entry.Exists {
				m.assetErr = "Selected ISO is already missing"
				return m, nil
			}
			if !isCachedISO(entry.Path) {
				m.assetErr = "Only ISO files inside the vmtui cache can be deleted here"
				return m, nil
			}
			if len(entry.OwnerIDs) > 0 {
				m.assetErr = "This ISO is still assigned to a VM; clear or change it before deleting"
				return m, nil
			}
			if m.assetPendingDeletePath != entry.Path {
				m.assetPendingDeletePath = entry.Path
				m.assetErr = fmt.Sprintf("Press x again to delete cached ISO %q", filepath.Base(entry.Path))
				return m, nil
			}
			if err := os.Remove(entry.Path); err != nil {
				m.assetErr = fmt.Sprintf("Failed to delete ISO: %v", err)
				return m, nil
			}
			m.assetPendingDeletePath = ""
			m.assetErr = ""
			m = openAssets(m)
			return m, nil
		}
	}
	return m, nil
}

func (s AssetsScreen) View(m RootModel) string {
	var sb strings.Builder
	sb.WriteString(styleTitle.Render("Assets"))
	sb.WriteString("\n\n")

	if m.assetErr != "" {
		sb.WriteString(styleError.Render("✗ " + m.assetErr))
		sb.WriteString("\n\n")
	}

	sb.WriteString(styleSection.Render("Disks"))
	sb.WriteString("\n")
	if len(m.assetDisks) == 0 {
		sb.WriteString(styleDim.Render("No VM disks found"))
		sb.WriteString("\n")
	} else {
		cursor := 0
		for _, entry := range m.assetDisks {
			title := styleNormal.Render(entry.Title)
			if cursor == m.assetCursor {
				title = styleSelected.Render(entry.Title)
			}
			sb.WriteString(title)
			sb.WriteString("\n")
			metaStyle := styleError
			if entry.Exists {
				metaStyle = styleSuccess
			}
			sb.WriteString(metaStyle.Render("  " + entry.Meta))
			sb.WriteString("\n")
			sb.WriteString(styleDim.Render("  " + entry.Path))
			sb.WriteString("\n")
			cursor++
		}
	}

	sb.WriteString("\n")
	sb.WriteString(styleSection.Render("Cached ISO"))
	sb.WriteString("\n")
	if len(m.assetISOs) == 0 {
		sb.WriteString(styleDim.Render("No cached ISO found"))
		sb.WriteString("\n")
	} else {
		cursor := len(m.assetDisks)
		for _, entry := range m.assetISOs {
			title := styleNormal.Render(entry.Title)
			if cursor == m.assetCursor {
				title = styleSelected.Render(entry.Title)
			}
			sb.WriteString(title)
			sb.WriteString("\n")
			metaStyle := styleError
			if entry.Exists {
				metaStyle = styleSuccess
			}
			sb.WriteString(metaStyle.Render("  " + entry.Meta))
			sb.WriteString("\n")
			sb.WriteString(styleDim.Render("  " + entry.Path))
			sb.WriteString("\n")
			cursor++
		}
	}

	sb.WriteString("\n")
	sb.WriteString(styleStatusBar.Render("↑↓ navigate  •  l VM log  •  x delete cached ISO  •  esc back  •  q quit"))
	return renderScreen(m.width, m.height, sb.String())
}

func openAssets(m RootModel) RootModel {
	disks, isos, err := collectAssets(m.store)
	m.assetDisks = disks
	m.assetISOs = isos
	m.assetErr = ""
	m.assetPendingDeletePath = ""
	if err != nil {
		m.assetErr = err.Error()
	}
	if m.assetCursor >= assetCount(m) {
		m.assetCursor = max(0, assetCount(m)-1)
	}
	m.screen = AssetsScreen{}
	return m
}

func assetCount(m RootModel) int {
	return len(m.assetDisks) + len(m.assetISOs)
}

func selectedAsset(m RootModel) *assetEntry {
	if m.assetCursor < 0 || m.assetCursor >= assetCount(m) {
		return nil
	}
	if m.assetCursor < len(m.assetDisks) {
		return &m.assetDisks[m.assetCursor]
	}
	idx := m.assetCursor - len(m.assetDisks)
	return &m.assetISOs[idx]
}

func isCachedISO(path string) bool {
	cacheDir := vm.ResolvePath(vm.ISODir())
	path = vm.ResolvePath(path)
	rel, err := filepath.Rel(cacheDir, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func collectAssets(store *vm.Store) ([]assetEntry, []assetEntry, error) {
	diskEntries := collectDiskAssets(store.AllVMs())
	isoEntries, err := collectISOAssets(store.AllVMs())
	return diskEntries, isoEntries, err
}

func collectDiskAssets(vms []vm.VMConfig) []assetEntry {
	type diskInfo struct {
		ownerIDs   []string
		ownerNames []string
		path       string
	}

	disks := make(map[string]*diskInfo)
	for _, cfg := range vms {
		if strings.TrimSpace(cfg.DiskFile) == "" {
			continue
		}
		path := vm.ResolvePath(cfg.DiskFile)
		info := disks[path]
		if info == nil {
			info = &diskInfo{path: path}
			disks[path] = info
		}
		info.ownerIDs = append(info.ownerIDs, cfg.ID)
		info.ownerNames = append(info.ownerNames, cfg.Name)
	}

	paths := make([]string, 0, len(disks))
	for path := range disks {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	var out []assetEntry
	for _, path := range paths {
		info := disks[path]
		title := strings.Join(info.ownerNames, ", ")
		meta := "missing"
		exists := false
		if st, err := os.Stat(path); err == nil {
			meta = formatBytes(st.Size())
			exists = true
		}
		out = append(out, assetEntry{
			Kind:       "disk",
			Title:      title,
			Meta:       meta,
			Path:       path,
			Exists:     exists,
			OwnerIDs:   info.ownerIDs,
			OwnerNames: info.ownerNames,
		})
	}
	return out
}

func collectISOAssets(vms []vm.VMConfig) ([]assetEntry, error) {
	usedByNames := make(map[string][]string)
	usedByIDs := make(map[string][]string)
	for _, cfg := range vms {
		if strings.TrimSpace(cfg.ISOPath) == "" {
			continue
		}
		path := vm.ResolvePath(cfg.ISOPath)
		usedByNames[path] = append(usedByNames[path], cfg.Name)
		usedByIDs[path] = append(usedByIDs[path], cfg.ID)
	}

	paths, err := vm.ListCachedISOs()
	if err != nil {
		return nil, err
	}
	for path := range usedByNames {
		found := false
		for _, existing := range paths {
			if vm.ResolvePath(existing) == path {
				found = true
				break
			}
		}
		if !found {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)

	var out []assetEntry
	for _, path := range paths {
		label := filepath.Base(path)
		meta := "missing"
		exists := false
		if st, err := os.Stat(path); err == nil {
			meta = formatBytes(st.Size())
			exists = true
		}
		if owners := usedByNames[vm.ResolvePath(path)]; len(owners) > 0 {
			meta += "  ·  used by " + strings.Join(owners, ", ")
		}
		out = append(out, assetEntry{
			Kind:       "iso",
			Title:      label,
			Meta:       meta,
			Path:       path,
			Exists:     exists,
			OwnerIDs:   usedByIDs[vm.ResolvePath(path)],
			OwnerNames: usedByNames[vm.ResolvePath(path)],
		})
	}
	return out, nil
}

func formatBytes(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(div), "KMGTPE"[exp])
}

func cachedISOChoices() []string {
	choices, err := vm.ListCachedISOs()
	if err != nil {
		return nil
	}
	return choices
}
