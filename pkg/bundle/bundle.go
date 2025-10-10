package bundle

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// Error sentinel values for bundle integrity validation.
var (
	ErrManifestMissing   = errors.New("bundle manifest missing")
	ErrChecksumMismatch  = errors.New("bundle checksum mismatch")
	ErrPathOutsideBundle = errors.New("bundle entry escapes extraction directory")
)

// Manifest describes the structure of the bundle.
type Manifest struct {
	Version   string            `yaml:"version"`
	Images    []ImageRecord     `yaml:"images"`
	Charts    []ChartRecord     `yaml:"helmCharts"`
	Binaries  []BinaryRecord    `yaml:"binaries"`
	Checksums map[string]string `yaml:"checksums"`
}

// ImageRecord captures a container image entry.
type ImageRecord struct {
	Name   string `yaml:"name"`
	Tag    string `yaml:"tag"`
	Digest string `yaml:"digest"`
}

// ChartRecord captures Helm chart metadata.
type ChartRecord struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Path    string `yaml:"path"`
}

// BinaryRecord captures auxiliary binary information.
type BinaryRecord struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Path    string `yaml:"path"`
	OS      string `yaml:"os"`
	Arch    string `yaml:"arch"`
}

// Marshal serialises the manifest to YAML.
func (m Manifest) Marshal() ([]byte, error) {
	return yaml.Marshal(m)
}

// unmarshalManifest returns a Manifest from YAML bytes.
func unmarshalManifest(data []byte) (Manifest, error) {
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return Manifest{}, err
	}
	if m.Checksums == nil {
		m.Checksums = map[string]string{}
	}
	return m, nil
}

// Bundle represents an extracted bundle on disk.
type Bundle struct {
	Path      string
	CacheRoot string
	Extracted string
	Manifest  Manifest
}

// AssetPath returns the absolute path for a file inside the extracted bundle.
func (b *Bundle) AssetPath(rel string) string {
	cleaned := filepath.Clean(rel)
	return filepath.Join(b.Extracted, cleaned)
}

// Load extracts the bundle tarball into cacheRoot (creating a hashed directory) and validates checksums.
func Load(tarballPath, cacheRoot string) (*Bundle, error) {
	if tarballPath == "" {
		return nil, fmt.Errorf("tarball path required")
	}
	if cacheRoot == "" {
		cacheRoot = filepath.Dir(tarballPath)
	}

	data, err := os.ReadFile(tarballPath)
	if err != nil {
		return nil, fmt.Errorf("read bundle: %w", err)
	}

	hash := sha256.Sum256(data)
	bundleID := hex.EncodeToString(hash[:])
	extractDir := filepath.Join(cacheRoot, bundleID)

	if err := os.RemoveAll(extractDir); err != nil {
		return nil, fmt.Errorf("reset bundle cache: %w", err)
	}
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return nil, fmt.Errorf("create bundle cache: %w", err)
	}
	manifest, err := extractTarToDir(data, extractDir)
	if err != nil {
		return nil, err
	}

	if err := validateChecksums(extractDir, manifest.Checksums); err != nil {
		return nil, err
	}

	return &Bundle{
		Path:      tarballPath,
		CacheRoot: cacheRoot,
		Extracted: extractDir,
		Manifest:  manifest,
	}, nil
}

// extractTarToDir extracts a tar archive byte slice into extractDir and returns the parsed manifest.
func extractTarToDir(data []byte, extractDir string) (Manifest, error) {
	tr := tar.NewReader(bytes.NewReader(data))
	var manifestBytes []byte
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return Manifest{}, fmt.Errorf("read tar entry: %w", err)
		}
		if err := extractEntry(tr, hdr, extractDir, &manifestBytes); err != nil {
			return Manifest{}, err
		}
	}
	if manifestBytes == nil {
		return Manifest{}, ErrManifestMissing
	}
	m, err := unmarshalManifest(manifestBytes)
	if err != nil {
		return Manifest{}, fmt.Errorf("parse manifest: %w", err)
	}
	return m, nil
}

// extractEntry handles a single tar header extraction and updates manifestBytes when the manifest is encountered.
func extractEntry(tr *tar.Reader, hdr *tar.Header, root string, manifestBytes *[]byte) error {
	switch hdr.Typeflag {
	case tar.TypeDir:
		targetDir, err := safeJoin(root, hdr.Name)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(targetDir, os.FileMode(hdr.Mode)); err != nil {
			return fmt.Errorf("create dir %s: %w", hdr.Name, err)
		}
		return nil
	default:
		targetPath, err := safeJoin(root, hdr.Name)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("create parent dir for %s: %w", hdr.Name, err)
		}
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, tr); err != nil {
			return fmt.Errorf("copy tar entry %s: %w", hdr.Name, err)
		}
		if hdr.Name == "bundle.yaml" {
			*manifestBytes = buf.Bytes()
			return nil
		}
		if err := os.WriteFile(targetPath, buf.Bytes(), os.FileMode(hdr.Mode)); err != nil {
			return fmt.Errorf("write file %s: %w", hdr.Name, err)
		}
		return nil
	}
}

// validateChecksums verifies that files listed in checksums map match their sha256 sums.
func validateChecksums(root string, checksums map[string]string) error {
	for rel, expected := range checksums {
		abs, err := safeJoin(root, rel)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			return fmt.Errorf("read asset %s: %w", rel, err)
		}
		sum := sha256.Sum256(data)
		actual := hex.EncodeToString(sum[:])
		if !strings.EqualFold(actual, expected) {
			return fmt.Errorf("%w: %s", ErrChecksumMismatch, rel)
		}
	}
	return nil
}

func safeJoin(root, name string) (string, error) {
	base, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve base path: %w", err)
	}

	cleaned := filepath.Clean(name)
	if filepath.IsAbs(cleaned) {
		return "", ErrPathOutsideBundle
	}
	if vol := filepath.VolumeName(cleaned); vol != "" {
		return "", ErrPathOutsideBundle
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) {
		return "", ErrPathOutsideBundle
	}

	target := filepath.Join(base, cleaned)
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", ErrPathOutsideBundle
	}
	return target, nil
}
