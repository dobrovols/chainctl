package config

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestCobraCatalogCapturesCommandsAndFlags(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	root.PersistentFlags().String("global", "", "global flag")

	install := &cobra.Command{Use: "install"}
	install.Flags().Bool("dry-run", false, "dry run")
	install.Flags().StringSlice("roles", []string{}, "roles")

	hidden := &cobra.Command{Use: "hidden", Hidden: true}

	root.AddCommand(install, hidden)

	catalog := NewCobraCatalog(root)

	commands := catalog.Commands()
	if len(commands) != 2 {
		t.Fatalf("expected 2 commands (root and root install), got %d: %#v", len(commands), commands)
	}
	if !catalog.IsCommandSupported("root") {
		t.Fatalf("expected root command to be supported")
	}
	if !catalog.IsCommandSupported("root install") {
		t.Fatalf("expected root install command to be supported")
	}
	if catalog.IsCommandSupported("root hidden") {
		t.Fatalf("expected hidden command to be skipped")
	}

	if flagType, ok := catalog.FlagType("root install", "dry-run"); !ok || flagType != FlagTypeBool {
		t.Fatalf("expected dry-run flag to be bool, got ok=%t type=%v", ok, flagType)
	}
	if flagType, ok := catalog.FlagType("root install", "roles"); !ok || flagType != FlagTypeStringSlice {
		t.Fatalf("expected roles flag to be string slice, got ok=%t type=%v", ok, flagType)
	}
	if _, ok := catalog.FlagType("root install", "nonexistent"); ok {
		t.Fatalf("expected nonexistent flag lookup to fail")
	}

	if flagType, ok := catalog.FlagType("root install", "global"); !ok || flagType != FlagTypeString {
		t.Fatalf("expected inherited global flag to be string, got ok=%t type=%v", ok, flagType)
	}
	if flagType, ok := catalog.AnyFlagType("global"); !ok || flagType != FlagTypeString {
		t.Fatalf("expected AnyFlagType to return string for global flag, got ok=%t type=%v", ok, flagType)
	}
}
