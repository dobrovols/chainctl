package cli_test

import (
	"testing"

	"github.com/dobrovols/chainctl/internal/cli"
)

func TestNewRootCommandRegistersSubcommands(t *testing.T) {
	cmd := cli.NewRootCommand()
	if cmd.Use != "chainctl" {
		t.Fatalf("expected use chainctl, got %s", cmd.Use)
	}
	names := map[string]bool{}
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
	}
	for _, expected := range []string{"encrypt-values", "node", "cluster", "app"} {
		if !names[expected] {
			t.Fatalf("expected subcommand %s to be registered", expected)
		}
	}
}
