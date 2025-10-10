package logging

import (
	"strings"
	"testing"
)

func TestSanitizeCommandRedactsInlineSecrets(t *testing.T) {
	args := []string{"helm", "upgrade", "myapp", "--token=abcd1234", "--namespace", "demo"}

	sanitized := SanitizeCommand(args)

	if sanitized == "" {
		t.Fatalf("expected sanitized command, got empty string")
	}

	if expected := "--token=***"; !containsToken(sanitized, expected) {
		t.Fatalf("expected inline secret to be redacted; sanitized=%q", sanitized)
	}

	if containsToken(sanitized, "--token=abcd1234") {
		t.Fatalf("expected original token to be removed; sanitized=%q", sanitized)
	}

	if !containsToken(sanitized, "--namespace demo") {
		t.Fatalf("expected non-sensitive flag to remain; sanitized=%q", sanitized)
	}
}

func TestSanitizeCommandRedactsSeparatedSecrets(t *testing.T) {
	args := []string{"helm", "upgrade", "myapp", "--password", "super-secret", "--context", "prod"}

	sanitized := SanitizeCommand(args)

	if containsToken(sanitized, "super-secret") {
		t.Fatalf("expected separated value to be redacted; sanitized=%q", sanitized)
	}

	if !containsToken(sanitized, "--password ***") {
		t.Fatalf("expected password flag to be redacted; sanitized=%q", sanitized)
	}

	if !containsToken(sanitized, "--context prod") {
		t.Fatalf("expected allowlisted flag to remain; sanitized=%q", sanitized)
	}

	args = []string{"helm", "upgrade", "myapp", "--values-passphrase", "topsecret"}
	sanitized = SanitizeCommand(args)
	if containsToken(sanitized, "topsecret") {
		t.Fatalf("expected passphrase redacted; sanitized=%q", sanitized)
	}
}

func TestSanitizeCommandRedactsSetExpressions(t *testing.T) {
	args := []string{"helm", "upgrade", "myapp", "--set", "adminPassword=topsecret", "--set-string", "imagePullSecret=abc"}

	sanitized := SanitizeCommand(args)

	if containsToken(sanitized, "adminPassword=topsecret") {
		t.Fatalf("expected admin password to be redacted; sanitized=%q", sanitized)
	}

	if !containsToken(sanitized, "adminPassword=***") {
		t.Fatalf("expected admin password placeholder; sanitized=%q", sanitized)
	}

	if !containsToken(sanitized, "imagePullSecret=***") {
		t.Fatalf("expected image pull secret to be redacted; sanitized=%q", sanitized)
	}
}

func TestSanitizeEnvMasksSensitiveVariables(t *testing.T) {
	env := map[string]string{
		"KUBECONFIG":               "/tmp/kubeconfig",
		"HELM_REPOSITORY_PASSWORD": "hunter2",
		"TOKEN":                    "abcd",
	}

	sanitized := SanitizeEnv(env)

	if sanitized["KUBECONFIG"] != "/tmp/kubeconfig" {
		t.Fatalf("expected allowlisted env to remain, got %q", sanitized["KUBECONFIG"])
	}

	if sanitized["HELM_REPOSITORY_PASSWORD"] != "***" {
		t.Fatalf("expected helm password to be redacted, got %q", sanitized["HELM_REPOSITORY_PASSWORD"])
	}

	if sanitized["TOKEN"] != "***" {
		t.Fatalf("expected generic token to be redacted, got %q", sanitized["TOKEN"])
	}
}

func TestSanitizeTextRedactsKeyValuePairs(t *testing.T) {
	input := "error: token=abcd password=topsecret still here"
	got := SanitizeText(input)
	if strings.Contains(got, "abcd") || strings.Contains(got, "topsecret") {
		t.Fatalf("expected sensitive values to be redacted, got %q", got)
	}
	if !strings.Contains(got, "token=***") {
		t.Fatalf("expected token placeholder, got %q", got)
	}
}

func containsToken(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
