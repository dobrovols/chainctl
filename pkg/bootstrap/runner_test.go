package bootstrap

import (
	"errors"
	"reflect"
	"testing"

	"github.com/dobrovols/chainctl/pkg/telemetry"
)

type fakeStructuredLogger struct {
	entries []telemetry.Entry
}

func (f *fakeStructuredLogger) Emit(entry telemetry.Entry) error {
	f.entries = append(f.entries, entry)
	return nil
}

func TestLoggingRunnerLogsSuccessfulCommand(t *testing.T) {
	expectedCmd := []string{"helm", "upgrade", "myapp"}
	env := map[string]string{"TOKEN": "secret", "NAMESPACE": "demo"}
	logger := &fakeStructuredLogger{}

	runner := NewLoggingRunner(func(cmd []string, receivedEnv map[string]string) CommandResult {
		if !reflect.DeepEqual(cmd, expectedCmd) {
			t.Fatalf("unexpected command execution: %v", cmd)
		}
		if !reflect.DeepEqual(receivedEnv, env) {
			t.Fatalf("unexpected environment passed to executor: %v", receivedEnv)
		}
		return CommandResult{ExitCode: 0}
	}, logger,
		func(args []string) string { return "sanitized command" },
		func(m map[string]string) map[string]string {
			return map[string]string{"TOKEN": "***", "NAMESPACE": "demo"}
		},
		func(out string) string { return out },
		64,
	)

	if err := runner.Run(expectedCmd, env); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if len(logger.entries) != 2 {
		t.Fatalf("expected two log entries (start + completion), got %d", len(logger.entries))
	}

	start := logger.entries[0]
	if start.Severity != telemetry.SeverityInfo {
		t.Fatalf("expected start severity info, got %s", start.Severity)
	}
	if start.Command != "sanitized command" {
		t.Fatalf("expected sanitized command in start entry, got %q", start.Command)
	}

	complete := logger.entries[1]
	if complete.Severity != telemetry.SeverityInfo {
		t.Fatalf("expected completion severity info, got %s", complete.Severity)
	}
	if complete.Command != "sanitized command" {
		t.Fatalf("expected sanitized command in completion entry, got %q", complete.Command)
	}
	if exit := complete.Metadata["exitCode"]; exit != "0" {
		t.Fatalf("expected exit code 0, got %q", exit)
	}
	if complete.StderrExcerpt != "" {
		t.Fatalf("expected empty stderr excerpt on success, got %q", complete.StderrExcerpt)
	}
}

func TestLoggingRunnerLogsFailureWithSanitizedStderr(t *testing.T) {
	expectedCmd := []string{"sh", "-c", "do something"}
	env := map[string]string{"PASSWORD": "hunter2"}
	logger := &fakeStructuredLogger{}

	runner := NewLoggingRunner(func(cmd []string, receivedEnv map[string]string) CommandResult {
		return CommandResult{
			ExitCode: 23,
			Stderr:   "token=abc123\npermission denied",
			Err:      errors.New("execution failed"),
		}
	}, logger,
		func(args []string) string { return "sanitized exec" },
		func(m map[string]string) map[string]string { return map[string]string{"PASSWORD": "***"} },
		func(out string) string { return "sanitized stderr" },
		32,
	)

	err := runner.Run(expectedCmd, env)
	if err == nil {
		t.Fatalf("expected error from failing command")
	}

	if len(logger.entries) != 2 {
		t.Fatalf("expected two log entries, got %d", len(logger.entries))
	}

	complete := logger.entries[1]
	if complete.Severity != telemetry.SeverityError {
		t.Fatalf("expected error severity, got %s", complete.Severity)
	}
	if complete.Command != "sanitized exec" {
		t.Fatalf("expected sanitized command, got %q", complete.Command)
	}
	if exit := complete.Metadata["exitCode"]; exit != "23" {
		t.Fatalf("expected exit code 23, got %q", exit)
	}
	if complete.StderrExcerpt != "sanitized stderr" {
		t.Fatalf("expected sanitized stderr excerpt, got %q", complete.StderrExcerpt)
	}
}
