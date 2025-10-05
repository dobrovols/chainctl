package secrets

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/scrypt"
)

const (
	envelopeMagic   = "CENC"
	envelopeVersion = byte(1)
	saltSize        = 16
	nonceSize       = 12
)

// EncryptOptions captures configuration for EncryptFile.
type EncryptOptions struct {
	InputPath  string
	OutputPath string
	Passphrase string
	Overwrite  bool
}

// EncryptResult describes the outcome of encryption.
type EncryptResult struct {
	OutputPath string
	Checksum   string
}

// DecryptOptions captures configuration for DecryptFile.
type DecryptOptions struct {
	InputPath  string
	Passphrase string
}

// EncryptFile encrypts a plaintext file and writes the binary envelope to disk.
func EncryptFile(opts EncryptOptions) (*EncryptResult, error) {
	if opts.InputPath == "" || opts.OutputPath == "" {
		return nil, NewError(ErrCodeValidation, errors.New("input and output paths are required"))
	}
	if opts.Passphrase == "" {
		return nil, NewError(ErrCodeValidation, errors.New("passphrase cannot be empty"))
	}

	if !opts.Overwrite {
		if _, err := os.Stat(opts.OutputPath); err == nil {
			return nil, NewError(ErrCodeValidation, fmt.Errorf("output file %s already exists (use --confirm to overwrite)", opts.OutputPath))
		}
	}

	plaintext, err := os.ReadFile(opts.InputPath)
	if err != nil {
		return nil, NewError(ErrCodeValidation, fmt.Errorf("read input file: %w", err))
	}
	if len(bytes.TrimSpace(plaintext)) == 0 {
		return nil, NewError(ErrCodeValidation, errors.New("input file is empty"))
	}

	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, NewError(ErrCodeEncryption, fmt.Errorf("generate salt: %w", err))
	}

	passBytes := []byte(opts.Passphrase)
	key, err := scrypt.Key(passBytes, salt, 1<<15, 8, 1, 32)
	if err != nil {
		return nil, NewError(ErrCodeEncryption, fmt.Errorf("derive key: %w", err))
	}
	defer zeroBytes(key)
	zeroBytes(passBytes)
	// Prevent compiler optimization from removing zeroing
	runtime.KeepAlive(passBytes)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, NewError(ErrCodeEncryption, fmt.Errorf("create cipher: %w", err))
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, NewError(ErrCodeEncryption, fmt.Errorf("generate nonce: %w", err))
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, NewError(ErrCodeEncryption, fmt.Errorf("create gcm: %w", err))
	}

	additionalData := append([]byte(envelopeMagic), envelopeVersion)
	ciphertext := aead.Seal(nil, nonce, plaintext, additionalData)

	buf := bytes.NewBuffer(nil)
	buf.Write(additionalData)
	buf.Write(salt)
	buf.Write(nonce)
	buf.Write(ciphertext)

	if err := os.WriteFile(opts.OutputPath, buf.Bytes(), 0o600); err != nil {
		return nil, NewError(ErrCodeEncryption, fmt.Errorf("write output file: %w", err))
	}

	checksum := sha256.Sum256(ciphertext)

	zeroBytes(plaintext)

	return &EncryptResult{
		OutputPath: opts.OutputPath,
		Checksum:   fmt.Sprintf("%x", checksum),
	}, nil
}

// DecryptFile decrypts an encrypted values file and returns plaintext bytes.
func DecryptFile(opts DecryptOptions) ([]byte, error) {
	if opts.InputPath == "" {
		return nil, NewError(ErrCodeValidation, errors.New("input path is required"))
	}
	if opts.Passphrase == "" {
		return nil, NewError(ErrCodeValidation, errors.New("passphrase cannot be empty"))
	}

	payload, err := os.ReadFile(opts.InputPath)
	if err != nil {
		return nil, NewError(ErrCodeValidation, fmt.Errorf("read input file: %w", err))
	}

	headerSize := len(envelopeMagic) + 1
	minSize := headerSize + saltSize + nonceSize + 1
	if len(payload) < minSize {
		return nil, NewError(ErrCodeEncryption, errors.New("encrypted payload too small"))
	}

	if string(payload[:len(envelopeMagic)]) != envelopeMagic {
		return nil, NewError(ErrCodeEncryption, errors.New("invalid envelope header"))
	}
	if payload[len(envelopeMagic)] != envelopeVersion {
		return nil, NewError(ErrCodeEncryption, fmt.Errorf("unsupported envelope version %d", payload[len(envelopeMagic)]))
	}

	offset := headerSize
	salt := payload[offset : offset+saltSize]
	offset += saltSize
	nonce := payload[offset : offset+nonceSize]
	offset += nonceSize
	ciphertext := payload[offset:]

	passBytes := []byte(opts.Passphrase)
	key, err := scrypt.Key(passBytes, salt, 1<<15, 8, 1, 32)
	zeroBytes(passBytes)
	if err != nil {
		return nil, NewError(ErrCodeEncryption, fmt.Errorf("derive key: %w", err))
	}
	defer zeroBytes(key)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, NewError(ErrCodeEncryption, fmt.Errorf("create cipher: %w", err))
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, NewError(ErrCodeEncryption, fmt.Errorf("create gcm: %w", err))
	}

	additionalData := append([]byte(envelopeMagic), envelopeVersion)
	plaintext, err := aead.Open(nil, nonce, ciphertext, additionalData)
	if err != nil {
		return nil, NewError(ErrCodeEncryption, fmt.Errorf("decrypt: %w", err))
	}

	result := make([]byte, len(plaintext))
	copy(result, plaintext)
	zeroBytes(plaintext)

	return result, nil
}

func zeroBytes(buf []byte) {
	for i := range buf {
		buf[i] = 0
	}
}
