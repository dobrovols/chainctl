package logging

import (
	"regexp"
	"strings"
)

const redactionPlaceholder = "***"

var allowlistedEnvKeys = map[string]struct{}{
	"PATH":       {},
	"HOME":       {},
	"USER":       {},
	"SHELL":      {},
	"KUBECONFIG": {},
	"PWD":        {},
	"LANG":       {},
	"LC_ALL":     {},
	"TMPDIR":     {},
	"TMP":        {},
	"TERM":       {},
	"LOGNAME":    {},
	"EDITOR":     {},
}

// SanitizeCommand returns a sanitized string representation of the provided command arguments.
// Sensitive tokens (passwords, tokens, secrets) are redacted while leaving the overall structure intact.

func SanitizeCommand(args []string) string {
	if len(args) == 0 {
		return ""
	}
	sanitized := make([]string, len(args))
	var nextTransform func(string) string
	for i, arg := range args {
		if nextTransform != nil {
			sanitized[i] = nextTransform(arg)
			nextTransform = nil
			continue
		}

		lower := strings.ToLower(arg)

		// Handle --set/--set-string style flags where the next argument contains key=value pairs.
		if lower == "--set" || lower == "--set-string" {
			sanitized[i] = arg
			nextTransform = func(value string) string {
				return sanitizeSetExpressions(value)
			}
			continue
		}

		if eq := strings.Index(arg, "="); eq > 0 {
			flag := arg[:eq]
			value := arg[eq+1:]
			if isSensitiveFlag(flag) {
				sanitized[i] = flag + "=" + redactionPlaceholder
				continue
			}
			if isSensitiveKey(value) && strings.HasPrefix(lower, "--") {
				sanitized[i] = flag + "=" + redactionPlaceholder
				continue
			}
			if strings.HasPrefix(lower, "--set") {
				sanitized[i] = flag + "=" + sanitizeSetExpressions(value)
				continue
			}
		}

		if isSensitiveFlag(arg) {
			sanitized[i] = arg
			nextTransform = func(string) string { return redactionPlaceholder }
			continue
		}

		sanitized[i] = arg
	}

	if nextTransform != nil {
		sanitized = append(sanitized, redactionPlaceholder)
	}

	return strings.Join(sanitized, " ")
}

// SanitizeEnv returns a sanitized copy of the provided environment variables.
// Sensitive values are replaced with a placeholder while preserving allowlisted keys.
func SanitizeEnv(env map[string]string) map[string]string {
	if len(env) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(env))
	for key, value := range env {
		if _, ok := allowlistedEnvKeys[key]; ok {
			out[key] = value
			continue
		}
		if isSensitiveKey(key) {
			out[key] = redactionPlaceholder
			continue
		}
		out[key] = value
	}
	return out
}

var sensitivePattern = regexp.MustCompile(`(?i)(password|passphrase|secret|token|apikey|privatekey)=([^\s]{1,128})`)

// SanitizeText redacts sensitive key/value pairs inside freeform strings.
func SanitizeText(text string) string {
	if text == "" {
		return ""
	}
	return sensitivePattern.ReplaceAllStringFunc(text, func(match string) string {
		parts := strings.SplitN(match, "=", 2)
		if len(parts) != 2 {
			return match
		}
		return parts[0] + "=" + redactionPlaceholder
	})
}

func sanitizeSetExpressions(expression string) string {
	pairs := strings.Split(expression, ",")
	for i, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := kv[0]
		value := kv[1]
		if isSensitiveKey(key) || isSensitiveKey(value) {
			pairs[i] = key + "=" + redactionPlaceholder
			continue
		}
		pairs[i] = key + "=" + value
	}
	return strings.Join(pairs, ",")
}

func isSensitiveFlag(flag string) bool {
	flagLower := strings.ToLower(flag)
	return strings.Contains(flagLower, "password") ||
		strings.Contains(flagLower, "passphrase") ||
		strings.Contains(flagLower, "token") ||
		strings.Contains(flagLower, "secret") ||
		strings.Contains(flagLower, "credential")
}

func isSensitiveKey(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(lower, "password") ||
		strings.Contains(lower, "passphrase") ||
		strings.Contains(lower, "secret") ||
		strings.Contains(lower, "token") ||
		strings.Contains(lower, "apikey") ||
		strings.Contains(lower, "privatekey")
}
