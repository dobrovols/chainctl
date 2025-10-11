package config

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// FlagType describes the expected Go type for a flag value.
type FlagType int

const (
	FlagTypeString FlagType = iota
	FlagTypeBool
	FlagTypeStringSlice
)

// FlagCatalog exposes metadata about supported commands and flags.
type FlagCatalog interface {
	IsCommandSupported(command string) bool
	FlagType(command, flag string) (FlagType, bool)
	AnyFlagType(flag string) (FlagType, bool)
	Commands() []string
}

// cobraCatalog implements FlagCatalog using a Cobra command tree.
type cobraCatalog struct {
	commands map[string]map[string]FlagType
	index    map[string]FlagType
	order    []string
}

// NewCobraCatalog builds a flag catalog from the provided Cobra root command.
func NewCobraCatalog(root *cobra.Command) FlagCatalog {
	c := &cobraCatalog{
		commands: make(map[string]map[string]FlagType),
		index:    make(map[string]FlagType),
	}
	traverseCommands(root, nil, c)
	return c
}

func traverseCommands(cmd *cobra.Command, parents []string, catalog *cobraCatalog) {
	path := append(parents, cmd.Use)
	commandPath := strings.Join(path, " ")
	flags := make(map[string]FlagType)

	cmd.InheritedFlags().VisitAll(func(flag *pflag.Flag) {
		flags[flag.Name] = flagType(flag)
	})

	cmd.NonInheritedFlags().VisitAll(func(flag *pflag.Flag) {
		flags[flag.Name] = flagType(flag)
	})

	catalog.commands[commandPath] = flags
	catalog.order = append(catalog.order, commandPath)

	for name, flagType := range flags {
		if _, ok := catalog.index[name]; !ok {
			catalog.index[name] = flagType
		}
	}

	for _, child := range cmd.Commands() {
		if !child.Hidden {
			traverseCommands(child, path, catalog)
		}
	}
}

func (c *cobraCatalog) IsCommandSupported(command string) bool {
	_, ok := c.commands[command]
	return ok
}

func (c *cobraCatalog) FlagType(command, flag string) (FlagType, bool) {
	if flags, ok := c.commands[command]; ok {
		t, found := flags[flag]
		return t, found
	}
	return 0, false
}

func (c *cobraCatalog) AnyFlagType(flag string) (FlagType, bool) {
	t, ok := c.index[flag]
	return t, ok
}

func (c *cobraCatalog) Commands() []string {
	return append([]string(nil), c.order...)
}

func flagType(flag *pflag.Flag) FlagType {
	switch flag.Value.Type() {
	case "bool":
		return FlagTypeBool
	case "stringSlice", "stringArray":
		return FlagTypeStringSlice
	default:
		return FlagTypeString
	}
}
