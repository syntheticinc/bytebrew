# ByteBrew Cloud API

REST API server for managing users, licenses, billing (Stripe), and teams.

## Stack

- **Go 1.24** + chi/v5 (HTTP router)
- **PostgreSQL 17** (pgx/v5 + SQLC)
- **Stripe** (checkout, webhooks, portal)
- **Ed25519** (license JWT signing)
- **golang-migrate** (migrations embedded in binary)

## Quick Start

### 1. PostgreSQL

```bash
# Docker (recommended)
docker compose up -d

# Or manually — create DB bytebrew with user bytebrew:bytebrew
# Default port: 5432
```

### 2. Configuration

```bash
cp config.example.yaml config.yaml
```

Minimal changes in `config.yaml`:

| Parameter | Description | Required |
|-----------|-------------|:---:|
| `database.url` | PostgreSQL connection string | yes |
| `auth.jwt_secret` | Auth JWT secret (any random string) | yes |
| `license.private_key_hex` | Ed25519 private key (hex) | yes |
| `stripe.secret_key` | Stripe secret key (`sk_test_...`) | for billing |
| `stripe.webhook_secret` | Stripe webhook signing secret (`whsec_...`) | for webhooks |
| `stripe.prices.*` | Stripe Price IDs | for billing |
| `deepinfra.api_key` | DeepInfra API key | for LLM proxy |
| `email.resend_api_key` | Resend API key | for email (optional) |

### 3. Generate Ed25519 Keys

```bash
go run ./cmd/keygen
```

Copy `private_key_hex` to `config.yaml` → `license.private_key_hex`.
Copy `public_key_hex` to bytebrew-srv config → `license.public_key_hex`.

### 4. Set Up Stripe Products / Prices

```bash
STRIPE_SECRET_KEY=sk_test_... go run ./cmd/stripe-setup
```

The script idempotently creates Products (Personal $20/mo, Teams $30/user/mo) and Prices in Stripe. Copy the printed Price IDs to `config.yaml` → `stripe.prices.*`.

### 5. Run

```bash
go run ./cmd/server

# Or with a specific config
go run ./cmd/server --config /path/to/config.yaml

# Migrations only (without starting the server)
go run ./cmd/server --migrate-only
```

Server listens on the port from `config.yaml` (default `:8080`, in dev usually `:60402`).

Migrations are applied automatically on startup.

## Stripe Webhook Forwarding (Local Development)

To handle Stripe webhooks locally (without a public domain), use Stripe CLI.

### Install Stripe CLI

```bash
# Windows (scoop)
scoop install stripe

# macOS
brew install stripe/stripe-cli/stripe

# Linux
# Download from https://github.com/stripe/stripe-cli/releases
```

### Authenticate

```bash
stripe login
```

### Start Forwarding

```bash
stripe listen --forward-to localhost:60402/api/v1/webhooks/stripe
```

Stripe CLI will output a **webhook signing secret** like `whsec_...`. Copy it to `config.yaml` → `stripe.webhook_secret`, then restart the server.

> Stripe CLI intercepts all events from your Stripe account and forwards them to your local server. While `stripe listen` is running, webhooks work locally.

### Testing Webhooks

```bash
# In a separate terminal (while stripe listen is running)
stripe trigger checkout.session.completed
stripe trigger customer.subscription.updated
stripe trigger invoice.payment_succeeded
```

## API Endpoints

### Auth

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/auth/register` | Registration (email + password) |
| POST | `/api/v1/auth/login` | Login → access + refresh tokens |
| POST | `/api/v1/auth/refresh` | Refresh access token |

### License (requires auth)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/license/activate` | Activation → license JWT |
| POST | `/api/v1/license/refresh` | Refresh license JWT |

### Billing (requires auth)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/billing/checkout` | Create Stripe Checkout Session |
| POST | `/api/v1/billing/portal` | Create Stripe Customer Portal |
| POST | `/api/v1/webhooks/stripe` | Stripe webhook endpoint |

### Usage (requires auth)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/usage` | Usage statistics |

### Teams (requires auth)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/teams` | Create team |
| GET | `/api/v1/teams/members` | List members + invites |
| POST | `/api/v1/teams/invite` | Invite member |
| DELETE | `/api/v1/teams/members/:id` | Remove member |
| POST | `/api/v1/teams/accept` | Accept invitation |

### Proxy (requires auth)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/proxy/llm` | LLM proxy gateway (DeepInfra) |

## License Generation (Dev)

```bash
go run ./cmd/genlicense <private_key_hex> <email> [tier] [days]
```

| Parameter | Required | Description | Default |
|-----------|:---:|-------------|:---:|
| `private_key_hex` | yes | Ed25519 private key (128 hex chars) | — |
| `email` | yes | User email | — |
| `tier` | no | `personal`, `teams`, `trial` | `personal` |
| `days` | no | License duration in days | `365` |

```bash
# Personal license, 1 year (key from config.yaml → license.private_key_hex)
go run ./cmd/genlicense "PRIVATE_KEY_HEX" "user@example.com"

# Teams license, 90 days
go run ./cmd/genlicense "PRIVATE_KEY_HEX" "user@example.com" "teams" "90"

# Save to file
go run ./cmd/genlicense "KEY" "user@example.com" > ~/.bytebrew/license.jwt         # Linux/macOS
go run ./cmd/genlicense "KEY" "user@example.com" > %APPDATA%/bytebrew/license.jwt  # Windows
```

`bytebrew-srv` reads `license.jwt` on startup and verifies the Ed25519 signature with the public key.

## Project Structure

```
bytebrew-cloud-api/
├── cmd/
│   ├── server/         # Entry point
│   ├── keygen/         # Ed25519 key generator
│   ├── genlicense/     # License JWT generator (dev)
│   └── stripe-setup/   # Stripe Products/Prices setup
├── internal/
│   ├── domain/         # Entities (User, Subscription, Team, License)
│   ├── usecase/        # Business logic (1 package per operation)
│   ├── delivery/http/  # HTTP handlers + middleware
│   └── infrastructure/
│       ├── postgres/   # Repositories (pgx + SQLC)
│       ├── crypto/     # Ed25519 signing, bcrypt, JWT
│       ├── stripe/     # Stripe checkout, price resolver
│       ├── email/      # Resend sender + noop fallback
│       └── ratelimit/  # Trial rate limiter
├── migrations/         # SQL migrations (embed)
├── queries/            # SQLC SQL files
├── pkg/
│   ├── config/         # Configuration (viper)
│   └── errors/         # Domain errors
└── config.example.yaml
```

## Tests

```bash
# All unit tests
go test ./...

# Specific package
go test ./internal/usecase/activate/...
```

## Environment Variables

Environment variables override values from `config.yaml`:

| Variable | Description |
|----------|-------------|
| `STRIPE_SECRET_KEY` | Stripe API key |
| `DEEPINFRA_API_KEY` | DeepInfra API key |
