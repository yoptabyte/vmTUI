package vm

import (
	"os"
	"path/filepath"
	"strings"
)

// ExpandPath resolves "~" to the current user's home directory and cleans the path.
func ExpandPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	if path == "~" || strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			if path == "~" {
				path = home
			} else {
				path = filepath.Join(home, path[2:])
			}
		}
	}

	return filepath.Clean(path)
}

// ResolvePath expands "~" and converts relative paths to absolute paths.
func ResolvePath(path string) string {
	path = ExpandPath(path)
	if path == "" {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}

// DiskDir returns the default directory for VM qcow2 images.
func DiskDir() string {
	dataDir, err := os.UserHomeDir()
	if err != nil || dataDir == "" {
		return filepath.Join(".", ".local", "share", "vmtui", "disks")
	}
	return filepath.Join(dataDir, ".local", "share", "vmtui", "disks")
}

// DefaultDiskPath returns a default qcow2 path for a VM name.
func DefaultDiskPath(name string) string {
	slug := slugifyASCII(name)
	if slug == "" {
		slug = "vm"
	}
	return filepath.Join(DiskDir(), slug+".qcow2")
}

func slugifyASCII(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		case !prevDash:
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
