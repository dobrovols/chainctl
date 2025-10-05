package node_test

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/spf13/cobra"

	nodecmd "github.com/dobrovols/chainctl/cmd/chainctl/node"
	"github.com/dobrovols/chainctl/internal/cli"
	"github.com/dobrovols/chainctl/pkg/tokens"
)

type fakeTokenStore struct {
	created *tokens.CreatedToken
	err     error
}

func (f *fakeTokenStore) Create(opts tokens.CreateOptions) (*tokens.CreatedToken, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.created != nil {
		return f.created, nil
	}
	return &tokens.CreatedToken{
		ID:        "abc123",
		Scope:     opts.Scope,
		ExpiresAt: time.Now().Add(opts.TTL),
		Token:     "abc123.def456",
	}, nil
}

func TestNodeTokenCommand_TextOutput(t *testing.T) {
	root := cli.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"node", "token", "--role", "worker", "--ttl", "1h"})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if !bytes.Contains(out.Bytes(), []byte("Token:")) {
		t.Fatalf("expected token in output: %s", out.String())
	}
}

func TestNodeTokenCommand_JSONOutput(t *testing.T) {
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	opts := nodecmd.TokenCommandOptions{Role: "control-plane", TTL: "30m", Output: "json"}
	store := &fakeTokenStore{
		created: &tokens.CreatedToken{
			ID:        "id1",
			Scope:     tokens.ScopeControlPlane,
			ExpiresAt: time.Unix(0, 0).UTC(),
			Token:     "id1.secret",
		},
	}

	if err := nodecmd.RunTokenCreateForTest(cmd, opts, store); err != nil {
		t.Fatalf("run: %v", err)
	}

	if !bytes.Contains(out.Bytes(), []byte("\"token\": \"id1.secret\"")) {
		t.Fatalf("expected json token, got %s", out.String())
	}
}

func TestNodeTokenCommand_InvalidRole(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	err := nodecmd.RunTokenCreateForTest(cmd, nodecmd.TokenCommandOptions{Role: "invalid", TTL: "1h"}, &fakeTokenStore{})
	if err == nil {
		t.Fatalf("expected error for invalid role")
	}
	if !errors.Is(err, nodecmd.ErrInvalidRole()) {
		t.Fatalf("expected invalid role error, got %v", err)
	}
}
