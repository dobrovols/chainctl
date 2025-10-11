package declarative

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	internalconfig "github.com/dobrovols/chainctl/internal/config"
	pkgconfig "github.com/dobrovols/chainctl/pkg/config"
	"github.com/dobrovols/chainctl/pkg/telemetry"
)

const configFlagName = "config"

// Annotation keys used to mark commands that participate in declarative configuration.
const (
	AnnotationEnabled = "declarative-config"
)

// Manager wires declarative configuration discovery and application into annotated commands.
type Manager struct {
	catalog internalconfig.FlagCatalog
	loader  *internalconfig.Loader
}

// NewManager constructs a manager for the provided root command.
func NewManager(root *cobra.Command) *Manager {
	catalog := internalconfig.NewCobraCatalog(root)
	return &Manager{
		catalog: catalog,
		loader:  internalconfig.NewLoader(catalog),
	}
}

// Bind walks the command tree and attaches declarative configuration behaviour to each annotated command.
func (m *Manager) Bind(root *cobra.Command) {
	walkCommands(root, func(cmd *cobra.Command) {
		if cmd.Annotations == nil || strings.ToLower(cmd.Annotations[AnnotationEnabled]) != "true" {
			return
		}
		if cmd.Flags().Lookup(configFlagName) == nil {
			cmd.Flags().String(configFlagName, "", "Path to declarative configuration file")
		}
		existing := cmd.PreRunE
		cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
			if err := m.apply(cmd); err != nil {
				return err
			}
			if existing != nil {
				return existing(cmd, args)
			}
			return nil
		}
	})
}

func (m *Manager) apply(cmd *cobra.Command) error {
	flagSet := cmd.Flags()
	if flagSet == nil {
		return nil
	}

	location, err := resolveConfigLocation(flagSet)
	if err != nil {
		return err
	}
	if location == nil {
		return nil
	}

	runtime, err := collectRuntimeOverrides(cmd)
	if err != nil {
		return err
	}

	resolved, err := m.buildResolvedInvocation(location.Path, cmd, runtime)
	if err != nil {
		return err
	}

	if err := applyResolvedFlags(cmd, resolved); err != nil {
		return err
	}

	storeResolvedInvocation(cmd, resolved)

	return emitInvocationSummary(cmd, flagSet, resolved)
}

func resolveConfigLocation(flagSet *pflag.FlagSet) (*internalconfig.LocationResult, error) {
	explicitPath, err := flagSet.GetString(configFlagName)
	if err != nil {
		return nil, fmt.Errorf("read --config flag: %w", err)
	}

	if strings.TrimSpace(explicitPath) != "" {
		location, locErr := internalconfig.LocateConfig(explicitPath)
		if locErr != nil {
			return nil, locErr
		}
		return &location, nil
	}

	location, err := internalconfig.LocateConfig("")
	if err != nil {
		if errors.Is(err, internalconfig.ErrConfigNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &location, nil
}

func (m *Manager) buildResolvedInvocation(configPath string, cmd *cobra.Command, runtime pkgconfig.FlagSet) (*pkgconfig.ResolvedInvocation, error) {
	profile, err := m.loader.Load(configPath)
	if err != nil {
		return nil, err
	}
	commandPath := cmd.CommandPath()
	return pkgconfig.ResolveInvocation(profile, commandPath, runtime)
}

func emitInvocationSummary(cmd *cobra.Command, flagSet *pflag.FlagSet, resolved *pkgconfig.ResolvedInvocation) error {
	format := determineOutputFormat(flagSet)
	summary, err := pkgconfig.FormatSummary(resolved, format)
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.ErrOrStderr(), summary)
	return nil
}

func determineOutputFormat(flagSet *pflag.FlagSet) string {
	if flagSet == nil {
		return pkgconfig.SummaryFormatText
	}
	format, _ := flagSet.GetString("output")
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		return pkgconfig.SummaryFormatText
	}
	if format != pkgconfig.SummaryFormatText && format != pkgconfig.SummaryFormatJSON {
		return pkgconfig.SummaryFormatText
	}
	return format
}

func collectRuntimeOverrides(cmd *cobra.Command) (pkgconfig.FlagSet, error) {
	runtime := pkgconfig.FlagSet{}
	var firstErr error
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if firstErr != nil || shouldSkipRuntimeFlag(flag) {
			return
		}
		if err := captureRuntimeFlag(cmd, flag, runtime); err != nil {
			firstErr = err
		}
	})
	if firstErr != nil {
		return nil, firstErr
	}
	if len(runtime) == 0 {
		return nil, nil
	}
	return runtime, nil
}

func applyResolvedFlags(cmd *cobra.Command, resolved *pkgconfig.ResolvedInvocation) error {
	for name, flagValue := range resolved.Flags {
		if err := applyResolvedFlag(cmd, resolved, name, flagValue); err != nil {
			return err
		}
	}
	return nil
}

func shouldSkipRuntimeFlag(flag *pflag.Flag) bool {
	return !flag.Changed || flag.Name == configFlagName
}

func captureRuntimeFlag(cmd *cobra.Command, flag *pflag.Flag, runtime pkgconfig.FlagSet) error {
	value, err := readRuntimeFlagValue(cmd, flag)
	if err != nil {
		return err
	}
	runtime[flag.Name] = pkgconfig.FlagValue{Value: value, Source: pkgconfig.ValueSourceRuntime}
	return nil
}

func readRuntimeFlagValue(cmd *cobra.Command, flag *pflag.Flag) (any, error) {
	switch flag.Value.Type() {
	case "bool":
		return cmd.Flags().GetBool(flag.Name)
	case "stringSlice", "stringArray":
		value, err := cmd.Flags().GetStringSlice(flag.Name)
		if err != nil {
			return nil, err
		}
		return append([]string(nil), value...), nil
	default:
		return cmd.Flags().GetString(flag.Name)
	}
}

func applyResolvedFlag(cmd *cobra.Command, resolved *pkgconfig.ResolvedInvocation, name string, flagValue pkgconfig.FlagValue) error {
	if name == configFlagName {
		return nil
	}
	flag := cmd.Flags().Lookup(name)
	if flag == nil {
		resolved.Warnings = append(resolved.Warnings, fmt.Sprintf("flag %q ignored (not recognised by command)", name))
		return nil
	}
	value, err := formatResolvedFlagValue(flag, flagValue.Value, name)
	if err != nil {
		return err
	}
	if err := cmd.Flags().Set(name, value); err != nil {
		return fmt.Errorf("apply flag %q: %w", name, err)
	}
	return nil
}

func formatResolvedFlagValue(flag *pflag.Flag, raw any, name string) (string, error) {
	switch flag.Value.Type() {
	case "bool":
		boolVal, ok := raw.(bool)
		if !ok {
			return "", fmt.Errorf("%w: %s expects boolean", internalconfig.ErrInvalidFlagType, name)
		}
		return fmt.Sprintf("%t", boolVal), nil
	case "stringSlice", "stringArray":
		slice, err := toStringSlice(raw)
		if err != nil {
			return "", fmt.Errorf("%w: %s expects string list", internalconfig.ErrInvalidFlagType, name)
		}
		return strings.Join(slice, ","), nil
	default:
		return fmt.Sprint(raw), nil
	}
}

func toStringSlice(value any) ([]string, error) {
	switch v := value.(type) {
	case []string:
		return append([]string(nil), v...), nil
	case []interface{}:
		out := make([]string, len(v))
		for i, item := range v {
			out[i] = fmt.Sprint(item)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported slice type %T", value)
	}
}

func walkCommands(cmd *cobra.Command, fn func(*cobra.Command)) {
	fn(cmd)
	for _, child := range cmd.Commands() {
		walkCommands(child, fn)
	}
}

type resolvedContextKey struct{}

func storeResolvedInvocation(cmd *cobra.Command, resolved *pkgconfig.ResolvedInvocation) {
	if cmd == nil || resolved == nil {
		return
	}
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	cmd.SetContext(context.WithValue(ctx, resolvedContextKey{}, resolved))
}

// ResolvedInvocationFromContext retrieves the resolved invocation stored for the command.
func ResolvedInvocationFromContext(cmd *cobra.Command) (*pkgconfig.ResolvedInvocation, bool) {
	if cmd == nil {
		return nil, false
	}
	if ctx := cmd.Context(); ctx != nil {
		if resolved, ok := ctx.Value(resolvedContextKey{}).(*pkgconfig.ResolvedInvocation); ok {
			return resolved, true
		}
	}
	return nil, false
}

// EmitTelemetry emits a configuration summary via the provided structured logger.
func EmitTelemetry(logger telemetry.StructuredLogger, resolved *pkgconfig.ResolvedInvocation) {
	if logger == nil || resolved == nil {
		return
	}

	metadata := map[string]string{
		"command": resolved.CommandPath,
	}
	if resolved.SourcePath != "" {
		metadata["sourcePath"] = resolved.SourcePath
	}
	if len(resolved.Profiles) > 0 {
		metadata["profiles"] = strings.Join(resolved.Profiles, ",")
	}
	if len(resolved.Overrides) > 0 {
		metadata["overrides"] = strings.Join(resolved.Overrides, ",")
	}
	for name, value := range resolved.Flags {
		metadata["flag."+name] = fmt.Sprint(value.Value)
		metadata["flag."+name+".source"] = string(value.Source)
	}

	_ = logger.Emit(telemetry.Entry{
		Category: telemetry.CategoryConfig,
		Message:  "declarative configuration resolved",
		Severity: telemetry.SeverityInfo,
		Metadata: metadata,
	})
}
