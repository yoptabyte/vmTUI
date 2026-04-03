package tui

import (
	"path/filepath"
	"strings"
	"testing"

	"vmtui/vm"
)

func TestFormScreenEditSavesAndReturnsToList(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("XDG_CACHE_HOME", tempDir)

	store, err := vm.LoadStore()
	if err != nil {
		t.Fatalf("vm.LoadStore() error = %v", err)
	}
	pids := vm.LoadPIDStore()

	saved, err := store.Add(vm.VMConfig{
		Name:        "debian",
		MemMiB:      1024,
		VCPUs:       1,
		DiskFile:    filepath.Join(tempDir, "debian.qcow2"),
		DiskSizeGiB: 20,
		SSHPort:     2222,
		ISOPath:     filepath.Join(tempDir, "old.iso"),
	})
	if err != nil {
		t.Fatalf("store.Add() error = %v", err)
	}

	m := NewRootModel(store, pids)
	m.editingVMID = saved.ID
	m.form = FormModel{
		done: true,
		result: vm.VMConfig{
			ID:          saved.ID,
			Name:        "Debian Updated",
			Type:        vm.TypeQEMU,
			MemMiB:      2048,
			VCPUs:       2,
			DiskFile:    saved.DiskFile,
			DiskSizeGiB: saved.DiskSizeGiB,
			SSHPort:     2225,
			ISOPath:     filepath.Join(tempDir, "new.iso"),
		},
	}
	m.screen = FormScreen{}

	updated, cmd := FormScreen{}.Update(m, nil)

	if cmd != nil {
		t.Fatal("cmd != nil, want nil after save")
	}
	if _, ok := updated.screen.(ListScreen); !ok {
		t.Fatalf("updated.screen = %T, want ListScreen", updated.screen)
	}
	if updated.editingVMID != "" {
		t.Fatalf("updated.editingVMID = %q, want empty", updated.editingVMID)
	}
	if got := len(updated.store.VMs); got != 1 {
		t.Fatalf("len(updated.store.VMs) = %d, want 1", got)
	}

	got := updated.store.VMs[0]
	if got.Name != "Debian Updated" {
		t.Fatalf("got.Name = %q, want %q", got.Name, "Debian Updated")
	}
	if got.ISOPath != filepath.Join(tempDir, "new.iso") {
		t.Fatalf("got.ISOPath = %q, want updated ISO path", got.ISOPath)
	}
	if !strings.Contains(updated.list.msg, "updated") {
		t.Fatalf("updated.list.msg = %q, want update success message", updated.list.msg)
	}
	if updated.list.msgErr {
		t.Fatal("updated.list.msgErr = true, want false")
	}
}
