package vm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// VMType distinguishes VM launch backends.
type VMType string

const (
	TypeQEMU VMType = "qemu"

	legacyTypeCustom VMType = "custom"
)

// VMConfig holds everything needed to launch and describe a VM.
type VMConfig struct {
	ID          string `json:"id"`   // unique slug
	Name        string `json:"name"` // display name
	Type        VMType `json:"type"`
	MemMiB      int    `json:"memMiB"`
	VCPUs       int    `json:"vcpus"`
	DiskFile    string `json:"diskFile"` // path to .qcow2
	DiskSizeGiB int    `json:"diskSizeGiB,omitempty"`
	SSHPort     int    `json:"sshPort"`
	FlakeApp    string `json:"flakeApp,omitempty"` // legacy field kept for compatibility
	ISOPath     string `json:"isoPath,omitempty"`
}

// Store persists user-created VM configs to a JSON file.
type Store struct {
	path string
	VMs  []VMConfig `json:"vms"`
}

func storeFile() string {
	dir, _ := os.UserConfigDir()
	return filepath.Join(dir, "vmtui", "vms.json")
}

// LoadStore reads the store from disk (or returns empty store if not found).
func LoadStore() (*Store, error) {
	s := &Store{path: storeFile()}
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}
	for i := range s.VMs {
		s.VMs[i] = normalizeVMConfig(s.VMs[i])
	}
	return s, nil
}

// Save writes the store to disk.
func (s *Store) Save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

// Add appends a new VM config and persists.
func (s *Store) Add(cfg VMConfig) (VMConfig, error) {
	if strings.TrimSpace(cfg.Name) == "" {
		return VMConfig{}, fmt.Errorf("vm name cannot be empty")
	}
	cfg = normalizeVMConfig(cfg)
	if cfg.ID == "" {
		cfg.ID = slugify(cfg.Name)
	}
	cfg.ID = uniqueVMID(cfg.ID, s.AllVMs())
	if err := s.ensureUniqueDiskFile(cfg, ""); err != nil {
		return VMConfig{}, err
	}
	s.VMs = append(s.VMs, cfg)
	return cfg, s.Save()
}

// AllVMs returns all stored VMs.
func (s *Store) AllVMs() []VMConfig {
	return slices.Clone(s.VMs)
}

// Update replaces a stored user VM config with the same ID.
func (s *Store) Update(cfg VMConfig) error {
	cfg = normalizeVMConfig(cfg)
	if err := s.ensureUniqueDiskFile(cfg, cfg.ID); err != nil {
		return err
	}
	for i := range s.VMs {
		if s.VMs[i].ID == cfg.ID {
			s.VMs[i] = cfg
			return s.Save()
		}
	}
	return fmt.Errorf("vm not found: %s", cfg.ID)
}

// Remove deletes a stored user VM config and returns the removed record.
func (s *Store) Remove(id string) (VMConfig, error) {
	for i := range s.VMs {
		if s.VMs[i].ID == id {
			removed := s.VMs[i]
			s.VMs = slices.Delete(s.VMs, i, i+1)
			return removed, s.Save()
		}
	}
	return VMConfig{}, fmt.Errorf("vm not found: %s", id)
}

func (s *Store) ensureUniqueDiskFile(cfg VMConfig, selfID string) error {
	if strings.TrimSpace(cfg.DiskFile) == "" {
		return nil
	}
	want := ResolvePath(cfg.DiskFile)
	for _, existing := range s.AllVMs() {
		if existing.ID == selfID || existing.ID == cfg.ID {
			continue
		}
		if strings.TrimSpace(existing.DiskFile) == "" {
			continue
		}
		if ResolvePath(existing.DiskFile) == want {
			return fmt.Errorf("disk file is already used by %q: %s", existing.Name, want)
		}
	}
	return nil
}

func uniqueVMID(base string, existing []VMConfig) string {
	if base == "" {
		base = "vm"
	}
	used := make(map[string]struct{}, len(existing))
	for _, cfg := range existing {
		used[cfg.ID] = struct{}{}
	}
	if _, ok := used[base]; !ok {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if _, ok := used[candidate]; !ok {
			return candidate
		}
	}
}

func slugify(s string) string {
	out := slugifyASCII(s)
	if out == "" {
		return "vm"
	}
	return out
}

func normalizeVMConfig(cfg VMConfig) VMConfig {
	switch cfg.Type {
	case "", legacyTypeCustom:
		cfg.Type = TypeQEMU
	}
	if cfg.DiskSizeGiB <= 0 {
		cfg.DiskSizeGiB = 40
	}
	cfg.FlakeApp = ""
	return cfg
}
