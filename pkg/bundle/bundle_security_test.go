package bundle

import (
    "archive/tar"
    "bytes"
    "crypto/sha256"
    "encoding/hex"
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
            Mode: 0644,
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
        "bundle.yaml": "checksums:\n  app/ok.txt: \nversion: v1\n",
        "app/ok.txt":  "ok",
        "../../evil":  "nope",
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
    root := "/tmp/extract"
    if _, err := safeJoin(root, "../../etc/passwd"); err == nil {
        t.Fatal("expected error when path escapes root")
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
        "bundle.yaml":     manifest,
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
