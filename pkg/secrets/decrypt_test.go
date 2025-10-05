package secrets_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/dobrovols/chainctl/pkg/secrets"
)

func TestDecryptFile_Success(t *testing.T) {
	tempDir := t.TempDir()
	input := filepath.Join(tempDir, "values.yaml")
	if err := os.WriteFile(input, []byte("replicaCount: 2\n"), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}

	output := filepath.Join(tempDir, "values.enc")
	if _, err := secrets.EncryptFile(secrets.EncryptOptions{
		InputPath:  input,
		OutputPath: output,
		Passphrase: "secret",
		Overwrite:  true,
	}); err != nil {
		t.Fatalf("encrypt file: %v", err)
	}

	plaintext, err := secrets.DecryptFile(secrets.DecryptOptions{
		InputPath:  output,
		Passphrase: "secret",
	})
	if err != nil {
		t.Fatalf("decrypt file: %v", err)
	}

	if string(plaintext) != "replicaCount: 2\n" {
		t.Fatalf("expected decrypted content to match original, got %q", string(plaintext))
	}
}

func TestDecryptFile_WrongPassphrase(t *testing.T) {
	tempDir := t.TempDir()
	input := filepath.Join(tempDir, "values.yaml")
	if err := os.WriteFile(input, []byte("image: latest\n"), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}

	output := filepath.Join(tempDir, "values.enc")
	if _, err := secrets.EncryptFile(secrets.EncryptOptions{
		InputPath:  input,
		OutputPath: output,
		Passphrase: "correct",
		Overwrite:  true,
	}); err != nil {
		t.Fatalf("encrypt file: %v", err)
	}

	if _, err := secrets.DecryptFile(secrets.DecryptOptions{
		InputPath:  output,
		Passphrase: "wrong",
	}); err == nil {
		t.Fatalf("expected error when passphrase is incorrect")
	} else {
		var serr *secrets.Error
		if !errors.As(err, &serr) {
			t.Fatalf("expected secrets.Error, got %T", err)
		}
		if serr.Code != secrets.ErrCodeEncryption {
			t.Fatalf("expected encryption error code, got %d", serr.Code)
		}
	}
}

func TestDecryptFile_InvalidEnvelope(t *testing.T) {
	tempDir := t.TempDir()
	bogus := filepath.Join(tempDir, "corrupt.enc")
	if err := os.WriteFile(bogus, []byte("not-a-valid-envelope"), 0o600); err != nil {
		t.Fatalf("write bogus file: %v", err)
	}

	if _, err := secrets.DecryptFile(secrets.DecryptOptions{
		InputPath:  bogus,
		Passphrase: "anything",
	}); err == nil {
		t.Fatalf("expected error for invalid envelope")
	}
}

func TestEncryptFileValidatesPaths(t *testing.T) {
	if _, err := secrets.EncryptFile(secrets.EncryptOptions{}); err == nil {
		t.Fatalf("expected error when input/output missing")
	}
}

func TestEncryptFileRequiresPassphrase(t *testing.T) {
	tempDir := t.TempDir()
	input := filepath.Join(tempDir, "values.yaml")
	if err := os.WriteFile(input, []byte("image: test\n"), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}

	if _, err := secrets.EncryptFile(secrets.EncryptOptions{InputPath: input, OutputPath: filepath.Join(tempDir, "values.enc")}); err == nil {
		t.Fatalf("expected passphrase validation error")
	}
}

func TestEncryptFilePreventOverwrite(t *testing.T) {
	tempDir := t.TempDir()
	input := filepath.Join(tempDir, "values.yaml")
	output := filepath.Join(tempDir, "values.enc")

	if err := os.WriteFile(input, []byte("foo: bar\n"), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}
	if err := os.WriteFile(output, []byte("existing"), 0o600); err != nil {
		t.Fatalf("write output: %v", err)
	}

	if _, err := secrets.EncryptFile(secrets.EncryptOptions{
		InputPath:  input,
		OutputPath: output,
		Passphrase: "secret",
	}); err == nil {
		t.Fatalf("expected error when overwrite not confirmed")
	}
}

func TestEncryptFileRejectsEmptyPayload(t *testing.T) {
	tempDir := t.TempDir()
	input := filepath.Join(tempDir, "values.yaml")
	output := filepath.Join(tempDir, "values.enc")

	if err := os.WriteFile(input, []byte("   \n"), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}

	if _, err := secrets.EncryptFile(secrets.EncryptOptions{
		InputPath:  input,
		OutputPath: output,
		Passphrase: "secret",
		Overwrite:  true,
	}); err == nil {
		t.Fatalf("expected error for empty input file")
	}
}
