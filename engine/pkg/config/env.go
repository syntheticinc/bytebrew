package config

import (
	"os"
	"regexp"
	"strings"
)

// envVarPattern matches ${VAR_NAME} placeholders in strings.
var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// expandEnvVars replaces all ${VAR} occurrences in s with the value of
// the corresponding environment variable. Unknown variables resolve to "".
func expandEnvVars(s string) string {
	if s == "" {
		return s
	}
	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		// Extract variable name from ${VAR}
		varName := envVarPattern.FindStringSubmatch(match)[1]
		return os.Getenv(varName)
	})
}

// splitAndTrim splits s by sep and trims whitespace from each element.
// Empty elements are excluded.
func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
