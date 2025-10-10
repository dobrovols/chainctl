package node

import "github.com/dobrovols/chainctl/pkg/telemetry"

const (
	stepNodeJoin    = "node-join"
	stepTokenCreate = "token-create"
)

func logWorkflowStart(logger telemetry.StructuredLogger, step string, metadata map[string]string) {
	logWorkflowEntry(logger, step, step+" workflow started", telemetry.SeverityInfo, metadata, nil)
}

func logWorkflowSuccess(logger telemetry.StructuredLogger, step string, metadata map[string]string) {
	logWorkflowEntry(logger, step, step+" workflow completed", telemetry.SeverityInfo, metadata, nil)
}

func logWorkflowFailure(logger telemetry.StructuredLogger, step string, metadata map[string]string, err error) {
	logWorkflowEntry(logger, step, step+" workflow failed", telemetry.SeverityError, metadata, err)
}

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
