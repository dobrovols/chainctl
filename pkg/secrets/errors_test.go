package secrets_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/dobrovols/chainctl/pkg/secrets"
)

func TestErrorWrapper(t *testing.T) {
	inner := errors.New("boom")
	err := secrets.NewError(secrets.ErrCodeValidation, inner)
	if err.Error() != "boom" {
		t.Fatalf("unexpected error string: %s", err.Error())
	}
	if !errors.Is(err, inner) {
		t.Fatalf("expected unwrap to match inner error")
	}
	if !strings.Contains(err.String(), "code=") {
		t.Fatalf("expected String to include code, got %s", err.String())
	}
}
