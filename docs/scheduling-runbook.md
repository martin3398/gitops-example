# Scheduling Runbook (Kyverno + Placement, Dev)

This runbook covers the current scheduling and placement policy layer.

## Components

- Kyverno operator: `kubernetes/infrastructure/base/core-services/operators/kyverno/`
- Cluster policy stage: `kubernetes/clusters/dev/apps/kustomization-app-policies.yaml`
- Policy overlay: `kubernetes/apps/overlays/dev/policies/`
- Ceph placement guidance: `kubernetes/infrastructure/base/data-ceph/`

## Current Policies

- `clusterpolicy-tiny-pod-requests.yaml`
  - enforces small default pod requests for workload manifests
- `clusterpolicy-prefer-non-ceph-nodes.yaml`
  - adds soft node affinity toward nodes without `storage=ceph`

The Ceph-aware placement policy excludes system and platform namespaces and can be bypassed per workload with:

```text
placement.gitops-showcase.io/allow-ceph-nodes: "true"
```

## Node Labeling

Ceph workers are labeled with:

```bash
kubectl label node worker-1 storage=ceph
kubectl label node worker-2 storage=ceph
kubectl label node worker-3 storage=ceph
```

## Validation

Use a workstation with cluster access.

```bash
kubectl -n flux-system get kustomizations,helmreleases
kubectl get clusterpolicy
kubectl get clusterpolicy prefer-non-ceph-nodes -o yaml
kubectl -n kyverno get pods
```

Then spot-check a workload and confirm the injected node affinity is present.

## Troubleshooting

- If policies do not appear, confirm the `kustomization-app-policies` Flux stage is Ready.
- If mutation is not visible, check namespace exclusions and the opt-out label.
- If Kyverno pods are unhealthy, inspect the `kyverno` namespace and the HelmRelease status.
