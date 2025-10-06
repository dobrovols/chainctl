package e2e

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/dobrovols/chainctl/pkg/bundle"
)

func TestAppInstallUpdateQuickstart(t *testing.T) {
	if os.Getenv("CHAINCTL_E2E") == "" {
		t.Skip("skip app install e2e: set CHAINCTL_E2E=1")
	}

	tempDir := t.TempDir()
	bundlePath := filepath.Join(tempDir, "quickstart-bundle.tar")
	createQuickstartBundle(t, bundlePath)

	statePath := filepath.Join(tempDir, "state", "app.json")

	env := append(os.Environ(),
		"GO111MODULE=on",
		"XDG_CONFIG_HOME="+filepath.Join(tempDir, "xdg"),
	)

	valuesFile := envOrDefault("CHAINCTL_VALUES_FILE", "test/e2e/testdata/values.enc")
	passphrase := envOrDefault("CHAINCTL_VALUES_PASSPHRASE", "secret")
	clusterEndpoint := envOrDefault("CHAINCTL_CLUSTER_ENDPOINT", "https://cluster.local")

	run := func(args ...string) ([]byte, error) {
		cmd := exec.Command("go", append([]string{"run", "./cmd/chainctl"}, args...)...)
		cmd.Dir = projectRoot(t)
		cmd.Env = env
		return cmd.CombinedOutput()
	}

	installArgs := []string{
		"app", "install",
		"--bundle-path", bundlePath,
		"--values-file", valuesFile,
		"--values-passphrase", passphrase,
		"--release-name", "quickstart",
		"--namespace", "quickstart",
		"--state-file", statePath,
		"--output", "json",
	}

	installOut, err := run(installArgs...)
	if err != nil {
		t.Fatalf("app install failed: %v\n%s", err, string(installOut))
	}

	installPayload := parseCLIJSON(t, installOut)
	assertEqual(t, installPayload["status"], "success", "unexpected install status")
	assertEqual(t, installPayload["action"], "install", "unexpected install action")
	assertEqual(t, installPayload["stateFile"], statePath, "install state path mismatch")

	stateRecord := readStateRecord(t, statePath)
	if stateRecord.LastAction != "install" {
		t.Fatalf("expected last action install, got %s", stateRecord.LastAction)
	}
	if stateRecord.Chart.Reference != bundlePath {
		t.Fatalf("expected chart reference %s, got %s", bundlePath, stateRecord.Chart.Reference)
	}

	upgradeArgs := []string{
		"app", "upgrade",
		"--bundle-path", bundlePath,
		"--values-file", valuesFile,
		"--values-passphrase", passphrase,
		"--cluster-endpoint", clusterEndpoint,
		"--release-name", "quickstart",
		"--namespace", "quickstart",
		"--state-file", statePath,
		"--output", "json",
	}

	upgradeOut, err := run(upgradeArgs...)
	if err != nil {
		t.Fatalf("app upgrade failed: %v\n%s", err, string(upgradeOut))
	}

	upgradePayload := parseCLIJSON(t, upgradeOut)
	assertEqual(t, upgradePayload["action"], "upgrade", "unexpected upgrade action")
	assertEqual(t, upgradePayload["stateFile"], statePath, "upgrade state path mismatch")
	assertEqual(t, upgradePayload["cluster"], clusterEndpoint, "upgrade cluster endpoint mismatch")

	stateRecord = readStateRecord(t, statePath)
	if stateRecord.LastAction != "upgrade" {
		t.Fatalf("expected last action upgrade, got %s", stateRecord.LastAction)
	}
	if stateRecord.ClusterEndpoint != clusterEndpoint {
		t.Fatalf("expected cluster endpoint %s, got %s", clusterEndpoint, stateRecord.ClusterEndpoint)
	}
}

func createQuickstartBundle(t *testing.T, bundlePath string) {
	t.Helper()

	file, err := os.Create(bundlePath)
	if err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	defer file.Close()

	tw := tar.NewWriter(file)
	defer tw.Close()

	payload := map[string][]byte{
		"charts/app.tgz": []byte("dummy chart"),
	}

	manifest := bundle.Manifest{
		Version: "1.0.0",
		Charts: []bundle.ChartRecord{
			{
				Name:    "quickstart",
				Version: "1.0.0",
				Path:    "charts/app.tgz",
			},
		},
		Checksums: map[string]string{},
	}

	for name, data := range payload {
		sum := sha256.Sum256(data)
		manifest.Checksums[name] = hex.EncodeToString(sum[:])
	}

	manifestBytes, err := manifest.Marshal()
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	writeTarDir(t, tw, "charts/")
	writeTarFile(t, tw, "bundle.yaml", manifestBytes)
	for name, data := range payload {
		writeTarFile(t, tw, name, data)
	}
}

func writeTarDir(t *testing.T, tw *tar.Writer, name string) {
	t.Helper()

	hdr := &tar.Header{
		Name:     name,
		Mode:     0o755,
		Typeflag: tar.TypeDir,
		ModTime:  time.Unix(0, 0),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("write tar dir %s: %v", name, err)
	}
}

func writeTarFile(t *testing.T, tw *tar.Writer, name string, data []byte) {
	t.Helper()

	hdr := &tar.Header{
		Name:    name,
		Mode:    0o644,
		Size:    int64(len(data)),
		ModTime: time.Unix(0, 0),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("write tar file %s: %v", name, err)
	}
	if _, err := tw.Write(data); err != nil {
		t.Fatalf("write tar data %s: %v", name, err)
	}
}

type appStateRecord struct {
	LastAction      string `json:"lastAction"`
	ClusterEndpoint string `json:"clusterEndpoint"`
	Chart           struct {
		Reference string `json:"reference"`
		Type      string `json:"type"`
	} `json:"chart"`
}

func readStateRecord(t *testing.T, path string) appStateRecord {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read state file: %v", err)
	}

	var record appStateRecord
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatalf("decode state file: %v", err)
	}

	return record
}

func parseCLIJSON(t *testing.T, data []byte) map[string]any {
	t.Helper()

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		t.Fatalf("decode CLI output: empty output")
	}
	lines := bytes.Split(trimmed, []byte("\n"))
	payloadLine := lines[len(lines)-1]

	var payload map[string]any
	if err := json.Unmarshal(payloadLine, &payload); err != nil {
		t.Fatalf("decode CLI output: %v\n%s", err, string(trimmed))
	}
	return payload
}

func assertEqual(t *testing.T, got any, want any, msg string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s: got %v want %v", msg, got, want)
	}
}
