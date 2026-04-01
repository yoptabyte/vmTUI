package vm

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// CheckVersion performs a lightweight HTTP HEAD against the ISO URL to
// detect whether the remote file has changed since our catalog was last
// updated. It returns a warning string when the remote Content-Length
// differs significantly from the catalog SizeMiB, or when the
// Last-Modified header is more recent than a cutoff we embed.
//
// For rolling distros (Arch) and instruction-only entries (Windows) it
// returns an empty string.
func CheckVersion(entry CatalogEntry, variant ImageVariant) string {
	if variant.URL == "" {
		return ""
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequest("HEAD", variant.URL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "vmctl/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Sprintf("remote returned HTTP %d — URL may be broken", resp.StatusCode)
	}

	var warnings []string

	contentLen := resp.ContentLength
	if contentLen > 0 && variant.SizeMiB > 0 {
		expectedBytes := int64(variant.SizeMiB) * 1024 * 1024
		diff := contentLen - expectedBytes
		if diff < 0 {
			diff = -diff
		}
		pct := float64(diff) / float64(expectedBytes) * 100
		if pct > 15 {
			actualMiB := contentLen / 1024 / 1024
			warnings = append(warnings, fmt.Sprintf("size mismatch: catalog says ~%d MiB, remote is %d MiB (newer version may exist)", variant.SizeMiB, actualMiB))
		}
	}

	if lm := resp.Header.Get("Last-Modified"); lm != "" {
		if isLikelyStale(entry.ID, lm) {
			warnings = append(warnings, fmt.Sprintf("remote Last-Modified: %s — check upstream for newer release", lm))
		}
	}

	if len(warnings) == 0 {
		return ""
	}
	return strings.Join(warnings, "; ")
}

// knownReleaseDates maps distro IDs to the date the catalog entry was
// last verified. If the remote Last-Modified is significantly newer we
// warn the user.
var knownReleaseDates = map[string]string{
	"alpine":      "2026-01-27",
	"arch":        "2026-03-01",
	"debian":      "2026-03-14",
	"debian-hurd": "2025-08-07",
	"fedora":      "2025-10-23",
	"freebsd":     "2026-03-06",
	"guix":        "2026-01-22",
	"kali":        "2026-01-28",
	"netbsd":      "2024-12-17",
	"nixos":       "2025-11-30",
	"openbsd":     "2025-10-12",
	"ubuntu":      "2026-02-10",
}

func isLikelyStale(distroID, lastModified string) bool {
	remoteTime, err := http.ParseTime(lastModified)
	if err != nil {
		return false
	}

	cutoffStr, ok := knownReleaseDates[distroID]
	if !ok {
		return false
	}
	cutoff, err := time.Parse("2006-01-02", cutoffStr)
	if err != nil {
		return false
	}

	return remoteTime.After(cutoff.Add(60 * 24 * time.Hour))
}

// IsInstructionOnly returns true when the variant has no direct ISO URL
// and instead points the user to a download page.
func IsInstructionOnly(v ImageVariant) bool {
	return v.URL == "" && v.InstructionURL != ""
}
