# bytebrew-engine Helm Chart

Helm chart for deploying the **ByteBrew AI Agent Engine** (Community Edition) on Kubernetes.

## Quick Install

```bash
helm install bytebrew-engine oci://ghcr.io/syntheticinc/charts/bytebrew-engine \
  --version 0.4.0 \
  --set image.tag=v1.0.0 \
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
