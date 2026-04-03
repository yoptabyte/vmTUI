package vm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// PIDStore persists PIDs of launched VMs so status survives TUI restarts.
type PIDStore struct {
	path string
	pids map[string]int // vm.ID -> PID
}

func pidFile() string {
	dir, _ := os.UserCacheDir()
	return filepath.Join(dir, "vmtui", "pids.json")
}

func LoadPIDStore() *PIDStore {
	s := &PIDStore{
		path: pidFile(),
		pids: make(map[string]int),
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		return s
	}
	_ = json.Unmarshal(data, &s.pids)
	return s
}

func (s *PIDStore) save() {
	_ = os.MkdirAll(filepath.Dir(s.path), 0o700)
	data, _ := json.Marshal(s.pids)
	_ = os.WriteFile(s.path, data, 0o600)
}

// Set records a PID for a VM and persists.
func (s *PIDStore) Set(id string, pid int) {
	s.pids[id] = pid
	s.save()
}

// Clear removes the PID entry for a VM.
func (s *PIDStore) Clear(id string) {
	delete(s.pids, id)
	s.save()
}

// IsRunning checks if the VM's recorded process is still alive.
func (s *PIDStore) IsRunning(id string) bool {
	pid, ok := s.pids[id]
	if !ok || pid <= 0 {
		return false
	}
	return processAlive(pid)
}

// PID returns the stored PID, 0 if not set.
func (s *PIDStore) PID(id string) int {
	return s.pids[id]
}

// StatusAll returns a snapshot map of id -> running for all known VMs.
func (s *PIDStore) StatusAll(ids []string) map[string]VMStatus {
	result := make(map[string]VMStatus, len(ids))
	changed := false
	for _, id := range ids {
		pid := s.pids[id]
		alive := pid > 0 && processAlive(pid)
		if !alive && pid > 0 {
			// stale — clean up
			delete(s.pids, id)
			changed = true
		}
		result[id] = VMStatus{Running: alive, PID: pid}
	}
	if changed {
		s.save()
	}
	return result
}

// VMStatus holds runtime state for one VM.
type VMStatus struct {
	Running bool
	PID     int
}

// String formats the status for display.
func (vs VMStatus) String() string {
	if vs.Running {
		return fmt.Sprintf("running (pid %d)", vs.PID)
	}
	return "stopped"
}

// processAlive returns true if the PID exists and is not a zombie.
func processAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0: no signal sent, just check existence.
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}
