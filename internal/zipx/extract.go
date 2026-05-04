// Package zipx provides a small stdlib-only zip extraction helper that
// preserves directory structure and refuses zip-slip paths.
//
// It exists so gismanager can drop the deprecated github.com/mholt/archiver
// dependency. The only thing the rest of the codebase needs is "extract this
// zipped shapefile bundle into a temp directory".
package zipx

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ErrZipSlip indicates that a zip entry's name resolved outside the
// destination directory after path cleaning. Refusing to extract is the
// correct behavior; an attacker-controlled archive must not be able to
// write through the host filesystem (CVE-2018-1002200).
var ErrZipSlip = errors.New("zipx: archive entry escapes destination directory")

// Extract opens the zip file at zipPath and writes its contents into destDir.
// destDir is created (with parents) if it does not exist. Any entry whose
// resolved path escapes destDir produces [ErrZipSlip] and aborts the
// extraction without leaving partial files for that entry.
//
// File modes from the archive are preserved (subject to the running process's
// umask). Symlinks inside the archive are not followed and are extracted as
// regular files containing the link target — same posture as the upstream
// shapefile/GeoPackage tooling that produced these archives.
//
// Per-entry decompressed size is capped at [MaxBytesPerEntry] to refuse
// zip-bomb-style archives (gosec G110).
func Extract(zipPath, destDir string) (retErr error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("zipx: open %q: %w", zipPath, err)
	}
	defer func() {
		if cerr := zr.Close(); cerr != nil && retErr == nil {
			retErr = fmt.Errorf("zipx: close reader: %w", cerr)
		}
	}()

	absDest, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("zipx: resolve destination %q: %w", destDir, err)
	}
	if err := os.MkdirAll(absDest, 0o750); err != nil {
		return fmt.Errorf("zipx: create destination %q: %w", absDest, err)
	}

	for _, f := range zr.File {
		if err := extractEntry(f, absDest); err != nil {
			return err
		}
	}
	return nil
}

// MaxBytesPerEntry caps the decompressed size of any single zip entry. Set
// generously enough that legitimate shapefile and GeoPackage bundles fit
// (geospatial vector datasets routinely run into hundreds of MB), but small
// enough that a maliciously-crafted highly-compressed entry can't exhaust
// memory or disk.
const MaxBytesPerEntry int64 = 2 << 30 // 2 GiB

func extractEntry(f *zip.File, absDest string) error {
	// G305 (file traversal): the joined path is validated against absDest below
	// before any filesystem I/O; zip-slip names produce ErrZipSlip, not writes.
	target := filepath.Join(absDest, f.Name) //nolint:gosec // G305: validated against absDest before use.
	cleanTarget, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("zipx: resolve %q: %w", f.Name, err)
	}

	// Reject any path that does not stay under absDest. Compare with a
	// trailing separator so /foo/bar matches but /foo/barbaz does not.
	prefix := absDest + string(filepath.Separator)
	if cleanTarget != absDest && !strings.HasPrefix(cleanTarget, prefix) {
		return fmt.Errorf("%w: %q -> %q", ErrZipSlip, f.Name, cleanTarget)
	}

	if f.FileInfo().IsDir() {
		return os.MkdirAll(cleanTarget, f.Mode())
	}

	if err := os.MkdirAll(filepath.Dir(cleanTarget), 0o750); err != nil {
		return fmt.Errorf("zipx: create parent for %q: %w", f.Name, err)
	}

	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("zipx: open entry %q: %w", f.Name, err)
	}
	defer func() { _ = rc.Close() }()

	// The destination path was validated above (absolute, under absDest); the
	// gosec G304 / G305 false-positives don't apply to the OpenFile call.
	out, err := os.OpenFile(cleanTarget, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode()) //nolint:gosec // path validated against absDest above.
	if err != nil {
		return fmt.Errorf("zipx: create %q: %w", cleanTarget, err)
	}
	// Bound the decompressed size to refuse zip-bombs (gosec G110). CopyN
	// stops at the limit; we then verify the entry is actually exhausted by
	// reading one more byte. If the source still has data, the entry exceeds
	// the cap and we abort.
	written, err := io.CopyN(out, rc, MaxBytesPerEntry)
	if err != nil && !errors.Is(err, io.EOF) {
		_ = out.Close()
		return fmt.Errorf("zipx: write %q: %w", cleanTarget, err)
	}
	if written == MaxBytesPerEntry {
		var probe [1]byte
		n, _ := rc.Read(probe[:])
		if n > 0 {
			_ = out.Close()
			return fmt.Errorf("zipx: entry %q exceeds %d-byte cap", f.Name, MaxBytesPerEntry)
		}
	}
	return out.Close()
}
