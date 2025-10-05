package tokens

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Scope enumerates valid token scopes.
type Scope string

const (
	ScopeWorker       Scope = "worker"
	ScopeControlPlane Scope = "control-plane"

	maxTTL = 24 * time.Hour
)

var (
	errTokenNotFound   = errors.New("token not found")
	errTokenConsumed   = errors.New("token already consumed")
	errTokenExpired    = errors.New("token expired")
	errScopeMismatch   = errors.New("token scope mismatch")
	errInvalidTokenFmt = errors.New("invalid token format")
)

// CreateOptions defines how tokens are generated.
type CreateOptions struct {
	Scope       Scope
	TTL         time.Duration
	CreatedBy   string
	Description string
}

// Token encapsulates metadata stored for validation.
type Token struct {
	ID           string
	Scope        Scope
	ExpiresAt    time.Time
	HashedSecret string
	CreatedBy    string
	Description  string
	Consumed     bool
}

// CreatedToken is returned to callers after creation.
type CreatedToken struct {
	ID          string
	Scope       Scope
	ExpiresAt   time.Time
	CreatedBy   string
	Description string
	Token       string
}

// MemoryStore stores tokens in-memory (suitable for unit tests).
type MemoryStore struct {
	mu     sync.Mutex
	tokens map[string]*Token
}

// NewMemoryStore constructs a store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{tokens: map[string]*Token{}}
}

// Create generates a new token and returns metadata plus the composite token string.
func (s *MemoryStore) Create(opts CreateOptions) (*CreatedToken, error) {
	if opts.Scope != ScopeWorker && opts.Scope != ScopeControlPlane {
		return nil, fmt.Errorf("unknown scope %q", opts.Scope)
	}
	ttl := opts.TTL
	if ttl <= 0 {
		ttl = 2 * time.Hour
	}
	if ttl > maxTTL {
		return nil, fmt.Errorf("ttl %s exceeds maximum %s", ttl, maxTTL)
	}

	id, err := randomHex(8)
	if err != nil {
		return nil, fmt.Errorf("generate id: %w", err)
	}
	secret, err := randomHex(16)
	if err != nil {
		return nil, fmt.Errorf("generate secret: %w", err)
	}

	hashed := hashSecret(id, secret)

	record := &Token{
		ID:           id,
		Scope:        opts.Scope,
		ExpiresAt:    time.Now().Add(ttl),
		HashedSecret: hashed,
		CreatedBy:    opts.CreatedBy,
		Description:  opts.Description,
	}

	s.mu.Lock()
	s.tokens[id] = record
	s.mu.Unlock()

	composite := fmt.Sprintf("%s.%s", id, secret)

	return &CreatedToken{
		ID:          record.ID,
		Scope:       record.Scope,
		ExpiresAt:   record.ExpiresAt,
		CreatedBy:   record.CreatedBy,
		Description: record.Description,
		Token:       composite,
	}, nil
}

// Consume validates and marks a token as used.
func (s *MemoryStore) Consume(composite string, expected Scope) error {
	id, secret, err := splitToken(composite)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.tokens[id]
	if !ok {
		return errTokenNotFound
	}
	if record.Consumed {
		return errTokenConsumed
	}
	if time.Now().After(record.ExpiresAt) {
		return errTokenExpired
	}
	if record.Scope != expected {
		return errScopeMismatch
	}

	if !compareSecret(record, id, secret) {
		return errTokenNotFound
	}

	record.Consumed = true
	return nil
}

// ForceExpire is a helper for tests to simulate expiry.
func (s *MemoryStore) ForceExpire(composite string) {
	id, _, err := splitToken(composite)
	if err != nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if record, ok := s.tokens[id]; ok {
		record.ExpiresAt = time.Now().Add(-time.Minute)
	}
}

func splitToken(token string) (string, string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", errInvalidTokenFmt
	}
	return parts[0], parts[1], nil
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashSecret(id, secret string) string {
	sum := sha256.Sum256([]byte(id + ":" + secret))
	return hex.EncodeToString(sum[:])
}

func compareSecret(record *Token, id, secret string) bool {
	expected := hashSecret(id, secret)
	if len(expected) != len(record.HashedSecret) {
		return false
	}
	result := subtle.ConstantTimeCompare([]byte(expected), []byte(record.HashedSecret))
	return result == 1
}
