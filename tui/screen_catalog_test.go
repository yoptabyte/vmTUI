package tui

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"vmtui/vm"
)

func TestCatalogScreenUseISOTransitionsToInstallScreen(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("XDG_CACHE_HOME", tempDir)

	store, err := vm.LoadStore()
	if err != nil {
		t.Fatalf("vm.LoadStore() error = %v", err)
	}
	pids := vm.LoadPIDStore()

	saved, err := store.Add(vm.VMConfig{
		Name:     "alpine",
		MemMiB:   1024,
		VCPUs:    1,
		DiskFile: filepath.Join(tempDir, "alpine.qcow2"),
		SSHPort:  2222,
	})
	if err != nil {
		t.Fatalf("store.Add() error = %v", err)
	}

	m := NewRootModel(store, pids)
	m.catalog = NewCatalogModel()
	m.catalog.state = catalogDone
	m.catalog.ISOReady = true
	m.catalog.ISOPath = filepath.Join(tempDir, "alpine.iso")
	m.screen = CatalogScreen{}

	updated, _ := CatalogScreen{}.Update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	if _, ok := updated.screen.(InstallScreen); !ok {
		t.Fatalf("updated.screen = %T, want InstallScreen", updated.screen)
	}
	if updated.installTarget == nil {
		t.Fatal("updated.installTarget = nil, want selected VM")
	}
	if updated.installTarget.ID != saved.ID {
		t.Fatalf("updated.installTarget.ID = %q, want %q", updated.installTarget.ID, saved.ID)
	}
	if updated.isoInput != m.catalog.ISOPath {
		t.Fatalf("updated.isoInput = %q, want %q", updated.isoInput, m.catalog.ISOPath)
	}
}

func TestCatalogScreenRejectsNonISODownload(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("XDG_CACHE_HOME", tempDir)

	store, err := vm.LoadStore()
	if err != nil {
		t.Fatalf("vm.LoadStore() error = %v", err)
	}
	pids := vm.LoadPIDStore()

	if _, err := store.Add(vm.VMConfig{
		Name:     "netbsd",
		MemMiB:   1024,
		VCPUs:    1,
		DiskFile: filepath.Join(tempDir, "netbsd.qcow2"),
		SSHPort:  2222,
	}); err != nil {
		t.Fatalf("store.Add() error = %v", err)
	}

	m := NewRootModel(store, pids)
	m.catalog = NewCatalogModel()
	m.catalog.state = catalogDone
	m.catalog.ISOReady = true
	m.catalog.ISOPath = filepath.Join(tempDir, "NetBSD-10.1-amd64-install.img.gz")
	m.screen = CatalogScreen{}

	updated, _ := CatalogScreen{}.Update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	if _, ok := updated.screen.(ListScreen); !ok {
		t.Fatalf("updated.screen = %T, want ListScreen", updated.screen)
	}
	if updated.installTarget != nil {
		t.Fatal("updated.installTarget != nil, want nil for non-ISO file")
	}
	if !updated.list.msgErr {
		t.Fatal("updated.list.msgErr = false, want true")
	}
	if !strings.Contains(updated.list.msg, "not an ISO") {
		t.Fatalf("updated.list.msg = %q, want non-ISO error", updated.list.msg)
	}
}
