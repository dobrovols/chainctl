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

	explicitPath, err := flagSet.GetString(configFlagName)
	if err != nil {
		return fmt.Errorf("read --config flag: %w", err)
	}

	var (
		location internalconfig.LocationResult
		locErr   error
	)
	if strings.TrimSpace(explicitPath) != "" {
		location, locErr = internalconfig.LocateConfig(explicitPath)
	} else {
		location, locErr = internalconfig.LocateConfig("")
		if errors.Is(locErr, internalconfig.ErrConfigNotFound) {
			return nil
		}
	}
	if locErr != nil {
		return locErr
	}

	runtime, err := collectRuntimeOverrides(cmd)
	if err != nil {
		return err
	}

	profile, err := m.loader.Load(location.Path)
	if err != nil {
		return err
	}

	commandPath := cmd.CommandPath()
	resolved, err := pkgconfig.ResolveInvocation(profile, commandPath, runtime)
	if err != nil {
		return err
	}

	if err := applyResolvedFlags(cmd, resolved); err != nil {
		return err
	}

	storeResolvedInvocation(cmd, resolved)

	outputFormat, _ := flagSet.GetString("output")
	if outputFormat == "" {
		outputFormat = pkgconfig.SummaryFormatText
	} else {
		outputFormat = strings.ToLower(outputFormat)
	}
	if outputFormat != pkgconfig.SummaryFormatText && outputFormat != pkgconfig.SummaryFormatJSON {
		outputFormat = pkgconfig.SummaryFormatText
	}

	summary, err := pkgconfig.FormatSummary(resolved, outputFormat)
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.ErrOrStderr(), summary)
	return nil
}

func collectRuntimeOverrides(cmd *cobra.Command) (pkgconfig.FlagSet, error) {
	runtime := pkgconfig.FlagSet{}
	var firstErr error
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if !flag.Changed || flag.Name == configFlagName || firstErr != nil {
			return
		}
		switch flag.Value.Type() {
		case "bool":
			val, err := cmd.Flags().GetBool(flag.Name)
			if err != nil {
				firstErr = err
				return
			}
			runtime[flag.Name] = pkgconfig.FlagValue{Value: val, Source: pkgconfig.ValueSourceRuntime}
		case "stringSlice", "stringArray":
			val, err := cmd.Flags().GetStringSlice(flag.Name)
			if err != nil {
				firstErr = err
				return
			}
			runtime[flag.Name] = pkgconfig.FlagValue{Value: append([]string(nil), val...), Source: pkgconfig.ValueSourceRuntime}
		default:
			val, err := cmd.Flags().GetString(flag.Name)
			if err != nil {
				firstErr = err
				return
			}
			runtime[flag.Name] = pkgconfig.FlagValue{Value: val, Source: pkgconfig.ValueSourceRuntime}
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
		if name == configFlagName {
			continue
		}
		flag := cmd.Flags().Lookup(name)
		if flag == nil {
			resolved.Warnings = append(resolved.Warnings, fmt.Sprintf("flag %q ignored (not recognised by command)", name))
			continue
		}
		var value string
		switch flag.Value.Type() {
		case "bool":
			boolVal, ok := flagValue.Value.(bool)
			if !ok {
				return fmt.Errorf("%w: %s expects boolean", internalconfig.ErrInvalidFlagType, name)
			}
			if err := cmd.Flags().Set(name, fmt.Sprintf("%t", boolVal)); err != nil {
				return fmt.Errorf("apply flag %q: %w", name, err)
			}
			continue
		case "stringSlice", "stringArray":
			slice, err := toStringSlice(flagValue.Value)
			if err != nil {
				return fmt.Errorf("%w: %s expects string list", internalconfig.ErrInvalidFlagType, name)
			}
			value = strings.Join(slice, ",")
		default:
			value = fmt.Sprint(flagValue.Value)
		}
		if err := cmd.Flags().Set(name, value); err != nil {
			return fmt.Errorf("apply flag %q: %w", name, err)
		}
	}
	return nil
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
