package config

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFormatSummaryTextAndJSON(t *testing.T) {
	resolved := &ResolvedInvocation{
		CommandPath: "chainctl cluster install",
		SourcePath:  "/etc/chainctl/chainctl.yaml",
		Profiles:    []string{"staging"},
		Overrides:   []string{"runtime overrides namespace (was default)"},
		Flags: FlagSet{
			"namespace": {Value: "demo", Source: ValueSourceDefault},
			"dry-run":   {Value: true, Source: ValueSourceRuntime},
			"roles":     {Value: []string{"ops", "infra"}, Source: ValueSourceProfile},
			"notes":     {Value: []interface{}{"checked", 1}, Source: ValueSourceCommand},
		},
	}

	textSummary, err := FormatSummary(resolved, SummaryFormatText)
	if err != nil {
		t.Fatalf("text summary error: %v", err)
	}
	if !strings.Contains(textSummary, "Command:    chainctl cluster install") {
		t.Fatalf("text summary missing command header:\n%s", textSummary)
	}
	if !strings.Contains(textSummary, "roles      ops,infra  profile") {
		t.Fatalf("text summary missing roles row:\n%s", textSummary)
	}
	if !strings.Contains(textSummary, "notes      checked,1  command") {
		t.Fatalf("text summary missing notes row:\n%s", textSummary)
	}

	jsonSummary, err := FormatSummary(resolved, SummaryFormatJSON)
	if err != nil {
		t.Fatalf("json summary error: %v", err)
	}
	var payload struct {
		CommandPath string `json:"commandPath"`
		Flags       []struct {
			Name   string      `json:"name"`
			Value  interface{} `json:"value"`
			Source ValueSource `json:"source"`
		} `json:"flags"`
	}
	if err := json.Unmarshal([]byte(jsonSummary), &payload); err != nil {
		t.Fatalf("unmarshal json summary: %v", err)
	}
	if payload.CommandPath != "chainctl cluster install" {
		t.Fatalf("payload commandPath = %s", payload.CommandPath)
	}
	if len(payload.Flags) != 4 {
		t.Fatalf("expected 4 flags, got %d", len(payload.Flags))
	}

	if _, err := FormatSummary(resolved, "invalid-format"); err == nil {
		t.Fatalf("expected error for unsupported format")
	}
}
