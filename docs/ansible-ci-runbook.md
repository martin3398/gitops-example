# Ansible CI Runbook

This runbook describes how Ansible automation stages execute in GitHub Actions after infrastructure apply.

## Workflow Ordering

The Ansible workflow runs these ordered steps:

1. inventory generation
2. smoke
3. base
4. runtime
5. cluster bootstrap
6. Flux/core bootstrap
7. OpenBao bootstrap
8. verification

Inventory is generated from `infra/outputs.json` that `ansible-run` creates by reading OpenTofu remote state directly.

This order mirrors local `task pipeline:main` after the OpenTofu apply step.

## Trigger Flow

On `main` branch:

1. Run OpenTofu check/plan workflow
2. Manually trigger OpenTofu apply workflow (provision gate)
3. Manually trigger `ansible-run` workflow

The manual workflows use `concurrency: infra` to prevent concurrent infrastructure/cluster mutation.

## Pipeline Gates

- Gate 1 (provision): OpenTofu apply is manual.
- Gate 2 (deploy): Ansible staged deployment is manual.
- Gate 3 (destroy): OpenTofu destroy is manual and requires explicit `destroy_confirm=yes` input.

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
  - OpenTofu backend init fails (state config/credentials issue), or `tofu output -json` cannot read state outputs.
  - Validate `AWS_REGION`, `TF_STATE_BUCKET`, `TF_STATE_KEY`, and `TF_LOCK_TABLE`, then re-run.
- S3 transfer errors from Ansible SSM plugin:
  - CI identity lacks S3 permissions for `TF_STATE_BUCKET`.
- `ServiceMonitor` CRD errors during ingress or logging chart install:
  - The deployment order is wrong or Flux/core bootstrap did not complete.
  - `kube-prometheus-stack` must install Prometheus Operator CRDs before ingress-nginx, Loki, or Promtail render `ServiceMonitor` resources.
  - Re-run Flux/core bootstrap and OpenBao bootstrap after confirming `infrastructure-data-ceph` is Ready.

## Verification

Successful sequence should result in:

- inventory graph with 3 control planes + 3 workers
- smoke connectivity passing all hosts
- base and runtime playbooks converged
- cluster bootstrap completed with all nodes Ready
- Flux/core bootstrap completed with `dev-repo`, `infrastructure-core`, and `infrastructure-data-ceph` ready in `flux-system`
- OpenBao initialized, unsealed, and ready for External Secrets
- post-deploy verification completed against Flux, OpenBao, Postgres, Kafka, observability, ingress, and the visit demo workloads

Additional checks after bootstrap:

- `kubectl -n flux-system get imagerepositories,imagepolicies,imageupdateautomations`
- visit app image automation objects should appear from `kubernetes/apps/base/visit-web/`, `kubernetes/apps/base/visit-processing/`, and `kubernetes/apps/base/visit-loadgen/` after Flux reconciliation

## Flux Bootstrap Variables

Required for flux bootstrap step:

- `FLUX_GIT_SSH_PRIVATE_KEY_B64`: base64-encoded private key matching the deploy key used by Flux (must allow pushes for image automation updates)
- `FLUX_GIT_KNOWN_HOSTS_B64`: base64-encoded Git host SSH known_hosts line (for example `github.com ssh-ed25519 ...`)

Encoding example:

```bash
base64 -w0 flux-github
printf 'github.com ssh-ed25519 <host-key>' | base64 -w0
```
