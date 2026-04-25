package secrets

import "os"

// Lookup returns the value of the named environment variable, or "" if unset.
// Single chokepoint for runtime secret resolution by name when the name is
// supplied at runtime (e.g. mcp_servers.auth_key_env stored per-server in the
// catalog). All non-pkg/config code MUST use this helper instead of os.Getenv
// directly so audits/linters can locate every dynamic-name secret access by
// grepping `secrets.Lookup`.
//
// Note: this resolves env vars by name from process env. Per-tenant MCP
// secrets are tracked separately as tech debt — target architecture is
// auth_key_encrypted column in mcp_servers table (mirrors models.api_key_encrypted).
func Lookup(envVarName string) string {
	if envVarName == "" {
		return ""
	}
	return os.Getenv(envVarName)
}
