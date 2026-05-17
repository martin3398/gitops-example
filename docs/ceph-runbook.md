# Ceph Runbook (Rook, Dev)

This runbook covers the Ceph baseline deployed by Flux with Rook.

## Scope

- Rook operator and Ceph cluster run in `rook-ceph` namespace.
- Worker data disks from OpenTofu are consumed as OSD devices.
- Storage class `ceph-block` is created for dynamic PVC provisioning.
- `local-path` remains available and default during this phase.

## Manifests

- `kubernetes/platform/dev/core-services/storage/rook-ceph-operator/helmrelease-rook-ceph.yaml`
- `kubernetes/platform/dev/data-platform/ceph/cephcluster.yaml`
- `kubernetes/platform/dev/data-platform/ceph/cephblockpool.yaml`
- `kubernetes/platform/dev/data-platform/ceph/storageclass-ceph-block.yaml`
- `kubernetes/platform/dev/data-platform/ceph/toolbox.yaml`

## Reconciliation Order

Ceph is split across Flux Kustomizations to avoid CRD dry-run races:

- `platform-core` installs operators, including `rook-ceph`.
- `platform-data-ceph` applies Ceph CRs (`CephCluster`, `CephBlockPool`, `StorageClass`).
- `platform-data-services` applies service workloads (Kafka, Postgres) after Ceph.

The order is enforced by Flux `dependsOn` in:

- `kubernetes/flux/clusters/dev/kustomization-platform-core.yaml`
- `kubernetes/flux/clusters/dev/kustomization-platform-data-ceph.yaml`
- `kubernetes/flux/clusters/dev/kustomization-platform-data-services.yaml`

## Device Selection Model

Ceph is configured with an explicit device-path filter:

- `devicePathFilter: ^/dev/nvme1n1$`

This targets the first non-root NVMe disk on each worker that OpenTofu attaches for Ceph OSD use.

Current worker model:

- 6 workers total for workload capacity.
- Ceph OSD data disks are attached only to `worker-1`, `worker-2`, and `worker-3`.
- Ceph therefore uses only nodes that expose `/dev/nvme1n1`.

## Placement Policy for Non-Ceph Workloads

Kyverno applies a soft scheduling preference for general workloads to avoid Ceph nodes:

- Policy: `kubernetes/platform/dev/apps/policies/clusterpolicy-prefer-non-ceph-nodes.yaml`
- Behavior: injects `preferredDuringSchedulingIgnoredDuringExecution` for `storage NotIn [ceph]`
- Scope: Deployments/StatefulSets/DaemonSets/Jobs/CronJobs in application namespaces
- Excludes: system and platform namespaces (`kube-system`, `flux-system`, `kyverno`, `rook-ceph`, `local-path-storage`, `monitoring`, `data-kafka`, `data-postgres`)

Opt out for a specific workload by setting label:

- `placement.gitops-showcase.io/allow-ceph-nodes: "true"`

Label storage workers:

```bash
kubectl label node worker-1 storage=ceph
kubectl label node worker-2 storage=ceph
kubectl label node worker-3 storage=ceph
```

## Validation

1. Check Flux reconciliation:

```bash
kubectl -n flux-system get kustomizations
kubectl -n flux-system get helmreleases
```

2. Check Ceph control-plane resources:

```bash
kubectl -n rook-ceph get pods
kubectl -n rook-ceph get cephcluster
kubectl -n rook-ceph get cephblockpool
kubectl get storageclass
```

3. Check Ceph health from toolbox:

```bash
kubectl -n rook-ceph exec deploy/rook-ceph-tools -- ceph -s
kubectl -n rook-ceph exec deploy/rook-ceph-tools -- ceph osd tree
```

4. Smoke-test dynamic provisioning on `ceph-block`:

```bash
kubectl apply -f - <<'EOF'
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: ceph-block-smoke
  namespace: default
spec:
  accessModes:
    - ReadWriteOnce
  storageClassName: ceph-block
  resources:
    requests:
      storage: 1Gi
---
apiVersion: v1
kind: Pod
metadata:
  name: ceph-block-smoke
  namespace: default
spec:
  restartPolicy: Never
  containers:
    - name: test
      image: busybox:1.36
      command:
        - sh
        - -c
        - "echo ceph-ok > /data/out.txt && cat /data/out.txt && sleep 5"
      volumeMounts:
        - name: data
          mountPath: /data
  volumes:
    - name: data
      persistentVolumeClaim:
        claimName: ceph-block-smoke
EOF

kubectl get pvc ceph-block-smoke
kubectl logs ceph-block-smoke
kubectl delete pod ceph-block-smoke
kubectl delete pvc ceph-block-smoke
```

## Troubleshooting

- If no OSDs appear, verify worker disk visibility (`lsblk`) and confirm `/dev/nvme1n1` exists on every worker.
- If CephCluster is not ready, inspect operator and Ceph pod logs in `rook-ceph`.
- If PVC stays pending, inspect CSI pods and StorageClass parameters.
