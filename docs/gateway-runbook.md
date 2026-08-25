# Gateway API Runbook (Cilium, Dev)

This runbook covers the edge routing baseline implemented with the Kubernetes Gateway API and Cilium.

## Scope

- Gateway class: `cilium`
- Gateway: `gateway/dev-gateway`
- Protocol: HTTP on host port `30080` (mapped to public AWS NLB on port `80`)
- Security: Cross-namespace route attachment protected by `gateway-access: "true"` namespace label
- Supported routes: HTTPRoute and GRPCRoute

## Manifests

- `kubernetes/infrastructure/base/gateway/namespace.yaml`
- `kubernetes/infrastructure/base/gateway/gateway.yaml`
- `kubernetes/infrastructure/base/gateway/cilium-gatewayclassconfig-rbac.yaml`
- `kubernetes/infrastructure/base/gateway/cilium-gateway-endpointslice-rbac.yaml`
- `kubernetes/infrastructure/overlays/dev/gateway/kustomization.yaml`
- `kubernetes/clusters/dev/kustomization-infrastructure-gateway.yaml`

## Architecture & Data Flow

```text
[ Browser / Client ]
        │
        ▼ (HTTP :80)
[ AWS Network Load Balancer (NLB) ]
        │
        ▼ (TCP Forwarding to Worker Targets :30080)
[ Cilium Envoy Proxy (HostNetwork on Workers :30080) ]
        │
        ├── Host: gitops.local /api/*     ──► visit-web/visit-gateway:80
        ├── Host: gitops.local /*          ──► visit-web/visit-web:80
        ├── Host: grafana.gitops.local     ──► monitoring/kube-prometheus-stack-grafana:80
        ├── Host: prometheus.gitops.local  ──► monitoring/kube-prometheus-prometheus:9090
        └── Host: bao.gitops.local         ──► openbao/openbao:8200
```

## Route Inventory

| Hostname | Path Prefix | Backend Service | Namespace | Definition |
| :--- | :--- | :--- | :--- | :--- |
| `gitops.local` | `/api` | `visit-web-visit-gateway` | `visit-web` | `charts/visit-gateway/templates/httproute.yaml` |
| `gitops.local` | `/` | `visit-web-visit-web` | `visit-web` | `charts/visit-ui/templates/httproute.yaml` |
| `grafana.gitops.local` | `/` | `monitoring-kube-prometheus-stack-grafana` | `monitoring` | `helmrelease-kube-prometheus-stack.yaml` |
| `prometheus.gitops.local` | `/` | `monitoring-kube-prometheus-prometheus` | `monitoring` | `helmrelease-kube-prometheus-stack.yaml` |
| `bao.gitops.local` | `/` | `openbao` | `openbao` | `helmrelease.yaml` (OpenBao) |

## Cross-Namespace Route Permissions

The `dev-gateway` listener restricts route attachments to namespaces explicitly labeled with `gateway-access: "true"`:

```yaml
listeners:
  - name: http
    port: 30080
    protocol: HTTP
    allowedRoutes:
      namespaces:
        from: Selector
        selector:
          matchLabels:
            gateway-access: "true"
```

To expose a new workload through the Gateway:
1. Label the target namespace: `kubectl label ns <namespace> gateway-access=true`
2. Create an `HTTPRoute` referencing `dev-gateway` in the `gateway` namespace as `parentRefs`.

## Accessing Endpoints

### 1. Via Public AWS Ingress NLB (Recommended)
Generate `/etc/hosts` entries using the Taskfile helper:
```bash
task ingress:hosts_entries
```
Output:
```text
<NLB_IP> gitops.local grafana.gitops.local prometheus.gitops.local bao.gitops.local
```
Add the line to `/etc/hosts` on your workstation, then open `http://gitops.local` or `http://grafana.gitops.local` in your browser.

### 2. Via Direct Node IP
```bash
curl -H "Host: gitops.local" http://<WORKER_IP>:30080/api/v1/visits/count
```

### 3. Via Port-Forwarding
```bash
# Grafana
kubectl -n monitoring port-forward svc/monitoring-kube-prometheus-stack-grafana 3000:80
# OpenBao UI
kubectl -n openbao port-forward svc/openbao-ui 8200:8200
```

## Validation & Health Checks

Verify Gateway and HTTPRoute statuses:
```bash
kubectl get gateway -n gateway
kubectl get httproute -A
task pipeline:verify
```

Expected status:
- Gateway `Accepted=True`
- HTTPRoutes `ResolvedRefs=True`

Note: In Cilium `hostNetwork` mode without a cloud controller LoadBalancer integration, the Gateway reports `Programmed=False (AddressNotAssigned)`. This is normal behavior because Envoy binds directly to the node's host network rather than an allocated cloud IP.

## Troubleshooting

1. **HTTPRoute reports `ResolvedRefs: False`**:
   - Verify backend service and endpoints exist: `kubectl -n <ns> get endpoints <svc-name>`
   - Verify namespace has the `gateway-access: "true"` label: `kubectl get ns <ns> --show-labels`
   - Nudge Cilium route controller to resync: `kubectl -n <ns> annotate httproute <name> reconcile.force=$(date +%s) --overwrite`

2. **Public NLB returns `Connection Refused` on port 80**:
   - Check that `gateway.yaml` listener port is `30080` matching `ingress_nodeport_http` in Terraform.
   - Verify Envoy is listening on workers: `nc -zv <worker-ip> 30080`

3. **Traffic flow debugging with Hubble**:
   ```bash
   cilium hubble observe --http-status 200
   cilium hubble observe --namespace visit-web
   ```
