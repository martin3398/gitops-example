# GitHub Actions, GHCR, and Image Automation Runbook

This is the canonical CI/CD and registry automation doc for the repository.

## Workflows

- `.github/workflows/opentofu-check-plan.yml`
  - Runs `tofu fmt`, `tofu validate`, and `tofu plan`
  - Triggers on PR and push to `main` for `infra/` changes

- `.github/workflows/opentofu-apply-destroy.yml`
  - Manual workflow (`workflow_dispatch`)
  - `action=apply` runs plan + apply and exports outputs artifact
  - `action=destroy` requires `destroy_confirm=yes`

- `.github/workflows/ansible-check.yml`
  - Syntax checks all Ansible playbooks
  - Triggers on PR and push to `main` for `ansible/` changes

- `.github/workflows/ansible-run.yml`
  - Manual workflow for the full staged Ansible deployment chain
  - Reads OpenTofu outputs directly from remote state
  - Order: inventory, smoke, base, runtime, cluster bootstrap, Flux/core, OpenBao, verification

- `.github/workflows/apps-build-publish.yml`
  - Tests services on PR/push
  - On `main` push, builds and publishes images to GHCR
  - Tag format: `YYYYMMDDHHmmSS-<8-char-git-sha>` and `sha-<8-char-git-sha>`
  - Builds `visit-ui`, `visit-gateway`, `visit-processor`, and `visit-loadgen`

## Required Variables And Secrets

GitHub variables:

- `AWS_REGION`
- `TF_STATE_BUCKET`
- `TF_STATE_KEY`
- `TF_LOCK_TABLE`
- optional `GHCR_USERNAME`
- optional `GHCR_OWNER`

GitHub secrets:

- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- optional `AWS_SESSION_TOKEN`
- `FLUX_GIT_SSH_PRIVATE_KEY_B64` (for `ansible-run` Flux step)
- `FLUX_GIT_KNOWN_HOSTS_B64` (required by current `ansible-run` workflow)
- optional `OPENBAO_DEV_SEED` (overrides deterministic dev secret seeding for Postgres, visit demo, and Loki S3 credentials)
- optional `GHCR_TOKEN`

GHCR defaults in `apps-build-publish`:

- `GHCR_OWNER` defaults to `github.repository_owner` when not set.
- `GHCR_USERNAME` defaults to `github.actor` when not set.
- `GHCR_TOKEN` defaults to `${{ github.token }}` when not set.

## GHCR Setup

Create a GitHub Personal Access Token (classic) with:

- `write:packages`
- `read:packages`
- `repo` if package visibility requires it

Set repository variables/secrets in GitHub Actions:

- `GHCR_USERNAME`
- `GHCR_TOKEN`
- optional `GHCR_OWNER`

Current manifests point at `ghcr.io/martin3398/...`.
If you fork the repo or change ownership, update the image references in the visit demo HelmReleases and ImageRepository objects.

## Flux Image Automation

The registry and Flux image flow is:

1. GitHub Actions builds and pushes immutable tags on `main`.
2. Flux `ImageRepository` objects scan GHCR.
3. Flux `ImagePolicy` selects the newest timestamped tag.
4. Flux `ImageUpdateAutomation` commits the updated image reference back to Git.

Flux image objects live under:

- `kubernetes/apps/base/visit-web/`
- `kubernetes/apps/base/visit-processing/`
- `kubernetes/apps/base/visit-loadgen/`

## Registry Secrets

Preferred path: export `GHCR_USERNAME` and `GHCR_TOKEN` before running `task ansible:core`.
The Flux bootstrap role creates or updates:

- `ghcr-pull` in `visit-web`, `visit-processing`, and `visit-loadgen`
- `ghcr-registry` in `flux-system`

Manual fallback:

```bash
kubectl -n visit-web create secret docker-registry ghcr-pull \
  --docker-server=ghcr.io \
  --docker-username="$GHCR_USERNAME" \
  --docker-password="$GHCR_TOKEN"

kubectl -n visit-processing create secret docker-registry ghcr-pull \
  --docker-server=ghcr.io \
  --docker-username="$GHCR_USERNAME" \
  --docker-password="$GHCR_TOKEN"

kubectl -n visit-loadgen create secret docker-registry ghcr-pull \
  --docker-server=ghcr.io \
  --docker-username="$GHCR_USERNAME" \
  --docker-password="$GHCR_TOKEN"

kubectl -n flux-system create secret docker-registry ghcr-registry \
  --docker-server=ghcr.io \
  --docker-username="$GHCR_USERNAME" \
  --docker-password="$GHCR_TOKEN"
```

## Manual Runs

- `opentofu-apply-destroy`: run via Actions UI, choose `action=apply|destroy`
- `ansible-run`: run via Actions UI after successful OpenTofu apply

## Notes

- `ansible-run` and `opentofu-apply-destroy` are manual by design to preserve infrastructure safety gates.
- Workflows are intentionally split into check and manual execution paths for safer infrastructure operations.
