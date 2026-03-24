package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Auth      AuthConfig      `mapstructure:"auth"`
	License   LicenseConfig   `mapstructure:"license"`
	CORS      CORSConfig      `mapstructure:"cors"`
	Stripe    StripeConfig    `mapstructure:"stripe"`
	Trial     TrialConfig     `mapstructure:"trial"`
	DeepInfra DeepInfraConfig `mapstructure:"deepinfra"`
	Email        EmailConfig        `mapstructure:"email"`
	ModelRouting ModelRoutingConfig `mapstructure:"model_routing"`
	Google       GoogleConfig       `mapstructure:"google"`
}

// GoogleConfig holds Google OAuth settings. Optional -- if ClientID is empty, Google login is disabled.
type GoogleConfig struct {
	ClientID string `mapstructure:"client_id"`
}

// ModelRoutingConfig maps agent roles to LLM model IDs.
type ModelRoutingConfig struct {
	DefaultModel  string            `mapstructure:"default_model"`
	RoleOverrides map[string]string `mapstructure:"role_overrides"`
}

// RouteModel returns the model ID for a given agent role.
func (c ModelRoutingConfig) RouteModel(role string) string {
	if model, ok := c.RoleOverrides[role]; ok {
		return model
	}
	return c.DefaultModel
}

// EmailConfig holds email sending settings. Optional -- if APIKey is empty, emails are logged instead.
type EmailConfig struct {
	ResendAPIKey string `mapstructure:"resend_api_key"`
	FromEmail    string `mapstructure:"from_email"`
	FrontendURL  string `mapstructure:"frontend_url"`
}

// DeepInfraConfig holds DeepInfra LLM proxy settings. Optional -- if APIKey is empty, proxy endpoint is disabled.
type DeepInfraConfig struct {
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url"`
}

// TrialConfig holds trial-specific rate limiting settings.
type TrialConfig struct {
	StepsPerHour int `mapstructure:"steps_per_hour"`
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	Port int `mapstructure:"port"`
}

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	URL string `mapstructure:"url"`
}

// AuthConfig holds authentication settings
type AuthConfig struct {
	JWTSecret        string        `mapstructure:"jwt_secret"`
	AccessTokenTTL   time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL  time.Duration `mapstructure:"refresh_token_ttl"`
	PasswordResetTTL time.Duration `mapstructure:"password_reset_ttl"`
}

// LicenseConfig holds Ed25519 key settings for license generation
type LicenseConfig struct {
	PrivateKeyHex string `mapstructure:"private_key_hex"`
}

// CORSConfig holds CORS settings
type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

// StripeConfig holds Stripe billing settings. Optional — if SecretKey is empty, billing endpoints return 501.
type StripeConfig struct {
	SecretKey     string             `mapstructure:"secret_key"`
	WebhookSecret string            `mapstructure:"webhook_secret"`
	Prices        StripePricesConfig `mapstructure:"prices"`
	SuccessURL    string             `mapstructure:"success_url"`
	CancelURL     string             `mapstructure:"cancel_url"`
	TrialDays     int64              `mapstructure:"trial_days"`
}

// StripePricesConfig maps tier+period to Stripe Price IDs.
type StripePricesConfig struct {
	PersonalMonthly  string `mapstructure:"personal_monthly"`
	PersonalAnnual   string `mapstructure:"personal_annual"`
	TeamsMonthly     string `mapstructure:"teams_monthly"`
	TeamsAnnual      string `mapstructure:"teams_annual"`
	EngineEEMonthly  string `mapstructure:"engine_ee_monthly"`
	EngineEEAnnual   string `mapstructure:"engine_ee_annual"`
}

// Load reads configuration from a YAML file and optional .env file
func Load(configPath string) (*Config, error) {
	configDir := filepath.Dir(configPath)
	envPath := filepath.Join(configDir, ".env")
	if _, err := os.Stat(envPath); err == nil {
		if err := godotenv.Load(envPath); err != nil {
			return nil, fmt.Errorf("load .env: %w", err)
		}
	}

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Explicit env var bindings for secrets (Viper Unmarshal doesn't
	// check env vars for keys missing from YAML, so we bind them explicitly).
	for _, b := range []struct{ key, env string }{
		{"database.url", "DATABASE_URL"},
		{"auth.jwt_secret", "AUTH_JWT_SECRET"},
		{"license.private_key_hex", "LICENSE_PRIVATE_KEY_HEX"},
		{"stripe.secret_key", "STRIPE_SECRET_KEY"},
		{"stripe.webhook_secret", "STRIPE_WEBHOOK_SECRET"},
		{"stripe.prices.personal_monthly", "STRIPE_PRICES_PERSONAL_MONTHLY"},
		{"stripe.prices.personal_annual", "STRIPE_PRICES_PERSONAL_ANNUAL"},
		{"stripe.prices.teams_monthly", "STRIPE_PRICES_TEAMS_MONTHLY"},
		{"stripe.prices.teams_annual", "STRIPE_PRICES_TEAMS_ANNUAL"},
		{"stripe.prices.engine_ee_monthly", "STRIPE_PRICES_ENGINE_EE_MONTHLY"},
		{"stripe.prices.engine_ee_annual", "STRIPE_PRICES_ENGINE_EE_ANNUAL"},
		{"deepinfra.api_key", "DEEPINFRA_API_KEY"},
		{"email.resend_api_key", "EMAIL_RESEND_API_KEY"},
		{"google.client_id", "GOOGLE_CLIENT_ID"},
	} {
		_ = v.BindEnv(b.key, b.env)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

// Validate checks that all required configuration fields are set
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	if c.Database.URL == "" {
		return fmt.Errorf("database URL is required")
	}
	if c.Auth.JWTSecret == "" {
		return fmt.Errorf("JWT secret is required")
	}
	if c.Auth.AccessTokenTTL == 0 {
		c.Auth.AccessTokenTTL = 15 * time.Minute
	}
	if c.Auth.RefreshTokenTTL == 0 {
		c.Auth.RefreshTokenTTL = 7 * 24 * time.Hour
	}
	if c.Auth.PasswordResetTTL == 0 {
		c.Auth.PasswordResetTTL = 1 * time.Hour
	}
	if c.License.PrivateKeyHex == "" {
		return fmt.Errorf("license private key hex is required (generate with cmd/keygen)")
	}
	if c.Trial.StepsPerHour <= 0 {
		c.Trial.StepsPerHour = 20
	}
	if c.DeepInfra.BaseURL == "" {
		c.DeepInfra.BaseURL = "https://api.deepinfra.com/v1/openai"
	}
	if c.ModelRouting.DefaultModel == "" {
		c.ModelRouting.DefaultModel = "zai-org/GLM-5"
	}
	if c.ModelRouting.RoleOverrides == nil {
		c.ModelRouting.RoleOverrides = map[string]string{
			"reviewer": "zai-org/GLM-4.7",
			"tester":   "zai-org/GLM-4.7",
		}
	}
	return nil
}
