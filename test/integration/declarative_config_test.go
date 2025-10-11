package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dobrovols/chainctl/internal/cli"
	internalconfig "github.com/dobrovols/chainctl/internal/config"
	pkgconfig "github.com/dobrovols/chainctl/pkg/config"
)

const (
	sampleNamespace     = "runtime-ns"
	sampleValuesFileKey = "values-file"
)

func TestDeclarativeConfigPipelineResolvesProfilesAndPrecedence(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "chainctl.yaml")
	valuesPath := filepath.Join(tmpDir, "values.enc")
	if err := os.WriteFile(valuesPath, []byte("encrypted"), 0o600); err != nil {
		t.Fatalf("write values file: %v", err)
	}

	writeYAML(t, configPath, `
metadata:
  name: declarative-demo
defaults:
  namespace: demo
  values-file: "`+valuesPath+`"
profiles:
  staging:
    namespace: staging
commands:
  chainctl cluster install:
    flags:
      dry-run: true
      output: text
  chainctl app install:
    profiles:
      - staging
    flags:
      chart: oci://registry.example.com/app/demo:1.0.0
      release-name: demo-staging
`)

	t.Setenv("CHAINCTL_CONFIG", configPath)

	root := cli.NewRootCommand()
	catalog := internalconfig.NewCobraCatalog(root)
	located, err := internalconfig.LocateConfig("")
	if err != nil {
		t.Fatalf("LocateConfig: %v", err)
	}

	loader := internalconfig.NewLoader(catalog)
	profile, err := loader.Load(located.Path)
	if err != nil {
		t.Fatalf("Load profile: %v", err)
	}

	clusterRuntime := pkgconfig.FlagSet{
		"namespace":         {Value: sampleNamespace, Source: pkgconfig.ValueSourceRuntime},
		"values-passphrase": {Value: "runtime-pass", Source: pkgconfig.ValueSourceRuntime},
		"output":            {Value: "json", Source: pkgconfig.ValueSourceRuntime},
	}

	clusterResolved, err := pkgconfig.ResolveInvocation(profile, "chainctl cluster install", clusterRuntime)
	if err != nil {
		t.Fatalf("ResolveInvocation(cluster install): %v", err)
	}
	if clusterResolved.Flags["namespace"].Value != sampleNamespace {
		t.Fatalf("expected runtime namespace override, got %v", clusterResolved.Flags["namespace"].Value)
	}
	if clusterResolved.Flags["values-file"].Source != pkgconfig.ValueSourceDefault {
		t.Fatalf("expected values-file to come from defaults, got %s", clusterResolved.Flags["values-file"].Source)
	}

	textSummary, err := pkgconfig.FormatSummary(clusterResolved, pkgconfig.SummaryFormatText)
	if err != nil {
		t.Fatalf("FormatSummary(text): %v", err)
	}
	if !strings.Contains(textSummary, sampleNamespace) || !strings.Contains(textSummary, sampleValuesFileKey) {
		t.Fatalf("expected summary to include runtime namespace and values-file, got:\n%s", textSummary)
	}

	jsonSummary, err := pkgconfig.FormatSummary(clusterResolved, pkgconfig.SummaryFormatJSON)
	if err != nil {
		t.Fatalf("FormatSummary(json): %v", err)
	}
	var summaryPayload map[string]any
	if err := json.Unmarshal([]byte(jsonSummary), &summaryPayload); err != nil {
		t.Fatalf("unmarshal summary json: %v", err)
	}
	if summaryPayload["commandPath"] != "chainctl cluster install" {
		t.Fatalf("expected commandPath field, got %v", summaryPayload["commandPath"])
	}

	appResolved, err := pkgconfig.ResolveInvocation(profile, "chainctl app install", nil)
	if err != nil {
		t.Fatalf("ResolveInvocation(app install): %v", err)
	}
	if appResolved.Flags["namespace"].Value != "staging" {
		t.Fatalf("expected staging namespace from profile, got %v", appResolved.Flags["namespace"].Value)
	}
	if appResolved.Flags["chart"].Source != pkgconfig.ValueSourceCommand {
		t.Fatalf("expected chart to use command source, got %s", appResolved.Flags["chart"].Source)
	}
}

func writeYAML(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}
