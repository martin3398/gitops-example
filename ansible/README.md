# Ansible Iterations 1-5 (SSM-first)

Use this directory for the local Ansible bootstrap flow.

## What’s Here

- inventory handoff from OpenTofu outputs
- SSM-based connectivity model
- smoke validation playbook
- base, runtime, cluster bootstrap, Flux bootstrap, and OpenBao playbooks

## Bootstrap Flow

- Iteration 2: base node preparation
- Iteration 3: runtime preparation for RKE2
- Iteration 4: RKE2 bootstrap + bundled Cilium
- Iteration 5: Flux bootstrap handoff

## Key Paths

- inventory path: `ansible/inventories/dev/hosts.yml`
- Flux Git source object: `GitRepository/dev-repo`
- Kubernetes bootstrap: `ansible/playbooks/cluster-bootstrap.yml`
- Flux bootstrap: `ansible/playbooks/flux-bootstrap.yml`

## Reference

- Full staged deployment flow: `docs/deployment-pipeline-runbook.md`
- CI-specific notes: `docs/ansible-ci-runbook.md`
- GHCR and image automation: `docs/github-actions-runbook.md`

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

## Common Playbooks

```bash
cd ansible
ansible-playbook -i inventories/dev/hosts.yml playbooks/base.yml --tags base
ansible-playbook -i inventories/dev/hosts.yml playbooks/runtime.yml --tags runtime
ansible-playbook -i inventories/dev/hosts.yml playbooks/cluster-bootstrap.yml --tags bootstrap
ansible-playbook -i inventories/dev/hosts.yml playbooks/flux-bootstrap.yml --tags flux
```

See `docs/github-actions-runbook.md` for GHCR/image automation variables and `docs/deployment-pipeline-runbook.md` for the ordered pipeline.

Fetch kubeconfig for local kubectl usage:

```bash
task ansible:get_kubeconfig_public
```

That task fetches the RKE2 kubeconfig from the first control plane and rewrites the `kubernetes` context to the current cluster endpoint.

Use it locally:

```bash
KUBECONFIG=./kubeconfig.dev kubectl get nodes
```

Notes for RKE2 bundled Cilium:
- Cilium is enabled through RKE2's bundled chart, with a `HelmChartConfig` manifest in `/var/lib/rancher/rke2/server/manifests/`.
- WireGuard encryption is enabled in the chart config.
- The bootstrap and verification roles wait on the `cilium` DaemonSet directly.

## Groups and Variables

- `control_plane` and `workers` are generated from OpenTofu output
- `k8s_cluster` includes both groups
- SSM connection variables are written directly into generated inventory (`all.vars`) so runs do not depend on execution path
