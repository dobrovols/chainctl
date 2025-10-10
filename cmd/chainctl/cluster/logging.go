package cluster

import (
	"fmt"

	"github.com/dobrovols/chainctl/internal/cli/logging"
	"github.com/dobrovols/chainctl/pkg/telemetry"
)

const (
	stepInstall     = "install"
	stepBootstrap   = "bootstrap"
	stepHelm        = "helm"
	stepUpgrade     = "upgrade"
	stepUpgradePlan = "upgrade-plan"
)

func logWorkflowEntry(logger telemetry.StructuredLogger, step, message string, severity telemetry.Severity, metadata map[string]string, err error) {
	if logger == nil {
		return
	}
	entryMetadata := cloneMetadata(metadata)
	_ = logger.Emit(telemetry.Entry{
		Category: telemetry.CategoryWorkflow,
		Message:  message,
		Severity: severity,
		Step:     step,
		Metadata: entryMetadata,
		Error:    err,
	})
}

func logWorkflowStart(logger telemetry.StructuredLogger, step string, metadata map[string]string) {
	logWorkflowEntry(logger, step, fmt.Sprintf("%s workflow started", step), telemetry.SeverityInfo, metadata, nil)
}

func logWorkflowSuccess(logger telemetry.StructuredLogger, step string, metadata map[string]string) {
	logWorkflowEntry(logger, step, fmt.Sprintf("%s workflow completed", step), telemetry.SeverityInfo, metadata, nil)
}

func logWorkflowFailure(logger telemetry.StructuredLogger, step string, metadata map[string]string, err error) {
	logWorkflowEntry(logger, step, fmt.Sprintf("%s workflow failed", step), telemetry.SeverityError, metadata, err)
}

func logCommandEntry(logger telemetry.StructuredLogger, step string, args []string, stderr string, severity telemetry.Severity, metadata map[string]string, err error) {
	if logger == nil {
		return
	}
	sanitizedCommand := logging.SanitizeCommand(args)
	sanitizedStderr := logging.SanitizeText(stderr)
	entry := telemetry.Entry{
		Category:      telemetry.CategoryCommand,
		Message:       fmt.Sprintf("%s command", step),
		Severity:      severity,
		Command:       sanitizedCommand,
		StderrExcerpt: sanitizedStderr,
		Metadata:      cloneMetadata(metadata),
		Error:         err,
		Step:          step,
	}
	_ = logger.Emit(entry)
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
