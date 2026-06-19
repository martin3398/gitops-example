# Kyverno Runbook (Dev)

This runbook covers the Kyverno baseline deployed by Flux and the currently active policy set.

## Components

- Kyverno Helm source: `kubernetes/platform/dev/core-services/operators/kyverno/helmrepository.yaml`
- Kyverno Helm release: `kubernetes/platform/dev/core-services/operators/kyverno/helmrelease.yaml`
- Namespace: `kyverno`
- Flux platform core dependency: `kubernetes/flux/clusters/dev/core/kustomization-platform-core.yaml`

## Current Baseline

- Kyverno chart `kyverno` version `3.3.7` is managed by Flux.
- `admissionController` runs with `2` replicas for webhook availability.
- `backgroundController`, `cleanupController`, and `reportsController` are enabled with explicit resource requests/limits.
- Kyverno is deployed as part of `platform-core` (`kubernetes/platform/dev/core-services/kustomization.yaml`).

## Policy Inventory (Current)

- Policy set entrypoint: `kubernetes/platform/dev/apps/policies/kustomization.yaml`
- Active policy: `kubernetes/platform/dev/apps/policies/clusterpolicy-prefer-non-ceph-nodes.yaml`

Policy behavior summary:

- Type: `ClusterPolicy` mutate rule
- Action mode: `validationFailureAction: Audit`
- Rule injects soft node affinity (`preferredDuringSchedulingIgnoredDuringExecution`) to prefer nodes without `storage=ceph`.
- Target kinds: `Deployment`, `StatefulSet`, `DaemonSet`, `Job`, `CronJob`
- Excludes system/platform namespaces and workloads with label `placement.gitops-showcase.io/allow-ceph-nodes: "true"`

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

3. Verify policy registration:

```bash
kubectl get clusterpolicy
kubectl get clusterpolicy prefer-non-ceph-nodes -o yaml
```

4. Spot-check mutation on a sample deployment:

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
- If policies are missing, confirm `kubernetes/platform/dev/apps/policies/` is included by `kubernetes/platform/dev/apps/kustomization.yaml` and Flux app kustomizations are ready.
- If mutation is not visible on a workload, check policy exclusions (namespace or opt-out label).
