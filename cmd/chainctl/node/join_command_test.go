package node_test

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"

	nodecmd "github.com/dobrovols/chainctl/cmd/chainctl/node"
	"github.com/dobrovols/chainctl/pkg/tokens"
)

type fakeConsumer struct {
	consumed string
	scope    tokens.Scope
	err      error
}

func (f *fakeConsumer) Consume(token string, scope tokens.Scope) error {
	if f.err != nil {
		return f.err
	}
	f.consumed = token
	f.scope = scope
	return nil
}

func TestNodeJoinCommand_TextOutput(t *testing.T) {
	opts := nodecmd.JoinCommandOptions{
		ClusterEndpoint: "https://cluster.local",
		Role:            "worker",
		Token:           "id.secret",
		Output:          "text",
	}
	store := &fakeConsumer{}
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := nodecmd.RunJoinForTest(cmd, opts, store); err != nil {
		t.Fatalf("run join: %v", err)
	}

	if store.consumed != "id.secret" {
		t.Fatalf("expected token to be consumed")
	}
	if !bytes.Contains(out.Bytes(), []byte("Validated token")) {
		t.Fatalf("expected success message, got %s", out.String())
	}
}

func TestNodeJoinCommand_JSONOutput(t *testing.T) {
	opts := nodecmd.JoinCommandOptions{
		ClusterEndpoint: "https://cluster.local",
		Role:            "control-plane",
		Token:           "foo.bar",
		Output:          "json",
	}
	store := &fakeConsumer{}
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := nodecmd.RunJoinForTest(cmd, opts, store); err != nil {
		t.Fatalf("run join: %v", err)
	}

	if !bytes.Contains(out.Bytes(), []byte("\"status\": \"ready\"")) {
		t.Fatalf("expected json output, got %s", out.String())
	}
}

func TestNodeJoinCommand_ValidatesInputs(t *testing.T) {
	opts := nodecmd.JoinCommandOptions{Role: "worker", Token: "", ClusterEndpoint: "https://cluster"}
	err := nodecmd.RunJoinForTest(&cobra.Command{}, opts, &fakeConsumer{})
	if err == nil {
		t.Fatalf("expected error for missing token")
	}
	if err != nodecmd.ErrTokenRequired() {
		t.Fatalf("expected errTokenRequired, got %v", err)
	}

	opts = nodecmd.JoinCommandOptions{Role: "worker", Token: "x", ClusterEndpoint: ""}
	err = nodecmd.RunJoinForTest(&cobra.Command{}, opts, &fakeConsumer{})
	if err != nodecmd.ErrClusterEndpoint() {
		t.Fatalf("expected cluster endpoint error, got %v", err)
	}

	opts = nodecmd.JoinCommandOptions{Role: "invalid", Token: "x", ClusterEndpoint: "https://cluster"}
	err = nodecmd.RunJoinForTest(&cobra.Command{}, opts, &fakeConsumer{})
	if err == nil {
		t.Fatalf("expected invalid role error")
	}
}
