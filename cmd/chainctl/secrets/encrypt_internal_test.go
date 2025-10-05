package secrets

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestPromptForPassphraseNonInteractive(t *testing.T) {
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = oldStdin
		r.Close()
		w.Close()
	})

	_, perr := promptForPassphrase(io.Discard)
	if perr == nil || !strings.Contains(perr.Error(), "non-interactive") {
		t.Fatalf("expected non-interactive error, got %v", perr)
	}
}

func TestPromptForPassphraseSuccess(t *testing.T) {
	origIsTerminal := isTerminal
	origReadPassword := readPassword
	origStdinFD := stdinFD
	t.Cleanup(func() {
		isTerminal = origIsTerminal
		readPassword = origReadPassword
		stdinFD = origStdinFD
	})

	callCount := 0
	isTerminal = func(int) bool { return true }
	readPassword = func(int) ([]byte, error) {
		callCount++
		return []byte("s3cret"), nil
	}
	stdinFD = func() int { return 0 }

	pass, err := promptForPassphrase(io.Discard)
	if err != nil {
		t.Fatalf("prompt failed: %v", err)
	}
	if pass != "s3cret" {
		t.Fatalf("expected passphrase to match, got %s", pass)
	}
	if callCount != 2 {
		t.Fatalf("expected readPassword called twice, got %d", callCount)
	}
}

func TestPromptForPassphraseMismatch(t *testing.T) {
	origIsTerminal := isTerminal
	origReadPassword := readPassword
	origStdinFD := stdinFD
	t.Cleanup(func() {
		isTerminal = origIsTerminal
		readPassword = origReadPassword
		stdinFD = origStdinFD
	})

	isTerminal = func(int) bool { return true }
	stdinFD = func() int { return 0 }
	call := 0
	readPassword = func(int) ([]byte, error) {
		call++
		if call == 1 {
			return []byte("first"), nil
		}
		return []byte("second"), nil
	}

	_, err := promptForPassphrase(io.Discard)
	if err == nil || !strings.Contains(err.Error(), "do not match") {
		t.Fatalf("expected mismatch error, got %v", err)
	}
}

func TestZero(t *testing.T) {
	data := []byte{1, 2, 3}
	zero(data)
	for i, b := range data {
		if b != 0 {
			t.Fatalf("expected zero at index %d, got %d", i, b)
		}
	}
}
