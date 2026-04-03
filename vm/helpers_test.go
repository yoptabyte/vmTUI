package vm

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAria2Percent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		line    string
		wantPct int
		wantOK  bool
	}{
		{name: "progress line", line: "[#123 4.0MiB/100MiB(42%)]", wantPct: 42, wantOK: true},
		{name: "no progress marker", line: "download complete", wantPct: 0, wantOK: false},
		{name: "bad percent", line: "(abc%)", wantPct: 0, wantOK: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotPct, gotOK := parseAria2Percent(tc.line)
			if gotPct != tc.wantPct || gotOK != tc.wantOK {
				t.Fatalf("parseAria2Percent(%q) = (%d, %v), want (%d, %v)", tc.line, gotPct, gotOK, tc.wantPct, tc.wantOK)
			}
		})
	}
}

func TestLocalISOPathInDirUsesURLBaseAndFallback(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	entry := CatalogEntry{ID: "debian"}

	got := LocalISOPathInDir(dir, entry, ImageVariant{
		URL: "https://example.com/releases/debian-13.4.0-amd64-netinst.iso?mirror=1",
	})
	want := filepath.Join(dir, "debian-13.4.0-amd64-netinst.iso")
	if got != want {
		t.Fatalf("LocalISOPathInDir(url) = %q, want %q", got, want)
	}

	got = LocalISOPathInDir(dir, entry, ImageVariant{})
	want = filepath.Join(dir, "debian.iso")
	if got != want {
		t.Fatalf("LocalISOPathInDir(fallback) = %q, want %q", got, want)
	}
}

func TestVerifySHA256(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "sample.iso")
	content := []byte("vmtui test payload")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	sum := sha256.Sum256(content)
	expected := fmt.Sprintf("%x", sum[:])

	ok, err := verifySHA256(path, expected)
	if err != nil {
		t.Fatalf("verifySHA256() error = %v", err)
	}
	if !ok {
		t.Fatal("verifySHA256() = false, want true")
	}

	ok, err = verifySHA256(path, strings.Repeat("0", 64))
	if err != nil {
		t.Fatalf("verifySHA256() mismatch error = %v", err)
	}
	if ok {
		t.Fatal("verifySHA256() = true for wrong digest, want false")
	}
}

func TestIsLikelyStale(t *testing.T) {
	t.Parallel()

	if !isLikelyStale("ubuntu", "Tue, 21 Apr 2026 10:00:00 GMT") {
		t.Fatal("isLikelyStale() = false, want true for much newer Last-Modified")
	}
	if isLikelyStale("ubuntu", "Tue, 10 Feb 2026 10:00:00 GMT") {
		t.Fatal("isLikelyStale() = true, want false at release date")
	}
	if isLikelyStale("unknown", "Tue, 21 Apr 2026 10:00:00 GMT") {
		t.Fatal("isLikelyStale() = true, want false for unknown distro")
	}
}
