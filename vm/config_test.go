package vm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreAddNormalizesAndPersists(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Chdir(tempDir)

	store, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error = %v", err)
	}

	first, err := store.Add(VMConfig{
		Name:     "  My VM  ",
		MemMiB:   2048,
		VCPUs:    2,
		DiskFile: "./disk.qcow2",
		SSHPort:  2222,
		FlakeApp: "legacy.app",
	})
	if err != nil {
		t.Fatalf("store.Add(first) error = %v", err)
	}

	if first.ID != "my-vm" {
		t.Fatalf("first.ID = %q, want %q", first.ID, "my-vm")
	}
	if first.Type != TypeQEMU {
		t.Fatalf("first.Type = %q, want %q", first.Type, TypeQEMU)
	}
	if first.DiskSizeGiB != 40 {
		t.Fatalf("first.DiskSizeGiB = %d, want 40", first.DiskSizeGiB)
	}
	if first.FlakeApp != "" {
		t.Fatalf("first.FlakeApp = %q, want empty", first.FlakeApp)
	}
	if first.DiskFile != "./disk.qcow2" {
		t.Fatalf("first.DiskFile = %q, want %q", first.DiskFile, "./disk.qcow2")
	}

	second, err := store.Add(VMConfig{
		Name:     "My VM",
		MemMiB:   1024,
		VCPUs:    1,
		DiskFile: "./disk-2.qcow2",
		SSHPort:  2223,
	})
	if err != nil {
		t.Fatalf("store.Add(second) error = %v", err)
	}

	if second.ID != "my-vm-2" {
		t.Fatalf("second.ID = %q, want %q", second.ID, "my-vm-2")
	}

	reloaded, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() after save error = %v", err)
	}
	if got := len(reloaded.VMs); got != 2 {
		t.Fatalf("len(reloaded.VMs) = %d, want 2", got)
	}
	if reloaded.VMs[0].ID != "my-vm" || reloaded.VMs[1].ID != "my-vm-2" {
		t.Fatalf("reloaded IDs = %q, %q, want my-vm and my-vm-2", reloaded.VMs[0].ID, reloaded.VMs[1].ID)
	}
}

func TestStoreRejectsDuplicateDiskAcrossPathForms(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	store := &Store{path: filepath.Join(tempDir, "vms.json")}
	if _, err := store.Add(VMConfig{
		Name:     "alpha",
		MemMiB:   1024,
		VCPUs:    1,
		DiskFile: "./shared.qcow2",
		SSHPort:  2222,
	}); err != nil {
		t.Fatalf("store.Add(first) error = %v", err)
	}

	absDisk := filepath.Join(tempDir, "shared.qcow2")
	if _, err := store.Add(VMConfig{
		Name:     "beta",
		MemMiB:   1024,
		VCPUs:    1,
		DiskFile: absDisk,
		SSHPort:  2223,
	}); err == nil {
		t.Fatal("store.Add(second) error = nil, want duplicate disk error")
	}
}

func TestLoadStoreNormalizesLegacyConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	storePath := filepath.Join(tempDir, "vmtui", "vms.json")
	if err := os.MkdirAll(filepath.Dir(storePath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	raw := Store{
		VMs: []VMConfig{
			{
				ID:       "legacy",
				Name:     "Legacy VM",
				Type:     legacyTypeCustom,
				DiskFile: "/tmp/legacy.qcow2",
				SSHPort:  2222,
				FlakeApp: "deprecated.field",
			},
		},
	}
	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(storePath, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error = %v", err)
	}
	if got := len(store.VMs); got != 1 {
		t.Fatalf("len(store.VMs) = %d, want 1", got)
	}

	cfg := store.VMs[0]
	if cfg.Type != TypeQEMU {
		t.Fatalf("cfg.Type = %q, want %q", cfg.Type, TypeQEMU)
	}
	if cfg.DiskSizeGiB != 40 {
		t.Fatalf("cfg.DiskSizeGiB = %d, want 40", cfg.DiskSizeGiB)
	}
	if cfg.FlakeApp != "" {
		t.Fatalf("cfg.FlakeApp = %q, want empty", cfg.FlakeApp)
	}
}

func TestStoreUpdateRejectsDuplicateDiskAcrossVMs(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	store := &Store{path: filepath.Join(tempDir, "vms.json")}
	first, err := store.Add(VMConfig{
		Name:     "alpha",
		MemMiB:   1024,
		VCPUs:    1,
		DiskFile: "./alpha.qcow2",
		SSHPort:  2222,
	})
	if err != nil {
		t.Fatalf("store.Add(first) error = %v", err)
	}
	second, err := store.Add(VMConfig{
		Name:     "beta",
		MemMiB:   1024,
		VCPUs:    1,
		DiskFile: "./beta.qcow2",
		SSHPort:  2223,
	})
	if err != nil {
		t.Fatalf("store.Add(second) error = %v", err)
	}

	second.DiskFile = filepath.Join(tempDir, "alpha.qcow2")
	err = store.Update(second)
	if err == nil {
		t.Fatal("store.Update() error = nil, want duplicate disk error")
	}

	if got := store.VMs[0].DiskFile; got != first.DiskFile {
		t.Fatalf("store.VMs[0].DiskFile = %q, want %q", got, first.DiskFile)
	}
	if got := store.VMs[1].DiskFile; got != "./beta.qcow2" {
		t.Fatalf("store.VMs[1].DiskFile = %q, want %q", got, "./beta.qcow2")
	}
}
