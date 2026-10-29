# Kyverno Runbook (Dev)

This runbook covers the Kyverno baseline deployed by Flux and the currently active policy set.

## Components

- Kyverno Helm source: `kubernetes/infrastructure/base/core-services/operators/kyverno/helmrepository.yaml`
- Kyverno Helm release: `kubernetes/infrastructure/base/core-services/operators/kyverno/helmrelease.yaml`
- Namespace: `kyverno`
- Flux platform core dependency: `kubernetes/clusters/dev/kustomization-infrastructure-core.yaml`

## Current Baseline

- Kyverno chart `kyverno` version `3.3.7` is managed by Flux.
- `admissionController` runs with `2` replicas for webhook availability.
- `backgroundController`, `cleanupController`, and `reportsController` are enabled with explicit resource requests/limits.
- Kyverno is deployed as part of `infrastructure-core` (`kubernetes/infrastructure/base/core-services/kustomization.yaml`).

## Policy Inventory (Active)

- Policy stages:
  - `kubernetes/infrastructure/base/scheduling/policies/clusterpolicy-require-pss-labels.yaml`
  - `kubernetes/infrastructure/base/scheduling/policies/clusterpolicy-tiny-pod-requests.yaml`
  - `kubernetes/apps/overlays/dev/policies/clusterpolicy-prefer-non-ceph-nodes.yaml`

Policy behavior summary:

1. **`require-pss-labels`**:
   - Type: `ClusterPolicy` validate rule
   - Action mode: `validationFailureAction: Enforce`
   - Rule enforces that every new or modified `Namespace` explicitly declares `pod-security.kubernetes.io/enforce` (`restricted`, `baseline`, or `privileged`) and `pod-security.kubernetes.io/enforce-version` (`v1.30` or `latest`).
   - Excludes system namespaces: `kube-system`, `kube-public`, `kube-node-lease`, `default`, `flux-system`, `kyverno`, `local-path-storage`.
   - Prevents unhardened or unclassified namespaces from being deployed into the cluster.

2. **`tiny-pod-requests`**:
   - Type: `ClusterPolicy` mutate rule
   - Action mode: `validationFailureAction: Enforce`
   - Rule injects default minimal requests (`cpu: 5m`, `memory: 32Mi`) onto container and initContainer definitions if omitted.
   - Excludes system namespaces: `kube-system`, `kyverno`, `rook-ceph`.

3. **`prefer-non-ceph-nodes`**:
   - Type: `ClusterPolicy` mutate rule
   - Action mode: `validationFailureAction: Enforce`
   - Rule injects soft node affinity (`preferredDuringSchedulingIgnoredDuringExecution`) to prefer nodes without `storage=ceph`.
   - Target kinds: `Deployment`, `StatefulSet`, `DaemonSet`, `Job`, `CronJob`.
   - Excludes system/platform namespaces and workloads with label `placement.gitops-showcase.io/allow-ceph-nodes: "true"`.

---

## Pod Security Standards (PSS) Integration (`TASK-P4-03`)

The cluster implements native Kubernetes Pod Security Admission across three tiers:

| Namespace Category | Namespaces | PSS Profile | Enforced Controls |
| :--- | :--- | :--- | :--- |
| **Application Workloads** | `visit-web`, `visit-processing`, `visit-loadgen` | `restricted` (`v1.30`) | Disallow root user (`runAsNonRoot: true`), require `RuntimeDefault` seccomp, block privilege escalation, drop all capabilities, disallow host namespaces/hostPath, require read-only root filesystems. |
| **Platform Services** | `data-postgres`, `data-kafka`, `openbao`, `gateway` | `baseline` (`v1.30`) | Block known privilege escalations, disallow host namespaces/hostPath, allow standard non-root platform operator workloads. |
| **Host Storage & Telemetry** | `rook-ceph`, `monitoring` | `privileged` (`v1.30`) / `warn: baseline` | Allows hostPath volumes and host networking required by NVMe Ceph OSDs, CSI drivers, and node telemetry daemons (`node-exporter`, `promtail`). |

### Workload Hardening Baseline

All visit-demo application microservices Helm charts (`visit-gateway`, `visit-processor`, `visit-ui`, `visit-loadgen`) enforce the `restricted` security context:

```yaml
spec:
  securityContext:
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault
  containers:
    - name: app
      securityContext:
        allowPrivilegeEscalation: false
        readOnlyRootFilesystem: true
        runAsNonRoot: true
        capabilities:
          drop:
            - ALL
      volumeMounts:
        - name: tmp
          mountPath: /tmp
  volumes:
    - name: tmp
      emptyDir: {}
```

---

## Validation

Use a workstation with cluster access.

1. Verify Flux reconciliation and Kyverno release health:

```bash
kubectl -n flux-system get kustomizations
kubectl -n flux-system get helmrelease kyverno
```

2. Verify Kyverno controllers are running:

```bash
kubectl -n kyverno get pods
kubectl -n kyverno get deploy
```

3. Verify policy registration and `Enforce` mode:

```bash
kubectl get clusterpolicy
kubectl get clusterpolicy require-pss-labels -o jsonpath='{.spec.validationFailureAction}'
kubectl get clusterpolicy prefer-non-ceph-nodes -o jsonpath='{.spec.validationFailureAction}'
kubectl get clusterpolicy tiny-pod-requests -o jsonpath='{.spec.validationFailureAction}'
```

4. Verify Pod Security Standards namespace labels:

```bash
kubectl get ns -L pod-security.kubernetes.io/enforce,pod-security.kubernetes.io/warn,pod-security.kubernetes.io/audit
```

5. Test Kyverno rejection of an unlabelled namespace:

```bash
kubectl apply --dry-run=server -f - <<'EOF'
apiVersion: v1
kind: Namespace
metadata:
  name: test-unlabelled-ns
EOF
```

Expected output:
```text
Error from server: error when creating "STDIN": admission webhook "validate.kyverno.svc-fail" denied the request: 

policy Namespace//test-unlabelled-ns for resource violation: 

require-pss-labels:
  check-pss-enforce-labels: 'Every tenant/platform namespace must explicitly define ''pod-security.kubernetes.io/enforce'' (restricted, baseline, or privileged) and ''pod-security.kubernetes.io/enforce-version''.'
```

6. Test Pod Security Admission rejection in application namespace:

```bash
kubectl -n visit-web apply --dry-run=server -f - <<'EOF'
apiVersion: v1
kind: Pod
metadata:
  name: pss-violation-smoke
spec:
  containers:
    - name: app
      image: busybox:1.36
      securityContext:
        privileged: true
EOF
```

Expected output:
```text
Error from server (Forbidden): pods "pss-violation-smoke" is forbidden: violates PodSecurity "restricted:v1.30": privileged (container "app" must not set securityContext.privileged=true)
```

6. Spot-check Kyverno node affinity mutation on a sample deployment:

```bash
kubectl apply -f - <<'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kyverno-affinity-smoke
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kyverno-affinity-smoke
  template:
    metadata:
      labels:
        app: kyverno-affinity-smoke
    spec:
      containers:
        - name: app
          image: busybox:1.36
          command:
            - sh
            - -c
            - "sleep 3600"
EOF

kubectl -n default get deploy kyverno-affinity-smoke -o yaml
kubectl -n default delete deploy kyverno-affinity-smoke
```

In the deployment spec, confirm `spec.template.spec.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution` is present.

## Troubleshooting

- If Kyverno pods are not ready, inspect events and deployment status in `kyverno` namespace.
- If policies are missing, confirm `kubernetes/clusters/dev/apps/kustomization-app-policies.yaml` and `infrastructure-scheduling` are Ready in `flux-system`.
- If a pod creation is blocked with `violates PodSecurity`, inspect pod spec against `restricted` standards (must specify `runAsNonRoot: true`, `seccompProfile.type: RuntimeDefault`, `allowPrivilegeEscalation: false`, `capabilities.drop: ["ALL"]`, and avoid host namespaces/volumes).
- If mutation is not visible on a workload, check policy exclusions (namespace or opt-out label).

## Implemented Hardening Steps

- **Pod Security Standards & Enforce Mode (`TASK-P4-03`)**: Implemented. Namespaces labeled with PSS (`restricted` / `baseline` / `privileged`), Kyverno policies switched to `validationFailureAction: Enforce`, application charts hardened with non-root security contexts, and automated verification integrated in `ansible/playbooks/verify.yml`.
