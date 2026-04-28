# Changelog

All notable changes to the `bytebrew-engine` Helm chart will be documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this chart adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0] - 2026-04-28

### Added
- Liquibase migrations Job (`pre-install,pre-upgrade` Helm hook). Runs the
  `bytebrew/engine-migrations` image against `DATABASE_URL` before every
  install or upgrade. Toggle via `migrations.enabled` (default: `true`).
- `brewctl` config-apply Job (`post-install,post-upgrade` Helm hook) for
  declarative GitOps reconcile via the `brewctl` CLI. Waits for engine
  readiness via an init container before applying. Optional via
  `configApply.enabled` (default: `false`).
- ConfigMap template (`configmap-bytebrew.yaml`) for inline `bytebrew.yaml`
  config-as-code. Rendered only when `configApply.enabled=true` and
  `configApply.config` is non-empty and `configApply.existingConfigMap` is unset.
- Example values for ByteBrew EE on-prem (`values-ee.yaml`) and Cloud
  multi-tenant (`values-cloud.yaml`).
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
