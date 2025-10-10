package loggingintegration

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"

	clustercmd "github.com/dobrovols/chainctl/cmd/chainctl/cluster"
	"github.com/dobrovols/chainctl/internal/config"
	"github.com/dobrovols/chainctl/pkg/bundle"
	"github.com/dobrovols/chainctl/pkg/telemetry"
)

type inspectorStub struct{}

func (inspectorStub) CPUCount() int               { return 8 }
func (inspectorStub) MemoryGiB() int              { return 16 }
func (inspectorStub) HasKernelModule(string) bool { return true }
func (inspectorStub) HasSudoPrivileges() bool     { return true }

func TestLoggingDryRunEmitsWorkflowEntries(t *testing.T) {
	logs := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(logs)
	cmd.SetErr(new(bytes.Buffer))

	helm := &fakeHelmInstaller{}

	deps := clustercmd.InstallDeps{
		Inspector: inspectorStub{},
		BundleLoader: func(string, string) (*bundle.Bundle, error) {
			return &bundle.Bundle{Manifest: bundle.Manifest{Version: "1.2.3"}}, nil
		},
		Bootstrapper:        noopBootstrap{},
		HelmInstaller:       helm,
		TelemetryEmitter:    func(w io.Writer) (*telemetry.Emitter, error) { return telemetry.NewEmitter(w) },
		ClusterValidator:    func(*rest.Config) error { return nil },
		ClusterConfigLoader: func(*config.Profile) (*rest.Config, error) { return nil, nil },
	}

	opts := clustercmd.InstallOptions{
		Bootstrap:        true,
		ValuesFile:       t.TempDir() + "/values.enc",
		ValuesPassphrase: "super-secret",
		DryRun:           true,
		Output:           "json",
	}

	if err := clustercmd.RunInstallForTest(cmd, opts, deps); err != nil {
		t.Fatalf("expected dry-run success, got error: %v", err)
	}

	records := parseLogs(t, logs.Bytes())
	if len(records) == 0 {
		t.Fatalf("expected structured logs to be emitted")
	}

	workflowEntries := filterByCategory(records, "workflow")
	if len(workflowEntries) < 2 {
		t.Fatalf("expected workflow start and completion entries, got %d", len(workflowEntries))
	}

	commandEntries := filterByCategory(records, "command")
	if len(commandEntries) == 0 {
		t.Fatalf("expected command logging entry")
	}
	command := commandEntries[0]
	if containsSecret(command.Command, "super-secret") {
		t.Fatalf("expected command to redact secrets, got %q", command.Command)
	}
	if !containsSecret(command.Command, "***") {
		t.Fatalf("expected command to contain redacted placeholder, got %q", command.Command)
	}

	if helm.called {
		t.Fatalf("expected helm install not to run during dry-run")
	}
}

func TestLoggingCapturesHelmFailure(t *testing.T) {
	logs := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(logs)
	cmd.SetErr(new(bytes.Buffer))

	helm := &fakeHelmInstaller{err: errors.New("boom: token=abcd"), stderr: "token=abcd\nforbidden"}

	deps := clustercmd.InstallDeps{
		Inspector: inspectorStub{},
		BundleLoader: func(string, string) (*bundle.Bundle, error) {
			return &bundle.Bundle{Manifest: bundle.Manifest{Version: "1.2.3"}}, nil
		},
		Bootstrapper:     noopBootstrap{},
		HelmInstaller:    helm,
		TelemetryEmitter: func(w io.Writer) (*telemetry.Emitter, error) { return telemetry.NewEmitter(w) },
	}

	opts := clustercmd.InstallOptions{
		Bootstrap:        true,
		ValuesFile:       t.TempDir() + "/values.enc",
		ValuesPassphrase: "another-secret",
		Output:           "json",
	}

	err := clustercmd.RunInstallForTest(cmd, opts, deps)
	if err == nil {
		t.Fatalf("expected helm failure to surface error")
	}

	records := parseLogs(t, logs.Bytes())
	commandEntries := filterByCategory(records, "command")
	if len(commandEntries) == 0 {
		t.Fatalf("expected command log entries on failure")
	}
	foundError := false
	for _, entry := range commandEntries {
		if entry.Severity == "error" {
			foundError = true
			if containsSecret(entry.StderrExcerpt, "abcd") {
				t.Fatalf("expected stderr excerpt to be sanitized, got %q", entry.StderrExcerpt)
			}
			if !containsSecret(entry.StderrExcerpt, "***") {
				t.Fatalf("expected stderr excerpt to contain placeholder, got %q", entry.StderrExcerpt)
			}
		}
	}
	if !foundError {
		t.Fatalf("expected at least one error severity command entry")
	}
}

func TestLoggingCapturesBootstrapFailure(t *testing.T) {
	logs := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(logs)
	cmd.SetErr(new(bytes.Buffer))

	bootstrap := &failingBootstrap{err: errors.New("bootstrap failed: token=abcd")}
	helm := &fakeHelmInstaller{}

	deps := clustercmd.InstallDeps{
		Inspector: inspectorStub{},
		BundleLoader: func(string, string) (*bundle.Bundle, error) {
			return &bundle.Bundle{Manifest: bundle.Manifest{Version: "1.2.3"}}, nil
		},
		Bootstrapper:  bootstrap,
		HelmInstaller: &fakeHelmInstaller{},
		TelemetryEmitter: func(w io.Writer) (*telemetry.Emitter, error) {
			return telemetry.NewEmitter(w)
		},
		ClusterValidator: func(*rest.Config) error { return nil },
	}

	opts := clustercmd.InstallOptions{
		Bootstrap:        true,
		ValuesFile:       t.TempDir() + "/values.enc",
		ValuesPassphrase: "cluster-secret",
		Output:           "json",
	}

	err := clustercmd.RunInstallForTest(cmd, opts, deps)
	if err == nil {
		t.Fatalf("expected bootstrap failure to surface")
	}

	records := parseLogs(t, logs.Bytes())
	commandEntries := filterByCategory(records, "command")
	if len(commandEntries) == 0 {
		t.Fatalf("expected command log entries")
	}

	workflowIDs := map[string]struct{}{}
	for _, entry := range records {
		if entry.WorkflowID == "" {
			t.Fatalf("expected workflowId on log entry: %+v", entry)
		}
		workflowIDs[entry.WorkflowID] = struct{}{}
	}
	if len(workflowIDs) != 1 {
		t.Fatalf("expected a single workflowId across logs, got: %v", workflowIDs)
	}

	var errorEntry structuredLog
	for _, entry := range commandEntries {
		if entry.Severity == "error" {
			errorEntry = entry
			break
		}
	}
	if errorEntry.Severity != "error" {
		t.Fatalf("expected error log for bootstrap failure, got %+v", commandEntries)
	}
	if containsSecret(errorEntry.StderrExcerpt, "abcd") {
		t.Fatalf("expected sanitized stderr, got %q", errorEntry.StderrExcerpt)
	}
	if helm.called {
		t.Fatalf("expected helm not to execute when bootstrap fails early")
	}
}

func TestLoggingDisabledBlocksExecution(t *testing.T) {
	logs := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(logs)
	cmd.SetErr(new(bytes.Buffer))

	bootstrap := &failingBootstrap{}
	helm := &fakeHelmInstaller{}

	deps := clustercmd.InstallDeps{
		Inspector: inspectorStub{},
		BundleLoader: func(string, string) (*bundle.Bundle, error) {
			return &bundle.Bundle{Manifest: bundle.Manifest{Version: "1.2.3"}}, nil
		},
		Bootstrapper:  bootstrap,
		HelmInstaller: helm,
		TelemetryEmitter: func(io.Writer) (*telemetry.Emitter, error) {
			return nil, errors.New("logging disabled")
		},
		ClusterValidator: func(*rest.Config) error { return nil },
	}

	opts := clustercmd.InstallOptions{
		Bootstrap:        true,
		ValuesFile:       t.TempDir() + "/values.enc",
		ValuesPassphrase: "disable-secret",
		Output:           "json",
	}

	err := clustercmd.RunInstallForTest(cmd, opts, deps)
	if err == nil {
		t.Fatalf("expected error when logging emitter is unavailable")
	}

	if bootstrap.called {
		t.Fatalf("expected bootstrapper not to execute when logging is disabled")
	}
	if helm.called {
		t.Fatalf("expected helm not to run when logging is disabled")
	}

	if records := parseLogs(t, logs.Bytes()); len(records) != 0 {
		t.Fatalf("expected no logs when initialization fails, got %v", records)
	}
}

func parseLogs(t *testing.T, data []byte) []structuredLog {
	t.Helper()
	lines := bytes.Split(data, []byte("\n"))
	var logs []structuredLog
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if line[0] != '{' {
			continue
		}
		var record structuredLog
		if err := json.Unmarshal(line, &record); err != nil {
			t.Fatalf("failed to decode log line %s: %v", line, err)
		}
		logs = append(logs, record)
	}
	return logs
}

func filterByCategory(logs []structuredLog, category string) []structuredLog {
	var out []structuredLog
	for _, l := range logs {
		if l.Category == category {
			out = append(out, l)
		}
	}
	return out
}

func containsSecret(input, value string) bool {
	return strings.Contains(input, value)
}

type structuredLog struct {
	Category      string            `json:"category"`
	Message       string            `json:"message"`
	Severity      string            `json:"severity"`
	Command       string            `json:"command"`
	StderrExcerpt string            `json:"stderrExcerpt"`
	Metadata      map[string]string `json:"metadata"`
	WorkflowID    string            `json:"workflowId"`
	Step          string            `json:"step"`
}

type fakeHelmInstaller struct {
	called bool
	err    error
	stderr string
}

func (f *fakeHelmInstaller) Install(*config.Profile, *bundle.Bundle) error {
	f.called = true
	return f.err
}

type failingBootstrap struct {
	err    error
	called bool
}

func (f *failingBootstrap) Bootstrap(*config.Profile) error {
	f.called = true
	return f.err
}

type noopBootstrap struct{}

func (noopBootstrap) Bootstrap(*config.Profile) error { return nil }
