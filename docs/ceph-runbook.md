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

## Device Selection Model

Ceph is configured with an explicit device-path filter:

- `devicePathFilter: ^/dev/nvme1n1$`

This targets the first non-root NVMe disk on each worker that OpenTofu attaches for Ceph OSD use.

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
