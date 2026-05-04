package zipx

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeZip(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	defer f.Close()
	w := zip.NewWriter(f)
	defer w.Close()
	for name, body := range entries {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatalf("create entry %q: %v", name, err)
		}
		if _, err := fw.Write([]byte(body)); err != nil {
			t.Fatalf("write entry %q: %v", name, err)
		}
	}
}

func TestExtract_Flat(t *testing.T) {
	tmp := t.TempDir()
	zipPath := filepath.Join(tmp, "src.zip")
	writeZip(t, zipPath, map[string]string{
		"hello.txt": "world",
		"data.bin":  "\x00\x01\x02",
	})

	dest := filepath.Join(tmp, "out")
	if err := Extract(zipPath, dest); err != nil {
		t.Fatalf("Extract: %v", err)
	}

	for name, want := range map[string]string{
		"hello.txt": "world",
		"data.bin":  "\x00\x01\x02",
	} {
		got, err := os.ReadFile(filepath.Join(dest, name))
		if err != nil {
			t.Errorf("read %s: %v", name, err)
			continue
		}
		if string(got) != want {
			t.Errorf("%s: got %q want %q", name, got, want)
		}
	}
}

func TestExtract_NestedDirs(t *testing.T) {
	tmp := t.TempDir()
	zipPath := filepath.Join(tmp, "src.zip")
	writeZip(t, zipPath, map[string]string{
		"a/b/c/leaf.txt": "deep",
		"a/sibling.txt":  "shallow",
	})

	dest := filepath.Join(tmp, "out")
	if err := Extract(zipPath, dest); err != nil {
		t.Fatalf("Extract: %v", err)
	}

	leaf, err := os.ReadFile(filepath.Join(dest, "a", "b", "c", "leaf.txt"))
	if err != nil {
		t.Fatalf("read leaf: %v", err)
	}
	if string(leaf) != "deep" {
		t.Errorf("leaf: got %q want %q", leaf, "deep")
	}

	sibling, err := os.ReadFile(filepath.Join(dest, "a", "sibling.txt"))
	if err != nil {
		t.Fatalf("read sibling: %v", err)
	}
	if string(sibling) != "shallow" {
		t.Errorf("sibling: got %q want %q", sibling, "shallow")
	}
}

func TestExtract_CreatesDestination(t *testing.T) {
	tmp := t.TempDir()
	zipPath := filepath.Join(tmp, "src.zip")
	writeZip(t, zipPath, map[string]string{"x.txt": "y"})

	dest := filepath.Join(tmp, "missing", "deeper", "still")
	if err := Extract(zipPath, dest); err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "x.txt")); err != nil {
		t.Fatalf("expected file in %s: %v", dest, err)
	}
}

// TestExtract_ZipSlip writes an archive whose entry name resolves outside the
// destination via ../ and asserts that Extract refuses. The rejection must
// happen before any bytes are written for that entry.
func TestExtract_ZipSlip(t *testing.T) {
	tmp := t.TempDir()
	zipPath := filepath.Join(tmp, "evil.zip")
	writeZip(t, zipPath, map[string]string{
		"../escape.txt": "nope",
	})

	dest := filepath.Join(tmp, "out")
	err := Extract(zipPath, dest)
	if !errors.Is(err, ErrZipSlip) {
		t.Fatalf("expected ErrZipSlip, got %v", err)
	}
	// The escape file must not have been written above the destination.
	if _, err := os.Stat(filepath.Join(tmp, "escape.txt")); !os.IsNotExist(err) {
		t.Fatalf("zip-slip wrote outside destination; stat err = %v", err)
	}
}

func TestExtract_BadZip(t *testing.T) {
	tmp := t.TempDir()
	bogus := filepath.Join(tmp, "not-a-zip.zip")
	if err := os.WriteFile(bogus, []byte("this is not a zip"), 0o644); err != nil {
		t.Fatalf("write bogus: %v", err)
	}
	if err := Extract(bogus, filepath.Join(tmp, "out")); err == nil {
		t.Fatal("expected error opening bogus zip, got nil")
	}
}

func TestExtract_MissingZip(t *testing.T) {
	tmp := t.TempDir()
	if err := Extract(filepath.Join(tmp, "does-not-exist.zip"), filepath.Join(tmp, "out")); err == nil {
		t.Fatal("expected error for missing zip, got nil")
	}
}
