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

---

## Cilium Network Policies & East-West Traffic Isolation

The platform enforces least-privilege east-west network segmentation across tenant namespaces using a **2-Tier Defense-in-Depth Architecture**.

### 1. Two-Tier Policy Architecture

```text
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                                 TIER 1: PLATFORM GUARDRAILS                             │
│                  CiliumClusterwideNetworkPolicy (infrastructure-security)               │
├───────────────────────────────┬───────────────────────────────┬─────────────────────────┤
│ CoreDNS Egress (:53)          │ Kubelet Health Probes (host)  │ Block AWS IMDSv1        │
│ monitoring Prometheus Ingress │ KubeAPI TokenReview (:443)    │ 169.254.169.254/32 Drop │
└───────────────────────────────┴───────────────────────────────┴─────────────────────────┘
                                             │
                                             ▼
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                           TIER 2: SERVICE-OWNED EAST-WEST ISOLATION                     │
│                             Namespaced CiliumNetworkPolicy (CNP)                        │
├───────────────────────────────┬───────────────────────────────┬─────────────────────────┤
│ data-postgres/                │ data-kafka/                   │ openbao/                │
│ Ingress :5432 (web/proc)      │ Ingress :9093 mTLS (web/proc) │ Ingress :8200 (GW, ESO) │
│ Egress Barman :80 (Ceph RGW)  │ KRaft peer ports :9090/:9091  │ Raft peer port :8201    │
├───────────────────────────────┴───────────────────────────────┴─────────────────────────┤
│ visit-web/ & visit-processing/ & visit-loadgen/                                         │
│ Envoy Gateway ingress, inter-service API routing, least-privilege DB/Kafka egress       │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```

1. **Tier 1 (Global Platform Guardrails - `CiliumClusterwideNetworkPolicy`)**:
   - Location: `kubernetes/infrastructure/base/security/policies/cluster-guardrails.yaml`
   - Governs all tenant namespaces (`data-postgres`, `data-kafka`, `openbao`, `visit-web`, `visit-processing`, `visit-loadgen`).
   - Globally enables CoreDNS egress (port 53), Kubelet health checks (`reserved:host`), Prometheus scraping from `monitoring`, and Kubernetes API egress (`:443`).
   - Globally blocks access to AWS EC2 instance metadata (`169.254.169.254/32` IMDSv1) via `egressDeny`.

2. **Tier 2 (Service-Owned Micro-Segmentation - `CiliumNetworkPolicy`)**:
   - Co-located with each subsystem in its respective directory:
     - `kubernetes/infrastructure/base/data-postgres/cilium-network-policy.yaml`: Port 5432 ingress restricted to `visit-gateway`, `visit-processor`, and peer replication; egress restricted to Ceph RGW (`rook-ceph:80`) for Barman WAL/base archiving.
     - `kubernetes/infrastructure/base/data-kafka/cilium-network-policy.yaml`: Port 9093 mTLS ingress restricted to `visit-gateway`, `visit-processor`, and `kafka-exporter`; KRaft peer ports 9090/9091 restricted to internal brokers.
     - `kubernetes/infrastructure/base/security/openbao/cilium-network-policy.yaml`: Port 8200 API ingress allowed from Envoy Gateway (`reserved:ingress`) and ESO; port 8201 Raft restricted to peers; snapshot job egress to Ceph RGW (`rook-ceph:80`).
     - `kubernetes/apps/base/visit-web/cilium-network-policy.yaml`: Port 3000/80 ingress from Envoy (`reserved:ingress`); port 8080 API ingress from Envoy, `visit-ui`, and `visit-loadgen`; egress to Postgres (`5432`) and Kafka (`9093`).
     - `kubernetes/apps/base/visit-processing/cilium-network-policy.yaml`: Consumer-only service; egress restricted to Kafka (`9093`) and Postgres (`5432`).
     - `kubernetes/apps/base/visit-loadgen/cilium-network-policy.yaml`: Egress restricted to `visit-gateway` (`8080`).

### 2. Inspecting Network Policies

```bash
# Check cluster-wide guardrails
kubectl get ccnp

# Check namespaced policies
kubectl get cnp -A
```

Expected output:
```text
NAME                  AGE
platform-guardrails   ...

NAMESPACE          NAME                               AGE
data-kafka         kafka-broker-isolation             ...
data-kafka         kafka-internal-clients-isolation   ...
data-postgres      postgres-isolation                 ...
openbao            openbao-server-isolation           ...
openbao            openbao-snapshot-isolation         ...
visit-loadgen      visit-loadgen-isolation            ...
visit-processing   visit-processor-isolation          ...
visit-web          visit-gateway-isolation            ...
visit-web          visit-ui-isolation                 ...
```

### 3. Traffic Flow & Drop Debugging with Hubble

```bash
# Follow dropped flows in real time
cilium hubble observe --verdict DROPPED -f

# Follow traffic between visit microservices
cilium hubble observe --from-namespace visit-web --to-namespace data-postgres
cilium hubble observe --from-namespace visit-processing --to-namespace data-kafka

# Filter by drop reason
cilium hubble observe --type drop
```

### 4. Testing Unauthorized Cross-Namespace Drops

Run a transient pod in an unauthorized namespace (e.g. `default` or `visit-loadgen`) attempting to reach PostgreSQL on port 5432 or OpenBao on port 8200:

```bash
# Attempt to reach postgres from default namespace (should time out / drop)
kubectl run -it --rm --restart=Never test-probe -n default \
  --image=curlimages/curl -- curl -m 3 http://postgres-rw.data-postgres.svc:5432 || echo "Dropped as expected"
```
Cilium eBPF drops the SYN packet immediately and Hubble registers a `Policy denied (ingress)` drop.

