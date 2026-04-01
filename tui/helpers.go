package tui

import (
	"os"
	"path/filepath"

	"vmctl/vm"
)

func beginInstall(m RootModel, cfg vm.VMConfig, prefill string) RootModel {
	m.installTarget = &cfg
	m.isoInput = prefill
	m.isoErr = ""
	m.isoChoices = cachedISOChoices()
	m.isoChoiceIdx = 0
	if prefill != "" {
		for i, choice := range m.isoChoices {
			if choice == prefill {
				m.isoChoiceIdx = i
				break
			}
		}
	}
	return m
}

func deleteVM(m RootModel, cfg vm.VMConfig) error {
	if cfg.DiskFile != "" {
		diskPath := vm.ExpandPath(cfg.DiskFile)
		if diskPath != "" {
			if !filepath.IsAbs(diskPath) {
				if abs, err := filepath.Abs(diskPath); err == nil {
					diskPath = abs
				}
			}
			if err := os.Remove(diskPath); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}

	if _, err := m.store.Remove(cfg.ID); err != nil {
		return err
	}
	m.pids.Clear(cfg.ID)
	return nil
}

func findUserVM(m RootModel, id string) *vm.VMConfig {
	for i := range m.store.VMs {
		if m.store.VMs[i].ID == id {
			return &m.store.VMs[i]
		}
	}
	return nil
}

func findVMByID(m RootModel, id string) *vm.VMConfig {
	for _, cfg := range m.store.AllVMs() {
		if cfg.ID == id {
			copy := cfg
			return &copy
		}
	}
	return nil
}
