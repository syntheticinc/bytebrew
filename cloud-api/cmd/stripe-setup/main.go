// Command stripe-setup creates Stripe Products and Prices for ByteBrew.
// It is idempotent: existing products/prices are found by metadata and reused.
//
// Usage:
//
//	STRIPE_SECRET_KEY=sk_test_... go run ./cmd/stripe-setup
//	# or read from config:
//	go run ./cmd/stripe-setup --config config.yaml
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/stripe/stripe-go/v82"
	portalconfig "github.com/stripe/stripe-go/v82/billingportal/configuration"
	"github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/product"
)

type productDef struct {
	productKey string // metadata key: "personal", "teams", or "engine_ee"
	name        string
	description string
	priceAmount int64  // in cents
	interval    string // "month"
	configKey   string // config.yaml key
}

var products = []productDef{
	{
		productKey:  "personal",
		name:        "ByteBrew Personal",
		description: "AI coding agent for individual developers. 1 seat, 300 proxy steps/month + unlimited BYOK.",
		priceAmount: 2000, // $20/mo
		interval:    "month",
		configKey:   "personal_monthly",
	},
	{
		productKey:  "personal",
		name:        "ByteBrew Personal",
		description: "AI coding agent for individual developers. 1 seat, 300 proxy steps/month + unlimited BYOK.",
		priceAmount: 20000, // $200/yr (~17% off)
		interval:    "year",
		configKey:   "personal_annual",
	},
	{
		productKey:  "teams",
		name:        "ByteBrew Teams",
		description: "AI coding agent for teams. N seats, 300 proxy steps per user/month + unlimited BYOK + admin panel.",
		priceAmount: 3000, // $30/mo
		interval:    "month",
		configKey:   "teams_monthly",
	},
	{
		productKey:  "teams",
		name:        "ByteBrew Teams",
		description: "AI coding agent for teams. N seats, 300 proxy steps per user/month + unlimited BYOK + admin panel.",
		priceAmount: 30000, // $300/seat/yr (~17% off)
		interval:    "year",
		configKey:   "teams_annual",
	},
	{
		productKey:  "engine_ee",
		name:        "ByteBrew Engine Enterprise",
		description: "Enterprise features for ByteBrew Engine: audit logs, configurable rate limiting, Prometheus metrics, and more.",
		priceAmount: 49900, // $499/mo
		interval:    "month",
		configKey:   "engine_ee_monthly",
	},
	{
		productKey:  "engine_ee",
		name:        "ByteBrew Engine Enterprise",
		description: "Enterprise features for ByteBrew Engine: audit logs, configurable rate limiting, Prometheus metrics, and more.",
		priceAmount: 499000, // $4,990/yr (~17% off)
		interval:    "year",
		configKey:   "engine_ee_annual",
	},
}

func main() {
	configPath := flag.String("config", "", "Path to config.yaml (reads stripe.secret_key)")
	flag.Parse()

	secretKey := os.Getenv("STRIPE_SECRET_KEY")
	if secretKey == "" && *configPath != "" {
		secretKey = readKeyFromConfig(*configPath)
	}
	if secretKey == "" {
		log.Fatal("Set STRIPE_SECRET_KEY env var or use --config config.yaml")
	}

	stripe.Key = secretKey

	fmt.Println("ByteBrew Stripe Setup")
	fmt.Println("====================")
	fmt.Println()

	results := make(map[string]string)

	for _, pd := range products {
		priceID, err := ensureProductAndPrice(pd)
		if err != nil {
			log.Fatalf("Failed to setup %s: %v", pd.name, err)
		}
		results[pd.configKey] = priceID
	}

	if err := ensurePortalConfiguration(); err != nil {
		log.Fatalf("Failed to setup portal configuration: %v", err)
	}

	fmt.Println()
	fmt.Println("Config values for config.yaml:")
	fmt.Println("==============================")
	fmt.Println("stripe:")
	fmt.Println("  prices:")
	for _, pd := range products {
		fmt.Printf("    %s: \"%s\"\n", pd.configKey, results[pd.configKey])
	}
	fmt.Println()
	fmt.Println("Done. Copy the price IDs above into your config.yaml.")
}

// ensureProductAndPrice finds or creates a Stripe Product and Price.
func ensureProductAndPrice(pd productDef) (string, error) {
	// Search for existing product by metadata.
	prodID, err := findProduct(pd.productKey)
	if err != nil {
		return "", fmt.Errorf("search product: %w", err)
	}

	if prodID == "" {
		// Create product.
		params := &stripe.ProductParams{
			Name:        stripe.String(pd.name),
			Description: stripe.String(pd.description),
		}
		params.AddMetadata("bytebrew_product", pd.productKey)

		p, err := product.New(params)
		if err != nil {
			return "", fmt.Errorf("create product: %w", err)
		}
		prodID = p.ID
		fmt.Printf("Created product: %s (%s)\n", pd.name, prodID)
	} else {
		fmt.Printf("Found existing product: %s (%s)\n", pd.name, prodID)
	}

	// Search for existing price on this product.
	priceID, err := findPrice(prodID, pd.priceAmount, pd.interval)
	if err != nil {
		return "", fmt.Errorf("search price: %w", err)
	}

	if priceID == "" {
		// Create price.
		params := &stripe.PriceParams{
			Product:    stripe.String(prodID),
			Currency:   stripe.String("usd"),
			UnitAmount: stripe.Int64(pd.priceAmount),
			Recurring: &stripe.PriceRecurringParams{
				Interval: stripe.String(pd.interval),
			},
		}
		params.AddMetadata("bytebrew_price", pd.configKey)

		pr, err := price.New(params)
		if err != nil {
			return "", fmt.Errorf("create price: %w", err)
		}
		priceID = pr.ID
		fmt.Printf("  Created price: $%d/%s (%s)\n", pd.priceAmount/100, pd.interval, priceID)
	} else {
		fmt.Printf("  Found existing price: $%d/%s (%s)\n", pd.priceAmount/100, pd.interval, priceID)
	}

	return priceID, nil
}

// findProduct searches for a product with metadata bytebrew_product=key.
func findProduct(productKey string) (string, error) {
	params := &stripe.ProductListParams{}
	params.Filters.AddFilter("active", "", "true")
	params.Filters.AddFilter("limit", "", "100")

	iter := product.List(params)
	for iter.Next() {
		p := iter.Product()
		if p.Metadata["bytebrew_product"] == productKey {
			return p.ID, nil
		}
	}
	if err := iter.Err(); err != nil {
		return "", err
	}
	return "", nil
}

// findPrice searches for an active recurring price on a product with matching amount and interval.
func findPrice(productID string, amount int64, interval string) (string, error) {
	params := &stripe.PriceListParams{
		Product: stripe.String(productID),
		Active:  stripe.Bool(true),
	}

	iter := price.List(params)
	for iter.Next() {
		p := iter.Price()
		if p.UnitAmount == amount && p.Recurring != nil && string(p.Recurring.Interval) == interval {
			return p.ID, nil
		}
	}
	if err := iter.Err(); err != nil {
		return "", err
	}
	return "", nil
}

// readKeyFromConfig reads stripe.secret_key from a YAML config file.
func readKeyFromConfig(path string) string {
	// Simple approach: use viper to read just the key we need.
	// We import viper since it's already a dependency.
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	// Quick parse — look for secret_key line under stripe section.
	// Full viper parsing would pull in the whole config package; keep this standalone.
	lines := splitLines(string(data))
	inStripe := false
	for _, line := range lines {
		trimmed := trimSpace(line)
		if trimmed == "stripe:" {
			inStripe = true
			continue
		}
		if inStripe && len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
			inStripe = false
		}
		if inStripe && contains(trimmed, "secret_key:") {
			val := trimSpace(after(trimmed, "secret_key:"))
			val = trimQuotes(val)
			return val
		}
	}
	return ""
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func trimSpace(s string) string {
	i, j := 0, len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t') {
		j--
	}
	return s[i:j]
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchString(s, sub) >= 0
}

func searchString(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func after(s, sep string) string {
	idx := searchString(s, sep)
	if idx < 0 {
		return s
	}
	return s[idx+len(sep):]
}

func trimQuotes(s string) string {
	if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		return s[1 : len(s)-1]
	}
	return s
}

// ensurePortalConfiguration creates or updates the Stripe Billing Portal Configuration.
// It is idempotent: lists existing configurations first, updates if found, creates otherwise.
func ensurePortalConfiguration() error {
	fmt.Println()
	fmt.Println("Setting up Billing Portal Configuration...")

	existingID, err := findPortalConfiguration()
	if err != nil {
		return fmt.Errorf("list portal configurations: %w", err)
	}

	if existingID != "" {
		return updatePortalConfiguration(existingID)
	}
	return createPortalConfiguration()
}

// findPortalConfiguration returns the ID of an existing active portal configuration, or empty string.
func findPortalConfiguration() (string, error) {
	params := &stripe.BillingPortalConfigurationListParams{}
	params.Filters.AddFilter("active", "", "true")
	params.Filters.AddFilter("limit", "", "1")

	iter := portalconfig.List(params)
	for iter.Next() {
		cfg := iter.BillingPortalConfiguration()
		return cfg.ID, nil
	}
	if err := iter.Err(); err != nil {
		return "", err
	}
	return "", nil
}

// createPortalConfiguration creates a new Billing Portal Configuration.
func createPortalConfiguration() error {
	params := &stripe.BillingPortalConfigurationParams{
		BusinessProfile: &stripe.BillingPortalConfigurationBusinessProfileParams{
			Headline: stripe.String("ByteBrew — AI Agent for Software Engineers"),
		},
		Features: &stripe.BillingPortalConfigurationFeaturesParams{
			SubscriptionCancel: &stripe.BillingPortalConfigurationFeaturesSubscriptionCancelParams{
				Enabled:           stripe.Bool(true),
				Mode:              stripe.String("at_period_end"),
				ProrationBehavior: stripe.String("none"),
			},
			SubscriptionUpdate: &stripe.BillingPortalConfigurationFeaturesSubscriptionUpdateParams{
				Enabled: stripe.Bool(false),
			},
			InvoiceHistory: &stripe.BillingPortalConfigurationFeaturesInvoiceHistoryParams{
				Enabled: stripe.Bool(true),
			},
			PaymentMethodUpdate: &stripe.BillingPortalConfigurationFeaturesPaymentMethodUpdateParams{
				Enabled: stripe.Bool(true),
			},
			CustomerUpdate: &stripe.BillingPortalConfigurationFeaturesCustomerUpdateParams{
				Enabled:        stripe.Bool(true),
				AllowedUpdates: stripe.StringSlice([]string{"email"}),
			},
		},
	}

	cfg, err := portalconfig.New(params)
	if err != nil {
		return fmt.Errorf("create portal configuration: %w", err)
	}
	fmt.Printf("Created portal configuration: %s\n", cfg.ID)
	return nil
}

// updatePortalConfiguration updates an existing Billing Portal Configuration.
func updatePortalConfiguration(id string) error {
	params := &stripe.BillingPortalConfigurationParams{
		BusinessProfile: &stripe.BillingPortalConfigurationBusinessProfileParams{
			Headline: stripe.String("ByteBrew — AI Agent for Software Engineers"),
		},
		Features: &stripe.BillingPortalConfigurationFeaturesParams{
			SubscriptionCancel: &stripe.BillingPortalConfigurationFeaturesSubscriptionCancelParams{
				Enabled:           stripe.Bool(true),
				Mode:              stripe.String("at_period_end"),
				ProrationBehavior: stripe.String("none"),
			},
			SubscriptionUpdate: &stripe.BillingPortalConfigurationFeaturesSubscriptionUpdateParams{
				Enabled: stripe.Bool(false),
			},
			InvoiceHistory: &stripe.BillingPortalConfigurationFeaturesInvoiceHistoryParams{
				Enabled: stripe.Bool(true),
			},
			PaymentMethodUpdate: &stripe.BillingPortalConfigurationFeaturesPaymentMethodUpdateParams{
				Enabled: stripe.Bool(true),
			},
			CustomerUpdate: &stripe.BillingPortalConfigurationFeaturesCustomerUpdateParams{
				Enabled:        stripe.Bool(true),
				AllowedUpdates: stripe.StringSlice([]string{"email"}),
			},
		},
	}

	cfg, err := portalconfig.Update(id, params)
	if err != nil {
		return fmt.Errorf("update portal configuration: %w", err)
	}
	fmt.Printf("Updated portal configuration: %s\n", cfg.ID)
	return nil
}
