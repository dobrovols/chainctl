package unit

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	chainctlcmd "github.com/dobrovols/chainctl/internal/cli"
	secreterrors "github.com/dobrovols/chainctl/pkg/secrets"
)

func TestEncryptValuesCommand_TextOutput(t *testing.T) {
	tempDir := t.TempDir()
	input := filepath.Join(tempDir, "values.yaml")
	output := filepath.Join(tempDir, "values.enc")

	if err := os.WriteFile(input, []byte("replicaCount: 1\n"), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}

	root := chainctlcmd.NewRootCommand()
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs([]string{
		"encrypt-values",
		"--input", input,
		"--output", output,
		"--passphrase", "pa55word!",
	})

	if err := root.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	info, err := os.Stat(output)
	if err != nil {
		t.Fatalf("stat output: %v", err)
	}
	if info.Size() <= 32 {
		t.Fatalf("expected encrypted file larger than header, got %d", info.Size())
	}

	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if string(data[:4]) != "CENC" {
		t.Fatalf("expected CENC magic header, got %x", data[:4])
	}

	if !bytes.Contains(stdout.Bytes(), []byte("Checksum:")) {
		t.Fatalf("expected checksum in stdout, got %s", stdout.String())
	}
}

func TestEncryptValuesCommand_OverwriteProtection(t *testing.T) {
	tempDir := t.TempDir()
	input := filepath.Join(tempDir, "values.yaml")
	output := filepath.Join(tempDir, "values.enc")

	if err := os.WriteFile(input, []byte("replicaCount: 1\n"), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}
	if err := os.WriteFile(output, []byte("existing"), 0o600); err != nil {
		t.Fatalf("write output: %v", err)
	}

	root := chainctlcmd.NewRootCommand()
	root.SetOut(new(bytes.Buffer))
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{
		"encrypt-values",
		"--input", input,
		"--output", output,
		"--passphrase", "pa55word!",
	})

	err := root.Execute()
	if err == nil {
		t.Fatalf("expected error when overwriting without confirm")
	}

	var se *secreterrors.Error
	if !errors.As(err, &se) {
		t.Fatalf("expected secrets.Error, got %T", err)
	}
	if se.Code != secreterrors.ErrCodeValidation {
		t.Fatalf("expected validation error code, got %d", se.Code)
	}

	contents, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if string(contents) != "existing" {
		t.Fatalf("expected output file to remain unchanged")
	}
}

func TestEncryptValuesCommand_JSONOutput(t *testing.T) {
	tempDir := t.TempDir()
	input := filepath.Join(tempDir, "values.yaml")
	output := filepath.Join(tempDir, "values.enc")

	if err := os.WriteFile(input, []byte("key: value\n"), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}

	root := chainctlcmd.NewRootCommand()
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs([]string{
		"encrypt-values",
		"--input", input,
		"--output", output,
		"--passphrase", "another-passphrase",
		"--format", "json",
	})

	if err := root.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	if string(data[:4]) != "CENC" {
		t.Fatalf("expected CENC header, got %x", data[:4])
	}
	checksum := extractChecksum(stdout.Bytes())
	if checksum == "" {
		t.Fatalf("expected checksum in json output, got %s", stdout.String())
	}
	if _, err := hex.DecodeString(checksum); err != nil {
		t.Fatalf("checksum not hex encoded: %v", err)
	}
}

func extractChecksum(raw []byte) string {
	var out struct {
		Checksum string `json:"checksum"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return ""
	}
	return out.Checksum
}
