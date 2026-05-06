# Ansible CI Runbook

This runbook describes how Ansible automation stages execute in GitHub Actions after infrastructure apply.

## Workflow Ordering

The Ansible workflow runs these ordered steps:

1. inventory generation
2. smoke
3. base
4. runtime
5. cluster bootstrap
6. flux bootstrap

Inventory is generated from `infra/outputs.json` exported from OpenTofu state.

## Trigger Flow

On `main` branch:

1. Run OpenTofu check/plan workflow
2. Manually trigger OpenTofu apply workflow (provision gate)
3. Manually trigger `ansible-run` workflow

The manual workflows use `concurrency: infra` to prevent concurrent infrastructure/cluster mutation.

## Pipeline Gates

- Gate 1 (provision): OpenTofu apply is manual.
- Gate 2 (destroy): OpenTofu destroy is manual and requires explicit `destroy_confirm=yes` input.

## Variables and Reuse

Required existing variables:

- `AWS_REGION`
- `TF_STATE_BUCKET`
- `TF_STATE_KEY`
- `TF_LOCK_TABLE`
- AWS credentials variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, optional `AWS_SESSION_TOKEN`) or OIDC equivalent

Ansible-specific behavior:

- Inventory generator uses `--region "$AWS_REGION"`.
- Inventory generator uses `--ssm-bucket "$TF_STATE_BUCKET"`.
- No additional SSM bucket variable is required.

## Expected Failure Modes

- `TargetNotConnected` in smoke/base/runtime/bootstrap:
  - Freshly created instances are not SSM-online yet.
  - Retry after a short delay.
- Inventory generation failure:
  - `tofu:apply` artifact missing or outdated.
  - Re-run `tofu:apply`.
- S3 transfer errors from Ansible SSM plugin:
  - CI identity lacks S3 permissions for `TF_STATE_BUCKET`.

## Verification

Successful sequence should result in:

- inventory graph with 3 control planes + 3 workers
- smoke connectivity passing all hosts
- base and runtime playbooks converged
- cluster bootstrap completed with all nodes Ready
- Flux bootstrap completed with `dev-repo`, `platform`, and `apps` ready in `flux-system`

Additional checks after bootstrap:

- `kubectl -n flux-system get imagerepositories,imagepolicies,imageupdateautomations`
- visit app image automation objects should appear from `kubernetes/apps/dev/visit-web/` and `kubernetes/apps/dev/visit-processing/` after `apps` becomes Ready

## Flux Bootstrap Variables

Required for flux bootstrap step:

- `FLUX_GIT_SSH_PRIVATE_KEY_B64`: base64-encoded private key matching the deploy key used by Flux (must allow pushes for image automation updates)
- optional `FLUX_GIT_KNOWN_HOSTS_B64`: base64-encoded Git host SSH known_hosts line (for example `github.com ssh-ed25519 ...`); if unset, workflow can generate it

Encoding example:

```bash
base64 -w0 flux-github
printf 'github.com ssh-ed25519 <host-key>' | base64 -w0
```
