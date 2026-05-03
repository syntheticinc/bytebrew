#!/usr/bin/env bash
# Chart smoke — runs after `helm install` against an in-cluster engine
# reachable via kubectl port-forward.
#
# NOTE: tests/values/*.yaml currently pin engine image.tag to "1.0.1" because
# 1.0.2 may not yet be on Docker Hub at PR-author time. Chart appVersion is
# 1.0.2 (engine fail-fast on invalid bootstrap token); valid-token happy path
# is identical between 1.0.1 and 1.0.2 so smoke runs identically. Bump pins
# to "1.0.2" in a follow-up once the cloud-web deploy workflow has published
# bytebrew/engine:1.0.2 + bytebrew/engine-migrations:1.0.2.
#
# Required env:
#   ADMIN_TOKEN   bb_<64-hex> Bearer token for engine REST
#   ENGINE_URL    typically http://localhost:18443 (port-forward target)
#
# Optional env:
#   NAMESPACE     k8s namespace (default: default)
#   RELEASE       Helm release name (default: bb)
#
# Exit non-zero on any failure.
set -euo pipefail

NAMESPACE=${NAMESPACE:-default}
RELEASE=${RELEASE:-bb}
TOKEN=${ADMIN_TOKEN:?ADMIN_TOKEN env required}
ENGINE_URL=${ENGINE_URL:-http://localhost:18443}

echo "==> Wait for engine deployment ready"
kubectl -n "$NAMESPACE" rollout status \
  "deploy/${RELEASE}-bytebrew-engine" --timeout=300s

echo "==> GET /api/v1/health"
curl -fsS "$ENGINE_URL/api/v1/health" | jq -e '.status == "ok" or .status == "healthy"'

# REST endpoints — engine returns either a plain array or {data: [...]}.
# Smoke accepts both to stay neutral on the response envelope.
for endpoint in agents schemas models; do
  echo "==> GET /api/v1/$endpoint"
  curl -fsS "$ENGINE_URL/api/v1/$endpoint" \
    -H "Authorization: Bearer $TOKEN" \
    | jq -e 'type == "array" or has("data")'
done

# configApply Job runs as post-install Helm hook — when the scenario enables
# it the Job should already be Complete by the time `helm install --wait`
# returned. Guard with `--ignore-not-found` for scenarios without it.
echo "==> Verify configApply Job (if scenario enabled it)"
if kubectl -n "$NAMESPACE" get \
    "job/${RELEASE}-bytebrew-engine-config-apply" --ignore-not-found \
    -o name | grep -q job; then
  kubectl -n "$NAMESPACE" wait --for=condition=complete \
    "job/${RELEASE}-bytebrew-engine-config-apply" --timeout=120s

  # Catch v0.4.2 false-positive: brewctl `apply -f <dir>` walks subdirs only,
  # missed top-level ConfigMap-mounted bytebrew.yaml → "No changes" → Job
  # Completed → looked successful but ZERO resources created. Assert the
  # smoke bundle actually landed in engine.
  #
  # Gated behind EXPECT_BREWCTL_RESOURCES=true so scenarios that intentionally
  # ship an empty `models: []` bundle (e.g. restricted-security, where the
  # focus is engine-boot-under-readOnlyRootFilesystem, not brewctl flow)
  # don't trip on missing resources.
  if [ "${EXPECT_BREWCTL_RESOURCES:-}" = "true" ]; then
    echo "==> Assert configApply created the smoke resources"
    models=$(curl -fsS "$ENGINE_URL/api/v1/models" \
      -H "Authorization: Bearer $TOKEN" | jq -e 'map(select(.name == "kind-smoke-model")) | length')
    agents=$(curl -fsS "$ENGINE_URL/api/v1/agents" \
      -H "Authorization: Bearer $TOKEN" | jq -e 'map(select(.name == "kind-smoke-agent")) | length')
    schemas=$(curl -fsS "$ENGINE_URL/api/v1/schemas" \
      -H "Authorization: Bearer $TOKEN" | jq -e 'map(select(.name == "kind-smoke-schema")) | length')
    if [ "$models" != "1" ] || [ "$agents" != "1" ] || [ "$schemas" != "1" ]; then
      echo "FAIL: brewctl reported success but smoke resources missing — models=$models agents=$agents schemas=$schemas"
      exit 1
    fi
    echo "OK: brewctl created kind-smoke-{model,agent,schema}"
  fi
fi

echo "✅ Smoke pass"
