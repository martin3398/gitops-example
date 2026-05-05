# GitHub Actions Runbook

This repository uses GitHub Actions for CI/CD workflows.

## Workflows

- `.github/workflows/opentofu-check-plan.yml`
  - Runs `tofu fmt`, `tofu validate`, and `tofu plan`
  - Triggers on PR and push to `main` for `infra/` changes

- `.github/workflows/opentofu-apply-destroy.yml`
  - Manual workflow (`workflow_dispatch`)
  - `action=apply` runs plan+apply and exports outputs artifact
  - `action=destroy` requires `destroy_confirm=yes`

- `.github/workflows/ansible-check.yml`
  - Syntax checks all Ansible playbooks
  - Triggers on PR and push to `main` for `ansible/` changes

- `.github/workflows/ansible-run.yml`
  - Manual workflow to run inventory/smoke/base/runtime/bootstrap/flux chain
  - Reads OpenTofu outputs directly from remote state

- `.github/workflows/apps-build-publish.yml`
  - Tests services on PR/push
  - On `main` push, builds and publishes images to GHCR
  - Tag format: `YYYYMMDDHHmmSS-<8-char-git-sha>` and `sha-<full-git-sha>`

## Required GitHub Secrets

Infrastructure/Ansible:

- `AWS_REGION`
- `TF_STATE_BUCKET`
- `TF_STATE_KEY`
- `TF_LOCK_TABLE`
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- optional `AWS_SESSION_TOKEN`
- `FLUX_GIT_SSH_PRIVATE_KEY_B64` (for `ansible-run` flux step)
- `FLUX_GIT_KNOWN_HOSTS_B64` (for `ansible-run` flux step)

Container publishing:

- optional `GHCR_OWNER` (defaults to repository owner)
- optional `GHCR_USERNAME` (defaults to GitHub actor)
- optional `GHCR_TOKEN` (defaults to GitHub token)

## Notes

- `ansible-run` and `opentofu-apply-destroy` are manual by design to preserve infrastructure safety gates.
- Workflows are intentionally split into check and manual execution paths for safer infrastructure operations.
