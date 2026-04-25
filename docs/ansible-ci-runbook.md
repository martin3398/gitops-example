# Ansible CI Runbook

This runbook describes how Ansible automation stages execute in GitLab CI after infrastructure apply.

## Pipeline Ordering

The Ansible jobs are split into ordered stages:

1. `ansible:inventory`
2. `ansible:smoke`
3. `ansible:base`
4. `ansible:runtime`
5. `ansible:bootstrap`
6. `ansible:flux_bootstrap`

`ansible:inventory` depends on `tofu:apply` and consumes its `infra/outputs.json` artifact.

## Trigger Flow

On `main` branch:

1. Run `tofu:plan`
2. Manually trigger `tofu:apply` (provision gate)
3. Ansible jobs run automatically in order

All Ansible jobs use `resource_group: infra` to prevent concurrent infrastructure/cluster mutation.

## Pipeline Gates

- Gate 1 (provision): `tofu:apply` is manual on `main` and triggers infra + ordered Ansible automation.
- Gate 2 (destroy): `tofu:destroy` is manual on `main` and requires `DESTROY_CONFIRM=yes`.

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

- inventory graph with 3 control planes + 2 workers
- smoke connectivity passing all hosts
- base and runtime playbooks converged
- cluster bootstrap completed with all nodes Ready
- Flux bootstrap completed with `dev-repo`, `platform`, and `apps` ready in `flux-system`

Additional checks after bootstrap:

- `kubectl -n flux-system get imagerepositories,imagepolicies,imageupdateautomations`
- `podinfo` image automation objects should appear from `kubernetes/apps/dev/podinfo/` after `apps` becomes Ready

## Flux Bootstrap Variables

Required for `ansible:flux_bootstrap`:

- `FLUX_GIT_SSH_PRIVATE_KEY_B64`: base64-encoded private key matching the GitLab deploy key used by Flux (must allow pushes for image automation updates)
- `FLUX_GIT_KNOWN_HOSTS_B64`: base64-encoded GitLab SSH known_hosts line (for example `gitlab.com ssh-ed25519 ...`)

Encoding example:

```bash
base64 -w0 flux-gitlab
printf 'gitlab.com ssh-ed25519 <host-key>' | base64 -w0
```
