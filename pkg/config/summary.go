package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"
)

// Supported summary output formats.
const (
	SummaryFormatText = "text"
	SummaryFormatJSON = "json"
)

// FormatSummary renders a resolved invocation summary in the requested format.
func FormatSummary(resolved *ResolvedInvocation, format string) (string, error) {
	if resolved == nil {
		return "", fmt.Errorf("resolved invocation is nil")
	}

	switch strings.ToLower(format) {
	case "", SummaryFormatText:
		return formatSummaryText(resolved)
	case SummaryFormatJSON:
		return formatSummaryJSON(resolved)
	default:
		return "", fmt.Errorf("unsupported summary format %q", format)
	}
}

func formatSummaryText(resolved *ResolvedInvocation) (string, error) {
	var buf bytes.Buffer
	tw := tabwriter.NewWriter(&buf, 0, 4, 2, ' ', 0)

	fmt.Fprintf(tw, "Command:\t%s\n", resolved.CommandPath)
	if resolved.SourcePath != "" {
		fmt.Fprintf(tw, "Source:\t%s\n", resolved.SourcePath)
	}
	if len(resolved.Profiles) > 0 {
		fmt.Fprintf(tw, "Profiles:\t%s\n", strings.Join(resolved.Profiles, ", "))
	}
	if len(resolved.Overrides) > 0 {
		fmt.Fprintf(tw, "Overrides:\t%s\n", strings.Join(resolved.Overrides, ", "))
	}
	if len(resolved.Warnings) > 0 {
		fmt.Fprintf(tw, "Warnings:\t%s\n", strings.Join(resolved.Warnings, ", "))
	}
	fmt.Fprintln(tw)
	fmt.Fprintln(tw, "Flag\tValue\tSource")

	names := make([]string, 0, len(resolved.Flags))
	for name := range resolved.Flags {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		value := resolved.Flags[name]
		fmt.Fprintf(tw, "%s\t%v\t%s\n", name, formatFlagValue(value.Value), value.Source)
	}

	if err := tw.Flush(); err != nil {
		return "", fmt.Errorf("flush summary: %w", err)
	}
	return buf.String(), nil
}

func formatSummaryJSON(resolved *ResolvedInvocation) (string, error) {
	type flagEntry struct {
		Name   string      `json:"name"`
		Value  interface{} `json:"value"`
		Source ValueSource `json:"source"`
	}

	names := make([]string, 0, len(resolved.Flags))
	for name := range resolved.Flags {
		names = append(names, name)
	}
	sort.Strings(names)

	flags := make([]flagEntry, 0, len(names))
	for _, name := range names {
		value := resolved.Flags[name]
		flags = append(flags, flagEntry{
			Name:   name,
			Value:  value.Value,
			Source: value.Source,
		})
	}

	payload := map[string]interface{}{
		"commandPath": resolved.CommandPath,
		"sourcePath":  resolved.SourcePath,
		"flags":       flags,
	}
	if len(resolved.Profiles) > 0 {
		payload["profiles"] = resolved.Profiles
	}
	if len(resolved.Overrides) > 0 {
		payload["overrides"] = resolved.Overrides
	}
	if len(resolved.Warnings) > 0 {
		payload["warnings"] = resolved.Warnings
	}

	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal summary json: %w", err)
	}
	return string(encoded), nil
}

func formatFlagValue(value any) any {
	switch v := value.(type) {
	case []string:
		return strings.Join(v, ",")
	case []interface{}:
		items := make([]string, len(v))
		for i, item := range v {
			items[i] = fmt.Sprint(item)
		}
		return strings.Join(items, ",")
	default:
		return v
	}
}
