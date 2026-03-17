package config

import (
	"os"
	"regexp"
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
