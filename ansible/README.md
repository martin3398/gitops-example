# Ansible Iteration 1 (SSM-first)

This directory contains the first Ansible iteration for the lab:
- inventory handoff from OpenTofu outputs
- SSM-based connectivity model
- smoke validation playbook

No host configuration or Kubernetes bootstrap is included yet.

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
tofu output -json ansible_inventory > ../ansible/inventories/lab/ansible_inventory.json
```

2. Generate `hosts.yml`:

```bash
cd ..
python3 ansible/scripts/generate_inventory.py \
  --input ansible/inventories/lab/ansible_inventory.json \
  --output ansible/inventories/lab/hosts.yml \
  --region eu-central-1 \
  --ssm-bucket gitops-showcase-tofu-state-<account-id>-eu-central-1
```

Alternative via environment variable:

```bash
export ANSIBLE_SSM_BUCKET=gitops-showcase-tofu-state-<account-id>-eu-central-1
python3 ansible/scripts/generate_inventory.py \
  --input ansible/inventories/lab/ansible_inventory.json \
  --output ansible/inventories/lab/hosts.yml \
  --region eu-central-1
```

## Verify Inventory and Connectivity

Inspect inventory graph:

```bash
ansible-inventory -i ansible/inventories/lab/hosts.yml --graph
```

Run smoke checks:

```bash
ANSIBLE_CONFIG=ansible/ansible.cfg \
ansible-playbook -i ansible/inventories/lab/hosts.yml ansible/playbooks/smoke.yml
```

## Groups and Variables

- `control_plane` and `workers` are generated from OpenTofu output
- `k8s_cluster` includes both groups
- SSM connection variables are written directly into generated inventory (`all.vars`) so runs do not depend on execution path
