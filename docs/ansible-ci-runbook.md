# Ansible CI Runbook

This runbook describes how to run the Ansible automation stages in GitLab CI after infrastructure apply.

## Pipeline Ordering

The Ansible jobs are split into manual, ordered stages:

1. `ansible:inventory`
2. `ansible:smoke`
3. `ansible:base`
4. `ansible:runtime`
5. `ansible:bootstrap`

`ansible:inventory` depends on `tofu:apply` and consumes its `infra/outputs.json` artifact.

## Trigger Flow

On `main` branch:

1. Run `tofu:plan`
2. Manually trigger `tofu:apply`
3. Manually trigger Ansible jobs in order

All Ansible jobs use `resource_group: infra` to prevent concurrent infrastructure/cluster mutation.

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
