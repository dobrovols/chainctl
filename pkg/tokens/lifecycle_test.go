package tokens_test

import (
	"testing"
	"time"

	"github.com/dobrovols/chainctl/pkg/tokens"
)

func TestCreateAndConsumeTokenSuccess(t *testing.T) {
	store := tokens.NewMemoryStore()

	token, err := store.Create(tokens.CreateOptions{
		Scope:     tokens.ScopeWorker,
		TTL:       2 * time.Hour,
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	if token.Token == "" {
		t.Fatalf("expected token value to be returned")
	}

	if err := store.Consume(token.Token, tokens.ScopeWorker); err != nil {
		t.Fatalf("consume token: %v", err)
	}

	if err := store.Consume(token.Token, tokens.ScopeWorker); err == nil {
		t.Fatalf("expected second consumption to fail")
	}
}

func TestCreateTokenTTLValidation(t *testing.T) {
	store := tokens.NewMemoryStore()

	_, err := store.Create(tokens.CreateOptions{
		Scope:     tokens.ScopeWorker,
		TTL:       25 * time.Hour,
		CreatedBy: "tester",
	})
	if err == nil {
		t.Fatalf("expected error for TTL exceeding limit")
	}
}

func TestConsumeTokenScopeMismatch(t *testing.T) {
	store := tokens.NewMemoryStore()

	token, err := store.Create(tokens.CreateOptions{
		Scope:     tokens.ScopeControlPlane,
		TTL:       30 * time.Minute,
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	if err := store.Consume(token.Token, tokens.ScopeWorker); err == nil {
		t.Fatalf("expected scope mismatch error")
	}
}

func TestConsumeTokenExpired(t *testing.T) {
	store := tokens.NewMemoryStore()

	token, err := store.Create(tokens.CreateOptions{
		Scope:     tokens.ScopeWorker,
		TTL:       time.Minute,
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	store.ForceExpire(token.Token)

	if err := store.Consume(token.Token, tokens.ScopeWorker); err == nil {
		t.Fatalf("expected expiration error")
	}
}
