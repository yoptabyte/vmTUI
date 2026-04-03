package vm

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

// RunOptions carries optional flags for launching a VM.
type RunOptions struct {
	ExtraArgs []string
}

// Launch starts a VM through qemu-system-x86_64.
// The process is detached so the TUI can keep running.
func Launch(cfg VMConfig, opts RunOptions) (*exec.Cmd, error) {
	cfg = normalizeVMConfig(cfg)
	if cfg.Type != TypeQEMU {
		return nil, fmt.Errorf("unknown vm type: %s", cfg.Type)
	}
	diskPath := ResolvePath(cfg.DiskFile)
	if diskPath == "" {
		return nil, fmt.Errorf("vm %q has no disk file configured", cfg.ID)
	}
	if _, err := os.Stat(diskPath); err != nil {
		return nil, fmt.Errorf("disk image not found: %s", diskPath)
	}

	hostfwd := fmt.Sprintf("user,id=net0,hostfwd=tcp:127.0.0.1:%d-:22", cfg.SSHPort)

	args := []string{
		"-enable-kvm",
		"-name", cfg.Name,
		"-m", strconv.Itoa(cfg.MemMiB),
		"-smp", strconv.Itoa(cfg.VCPUs),
		"-cpu", "host",
		"-drive", "file=" + diskPath + ",format=qcow2,if=virtio,cache=writeback",
		"-netdev", hostfwd,
		"-device", "virtio-net-pci,netdev=net0",
		"-device", "virtio-vga",
		"-display", "gtk,gl=off",
		"-audiodev", "pa,id=audio0",
		"-device", "ich9-intel-hda",
		"-device", "hda-output,audiodev=audio0",
		"-boot", "order=c",
	}
	args = append(args, opts.ExtraArgs...)

	cmd := exec.Command("qemu-system-x86_64", args...)
	cmd.Env = os.Environ()
	cmd.Stdin = nil
	logFile, err := openLogFile(cfg.ID)
	if err != nil {
		return nil, err
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, err
	}
	_ = logFile.Close()
	return cmd, nil
}

// CreateDisk creates a qcow2 disk image via qemu-img.
func CreateDisk(path string, sizeGB int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	cmd := exec.Command("qemu-img", "create", "-f", "qcow2", path,
		strconv.Itoa(sizeGB)+"G")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ResizeDisk resizes an existing qcow2 disk image to the given size in GB.
func ResizeDisk(path string, sizeGB int) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("disk image not found: %s", path)
	}
	cmd := exec.Command("qemu-img", "resize", "-f", "qcow2", path,
		strconv.Itoa(sizeGB)+"G")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// InstallISO boots a VM from ISO for OS installation.
func InstallISO(cfg VMConfig, isoPath string) (*exec.Cmd, error) {
	cfg = normalizeVMConfig(cfg)
	if absISO, err := filepath.Abs(isoPath); err == nil {
		isoPath = absISO
	}

	if cfg.Type != TypeQEMU {
		return nil, fmt.Errorf("unknown vm type: %s", cfg.Type)
	}

	diskPath := ResolvePath(cfg.DiskFile)
	args := []string{
		"-enable-kvm",
		"-name", cfg.Name + "-installer",
		"-m", strconv.Itoa(cfg.MemMiB),
		"-smp", strconv.Itoa(cfg.VCPUs),
		"-cpu", "host",
		"-drive", "file=" + diskPath + ",format=qcow2,if=virtio",
		"-cdrom", isoPath,
		"-boot", "order=d",
		"-netdev", "user,id=net0",
		"-device", "virtio-net-pci,netdev=net0",
		"-device", "virtio-vga",
		"-display", "gtk,gl=off",
		"-audiodev", "pa,id=audio0",
		"-device", "ich9-intel-hda",
		"-device", "hda-output,audiodev=audio0",
	}

	cmd := exec.Command("qemu-system-x86_64", args...)
	cmd.Env = os.Environ()
	cmd.Stdin = nil
	logFile, err := openLogFile(cfg.ID)
	if err != nil {
		return nil, err
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, err
	}
	_ = logFile.Close()
	return cmd, nil
}

func openLogFile(id string) (*os.File, error) {
	logDir := LogDir()
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, err
	}

	return os.Create(LogPath(id))
}

func LogDir() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil || cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), "vmtui")
	}
	return filepath.Join(cacheDir, "vmtui", "logs")
}

func LogPath(id string) string {
	return filepath.Join(LogDir(), id+".log")
}

func StopPID(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid pid")
	}
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		return err
	}
	return nil
}
