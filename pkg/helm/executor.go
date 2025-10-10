package helm

import (
	"github.com/dobrovols/chainctl/internal/cli/logging"
	"github.com/dobrovols/chainctl/internal/config"
	"github.com/dobrovols/chainctl/pkg/bundle"
	"github.com/dobrovols/chainctl/pkg/telemetry"
)

// loggingExecutor decorates an Executor with structured logging.
type loggingExecutor struct {
	next   Executor
	logger telemetry.StructuredLogger
}

// NewLoggingExecutor wraps the provided executor with structured logging support.
// If exec is nil, a noop executor is used. When logger is nil, the original executor is returned.
func NewLoggingExecutor(exec Executor, logger telemetry.StructuredLogger) Executor {
	if logger == nil {
		if exec == nil {
			exec = noopExecutor{}
		}
		return exec
	}
	if exec == nil {
		exec = noopExecutor{}
	}
	return &loggingExecutor{next: exec, logger: logger}
}

func (l *loggingExecutor) UpgradeRelease(profile *config.Profile, b *bundle.Bundle) error {
	args := buildHelmArgs(profile)
	metadata := map[string]string{}
	if profile.HelmNamespace != "" {
		metadata["namespace"] = profile.HelmNamespace
	}
	if profile.HelmRelease != "" {
		metadata["release"] = profile.HelmRelease
	}
	if profile.BundlePath != "" {
		metadata["bundlePath"] = profile.BundlePath
	}
	if b != nil && b.Manifest.Version != "" {
		metadata["bundleVersion"] = b.Manifest.Version
	}

	sanitized := logging.SanitizeCommand(args)
	entry := telemetry.Entry{
		Category: telemetry.CategoryCommand,
		Message:  "helm upgrade start",
		Severity: telemetry.SeverityInfo,
		Command:  sanitized,
		Metadata: cloneMetadata(metadata),
		Step:     "helm",
	}
	_ = l.logger.Emit(entry)

	err := l.next.UpgradeRelease(profile, b)
	severity := telemetry.SeverityInfo
	if err != nil {
		severity = telemetry.SeverityError
	}
	sanitizedErr := ""
	if err != nil {
		sanitizedErr = logging.SanitizeText(err.Error())
	}
	entry = telemetry.Entry{
		Category:      telemetry.CategoryCommand,
		Message:       "helm upgrade complete",
		Severity:      severity,
		Command:       sanitized,
		Metadata:      cloneMetadata(metadata),
		StderrExcerpt: sanitizedErr,
		Error:         err,
		Step:          "helm",
	}
	_ = l.logger.Emit(entry)

	return err
}

func buildHelmArgs(profile *config.Profile) []string {
	args := []string{"helm", "upgrade"}
	if profile.HelmRelease != "" {
		args = append(args, profile.HelmRelease)
	}
	if profile.HelmNamespace != "" {
		args = append(args, "--namespace", profile.HelmNamespace)
	}
	if profile.EncryptedFile != "" {
		args = append(args, "--values", profile.EncryptedFile)
	}
	if profile.Passphrase != "" {
		args = append(args, "--values-passphrase", profile.Passphrase)
	}
	if profile.BundlePath != "" {
		args = append(args, "--bundle-path", profile.BundlePath)
	}
	return args
}

func cloneMetadata(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
