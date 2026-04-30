# Changelog

## [Unreleased] — 2026-04-28

### Added
- `BYTEBREW_BOOTSTRAP_ADMIN_TOKEN` env support: when set, engine seeds an admin
  API token in `api_tokens` on first boot (idempotent — skipped when
  `name="bootstrap-admin"` already exists). Enables automated declarative
  GitOps reconcile via `brewctl config-apply` in k8s deployments without
  manual Admin UI token generation.
  Format: `bb_<64-hex>`. Generate: `echo "bb_$(openssl rand -hex 32)"`.
  Scope: admin (mask=16). Name: `bootstrap-admin`.

## Architecture — CE/EE/Cloud Unification (pre-release)

Initial canonical architecture for ByteBrew Engine. Frozen pre-release — no
prior production clients, no upgrade path.

### Identity
- End-user identity is external — JWT `sub` claim — persisted as varchar
  (`sessions.user_sub`, `memories.user_sub`, `audit_logs.actor_sub`,
  `api_tokens.user_sub`). No `users` table; no UUID FKs to a user record.

### Auth
- EdDSA (Ed25519) is the only JWT algorithm. No HS256 shared-secret path.
- `BYTEBREW_AUTH_MODE=local`: engine auto-generates an Ed25519 keypair under
  `BYTEBREW_JWT_KEYS_DIR` on first boot; admin sessions minted via
  `POST /api/v1/auth/local-session` (sub=`local-admin`, tenant_id empty).
  Single-replica use only.
- `BYTEBREW_AUTH_MODE=external`: engine loads the issuer's public key from
  `BYTEBREW_JWT_PUBLIC_KEY_PATH`; no local-session route. Multi-replica safe.
- Admin SPA selects flow at build time via `VITE_AUTH_MODE` (and
  `VITE_LANDING_URL` for external handoff).

### API Contract
- PATCH is the partial-update verb on every resource (agents, schemas,
  models, knowledge-bases, mcp-servers). PUT is strict full-replace and
  returns 400 when required fields are missing.
- Resource references accept UUID **or** name; a single resolver layer
  (`ResolveAgentRef`, `ResolveModelRef`, …) canonicalises before DB reach.
- `models.kind` ∈ {`chat`, `embedding`} — application-layer validation
  rejects kind-mismatches on agent/KB assignment.

### Multi-tenancy
- Every tenant-scoped table carries `tenant_id` (not nullable, default
  installs to `00000000-0000-0000-0000-000000000001`). Cross-tenant reads
  return 404; writes respect the JWT tenant claim.
- MCP transport policy is DI-injected per deployment: permissive (CE) or
  restricted (Cloud — stdio/shell transports rejected at `400`).

### Observability + Security
- Security headers applied to every HTTP response (nosniff, frame-ancestors,
  CSP, referrer-policy; HSTS when TLS/X-Forwarded-Proto https).
- CORS is whitelist-only — empty config means same-origin; no wildcard.
- Widget routes use a schema-scoped CSP with per-tenant `widget_embed_origins`
  read from the `settings` table.
- All `slog` calls use the `*Context` variant; ctx-lint + slog-lint enforce
  in CI.

