package vm

import (
	"net/url"
	"os"
	"path"
	"path/filepath"
)

// ImageVariant is a specific downloadable ISO within a distro family.
type ImageVariant struct {
	Name           string // e.g. "Installer (amd64)"
	URL            string // direct ISO download URL; empty for instruction-only entries
	SHA256         string // expected hex digest; empty = skip verification
	SizeMiB        int    // approximate, for display
	InstructionURL string // URL to a page with download instructions (e.g. Windows)
}

// CatalogEntry is one distro in the catalog list.
type CatalogEntry struct {
	ID       string         // slug used as disk/vm name
	Distro   string         // display name
	Desc     string         // one-liner
	Variants []ImageVariant // at least one
}

// Catalog returns the built-in list of downloadable OS images.
// URLs point to official mirrors; SHA256 matches the current stable release.
// Update these when new releases ship.
func Catalog() []CatalogEntry {
	return []CatalogEntry{
		{
			ID:     "alpine",
			Distro: "Alpine Linux",
			Desc:   "Security-oriented, lightweight distro (musl + busybox)",
			Variants: []ImageVariant{
				{
					Name:    "Standard 3.23.3 x86_64",
					URL:     "https://dl-cdn.alpinelinux.org/alpine/v3.23/releases/x86_64/alpine-standard-3.23.3-x86_64.iso",
					SHA256:  "966d6bf4d4c79958d43abde84a3e5bbeb4f8c757c164a49d3ec8432be6d36f16",
					SizeMiB: 200,
				},
				{
					Name:    "Extended 3.23.3 x86_64",
					URL:     "https://dl-cdn.alpinelinux.org/alpine/v3.23/releases/x86_64/alpine-extended-3.23.3-x86_64.iso",
					SHA256:  "",
					SizeMiB: 700,
				},
			},
		},
		{
			ID:     "arch",
			Distro: "Arch Linux",
			Desc:   "Lightweight, rolling release",
			Variants: []ImageVariant{
				{
					Name:    "Arch Linux 2026.03.01 x86_64",
					URL:     "https://geo.mirror.pkgbuild.com/iso/latest/archlinux-2026.03.01-x86_64.iso",
					SHA256:  "569f7331bbcb882d130035324ab5feb1cd9807ccc9a49aa61102d40121518db6",
					SizeMiB: 1500,
				},
			},
		},
		{
			ID:     "debian-hurd",
			Distro: "Debian GNU/Hurd",
			Desc:   "Debian on GNU/Hurd — experimental microkernel variant",
			Variants: []ImageVariant{
				{
					Name:    "NETINST ISO amd64 (2025)",
					URL:     "https://cdimage.debian.org/cdimage/ports/13.0/hurd-amd64/iso-cd/debian-hurd-2025-amd64-NETINST-1.iso",
					SHA256:  "9871d70bdc1e71b5571bb534a63280d69cbc72c0ac80da101ed454f122860bbe",
					SizeMiB: 305,
				},
				{
					Name:    "Pre-installed disk image amd64 (20250807)",
					URL:     "https://cdimage.debian.org/cdimage/ports/13.0/hurd-amd64/debian-hurd-amd64.img.gz",
					SHA256:  "7df64bd90d364029752347dece0799996ef67f257d19d1fe8f325cccadd3255c",
					SizeMiB: 482,
				},
			},
		},
		{
			ID:     "debian",
			Distro: "Debian",
			Desc:   "Stable, universal operating system",
			Variants: []ImageVariant{
				{
					Name:    "Debian 13.4 netinst amd64",
					URL:     "https://cdimage.debian.org/debian-cd/current/amd64/iso-cd/debian-13.4.0-amd64-netinst.iso",
					SHA256:  "",
					SizeMiB: 660,
				},
				{
					Name:    "Debian 13.4 DVD amd64",
					URL:     "https://cdimage.debian.org/debian-cd/current/amd64/iso-dvd/debian-13.4.0-amd64-DVD-1.iso",
					SHA256:  "",
					SizeMiB: 3700,
				},
			},
		},
		{
			ID:     "fedora",
			Distro: "Fedora",
			Desc:   "Cutting-edge RPM Linux, upstream for RHEL",
			Variants: []ImageVariant{
				{
					Name:    "Fedora 43 Workstation",
					URL:     "https://download.fedoraproject.org/pub/fedora/linux/releases/43/Workstation/x86_64/iso/Fedora-Workstation-Live-43-1.6.x86_64.iso",
					SHA256:  "2a4a16c009244eb5ab2198700eb04103793b62407e8596f30a3e0cc8ac294d77",
					SizeMiB: 2600,
				},
				{
					Name:    "Fedora 43 Server netinst",
					URL:     "https://download.fedoraproject.org/pub/fedora/linux/releases/43/Server/x86_64/iso/Fedora-Server-netinst-x86_64-43-1.6.iso",
					SHA256:  "",
					SizeMiB: 1100,
				},
			},
		},
		{
			ID:     "freebsd",
			Distro: "FreeBSD",
			Desc:   "Production-grade Unix descendant, ZFS & jails",
			Variants: []ImageVariant{
				{
					Name:    "FreeBSD 14.4-RELEASE bootonly amd64",
					URL:     "https://download.freebsd.org/releases/amd64/amd64/ISO-IMAGES/14.4/FreeBSD-14.4-RELEASE-amd64-bootonly.iso",
					SHA256:  "5a262316dd17badd83dba6355a259341479a5133cb29d54b2585cf20d46292c7",
					SizeMiB: 558,
				},
				{
					Name:    "FreeBSD 14.4-RELEASE DVD amd64",
					URL:     "https://download.freebsd.org/releases/amd64/amd64/ISO-IMAGES/14.4/FreeBSD-14.4-RELEASE-amd64-dvd1.iso",
					SHA256:  "953814f2a4dcfc32958902f3b1f30cf34b5ab872f405668793491481abfd177e",
					SizeMiB: 4093,
				},
			},
		},
		{
			ID:     "guix",
			Distro: "GNU Guix System",
			Desc:   "Functional package manager & declarative OS",
			Variants: []ImageVariant{
				{
					Name:    "Graphical Installer (1.5.0)",
					URL:     "https://ftp.gnu.org/gnu/guix/guix-system-install-1.5.0.x86_64-linux.iso",
					SHA256:  "",
					SizeMiB: 1100,
				},
			},
		},
		{
			ID:     "kali",
			Distro: "Kali Linux",
			Desc:   "Penetration testing & security research",
			Variants: []ImageVariant{
				{
					Name:    "Installer amd64 (2026.1)",
					URL:     "https://cdimage.kali.org/current/kali-linux-2026.1-installer-amd64.iso",
					SHA256:  "271477ad6ea2676c7346576971b9acc2d32fabd9c2bbaf0e6302397626149306",
					SizeMiB: 4400,
				},
				{
					Name:    "NetInstaller amd64 (2026.1)",
					URL:     "https://cdimage.kali.org/current/kali-linux-2026.1-installer-netinst-amd64.iso",
					SHA256:  "caf5ff7d7a4f73c85a6f1688300b936d3d7fd6965c52d80632e36709a09255a7",
					SizeMiB: 712,
				},
			},
		},
		{
			ID:     "netbsd",
			Distro: "NetBSD",
			Desc:   "Highly portable Unix-like OS, runs on everything",
			Variants: []ImageVariant{
				{
					Name:    "NetBSD 10.1 install amd64",
					URL:     "https://cdn.netbsd.org/pub/NetBSD/NetBSD-10.1/images/NetBSD-10.1-amd64-install.img.gz",
					SHA256:  "",
					SizeMiB: 400,
				},
			},
		},
		{
			ID:     "nixos",
			Distro: "NixOS",
			Desc:   "Declarative Linux, consistent with your flake host",
			Variants: []ImageVariant{
				{
					Name:    "NixOS 25.11 graphical amd64",
					URL:     "https://channels.nixos.org/nixos-25.11/latest-nixos-graphical-x86_64-linux.iso",
					SHA256:  "a5f8bb5c128feb3d0cadbad19bd0bfbdafc8fe69ca49a4e48c22ea07e36c8a45",
					SizeMiB: 3000,
				},
				{
					Name:    "NixOS 25.11 minimal amd64",
					URL:     "https://channels.nixos.org/nixos-25.11/latest-nixos-minimal-x86_64-linux.iso",
					SHA256:  "",
					SizeMiB: 1100,
				},
			},
		},
		{
			ID:     "openbsd",
			Distro: "OpenBSD",
			Desc:   "Security-first Unix, proactive exploit mitigation",
			Variants: []ImageVariant{
				{
					Name:    "OpenBSD 7.8 install amd64",
					URL:     "https://cdn.openbsd.org/pub/OpenBSD/7.8/amd64/install78.iso",
					SHA256:  "a228d0a1ef558b4d9ec84c698f0d3ffd13cd38c64149487cba0f1ad873be07b2",
					SizeMiB: 774,
				},
			},
		},
		{
			ID:     "ubuntu",
			Distro: "Ubuntu",
			Desc:   "General-purpose Linux, LTS releases",
			Variants: []ImageVariant{
				{
					Name:    "Ubuntu 24.04.4 LTS Desktop amd64",
					URL:     "https://releases.ubuntu.com/24.04/ubuntu-24.04.4-desktop-amd64.iso",
					SHA256:  "3a4c9877b483ab46d7c3fbe165a0db275e1ae3cfe56a5657e5a47c2f99a99d1e",
					SizeMiB: 6200,
				},
				{
					Name:    "Ubuntu 24.04.4 LTS Server amd64",
					URL:     "https://releases.ubuntu.com/24.04/ubuntu-24.04.4-live-server-amd64.iso",
					SHA256:  "e907d92eeec9df64163a7e454cbc8d7755e8ddc7ed42f99dbc80c40f1a138433",
					SizeMiB: 3200,
				},
				{
					Name:    "Ubuntu 22.04.5 LTS Desktop amd64",
					URL:     "https://releases.ubuntu.com/22.04/ubuntu-22.04.5-desktop-amd64.iso",
					SHA256:  "bfd1cee02bc4f35db939e69b934ba49a39a378797ce9aee20f6e3e3e728fefbf",
					SizeMiB: 4400,
				},
				{
					Name:    "Ubuntu 22.04.5 LTS Server amd64",
					URL:     "https://releases.ubuntu.com/22.04/ubuntu-22.04.5-live-server-amd64.iso",
					SHA256:  "9bc6028870aef3f74f4e16b900008179e78b130e6b0b9a140635434a46aa98b0",
					SizeMiB: 2000,
				},
				{
					Name:    "Ubuntu 26.04 LTS Beta Desktop amd64",
					URL:     "https://releases.ubuntu.com/26.04/ubuntu-26.04-beta-desktop-amd64.iso",
					SHA256:  "85ebc356ca56c272285aa5a88c79c1f9c384efbef4da2fbabdf6cc4e669b38ab",
					SizeMiB: 6500,
				},
				{
					Name:    "Ubuntu 26.04 LTS Beta Server amd64",
					URL:     "https://releases.ubuntu.com/26.04/ubuntu-26.04-beta-live-server-amd64.iso",
					SHA256:  "ec11c403e5ee44952f23f21ae3db51c8df15269af68c54ec0e1d4a5991633640",
					SizeMiB: 2300,
				},
			},
		},
		{
			ID:     "windows",
			Distro: "Windows 11",
			Desc:   "Microsoft Windows — download ISO manually via link below",
			Variants: []ImageVariant{
				{
					Name:           "Windows 11 (download via Microsoft)",
					URL:            "",
					SHA256:         "",
					SizeMiB:        5500,
					InstructionURL: "https://www.microsoft.com/software-download/windows11",
				},
			},
		},
	}
}

// ISODir returns the directory where downloaded ISOs are stored.
// Defaults to ~/.cache/vmctl/iso/
func ISODir() string {
	dir, err := os.UserCacheDir()
	if err != nil || dir == "" {
		return filepath.Join(".", ".cache", "vmctl", "iso")
	}
	return filepath.Join(dir, "vmctl", "iso")
}

// LocalISOPath returns the expected local path for a downloaded variant.
func LocalISOPath(entry CatalogEntry, v ImageVariant) string {
	return LocalISOPathInDir(ISODir(), entry, v)
}

func LocalISOPathInDir(dir string, entry CatalogEntry, v ImageVariant) string {
	if v.URL == "" {
		return filepath.Join(dir, entry.ID+".iso")
	}
	u, err := url.Parse(v.URL)
	if err == nil {
		if base := path.Base(u.Path); base != "" && base != "." && base != "/" {
			return filepath.Join(dir, base)
		}
	}
	return filepath.Join(dir, entry.ID+".iso")
}
