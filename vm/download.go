package vm

import (
	"bufio"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

const (
	aria2Connections = "16"
	aria2MinSplit    = "4M"
)

type DownloadProgress struct {
	Percent int
	Detail  string
}

var aria2PercentRE = regexp.MustCompile(`\((\d{1,3})%\)`)

// DownloadISO fetches an ISO with aria2c. It is a convenience wrapper
// around DownloadISOWithCancel that does not support cancellation.
func DownloadISO(
	entry CatalogEntry,
	variant ImageVariant,
	destDir string,
	onProgress func(DownloadProgress),
) (string, error) {
	return DownloadISOWithCancel(entry, variant, destDir, onProgress, nil)
}

// DownloadISOWithCancel fetches an ISO with aria2c and supports
// cancellation via the cancelCh channel. Close cancelCh to abort the
// download; the underlying aria2c process will be killed.
func DownloadISOWithCancel(
	entry CatalogEntry,
	variant ImageVariant,
	destDir string,
	onProgress func(DownloadProgress),
	cancelCh <-chan struct{},
) (string, error) {
	if _, err := exec.LookPath("aria2c"); err != nil {
		return "", fmt.Errorf("aria2c not found in PATH")
	}

	destDir = ExpandPath(destDir)
	if destDir == "" {
		destDir = ISODir()
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}

	destPath := LocalISOPathInDir(destDir, entry, variant)
	if variant.SHA256 != "" {
		if ok, err := verifySHA256(destPath, variant.SHA256); err == nil && ok {
			if onProgress != nil {
				onProgress(DownloadProgress{Percent: 100, Detail: "cached ISO verified"})
			}
			return destPath, nil
		}
		if _, err := os.Stat(destPath); err == nil {
			if err := os.Remove(destPath); err != nil {
				return "", fmt.Errorf("cached ISO checksum mismatch and could not be removed: %w", err)
			}
		}
	}

	args := []string{
		"--continue=true",
		"--allow-overwrite=true",
		"--auto-file-renaming=false",
		"--max-connection-per-server=" + aria2Connections,
		"--split=" + aria2Connections,
		"--min-split-size=" + aria2MinSplit,
		"--file-allocation=none",
		"--summary-interval=1",
		"--download-result=default",
		"--console-log-level=notice",
		"--max-tries=5",
		"--retry-wait=3",
		"--dir", destDir,
		"--out", filepath.Base(destPath),
		variant.URL,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if cancelCh != nil {
		go func() {
			<-cancelCh
			cancel()
		}()
	}

	cmd := exec.CommandContext(ctx, "aria2c", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	if err := cmd.Start(); err != nil {
		return "", err
	}

	var (
		outputMu sync.Mutex
		output   []string
		wg       sync.WaitGroup
	)

	recordLine := func(line string) {
		line = strings.TrimSpace(line)
		if line == "" {
			return
		}
		outputMu.Lock()
		output = append(output, line)
		if len(output) > 20 {
			output = output[len(output)-20:]
		}
		outputMu.Unlock()

		if onProgress == nil {
			return
		}
		if pct, ok := parseAria2Percent(line); ok {
			onProgress(DownloadProgress{Percent: pct, Detail: line})
			return
		}
		if strings.Contains(strings.ToLower(line), "download complete") {
			onProgress(DownloadProgress{Percent: 100, Detail: line})
		}
	}

	consume := func(r io.Reader) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)
		for scanner.Scan() {
			recordLine(scanner.Text())
		}
	}

	wg.Add(2)
	go consume(stdout)
	go consume(stderr)

	err = cmd.Wait()
	wg.Wait()

	if ctx.Err() != nil {
		if onProgress != nil {
			onProgress(DownloadProgress{Percent: 0, Detail: "download cancelled"})
		}
		return "", fmt.Errorf("download cancelled")
	}

	if err != nil {
		outputMu.Lock()
		msg := strings.TrimSpace(strings.Join(output, "\n"))
		outputMu.Unlock()
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("aria2c failed: %s", msg)
	}

	if variant.SHA256 != "" {
		ok, err := verifySHA256(destPath, variant.SHA256)
		if err != nil {
			return "", err
		}
		if !ok {
			return "", fmt.Errorf("sha256 mismatch for %s", filepath.Base(destPath))
		}
	}

	if onProgress != nil {
		onProgress(DownloadProgress{Percent: 100, Detail: "download complete"})
	}

	return destPath, nil
}

func parseAria2Percent(line string) (int, bool) {
	m := aria2PercentRE.FindStringSubmatch(line)
	if len(m) != 2 {
		return 0, false
	}
	pct, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, false
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return pct, true
}

func verifySHA256(path string, expected string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer f.Close()

	sum := sha256.New()
	if _, err := io.Copy(sum, f); err != nil {
		return false, err
	}

	got := fmt.Sprintf("%x", sum.Sum(nil))
	return strings.EqualFold(got, expected), nil
}
