package declarative

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	pkgconfig "github.com/dobrovols/chainctl/pkg/config"
	"github.com/dobrovols/chainctl/pkg/telemetry"
)

func TestBinderAppliesConfigAndRuntimeOverrides(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "chainctl.yaml")
	yaml := `
defaults:
  namespace: default-ns
  roles:
    - base
    - 42
commands:
  chainctl cluster install:
    flags:
      dry-run: "true"
      output: json
`
	if err := os.WriteFile(configPath, []byte(yaml), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	root := &cobra.Command{
		Use: "chainctl",
	}
	errBuf := new(bytes.Buffer)
	root.SetErr(errBuf)
	root.SetOut(&bytes.Buffer{})

	cluster := &cobra.Command{
		Use: "cluster",
	}
	cluster.SetErr(errBuf)
	cluster.SetOut(&bytes.Buffer{})

	install := &cobra.Command{
		Use: "install",
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, err := cmd.Flags().GetBool("dry-run")
			if err != nil {
				return err
			}
			if dryRun {
				t.Fatalf("expected runtime override to set dry-run=false")
			}

			namespace, err := cmd.Flags().GetString("namespace")
			if err != nil {
				return err
			}
			if namespace != "runtime-ns" {
				t.Fatalf("namespace = %q, want runtime-ns", namespace)
			}

			output, err := cmd.Flags().GetString("output")
			if err != nil {
				return err
			}
			if output != "json" {
				t.Fatalf("output = %q, want json from config file", output)
			}

			roles, err := cmd.Flags().GetStringSlice("roles")
			if err != nil {
				return err
			}
			expectedRoles := []string{"runtimeA", "runtimeB"}
			if len(roles) < 2 {
				t.Fatalf("roles = %#v, want at least two entries", roles)
			}
			if !reflect.DeepEqual(roles[len(roles)-2:], expectedRoles) {
				t.Fatalf("roles tail = %#v, want %#v", roles[len(roles)-2:], expectedRoles)
			}

			resolved, ok := ResolvedInvocationFromContext(cmd)
			if !ok {
				t.Fatalf("expected resolved invocation in context")
			}
			if resolved.Flags["namespace"].Source != pkgconfig.ValueSourceRuntime {
				t.Fatalf("namespace source = %s, want runtime", resolved.Flags["namespace"].Source)
			}
			if resolved.Flags["dry-run"].Source != pkgconfig.ValueSourceRuntime {
				t.Fatalf("dry-run source = %s, want runtime", resolved.Flags["dry-run"].Source)
			}
			if resolved.Flags["output"].Source != pkgconfig.ValueSourceCommand {
				t.Fatalf("output source = %s, want command", resolved.Flags["output"].Source)
			}
			if rolesVal, ok := resolved.Flags["roles"]; ok {
				if !reflect.DeepEqual(rolesVal.Value, expectedRoles) {
					t.Fatalf("resolved roles value = %#v, want %#v", rolesVal.Value, expectedRoles)
				}
			}
			return nil
		},
	}
	install.Annotations = map[string]string{AnnotationEnabled: "true"}
	install.SetErr(errBuf)
	install.SetOut(&bytes.Buffer{})

	install.Flags().String("namespace", "", "")
	install.Flags().Bool("dry-run", false, "")
	install.Flags().String("output", "text", "")
	install.Flags().StringSlice("roles", []string{}, "")

	cluster.AddCommand(install)
	root.AddCommand(cluster)

	t.Setenv("CHAINCTL_CONFIG", configPath)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")

	NewManager(root).Bind(root)

	root.SetArgs([]string{
		"cluster", "install",
		"--namespace", "runtime-ns",
		"--dry-run=false",
		"--roles", "runtimeA",
		"--roles", "runtimeB",
	})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute command: %v", err)
	}

	summary := errBuf.String()
	if !strings.Contains(summary, `"commandPath": "chainctl cluster install"`) {
		t.Fatalf("summary missing command path:\n%s", summary)
	}
	if !strings.Contains(summary, `"namespace"`) || !strings.Contains(summary, `"runtime-ns"`) {
		t.Fatalf("summary missing runtime namespace override:\n%s", summary)
	}
	if !strings.Contains(summary, `"roles"`) || !strings.Contains(summary, `"runtimeA"`) {
		t.Fatalf("summary missing runtime roles override:\n%s", summary)
	}
}

func TestEmitTelemetryIncludesFlagMetadata(t *testing.T) {
	logger := &stubStructuredLogger{}
	resolved := &pkgconfig.ResolvedInvocation{
		CommandPath: "chainctl cluster install",
		SourcePath:  "/configs/chainctl.yaml",
		Profiles:    []string{"staging"},
		Overrides:   []string{"runtime overrides namespace (was default)"},
		Flags: pkgconfig.FlagSet{
			"namespace": {Value: "demo", Source: pkgconfig.ValueSourceDefault},
			"dry-run":   {Value: true, Source: pkgconfig.ValueSourceRuntime},
		},
	}

	EmitTelemetry(logger, resolved)

	if len(logger.entries) != 1 {
		t.Fatalf("expected 1 telemetry entry, got %d", len(logger.entries))
	}
	entry := logger.entries[0]
	if entry.Category != telemetry.CategoryConfig {
		t.Fatalf("entry category = %s, want %s", entry.Category, telemetry.CategoryConfig)
	}
	if entry.Severity != telemetry.SeverityInfo {
		t.Fatalf("entry severity = %s, want info", entry.Severity)
	}
	if entry.Message == "" || !strings.Contains(entry.Message, "declarative configuration resolved") {
		t.Fatalf("unexpected message: %q", entry.Message)
	}
	if entry.Metadata["command"] != "chainctl cluster install" {
		t.Fatalf("metadata command = %s", entry.Metadata["command"])
	}
	if entry.Metadata["flag.dry-run"] != "true" {
		t.Fatalf("metadata flag.dry-run = %s", entry.Metadata["flag.dry-run"])
	}
	if entry.Metadata["flag.namespace.source"] != string(pkgconfig.ValueSourceDefault) {
		t.Fatalf("metadata flag.namespace.source = %s", entry.Metadata["flag.namespace.source"])
	}
}

type stubStructuredLogger struct {
	entries []telemetry.Entry
}

func (s *stubStructuredLogger) Emit(entry telemetry.Entry) error {
	s.entries = append(s.entries, entry)
	return nil
}

func TestCollectRuntimeOverridesErrorPath(t *testing.T) {
	cmd := &cobra.Command{Use: "root"}
	cmd.Flags().Int("count", 0, "int flag")
	// Mark as changed with string input; GetString will be invoked and should error.
	if err := cmd.Flags().Set("count", "5"); err != nil {
		t.Fatalf("set count flag: %v", err)
	}

	_, err := collectRuntimeOverrides(cmd)
	if err == nil {
		t.Fatalf("expected error when collecting non-string flag")
	}
}

func TestApplyResolvedFlagsHandlesUnknownAndTypeMismatch(t *testing.T) {
	cmd := &cobra.Command{Use: "root"}
	cmd.Flags().Bool("dry-run", false, "")

	resolved := &pkgconfig.ResolvedInvocation{
		Flags: pkgconfig.FlagSet{
			"dry-run": {Value: "not-bool", Source: pkgconfig.ValueSourceRuntime},
		},
	}
	if err := applyResolvedFlags(cmd, resolved); err == nil {
		t.Fatalf("expected type mismatch error for bool flag")
	}

	resolved.Flags["dry-run"] = pkgconfig.FlagValue{Value: true, Source: pkgconfig.ValueSourceRuntime}
	resolved.Flags["unknown"] = pkgconfig.FlagValue{Value: "value", Source: pkgconfig.ValueSourceRuntime}

	if err := applyResolvedFlags(cmd, resolved); err != nil {
		t.Fatalf("applyResolvedFlags returned error: %v", err)
	}
	if len(resolved.Warnings) == 0 {
		t.Fatalf("expected warning for unknown flag")
	}
	if !strings.Contains(resolved.Warnings[0], "unknown") {
		t.Fatalf("warning does not mention unknown flag: %s", resolved.Warnings[0])
	}
}

func TestToStringSliceVariants(t *testing.T) {
	values, err := toStringSlice([]interface{}{"one", 2})
	if err != nil {
		t.Fatalf("expected conversion success, got error: %v", err)
	}
	if !reflect.DeepEqual(values, []string{"one", "2"}) {
		t.Fatalf("converted values = %#v", values)
	}
	if _, err := toStringSlice(map[string]any{"bad": true}); err == nil {
		t.Fatalf("expected error for unsupported type")
	}
}

func TestStoreAndRetrieveResolvedInvocationNilSafety(t *testing.T) {
	storeResolvedInvocation(nil, nil)
	if _, ok := ResolvedInvocationFromContext(nil); ok {
		t.Fatalf("expected false for nil command")
	}

	cmd := &cobra.Command{Use: "root"}
	if _, ok := ResolvedInvocationFromContext(cmd); ok {
		t.Fatalf("expected false for command with no context")
	}
	expected := &pkgconfig.ResolvedInvocation{CommandPath: "root"}
	storeResolvedInvocation(cmd, expected)
	actual, ok := ResolvedInvocationFromContext(cmd)
	if !ok || actual != expected {
		t.Fatalf("expected resolved invocation to be stored and retrieved")
	}
}

func TestApplySkipsWhenConfigMissing(t *testing.T) {
	root := &cobra.Command{Use: "chainctl"}
	root.SetErr(&bytes.Buffer{})
	root.SetOut(&bytes.Buffer{})

	cmd := &cobra.Command{
		Use: "install",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	cmd.Annotations = map[string]string{AnnotationEnabled: "true"}
	cmd.Flags().Bool("dry-run", false, "")
	cmd.Flags().String("output", "text", "")
	root.AddCommand(cmd)

	manager := NewManager(root)
	manager.Bind(root)

	if err := root.Execute(); err != nil {
		t.Fatalf("execute without config: %v", err)
	}
}
