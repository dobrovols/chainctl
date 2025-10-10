package bootstrap

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	clilogging "github.com/dobrovols/chainctl/internal/cli/logging"
	"github.com/dobrovols/chainctl/pkg/telemetry"
)

// CommandResult captures the outcome of executing a command.
type CommandResult struct {
	ExitCode int
	Stderr   string
	Err      error
}

// CommandExecutor executes a bootstrap command and returns its result.
type CommandExecutor func(cmd []string, env map[string]string) CommandResult

// LoggingRunner executes commands while emitting structured logs.
type LoggingRunner struct {
	exec            CommandExecutor
	logger          telemetry.StructuredLogger
	sanitizeCommand func([]string) string
	sanitizeEnv     func(map[string]string) map[string]string
	sanitizeOutput  func(string) string
	stderrLimit     int
}

// NewLoggingRunner constructs a LoggingRunner that emits structured command logs.
func NewLoggingRunner(exec CommandExecutor, logger telemetry.StructuredLogger, sanitizeCommand func([]string) string, sanitizeEnv func(map[string]string) map[string]string, sanitizeOutput func(string) string, stderrLimit int) *LoggingRunner {
	if exec == nil {
		panic("bootstrap: command executor is required")
	}
	if logger == nil {
		panic("bootstrap: structured logger is required")
	}
	if sanitizeCommand == nil {
		sanitizeCommand = clilogging.SanitizeCommand
	}
	if sanitizeEnv == nil {
		sanitizeEnv = clilogging.SanitizeEnv
	}
	if sanitizeOutput == nil {
		sanitizeOutput = clilogging.SanitizeText
	}
	if stderrLimit <= 0 {
		stderrLimit = 4096
	}
	return &LoggingRunner{
		exec:            exec,
		logger:          logger,
		sanitizeCommand: sanitizeCommand,
		sanitizeEnv:     sanitizeEnv,
		sanitizeOutput:  sanitizeOutput,
		stderrLimit:     stderrLimit,
	}
}

// Run executes the command and returns an error when the command fails.
func (l *LoggingRunner) Run(cmd []string, env map[string]string) error {
	if l == nil {
		return fmt.Errorf("logging runner is nil")
	}
	sanitizedCommand := l.sanitizeCommand(cmd)
	sanitizedEnv := l.sanitizeEnv(env)
	l.emit(telemetry.Entry{
		Category: telemetry.CategoryCommand,
		Message:  "bootstrap command start",
		Severity: telemetry.SeverityInfo,
		Command:  sanitizedCommand,
		Metadata: envMetadata(sanitizedEnv),
	})

	result := l.exec(cmd, env)
	severity := telemetry.SeverityInfo
	exitCode := result.ExitCode
	if result.Err != nil {
		severity = telemetry.SeverityError
		if exitCode == 0 {
			exitCode = 1
		}
	} else if exitCode != 0 {
		severity = telemetry.SeverityError
	}

	stderr := result.Stderr
	if l.stderrLimit > 0 && len(stderr) > l.stderrLimit {
		stderr = stderr[:l.stderrLimit]
	}
	if stderr != "" {
		stderr = l.sanitizeOutput(stderr)
	}

	metadata := envMetadata(sanitizedEnv)
	metadata["exitCode"] = strconv.Itoa(exitCode)

	l.emit(telemetry.Entry{
		Category:      telemetry.CategoryCommand,
		Message:       "bootstrap command complete",
		Severity:      severity,
		Command:       sanitizedCommand,
		StderrExcerpt: stderr,
		Metadata:      metadata,
		Error:         result.Err,
	})

	if result.Err != nil {
		return result.Err
	}
	if exitCode != 0 {
		return fmt.Errorf("command exited with code %d", exitCode)
	}
	return nil
}

func (l *LoggingRunner) emit(entry telemetry.Entry) {
	if err := l.logger.Emit(entry); err != nil {
		log.Printf("bootstrap: structured log emit failed: %v", err)
	}
}

func envMetadata(env map[string]string) map[string]string {
	if len(env) == 0 {
		return map[string]string{}
	}
	meta := make(map[string]string, len(env))
	for key, value := range env {
		meta["env."+strings.ToLower(key)] = value
	}
	return meta
}
