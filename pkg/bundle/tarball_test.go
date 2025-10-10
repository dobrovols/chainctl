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

const (
	testChartPath       = "charts/app.tgz"
	testChartContent    = "dummy chart"
	testManifestVersion = "1.0.0"
	testBundleFileName  = "bundle.tar"
	testCacheDirName    = "cache"
	testAssetPath       = "assets/data.txt"
	testAssetDir        = "assets"
)

func TestLoadBundleSuccess(t *testing.T) {
	bundlePath, cacheDir := tempBundlePaths(t)

	payload := map[string][]byte{
		testChartPath: []byte(testChartContent),
	}

	manifest := bundle.Manifest{
		Version:   testManifestVersion,
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

	if result.Manifest.Version != testManifestVersion {
		t.Fatalf("expected version 1.0.0, got %s", result.Manifest.Version)
	}

	chartPath := result.AssetPath(testChartPath)
	data, err := os.ReadFile(chartPath)
	if err != nil {
		t.Fatalf("read extracted file: %v", err)
	}
	if string(data) != testChartContent {
		t.Fatalf("unexpected file content: %s", string(data))
	}
}

func TestLoadBundleChecksumMismatch(t *testing.T) {
	bundlePath, cacheDir := tempBundlePaths(t)

	payload := map[string][]byte{
		testChartPath: []byte(testChartContent),
	}

	manifest := bundle.Manifest{
		Version: testManifestVersion,
		Checksums: map[string]string{
			testChartPath: "deadbeef",
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
	bundlePath, cacheDir := tempBundlePaths(t)

	withTarWriter(t, bundlePath, func(tw *tar.Writer) {
		writeTarFile(t, tw, testChartPath, []byte("dummy"))
	})

	if _, err := bundle.Load(bundlePath, cacheDir); err == nil || !errors.Is(err, bundle.ErrManifestMissing) {
		t.Fatalf("expected manifest missing error, got %v", err)
	}
}

func TestLoadBundlePreventsPathEscape(t *testing.T) {
	bundlePath, cacheDir := tempBundlePaths(t)

	manifest := bundle.Manifest{
		Version:   testManifestVersion,
		Checksums: map[string]string{},
	}

	manifestBytes, err := manifest.Marshal()
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	withTarWriter(t, bundlePath, func(tw *tar.Writer) {
		writeTarFile(t, tw, bundle.ManifestFileName, manifestBytes)
		writeTarFile(t, tw, "../escape.txt", []byte("bad"))
	})

	if _, err := bundle.Load(bundlePath, cacheDir); err == nil || !errors.Is(err, bundle.ErrPathOutsideBundle) {
		t.Fatalf("expected path outside bundle error, got %v", err)
	}
}

func TestLoadRequiresTarballPath(t *testing.T) {
	if _, err := bundle.Load("", ""); err == nil || !strings.Contains(err.Error(), "tarball path required") {
		t.Fatalf("expected path required error, got %v", err)
	}
}

func TestLoadInvalidManifest(t *testing.T) {
	bundlePath, cacheDir := tempBundlePaths(t)

	withTarWriter(t, bundlePath, func(tw *tar.Writer) {
		writeTarFile(t, tw, bundle.ManifestFileName, []byte("not: [valid"))
	})

	if _, err := bundle.Load(bundlePath, cacheDir); err == nil || !strings.Contains(err.Error(), "parse manifest") {
		t.Fatalf("expected parse manifest error, got %v", err)
	}
}

func TestLoadDefaultsChecksumMap(t *testing.T) {
	bundlePath, cacheDir := tempBundlePaths(t)

	manifest := bundle.Manifest{Version: testManifestVersion}
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
	bundlePath, cacheDir := tempBundlePaths(t)

	manifest := bundle.Manifest{
		Version:   testManifestVersion,
		Checksums: map[string]string{},
	}
	hash := sha256.Sum256([]byte("payload"))
	manifest.Checksums[testAssetPath] = hex.EncodeToString(hash[:])
	manifestBytes, err := manifest.Marshal()
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	withTarWriter(t, bundlePath, func(tw *tar.Writer) {
		if err := tw.WriteHeader(&tar.Header{Name: testAssetDir + "/", Mode: 0o755, Typeflag: tar.TypeDir}); err != nil {
			t.Fatalf("write dir header: %v", err)
		}
		writeTarFile(t, tw, bundle.ManifestFileName, manifestBytes)
		writeTarFile(t, tw, testAssetPath, []byte("payload"))
	})

	result, err := bundle.Load(bundlePath, cacheDir)
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	if _, err := os.Stat(filepath.Join(result.Extracted, testAssetDir)); err != nil {
		t.Fatalf("expected assets directory to exist: %v", err)
	}
}

func TestLoadDefaultsCacheRoot(t *testing.T) {
	bundlePath, _ := tempBundlePaths(t)

	manifest := bundle.Manifest{Version: testManifestVersion}
	createBundle(t, bundlePath, manifest, map[string][]byte{})

	result, err := bundle.Load(bundlePath, "")
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	if result.CacheRoot != filepath.Dir(bundlePath) {
		t.Fatalf("expected cache root to default to tarball dir, got %s", result.CacheRoot)
	}
}

func tempBundlePaths(t *testing.T) (bundlePath, cacheDir string) {
	t.Helper()

	dir := t.TempDir()
	return filepath.Join(dir, testBundleFileName), filepath.Join(dir, testCacheDirName)
}

func withTarWriter(t *testing.T, path string, fn func(*tar.Writer)) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create tar: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Fatalf("close tar file: %v", err)
		}
	}()

	tw := tar.NewWriter(file)
	defer func() {
		if err := tw.Close(); err != nil {
			t.Fatalf("close tar writer: %v", err)
		}
	}()

	fn(tw)
}

func createBundle(t *testing.T, path string, manifest bundle.Manifest, payload map[string][]byte) {
	t.Helper()

	manifestBytes, err := manifest.Marshal()
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	withTarWriter(t, path, func(tw *tar.Writer) {
		writeTarFile(t, tw, bundle.ManifestFileName, manifestBytes)
		for name, data := range payload {
			writeTarFile(t, tw, name, data)
		}
	})
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
