# Ansible Iterations 1-5 (SSM-first)

This directory contains the first Ansible iteration for the environment:
- inventory handoff from OpenTofu outputs
- SSM-based connectivity model
- smoke validation playbook

Iteration 2 adds base node preparation:
- package baseline for Kubernetes host prerequisites
- kernel module and sysctl preparation
- swap disablement and validation checks

Iteration 3 adds runtime and Kubernetes node binaries:
- containerd installation and `SystemdCgroup = true` configuration
- Kubernetes apt repository setup and package installation (`kubelet`, `kubeadm`, `kubectl`)
- package hold and service validation for `containerd` and `kubelet`

Note: During iteration 3 (pre-bootstrap), `kubelet` is expected to be enabled but may not be `running` until kubeadm init/join is completed.

Iteration 4 adds cluster bootstrap automation (4A-4D):
- 4A: first control plane `kubeadm init`
- 4B: Cilium CNI installation via Cilium CLI
- 4C: join remaining control planes and workers
- 4D: cluster-level validation from first control plane

Kubernetes bootstrap (kubeadm init/join) is included in `playbooks/cluster-bootstrap.yml`.

Iteration 5 adds Flux bootstrap handoff automation:
- install Flux controllers on the first control plane
- apply Git source and root Kustomizations from `kubernetes/flux/clusters/dev`
- validate Flux `GitRepository` and `Kustomization` readiness

Active environment naming:
- inventory path: `ansible/inventories/dev/hosts.yml`
- Flux Git source object: `GitRepository/dev-repo`

Kubernetes API endpoint behavior:
- inventory generator writes `kube_api_endpoint` into `all.vars`
- `kubeadm init` uses `kube_api_endpoint` when present
- fallback is first control-plane private IP (`<cp-1-private-ip>:6443`) when no endpoint is provided

Flux image automation is configured under `kubernetes/apps/dev/podinfo/`:
- `ImageRepository` tracks available tags from `ghcr.io/stefanprodan/podinfo`
- `ImagePolicy` selects stable semver tags in range `>=6.0.0 <7.0.0`
- `ImageUpdateAutomation` writes selected image updates back to Git (`main`) using setters

## Prerequisites

- Ansible installed locally
- AWS credentials available in the shell environment
- Session Manager Plugin installed locally
- `amazon.aws` collection installed
- Python packages for AWS-backed Ansible connection plugins (`boto3`, `botocore`)

Install collection:

```bash
ansible-galaxy collection install -r ansible/requirements.yml
```

Install Python dependencies (user local):

```bash
python3 -m pip install --user -r ansible/python-requirements.txt
```

If your system Python is externally managed, create and use a virtual environment:

```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -r ansible/python-requirements.txt
```

## Generate Inventory from OpenTofu

1. Export OpenTofu inventory output:

```bash
cd infra
tofu output -json ansible_inventory > ../ansible/inventories/dev/ansible_inventory.json
```

2. Generate `hosts.yml`:

```bash
cd ..
python3 ansible/scripts/generate_inventory.py \
  --input ansible/inventories/dev/ansible_inventory.json \
  --output ansible/inventories/dev/hosts.yml \
  --region eu-central-1 \
  --ssm-bucket gitops-showcase-tofu-state-<account-id>-eu-central-1
```

Alternative via environment variable:

```bash
export ANSIBLE_SSM_BUCKET=gitops-showcase-tofu-state-<account-id>-eu-central-1
python3 ansible/scripts/generate_inventory.py \
  --input ansible/inventories/dev/ansible_inventory.json \
  --output ansible/inventories/dev/hosts.yml \
  --region eu-central-1
```

## Verify Inventory and Connectivity

Inspect inventory graph:

```bash
ansible-inventory -i ansible/inventories/dev/hosts.yml --graph
```

Run smoke checks:

```bash
ANSIBLE_CONFIG=ansible/ansible.cfg \
ansible-playbook -i ansible/inventories/dev/hosts.yml ansible/playbooks/smoke.yml
```

Run base preparation (iteration 2):

```bash
cd ansible
ansible-playbook -i inventories/dev/hosts.yml playbooks/base.yml --tags base
```

Idempotency check:

```bash
cd ansible
ansible-playbook -i inventories/dev/hosts.yml playbooks/base.yml --tags base
```

Second run should show minimal changes.

Run runtime preparation (iteration 3):

```bash
cd ansible
ansible-playbook -i inventories/dev/hosts.yml playbooks/runtime.yml --tags runtime
```

Runtime idempotency check:

```bash
cd ansible
ansible-playbook -i inventories/dev/hosts.yml playbooks/runtime.yml --tags runtime
```

Run full cluster bootstrap (iterations 4A-4D):

```bash
cd ansible
ansible-playbook -i inventories/dev/hosts.yml playbooks/cluster-bootstrap.yml --tags bootstrap
```

Run Flux bootstrap handoff (iteration 5):

```bash
cd ansible
export FLUX_GIT_SSH_PRIVATE_KEY_B64="$(base64 -w0 ../flux-gitlab)"
export FLUX_GIT_KNOWN_HOSTS_B64="$(printf 'gitlab.com ssh-ed25519 <host-key>' | base64 -w0)"
export FLUX_GIT_SSH_PRIVATE_KEY="$(printf '%s' "$FLUX_GIT_SSH_PRIVATE_KEY_B64" | base64 -d)"
export FLUX_GIT_KNOWN_HOSTS="$(printf '%s' "$FLUX_GIT_KNOWN_HOSTS_B64" | base64 -d)"
ansible-playbook -i inventories/dev/hosts.yml playbooks/flux-bootstrap.yml --tags flux
```

Fetch kubeconfig for local kubectl usage:

```bash
task ansible:get_kubeconfig
```

Use it locally:

```bash
KUBECONFIG=./kubeconfig.dev kubectl get nodes
```

Notes for 4B Cilium install:
- Cilium version is pinned to `1.19.3` in `roles/cni/defaults/main.yml`.
- Cilium CLI version is pinned separately (`v0.19.2`) and installed on the first control plane if missing or version-mismatched.

## Groups and Variables

- `control_plane` and `workers` are generated from OpenTofu output
- `k8s_cluster` includes both groups
- SSM connection variables are written directly into generated inventory (`all.vars`) so runs do not depend on execution path
