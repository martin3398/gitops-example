# Ansible CI Runbook

This runbook describes how Ansible automation stages execute in GitHub Actions after infrastructure apply.

See `docs/deployment-pipeline-runbook.md` for the canonical staged ordering.

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

## Flux Bootstrap Notes

- Flux bootstrap still requires the SSH key and known_hosts secrets described in the GitHub Actions runbook.
