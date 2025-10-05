package unit

import (
	"archive/tar"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/dobrovols/chainctl/pkg/bundle"
)

func TestLoadBundleSuccess(t *testing.T) {
	tempDir := t.TempDir()
	bundlePath := filepath.Join(tempDir, "bundle.tar")
	cacheDir := filepath.Join(tempDir, "cache")

	payload := map[string][]byte{
		"charts/app.tgz": []byte("dummy chart"),
	}

	manifest := bundle.Manifest{
		Version:   "1.0.0",
		Checksums: map[string]string{},
	}
	for name, data := range payload {
		sum := sha256.Sum256(data)
		manifest.Checksums[name] = hex.EncodeToString(sum[:])
	}

	createBundle(t, bundlePath, manifest, payload)

	result, err := bundle.Load(bundlePath, cacheDir)
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}

	if result.Manifest.Version != "1.0.0" {
		t.Fatalf("expected version 1.0.0, got %s", result.Manifest.Version)
	}

	chartPath := result.AssetPath("charts/app.tgz")
	data, err := os.ReadFile(chartPath)
	if err != nil {
		t.Fatalf("read extracted file: %v", err)
	}
	if string(data) != "dummy chart" {
		t.Fatalf("unexpected file content: %s", string(data))
	}
}

func TestLoadBundleChecksumMismatch(t *testing.T) {
	tempDir := t.TempDir()
	bundlePath := filepath.Join(tempDir, "bundle.tar")
	cacheDir := filepath.Join(tempDir, "cache")

	payload := map[string][]byte{
		"charts/app.tgz": []byte("dummy chart"),
	}

	manifest := bundle.Manifest{
		Version: "1.0.0",
		Checksums: map[string]string{
			"charts/app.tgz": "deadbeef",
		},
	}

	createBundle(t, bundlePath, manifest, payload)

	_, err := bundle.Load(bundlePath, cacheDir)
	if err == nil {
		t.Fatalf("expected checksum mismatch error")
	}
	if !errors.Is(err, bundle.ErrChecksumMismatch) {
		t.Fatalf("expected ErrChecksumMismatch, got %v", err)
	}
}

func createBundle(t *testing.T, path string, manifest bundle.Manifest, payload map[string][]byte) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create tar: %v", err)
	}
	defer file.Close()

	tw := tar.NewWriter(file)
	defer tw.Close()

	manifestBytes, err := manifest.Marshal()
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	writeTarFile(t, tw, "bundle.yaml", manifestBytes)

	for name, data := range payload {
		writeTarFile(t, tw, name, data)
	}
}

func writeTarFile(t *testing.T, tw *tar.Writer, name string, data []byte) {
	t.Helper()

	hdr := &tar.Header{
		Name: name,
		Mode: 0o600,
		Size: int64(len(data)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("write header %s: %v", name, err)
	}
	if _, err := tw.Write(data); err != nil {
		t.Fatalf("write data %s: %v", name, err)
	}
	if err := tw.Flush(); err != nil {
		t.Fatalf("flush tar: %v", err)
	}
}
