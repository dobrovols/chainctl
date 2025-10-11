package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	internalconfig "github.com/dobrovols/chainctl/internal/config"
	pkgconfig "github.com/dobrovols/chainctl/pkg/config"
)

const (
	loaderTestCommand = "chainctl cluster install"
	loaderTestConfig  = "chainctl.yaml"
)

func TestLoadProfileParsesYAMLDocument(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, loaderTestConfig)
	writeConfigFile(t, path, `
metadata:
  name: shared-demo
  description: Example profile
defaults:
  namespace: demo
  bundle-path: ./bundle.tgz
profiles:
  staging:
    namespace: staging
commands:
  chainctl cluster install:
    profiles:
      - staging
    flags:
      chart: oci://example/cluster:1.0.0
      dry-run: true
      values-file: ./values.enc
`)

	catalog := fakeCatalog{
		commands: map[string]map[string]internalconfig.FlagType{
			"chainctl cluster install": {
				"namespace":         internalconfig.FlagTypeString,
				"bundle-path":       internalconfig.FlagTypeString,
				"chart":             internalconfig.FlagTypeString,
				"dry-run":           internalconfig.FlagTypeBool,
				"values-file":       internalconfig.FlagTypeString,
				"values-passphrase": internalconfig.FlagTypeString,
			},
		},
	}

	loader := internalconfig.NewLoader(catalog)
	profile, err := loader.Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if profile == nil {
		t.Fatalf("expected profile, got nil")
	}
	if profile.SourcePath != path {
		t.Fatalf("expected source path %q, got %q", path, profile.SourcePath)
	}
	if profile.Metadata.Name != "shared-demo" {
		t.Fatalf("expected metadata name to be shared-demo, got %s", profile.Metadata.Name)
	}
	if profile.Defaults["namespace"].Value != "demo" {
		t.Fatalf("expected default namespace demo, got %v", profile.Defaults["namespace"].Value)
	}
	install := profile.Commands[loaderTestCommand]
	if len(install.Profiles) != 1 || install.Profiles[0] != "staging" {
		t.Fatalf("expected staging profile reference, got %#v", install.Profiles)
	}
	if install.Flags["dry-run"].Value != true {
		t.Fatalf("expected dry-run true, got %v", install.Flags["dry-run"].Value)
	}
	if install.Flags["chart"].Source != pkgconfig.ValueSourceCommand {
		t.Fatalf("expected chart source command, got %s", install.Flags["chart"].Source)
	}
}

func TestLoadProfileRejectsSecrets(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, loaderTestConfig)
	writeConfigFile(t, path, `
defaults:
  values-passphrase: super-secret
commands:
  chainctl cluster install:
    flags:
      chart: oci://example/cluster:1.0.0
`)

	catalog := fakeCatalog{
		commands: map[string]map[string]internalconfig.FlagType{
			loaderTestCommand: {
				"chart":             internalconfig.FlagTypeString,
				"values-passphrase": internalconfig.FlagTypeString,
			},
		},
	}

	loader := internalconfig.NewLoader(catalog)
	_, err := loader.Load(path)
	if !errors.Is(err, internalconfig.ErrSecretsDisallowed) {
		t.Fatalf("expected ErrSecretsDisallowed, got %v", err)
	}
}

func TestLoadProfileRejectsUnknownCommand(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, loaderTestConfig)
	writeConfigFile(t, path, `
commands:
  chainctl invalid action:
    flags:
      chart: oci://example/cluster:1.0.0
`)

	catalog := fakeCatalog{
		commands: map[string]map[string]internalconfig.FlagType{},
	}

	loader := internalconfig.NewLoader(catalog)
	_, err := loader.Load(path)
	if !errors.Is(err, internalconfig.ErrUnknownCommand) {
		t.Fatalf("expected ErrUnknownCommand, got %v", err)
	}
}

func TestLoadProfileRejectsUnknownFlag(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, loaderTestConfig)
	writeConfigFile(t, path, `
commands:
  chainctl cluster install:
    flags:
      unknown-flag: value
`)

	catalog := fakeCatalog{
		commands: map[string]map[string]internalconfig.FlagType{
			loaderTestCommand: {
				"chart": internalconfig.FlagTypeString,
			},
		},
	}

	loader := internalconfig.NewLoader(catalog)
	_, err := loader.Load(path)
	if !errors.Is(err, internalconfig.ErrUnknownFlag) {
		t.Fatalf("expected ErrUnknownFlag, got %v", err)
	}
}

func TestCoerceValueParsesBooleanString(t *testing.T) {
	catalog := fakeCatalog{
		commands: map[string]map[string]internalconfig.FlagType{
			loaderTestCommand: {
				"dry-run": internalconfig.FlagTypeBool,
			},
		},
	}
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, loaderTestConfig)
	writeConfigFile(t, path, `
commands:
  chainctl cluster install:
    flags:
      dry-run: "true"
`)
	loader := internalconfig.NewLoader(catalog)
	profile, err := loader.Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if v := profile.Commands[loaderTestCommand].Flags["dry-run"].Value; v != true {
		t.Fatalf("expected dry-run true, got %v", v)
	}
}

func TestCoerceValueStringSliceVariants(t *testing.T) {
	catalog := fakeCatalog{
		commands: map[string]map[string]internalconfig.FlagType{
			loaderTestCommand: {
				"roles": internalconfig.FlagTypeStringSlice,
			},
		},
	}
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "chainctl.yaml")
	writeConfigFile(t, path, `
commands:
  chainctl cluster install:
    flags:
      roles:
        - operator
        - 42
`)
	loader := internalconfig.NewLoader(catalog)
	profile, err := loader.Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	value := profile.Commands[loaderTestCommand].Flags["roles"].Value
	list, ok := value.([]string)
	if !ok {
		t.Fatalf("expected string slice, got %T", value)
	}
	if len(list) != 2 || list[0] != "operator" || list[1] != "42" {
		t.Fatalf("unexpected roles slice %#v", list)
	}
}

func TestIsSensitiveMatchesSubstrings(t *testing.T) {
	t.Helper()
	catalog := fakeCatalog{
		commands: map[string]map[string]internalconfig.FlagType{
			"chainctl cluster install": {
				"cluster-secret-token": internalconfig.FlagTypeString,
			},
		},
	}
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "chainctl.yaml")
	writeConfigFile(t, path, `
commands:
  chainctl cluster install:
    flags:
      cluster-secret-token: value
`)
	loader := internalconfig.NewLoader(catalog)
	if _, err := loader.Load(path); !errors.Is(err, internalconfig.ErrSecretsDisallowed) {
		t.Fatalf("expected ErrSecretsDisallowed for secret token, got %v", err)
	}
}

func writeConfigFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := mustMkdirAll(path); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func mustMkdirAll(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}

type fakeCatalog struct {
	commands map[string]map[string]internalconfig.FlagType
}

func (c fakeCatalog) IsCommandSupported(command string) bool {
	_, ok := c.commands[command]
	return ok
}

func (c fakeCatalog) FlagType(command, flag string) (internalconfig.FlagType, bool) {
	flags, ok := c.commands[command]
	if !ok {
		return 0, false
	}
	t, ok := flags[flag]
	return t, ok
}

func (c fakeCatalog) AnyFlagType(flag string) (internalconfig.FlagType, bool) {
	for _, flags := range c.commands {
		if t, ok := flags[flag]; ok {
			return t, true
		}
	}
	return 0, false
}

func (c fakeCatalog) Commands() []string {
	out := make([]string, 0, len(c.commands))
	for name := range c.commands {
		out = append(out, name)
	}
	return out
}
