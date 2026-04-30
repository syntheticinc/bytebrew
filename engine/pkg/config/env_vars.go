package config

// Env var names — single registry.
//
// All `os.Getenv` calls for application configuration MUST live inside
// `pkg/config` and reference these constants instead of raw strings. New env
// vars MUST be added here, bound via Viper in `bindEnvVars`, declared as a
// typed field in `BootstrapConfig`, and documented in `.env.example`.
//
// See `.claude/rules/code-review.md` for the enforced env-vars policy.
const (
	EnvDatabaseURL        = "DATABASE_URL"
	EnvEngineHost         = "ENGINE_HOST"
	EnvEnginePort         = "ENGINE_PORT"
	EnvInternalPort       = "BYTEBREW_INTERNAL_PORT"
	EnvCORSOrigins        = "BYTEBREW_CORS_ORIGINS"
	EnvAuthMode           = "BYTEBREW_AUTH_MODE"
	EnvJWTKeysDir         = "BYTEBREW_JWT_KEYS_DIR"
	EnvJWTPublicKeyPath   = "BYTEBREW_JWT_PUBLIC_KEY_PATH"
	EnvLocalSessionTTL    = "BYTEBREW_LOCAL_SESSION_TTL"
	EnvEmbedURL           = "EMBED_URL"
	EnvEmbedModel         = "EMBED_MODEL"
	EnvEmbedDim           = "EMBED_DIM"
	EnvDebugModel         = "BYTEBREW_DEBUG_MODEL"
	EnvDocsMCPURL         = "BYTEBREW_DOCS_MCP_URL"
	EnvDataDir            = "DATA_DIR"
	EnvDisableLSPDownload    = "BYTEBREW_DISABLE_LSP_DOWNLOAD"
	EnvVersionsURL           = "BYTEBREW_VERSIONS_URL"
	EnvBootstrapAdminToken   = "BYTEBREW_BOOTSTRAP_ADMIN_TOKEN"
)
