# Argo Rollouts Config Mapper Helm Chart

A Kubernetes mutating admission webhook that dynamically rewrites ConfigMap and Secret references in Pod specs during ArgoCD Rollouts preview deployments.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+

## Installing the Chart

### From GitHub Container Registry (Recommended)

Install the latest version:

```bash
helm install argo-rollouts-config-mapper oci://ghcr.io/lsdopen/argo-rollouts-config-mapper/charts/argo-rollouts-config-mapper
```

Install a specific version:

```bash
helm install argo-rollouts-config-mapper oci://ghcr.io/lsdopen/argo-rollouts-config-mapper/charts/argo-rollouts-config-mapper --version 0.1.0
```

Install with custom values:

```bash
helm install argo-rollouts-config-mapper oci://ghcr.io/lsdopen/argo-rollouts-config-mapper/charts/argo-rollouts-config-mapper --values values-production.yaml
```

### From Source

```bash
helm install argo-rollouts-config-mapper ./chart
```

## Uninstalling the Chart

```bash
helm delete argo-rollouts-config-mapper
```

## Configuration

The chart comes with sensible defaults and requires no configuration for basic deployment.

### Default Configuration

The chart automatically configures:

- Image: `ghcr.io/lsdopen/argo-rollouts-config-mapper:latest` with `IfNotPresent` pull policy
- Service: ClusterIP on port 443
- Webhook: 5-second timeout with "Fail" policy
- Certificates: Helm-generated self-signed certificates (1-year validity)

### Optional Parameters

| Parameter | Description | Default |
|---|---|---|
| `replicaCount` | Number of replicas | `1` |
| `imagePullSecrets` | Image pull secrets | `[]` |
| `serviceAccount.create` | Create service account | `true` |
| `serviceAccount.annotations` | Service account annotations | `{}` |
| `serviceAccount.name` | Service account name | `""` |
| `podAnnotations` | Pod annotations | `{}` |
| `podSecurityContext` | Pod security context | `{}` |
| `securityContext` | Container security context | `{}` |
| `resources` | Resource limits and requests | `{}` |
| `nodeSelector` | Node selector | `{}` |
| `tolerations` | Tolerations | `[]` |
| `affinity` | Affinity rules | `{}` |
| `topologySpreadConstraints` | Topology spread constraints | `[]` |
| `webhook.timeoutSeconds` | Webhook timeout | `5` |
| `webhook.failurePolicy` | Webhook failure policy | `Fail` |
| `webhook.objectSelector` | Additional object selector expressions | `{}` |
| `webhook.namespaceSelector` | Additional namespace selector expressions | `{}` |
| `labels` | Additional labels for all resources | `{}` |
| `annotations` | Additional annotations for all resources | `{}` |

### Certificate Management

The webhook requires TLS certificates. The chart supports three methods:

#### 1. Helm-Generated Certificates (Default)

Helm automatically generates self-signed certificates during installation:

```yaml
certificates:
  method: "helm"
  helm:
    duration: "8760h" # 1 year
    subject:
      organizationName: "Argo Rollouts Config Mapper Webhook"
```

#### 2. cert-manager Integration (Recommended for Production)

```yaml
certificates:
  method: "cert-manager"
  certManager:
    issuer:
      name: "letsencrypt-prod"
      kind: "ClusterIssuer"
    duration: "2160h"    # 90 days
    renewBefore: "720h"  # 30 days
```

#### 3. External Certificate Management

```yaml
certificates:
  method: "external"
  external:
    secretName: "argo-rollouts-config-mapper-certs"
    certFile: "tls.crt"
    keyFile: "tls.key"
    caBundle: "<base64-encoded-ca-bundle>"
```

## Example Configurations

### High Availability

```yaml
replicaCount: 3

resources:
  limits:
    cpu: "200m"
    memory: "256Mi"
  requests:
    cpu: "100m"
    memory: "128Mi"

affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchExpressions:
            - key: app.kubernetes.io/name
              operator: In
              values:
                - argo-rollouts-config-mapper
        topologyKey: kubernetes.io/hostname

topologySpreadConstraints:
  - maxSkew: 1
    topologyKey: topology.kubernetes.io/zone
    whenUnsatisfiable: DoNotSchedule
    labelSelector:
      matchLabels:
        app.kubernetes.io/name: argo-rollouts-config-mapper
```

## Troubleshooting

### Common Issues

**Webhook not intercepting pods:**
Check that the MutatingWebhookConfiguration is properly configured and the service is accessible.

```bash
kubectl get mutatingwebhookconfiguration argo-rollouts-config-mapper
```

**Certificate errors:**
Ensure the TLS certificates are valid and the CA bundle matches the certificate authority.

**Pods not being mutated:**
Verify the Pod has the opt-in annotation:

```bash
kubectl get pod <pod-name> -o jsonpath='{.metadata.annotations.config-mapper\.lsdopen\.io/mutate}'
```

### Debugging Commands

```bash
# Check webhook pods
kubectl get pods -l app.kubernetes.io/name=argo-rollouts-config-mapper

# View webhook logs
kubectl logs -l app.kubernetes.io/name=argo-rollouts-config-mapper -f

# Check webhook configuration
kubectl get mutatingwebhookconfiguration argo-rollouts-config-mapper -o yaml
```
