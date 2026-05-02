# Changelog

All notable changes to the `bytebrew-engine` Helm chart will be documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this chart adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.2] - 2026-04-30

### Fixed
- **ServiceAccount hook ordering** — SA was a regular resource, but the
  pre-install migrations Job depends on it. On first install Helm tried to
  run the Job before creating the SA → `serviceaccount "..." not found`.
  SA template now declares `helm.sh/hook: pre-install,pre-upgrade` with
  weight `-10` and `before-hook-creation` delete policy so it always
  exists in time for hook Jobs.
- **Engine `HOME` not set** — engine writes its `server.port` discovery
  file to `~/.local/share/bytebrew/`. Under `runAsUser: 1000` without
  `HOME` set, the path resolved to `/.local`, which is not writable →
  `mkdir /.local: permission denied` → CrashLoopBackOff. Deployment
  template now sets `HOME=/tmp` explicitly.
- **Migrations Job no args** — the `bytebrew/engine-migrations` image is
  stock liquibase (entrypoint `/liquibase/docker-entrypoint.sh`, default
  Cmd `--help`). The chart Job did not pass any args → migrations never
  ran, the Job exited 0 after printing liquibase help, engine then crashed
  on `relation "agents" does not exist`. Job now overrides `command` with
  a POSIX shell wrapper that parses libpq `DATABASE_URL` into JDBC URL +
  URL-decoded username/password, then `exec`s the entrypoint with
  `--changeLogFile=migrations/db.changelog-master.yaml update`.
- **brewctl image tag** — chart default was `v0.1.0`, but the published
  GHCR tag is `0.1.0` (no `v` prefix). Default values now use `"0.1.0"`.
- **brewctl `command` override** — chart used `command: ["brewctl", ...]`
  but the brewctl image entrypoint is `/brewctl` (binary at root, not in
  PATH) → `executable file not found in $PATH`. Job now uses `args: [...]`
  so the entrypoint stays in effect.

### Security
- **No password leak via process argv** — migrations Job previously passed
  `--password=$DB_PASS` on liquibase argv, exposing the DB password to
  anyone with `pods/exec` or `ps -ef` rights inside the container. Now
  passed via `LIQUIBASE_COMMAND_PASSWORD` env var which never appears in
  argv. Same for username (`LIQUIBASE_COMMAND_USERNAME`).
- **DSN URL-decoding** — `DATABASE_URL` containing URL-encoded characters
  (e.g. password with `@` → `%40`) is now decoded before handoff to
  Liquibase. JDBC PostgreSQL driver does NOT URL-decode credentials, so
  passing the encoded form would have failed authentication on real-world
  managed Postgres credentials. Decoder is POSIX `printf '%b' / sed`.

### Added
- **Auto `/tmp` emptyDir when `readOnlyRootFilesystem: true`** — Deployment,
  migrations Job, and configApply Job now automatically mount an in-memory
  `/tmp` when the security context enables read-only root. Previously users
  had to manually wire `extraVolumes` + `extraVolumeMounts` and engine /
  Liquibase / brewctl would CrashLoop on first temp file write.
- **`replicaCount=1` guard for `auth.mode=local`** — chart now `fail`s at
  template time with a clear message if `replicaCount > 1` while
  `auth.mode=local`. Local mode persists JWT keypair on a single-writer
  PVC; multi-replica would race.
- `tests/` directory with kind-based smoke fixtures (excluded from
  `helm package` artefact via `.helmignore`):
  - `tests/values/default.yaml` — vanilla install with in-kind
    postgres-pgvector, no bootstrap token, no configApply
  - `tests/values/single-shot.yaml` — chirp-mono2-style flow with
    bootstrap admin token + configApply Helm hook
  - `tests/fixtures/postgres-pgvector.yaml` — Secret + ConfigMap +
    StatefulSet + Service for an in-kind pgvector Postgres. Init script
    is idempotent (re-runnable across smoke iterations) and the
    readiness probe verifies the `bytebrew` DB exists before signalling
    ready, eliminating a init-race flake on slow CI runners.
  - `tests/scripts/smoke.sh` — `/health` + admin REST endpoints +
    configApply Job completion check
- GitHub Actions workflow `.github/workflows/chart-test.yml` — static
  helm lint + render matrix with **regression-pinned greps** for each
  fixed bug, plus a kind v1.30 integration job that explicitly verifies
  the migrations Job ran Liquibase (not the transitive symptom of
  engine boot success). Triggered on PRs and pushes touching the chart.
- `.helmignore` — excludes `tests/` from the deployable chart artefact
  (test admin-token + smoke scripts must not leak into the OCI package).

### Changed
- Bumped chart `version` to `0.4.2`.
- Bumped `appVersion` to `1.0.2` (engine fail-fast on invalid bootstrap
  admin token format — see engine v1.0.2 release).

### Known limitations
- The ServiceAccount is rendered as a Helm hook (so it exists before
  pre-install Jobs). Hook resources are not tracked as release resources,
  so `helm uninstall` does NOT delete the SA — it is orphaned in the
  namespace until the namespace itself is deleted or the SA is manually
  removed (`kubectl delete sa <release>-bytebrew-engine`). On `helm
  upgrade` there is a brief sub-second window during pre-upgrade hook
  re-creation where the SA does not exist; existing pods retain their
  cached token, but new pods scheduled in that window will retry
  creation. Both limitations will be removed in v0.5.0 by moving
  migrations from a pre-install Helm hook to a Deployment init container.
- `helm rollback` does NOT downgrade the database schema. Liquibase
  `update` is forward-only. If you roll back to a chart revision whose
  engine image expects an older schema, the engine will crash on first
  DB read. Take a `pg_dump` before any upgrade and restore alongside the
  chart rollback. Documented in README "Known limitations".

## [0.4.1] - 2026-04-30

### Added
- `bootstrapAdminToken` section: when enabled, engine pod receives
  `BYTEBREW_BOOTSTRAP_ADMIN_TOKEN` env from a Secret. Engine v1.0.1+ seeds
  an admin API token in `api_tokens` on first boot, enabling single-shot
  GitOps deploy with `configApply.enabled=true` (no manual Admin UI token
  generation).
- Pair with `configApply.tokenSecret` pointing at the same Secret/key for
  DRY config: one Vault entry, two consumers (engine boot + brewctl Job).

### Changed
- Bumped `appVersion` to `1.0.1` (BOOTSTRAP_ADMIN_TOKEN feature requires it).
- Bumped chart `version` to `0.4.1`.

## [0.4.0] - 2026-04-30

### Added
- HTTPRoute template (Gateway API support) — opt-in via `httpRoute.enabled`. Tested with Envoy Gateway, Cilium Gateway API, and Istio Gateway.
- ServiceAccount template with configurable annotations for AWS IRSA and GCP Workload Identity. Toggle via `serviceAccount.create` (default: `true`).
- NetworkPolicy template — opt-in via `networkPolicy.enabled`. Configurable `ingressFrom` selectors; egress unrestricted by default (DNS, Postgres, LLM API).
- Escape hatches: `extraEnv`, `extraEnvFrom`, `extraVolumes`, `extraVolumeMounts`, `extraInitContainers`, `podAnnotations`, `podLabels` — applied to Deployment and Job pods.
- `podSecurityContext` (defaults: `fsGroup: 1000`, `runAsNonRoot: true`, `runAsUser: 1000`) and `containerSecurityContext` (defaults: `allowPrivilegeEscalation: false`, `readOnlyRootFilesystem: false`, `capabilities.drop: [ALL]`, `seccompProfile: RuntimeDefault`). `readOnlyRootFilesystem` is `false` by default to avoid CrashLoopBackOff from engine `/tmp` writes; opt-in pattern documented in README.
- `service.annotations` for cloud load-balancer hints (e.g. `service.beta.kubernetes.io/aws-load-balancer-internal`).
- README "Integrations" section with copy-paste examples for helmfile, External Secrets Operator + Vault, AWS IRSA, GCP Workload Identity, Argo CD, and read-only root filesystem opt-in.

## [0.3.0] - 2026-04-29

### Added
- Liquibase migrations Job (`pre-install,pre-upgrade` Helm hook). Runs the
  `bytebrew/engine-migrations` image against `DATABASE_URL` before every
  install or upgrade. Toggle via `migrations.enabled` (default: `true`).
- `brewctl` config-apply Job (`post-install,post-upgrade` Helm hook) for
  declarative GitOps reconcile via the `brewctl` CLI. Waits for engine
  readiness via an init container before applying. Optional via
  `configApply.enabled` (default: `false`).
  **Prerequisites:** brewctl Docker image (`ghcr.io/syntheticinc/brewctl:v0.1.0`
  or later) must be published before enabling `configApply.enabled=true`. See
  [bytebrew-brewctl releases](https://github.com/syntheticinc/bytebrew-brewctl/releases).
- ConfigMap template (`configmap-bytebrew.yaml`) for inline `bytebrew.yaml`
  config-as-code. Rendered only when `configApply.enabled=true` and
  `configApply.config` is non-empty and `configApply.existingConfigMap` is unset.
- Argo CD Application example (`examples/argocd-application.yaml`) with both
  Git-based and OCI-based source variants.
- GitHub Actions workflow `release-helm.yaml` publishing chart to
  `ghcr.io/syntheticinc/charts/bytebrew-engine` on `helm/v*.*.*` tags.

### Changed
- Bumped chart version to `0.3.0`.
- `NOTES.txt` updated with config-apply bootstrap instructions (step 3).

## [0.2.0] - earlier

Initial CE chart with engine Deployment, Service, Ingress (sticky sessions for
SSE), PVC for JWT keys, HPA, ServiceMonitor, and ConfigMap for `agents.yaml`.
