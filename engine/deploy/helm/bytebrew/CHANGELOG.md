# Changelog

All notable changes to the `bytebrew-engine` Helm chart will be documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this chart adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
