package bundle_test

import (
	"archive/tar"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestLoadBundleManifestMissing(t *testing.T) {
	tempDir := t.TempDir()
	bundlePath := filepath.Join(tempDir, "bundle.tar")

	file, err := os.Create(bundlePath)
	if err != nil {
		t.Fatalf("create tar: %v", err)
	}
	defer file.Close()

	tw := tar.NewWriter(file)
	defer tw.Close()

	writeTarFile(t, tw, "charts/app.tgz", []byte("dummy"))

	if _, err := bundle.Load(bundlePath, filepath.Join(tempDir, "cache")); err == nil || !errors.Is(err, bundle.ErrManifestMissing) {
		t.Fatalf("expected manifest missing error, got %v", err)
	}
}

func TestLoadBundlePreventsPathEscape(t *testing.T) {
	tempDir := t.TempDir()
	bundlePath := filepath.Join(tempDir, "bundle.tar")

	manifest := bundle.Manifest{
		Version:   "1.0.0",
		Checksums: map[string]string{},
	}

	file, err := os.Create(bundlePath)
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
	writeTarFile(t, tw, "../escape.txt", []byte("bad"))

	if _, err := bundle.Load(bundlePath, filepath.Join(tempDir, "cache")); err == nil || !errors.Is(err, bundle.ErrPathOutsideBundle) {
		t.Fatalf("expected path outside bundle error, got %v", err)
	}
}

func TestLoadRequiresTarballPath(t *testing.T) {
	if _, err := bundle.Load("", ""); err == nil || !strings.Contains(err.Error(), "tarball path required") {
		t.Fatalf("expected path required error, got %v", err)
	}
}

func TestLoadInvalidManifest(t *testing.T) {
	tempDir := t.TempDir()
	bundlePath := filepath.Join(tempDir, "bundle.tar")

	file, err := os.Create(bundlePath)
	if err != nil {
		t.Fatalf("create tar: %v", err)
	}
	defer file.Close()

	tw := tar.NewWriter(file)
	defer tw.Close()

	writeTarFile(t, tw, "bundle.yaml", []byte("not: [valid"))

	if _, err := bundle.Load(bundlePath, filepath.Join(tempDir, "cache")); err == nil || !strings.Contains(err.Error(), "parse manifest") {
		t.Fatalf("expected parse manifest error, got %v", err)
	}
}

func TestLoadDefaultsChecksumMap(t *testing.T) {
	tempDir := t.TempDir()
	bundlePath := filepath.Join(tempDir, "bundle.tar")
	cacheDir := filepath.Join(tempDir, "cache")

	manifest := bundle.Manifest{Version: "1.0.0"}
	createBundle(t, bundlePath, manifest, map[string][]byte{})

	result, err := bundle.Load(bundlePath, cacheDir)
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	if result.Manifest.Checksums == nil {
		t.Fatalf("expected checksums map to be initialised")
	}
}

func TestLoadCreatesDirectories(t *testing.T) {
	tempDir := t.TempDir()
	bundlePath := filepath.Join(tempDir, "bundle.tar")
	cacheDir := filepath.Join(tempDir, "cache")

	file, err := os.Create(bundlePath)
	if err != nil {
		t.Fatalf("create tar: %v", err)
	}
	defer file.Close()

	tw := tar.NewWriter(file)
	defer tw.Close()

	manifest := bundle.Manifest{
		Version:   "1.0.0",
		Checksums: map[string]string{},
	}
	hash := sha256.Sum256([]byte("payload"))
	manifest.Checksums["assets/data.txt"] = hex.EncodeToString(hash[:])
	manifestBytes, err := manifest.Marshal()
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	if err := tw.WriteHeader(&tar.Header{Name: "assets/", Mode: 0o755, Typeflag: tar.TypeDir}); err != nil {
		t.Fatalf("write dir header: %v", err)
	}
	writeTarFile(t, tw, "bundle.yaml", manifestBytes)
	writeTarFile(t, tw, "assets/data.txt", []byte("payload"))

	result, err := bundle.Load(bundlePath, cacheDir)
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	if _, err := os.Stat(filepath.Join(result.Extracted, "assets")); err != nil {
		t.Fatalf("expected assets directory to exist: %v", err)
	}
}

func TestLoadDefaultsCacheRoot(t *testing.T) {
	tempDir := t.TempDir()
	bundlePath := filepath.Join(tempDir, "bundle.tar")

	manifest := bundle.Manifest{Version: "1.0.0"}
	createBundle(t, bundlePath, manifest, map[string][]byte{})

	result, err := bundle.Load(bundlePath, "")
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	if result.CacheRoot != filepath.Dir(bundlePath) {
		t.Fatalf("expected cache root to default to tarball dir, got %s", result.CacheRoot)
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
