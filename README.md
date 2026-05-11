# Argo Rollouts Config Mapper

A Kubernetes mutating admission webhook that dynamically rewrites ConfigMap and Secret references in Pod specs during ArgoCD Rollouts preview deployments.

## Overview

In ArgoCD Rollouts, during a "Preview" step, Pods receive an ephemeral label (`rollouts-preview-hash`). When this label is present, the webhook appends a suffix (default: `-preview`) to ConfigMap and Secret references declared in an allow-list annotation. When the label is absent (promotion), the suffix is stripped.

This enables preview Pods to consume preview-specific configuration without modifying the Rollout manifest.

## How It Works

1. A Pod is submitted for creation.
2. The webhook checks for the opt-in annotation `config-mapper.lsdopen.io/mutate: "true"`.
3. If opted in, it reads the allow-list annotations to determine which ConfigMaps/Secrets to mutate.
4. If the label `config-mapper.lsdopen.io/preview: "true"` is present → appends the suffix.
5. If the label is absent or not `"true"` → strips the suffix (promotion cleanup).

The preview label is injected automatically by ArgoCD Rollouts via `previewMetadata`:

```yaml
strategy:
  blueGreen:
    previewMetadata:
      labels:
        config-mapper.lsdopen.io/preview: "true"
```

## Annotations

| Annotation | Required | Description |
|---|---|---|
| `config-mapper.lsdopen.io/mutate` | Yes | Set to `"true"` to opt in |
| `config-mapper.lsdopen.io/configmaps` | No | Comma-separated list of ConfigMap names to mutate |
| `config-mapper.lsdopen.io/secrets` | No | Comma-separated list of Secret names to mutate |
| `config-mapper.lsdopen.io/suffix` | No | Custom suffix (default: `preview`) |

## Example

```yaml
apiVersion: v1
kind: Pod
metadata:
  annotations:
    config-mapper.lsdopen.io/mutate: "true"
    config-mapper.lsdopen.io/secrets: "performance-api-file-ext-secret"
  labels:
    config-mapper.lsdopen.io/preview: "true"  # Injected by ArgoCD Rollouts via previewMetadata
spec:
  containers:
    - name: app
      env:
        - name: API_KEY
          valueFrom:
            secretKeyRef:
              name: performance-api-file-ext-secret  # → becomes performance-api-file-ext-secret-preview
              key: api-key
```

## Mutation Scope

The webhook mutates references in:

- `spec.containers[*].env[*].valueFrom.configMapKeyRef.name`
- `spec.containers[*].env[*].valueFrom.secretKeyRef.name`
- `spec.containers[*].envFrom[*].configMapRef.name`
- `spec.containers[*].envFrom[*].secretRef.name`
- `spec.initContainers[*]` — same as containers
- `spec.ephemeralContainers[*]` — same as containers
- `spec.volumes[*].configMap.name`
- `spec.volumes[*].secret.secretName`

## Installation

### Using Helm (Recommended)

Install from GitHub Container Registry:

```bash
# Install the latest version
helm install argo-rollouts-config-mapper oci://ghcr.io/lsdopen/argo-rollouts-config-mapper/charts/argo-rollouts-config-mapper

# Install a specific version
helm install argo-rollouts-config-mapper oci://ghcr.io/lsdopen/argo-rollouts-config-mapper/charts/argo-rollouts-config-mapper --version 0.1.0

# Install with custom values
helm install argo-rollouts-config-mapper oci://ghcr.io/lsdopen/argo-rollouts-config-mapper/charts/argo-rollouts-config-mapper --values my-values.yaml
```

### From Source

```bash
helm install argo-rollouts-config-mapper ./chart
```

For detailed configuration options, see the [Helm Chart README](chart/README.md).

## Development

### Prerequisites

- Go 1.25+
- Docker (for container builds)

### Build

```bash
go build -o webhook .
```

### Test

```bash
go test ./...
```

### Container Build

```bash
docker build -f Containerfile -t argo-rollouts-config-mapper:dev .
```

### Multi-arch Build

```bash
docker buildx build -f Containerfile --platform linux/amd64,linux/arm64 -t argo-rollouts-config-mapper:dev .
```

## CI/CD

The project uses GitHub Actions with three workflows:

| Workflow | Trigger | Purpose |
|---|---|---|
| `ci.yml` | Pull requests | Lint, SAST, test, build, container scan |
| `release.yml` | Push to `main` (non-chart) | Semantic release + multi-arch Docker push to GHCR |
| `helm-release.yml` | Push to `main` (chart changes) | Package and push Helm chart to GHCR OCI |

Releases follow [Conventional Commits](https://www.conventionalcommits.org/) via semantic-release.

## License

Apache License 2.0
