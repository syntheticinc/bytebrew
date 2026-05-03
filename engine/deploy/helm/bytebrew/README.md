# bytebrew-engine Helm Chart

Helm chart for deploying the **ByteBrew AI Agent Engine** (Community Edition) on Kubernetes.

## Stability

Per OSS testing transparency convention (Kubernetes feature gates / Helm Artifact
Hub maturity / Apache Incubator pattern), each chart feature is labelled with how
it has been validated. Pick what your environment supports and treat
`Beta`/`Experimental` paths as community-feedback territory.

| Feature                                | Tier         | Tested how |
|----------------------------------------|--------------|------------|
| Default install (single-shot)          | **Stable**   | CI gate (kind v1.28/1.30/1.31, install + upgrade + rollback) + chirp-mono2 dev canary |
| External Postgres + ESO + Vault        | **Stable**   | CI gate + chirp-mono2 dev canary |
| Bootstrap admin token + configApply    | **Stable**   | CI gate (full single-shot flow) |
| Migrations Job (Liquibase)             | **Stable**   | CI gate asserts `databasechangelog` populated |
| HTTPRoute (Gateway API v1)             | **Stable**   | CI render-validated + chirp-mono2 dev canary against Envoy Gateway |
| `containerSecurityContext.readOnlyRootFilesystem: true` (auto /tmp emptyDir) | **Stable** | CI render-validated; smoke runs render-only on kind |
| `replicaCount=1` enforcement (`auth.mode=local`) | **Stable** | CI gate (template `fail` on `replicaCount > 1`) |
| AWS IRSA annotations                   | **Beta**     | CI render-validated only — no AWS account in CI; community feedback welcome |
| GCP Workload Identity annotations      | **Beta**     | CI render-validated only — no GCP account in CI |
| NetworkPolicy enabled                  | **Beta**     | CI render-validated only — kind default CNI does NOT enforce NetworkPolicy |
| Argo CD pull GitOps                    | **Experimental** | Example-only (`examples/argocd-application.yaml`); not exercised in CI |
| Flux CD HelmRelease                    | **Experimental** | Not exercised in CI |
| Multi-replica HA (`auth.mode=external`)| **Out of CE** | EE feature — requires external JWT IdP, not in this chart |

**Known limitations (v0.4.2):**

- **ServiceAccount as hook.** Rendered as a Helm pre-install/pre-upgrade hook
  so it exists in time for the migrations Job. Side effects: SA is not deleted
  on `helm uninstall` (orphan; clean up via `kubectl delete sa
  <release>-bytebrew-engine` or by deleting the namespace), and during
  `helm upgrade` there is a sub-second window during SA recreation where new
  pods cannot schedule. Both will be removed in v0.5.0 by relocating
  migrations from a hook to a Deployment init container.

- **`helm rollback` does NOT downgrade the database schema.** Liquibase
  `update` is forward-only. After rolling back to an older chart revision,
  the engine pod will still see the newer schema applied by the previous
  upgrade. If the older engine image references columns/tables that no
  longer exist (or expects narrower schema), it will crash on first DB read.
  **To roll back the engine version safely, also restore the database
  from a pre-upgrade snapshot** (e.g. `pg_dump` taken before `helm upgrade`).
  The chart cannot do this for you because rollback semantics are
  application-specific.

- **HPA + `auth.mode=local` is rejected at template time.** Local auth
  persists the JWT keypair on a single-writer PVC; multi-replica races and
  produces intermittent auth failures. The chart `fail`s at render if
  `autoscaling.enabled=true` while `autoscaling.maxReplicas > 1`. Use
  `auth.mode=external` for HA.

## Quick Install

```bash
helm install bytebrew-engine oci://ghcr.io/syntheticinc/charts/bytebrew-engine \
  --version 0.4.2 \
  --set image.tag=1.0.2 \
  --set postgresql.external.host=my-postgres \
  --set postgresql.external.password=secret
```

## Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.tag` | Engine image tag (pin to specific version) | `latest` |
| `replicaCount` | Number of replicas (local auth mode: 1 only) | `1` |
| `config.auth.mode` | Auth mode: `local` or `external` | `local` |
| `postgresql.external.host` | PostgreSQL host | `""` |
| `postgresql.external.existingSecret` | Existing Secret with `DATABASE_URL` key | `""` |
| `migrations.enabled` | Run Liquibase migrations Job on install/upgrade | `true` |
| `configApply.enabled` | Run brewctl config-apply Job on install/upgrade | `false` |
| `serviceAccount.create` | Create a ServiceAccount | `true` |
| `serviceAccount.annotations` | ServiceAccount annotations (IRSA, WI) | `{}` |
| `httpRoute.enabled` | Create Gateway API HTTPRoute | `false` |
| `networkPolicy.enabled` | Create NetworkPolicy | `false` |
| `podSecurityContext` | Pod-level security context | see values.yaml |
| `containerSecurityContext` | Container-level security context | see values.yaml |

See `values.yaml` for the full list of parameters.

## Integrations

### Helmfile (chirp-mono2 / similar GitOps stacks)

```yaml
repositories:
  - name: bytebrew
    url: oci://ghcr.io/syntheticinc/charts

releases:
  - name: bytebrew-engine
    chart: oci://ghcr.io/syntheticinc/charts/bytebrew-engine
    version: 0.4.0
    namespace: {{ .Environment.Name }}
    values:
      - ./values/{{ .Environment.Name }}/bytebrew-engine.yaml
    needs:
      - postgresql-bytebrew
```

### External Secrets Operator + Vault

Use `existingSecret` to skip the chart-managed Secret and pull `DATABASE_URL`
from your ESO-managed Secret:

```yaml
postgresql:
  external:
    existingSecret: bytebrew-config
    existingSecretKey: DATABASE_URL
```

### AWS IRSA

Bind an IAM Role to the engine pod via ServiceAccount annotation:

```yaml
serviceAccount:
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/bytebrew-engine
```

### GCP Workload Identity

```yaml
serviceAccount:
  annotations:
    iam.gke.io/gcp-service-account: bytebrew-engine@my-project.iam.gserviceaccount.com
```

### Argo CD

See `examples/argocd-application.yaml` for both Git-based and OCI-based source variants.

### Read-only root filesystem (opt-in security best practice)

Engine writes temp files to `/tmp`. To enable a read-only root filesystem:

```yaml
containerSecurityContext:
  readOnlyRootFilesystem: true

extraVolumes:
  - name: tmp
    emptyDir: {}

extraVolumeMounts:
  - name: tmp
    mountPath: /tmp
```

### Gateway API HTTPRoute (Envoy Gateway / Cilium / Istio)

```yaml
ingress:
  enabled: false

httpRoute:
  enabled: true
  routes:
    - nameSuffix: api
      hostnames:
        - bytebrew.dev.example.com
      parentRefs:
        - name: internal
          namespace: envoy-gateway-system
          sectionName: https-dev-example-com
      rules:
        - matches:
            - path: /
              pathType: PathPrefix
          servicePort: 8443
```

### NetworkPolicy

```yaml
networkPolicy:
  enabled: true
  ingressFrom:
    - podSelector: {}
    - namespaceSelector:
        matchLabels:
          kubernetes.io/metadata.name: envoy-gateway-system
```

### Single-shot deployment with `bootstrapAdminToken`

For fully-automated GitOps deploys (no manual Admin UI step), pre-mint
an admin token and reference it from both `bootstrapAdminToken.existingSecret`
(engine) and `configApply.tokenSecret` (brewctl Job):

```yaml
bootstrapAdminToken:
  enabled: true
  existingSecret: bytebrew-config

configApply:
  enabled: true
  tokenSecret: bytebrew-config
  apiKeysSecret: bytebrew-config
```

Engine seeds the token into `api_tokens` on first boot (idempotent — safe
to re-apply). Token format: `bb_<64-hex>` — generate via
`echo "bb_$(openssl rand -hex 32)"`.

Requires engine image **v1.0.1 or later** for `BYTEBREW_BOOTSTRAP_ADMIN_TOKEN`
soft seeding (invalid format → WARN log + skip seed). Engine **v1.0.2+** adds
fail-fast on invalid token format — process exits with a clear cause logged,
producing CrashLoopBackOff visible in `kubectl describe pod` (chart `appVersion`
1.0.2 reflects this).
