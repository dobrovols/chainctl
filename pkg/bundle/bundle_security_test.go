package bundle

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// buildTar constructs a tar archive from a map of name->content.
func buildTar(entries map[string]string) []byte {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for name, content := range entries {
		hdr := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}
		_ = tw.WriteHeader(hdr)
		_, _ = io.WriteString(tw, content)
	}
	_ = tw.Close()
	return buf.Bytes()
}

func writeBundle(t *testing.T, path string, tarData []byte) {
	t.Helper()
	if err := os.WriteFile(path, tarData, 0o600); err != nil {
		t.Fatalf("write temp tar: %v", err)
	}
}

func TestLoadRejectsPathTraversalEntries(t *testing.T) {
	t.Parallel()

	// Create a tar with a manifest and a traversal attempt.
	entries := map[string]string{
		ManifestFileName: "checksums:\n  app/ok.txt: \nversion: v1\n",
		"app/ok.txt":     "ok",
		"../../evil":     "nope",
	}
	tarData := buildTar(entries)

	tmp := t.TempDir()
	tarPath := filepath.Join(tmp, "bundle.tar")
	writeBundle(t, tarPath, tarData)

	if _, err := Load(tarPath, tmp); err == nil {
		t.Fatal("expected error for path traversal entry, got nil")
	}
}

func TestSafeJoinPreventsEscape(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cases := []struct {
		name   string
		target string
	}{
		{
			name:   "absolute path",
			target: filepath.Join(root, "other"),
		},
		{
			name:   "parent only",
			target: "..",
		},
		{
			name:   "parent traversal",
			target: filepath.Join("..", "etc", "passwd"),
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := safeJoin(root, tc.target); err == nil {
				t.Fatalf("expected error for %q, got nil", tc.target)
			}
		})
	}
}

func TestSafeJoinAllowsRelative(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	got, err := safeJoin(root, filepath.Join("charts", "app.tgz"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(root, "charts", "app.tgz")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestLoadAcceptsNormalEntriesAndValidatesChecksums(t *testing.T) {
	t.Parallel()

	// Prepare a valid tar with one file and a checksum in the manifest.
	content := "hello"
	sum := sha256.Sum256([]byte(content))
	checksum := hex.EncodeToString(sum[:])
	manifest := "checksums:\n  files/hello.txt: " + checksum + "\nversion: v1\n"
	entries := map[string]string{
		ManifestFileName:  manifest,
		"files/hello.txt": content,
	}
	tarData := buildTar(entries)

	tmp := t.TempDir()
	tarPath := filepath.Join(tmp, "bundle.tar")
	writeBundle(t, tarPath, tarData)

	b, err := Load(tarPath, tmp)
	if err != nil {
		t.Fatalf("unexpected error loading bundle: %v", err)
	}
	if b.Extracted == "" || b.Manifest.Version != "v1" {
		t.Fatalf("unexpected bundle metadata: %+v", b.Manifest)
	}
}

type tarEntry struct {
	name     string
	mode     int64
	typeflag byte
	body     string
}

func buildTarEntries(entries []tarEntry) []byte {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, e := range entries {
		hdr := &tar.Header{
			Name:     e.name,
			Mode:     e.mode,
			Typeflag: e.typeflag,
		}
		if hdr.Typeflag != tar.TypeDir {
			hdr.Size = int64(len(e.body))
		}
		_ = tw.WriteHeader(hdr)
		if hdr.Typeflag != tar.TypeDir {
			_, _ = io.WriteString(tw, e.body)
		}
	}
	_ = tw.Close()
	return buf.Bytes()
}

func TestExtractTarToDirRequiresManifest(t *testing.T) {
	t.Parallel()

	entries := []tarEntry{
		{name: "files/hello.txt", mode: 0o644, typeflag: tar.TypeReg, body: "hello"},
	}
	data := buildTarEntries(entries)

	tmp := t.TempDir()
	if _, err := extractTarToDir(data, tmp); !errors.Is(err, ErrManifestMissing) {
		t.Fatalf("expected ErrManifestMissing, got %v", err)
	}
}

func TestExtractTarToDirCreatesDirectories(t *testing.T) {
	t.Parallel()

	entries := []tarEntry{
		{name: "charts/", mode: 0o755, typeflag: tar.TypeDir},
		{name: ManifestFileName, mode: 0o644, typeflag: tar.TypeReg, body: "version: v1\n"},
		{name: "charts/values.yaml", mode: 0o600, typeflag: tar.TypeReg, body: "foo: bar\n"},
	}
	data := buildTarEntries(entries)

	tmp := t.TempDir()
	m, err := extractTarToDir(data, tmp)
	if err != nil {
		t.Fatalf("unexpected error extracting tar: %v", err)
	}
	if m.Version != "v1" {
		t.Fatalf("unexpected manifest: %+v", m)
	}
	info, err := os.Stat(filepath.Join(tmp, "charts"))
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("charts path is not a directory")
	}
}

func TestValidateChecksumsDetectsMismatch(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	assetPath := filepath.Join(tmp, "files", "bin")
	if err := os.MkdirAll(filepath.Dir(assetPath), 0o755); err != nil {
		t.Fatalf("create dirs: %v", err)
	}
	if err := os.WriteFile(assetPath, []byte("contents"), 0o600); err != nil {
		t.Fatalf("write asset: %v", err)
	}
	checksums := map[string]string{"files/bin": "deadbeef"}
	if err := validateChecksums(tmp, checksums); !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("expected ErrChecksumMismatch, got %v", err)
	}
}
