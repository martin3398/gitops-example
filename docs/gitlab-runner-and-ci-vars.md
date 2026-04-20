# GitLab Runner and CI Variables

This document describes the local Docker Compose runner setup and required CI variables for the OpenTofu and Ansible pipelines.

## Runner Setup (Local)

Runner compose file:
- `docker-compose.runner.yml`

Start runner:

```bash
docker compose -f docker-compose.runner.yml up -d
```

Register runner:

```bash
docker compose -f docker-compose.runner.yml exec gitlab-runner gitlab-runner register
```

Recommended registration values:
- Executor: `docker`
- Default image: `alpine:3.20` (fallback only)
- Allow untagged jobs: enabled (or add tags in pipeline jobs)

## Pipeline Files

- `.gitlab-ci.yml`
- `.gitlab/ci/opentofu.yml`
- `.gitlab/ci/ansible.yml`

## Required GitLab CI/CD Variables

Define in project settings: `Settings -> CI/CD -> Variables`

Required:
- `AWS_REGION` = `eu-central-1`
- `TF_STATE_BUCKET` = `gitops-showcase-tofu-state-<account-id>-eu-central-1`
- `TF_STATE_KEY` = `gitops-showcase/lab/infra.tfstate`
- `TF_LOCK_TABLE` = `gitops-showcase-tofu-locks`

AWS auth (choose one model):

1. Static keys (initial lab simplicity):
- `AWS_ACCESS_KEY_ID` (masked, protected)
- `AWS_SECRET_ACCESS_KEY` (masked, protected)

2. OIDC role assumption (preferred long-term):
- Configure trust and role in AWS, then use GitLab OIDC token flow

Optional (temporary credentials):
- `AWS_SESSION_TOKEN`

The Ansible pipeline reuses:
- `AWS_REGION` for SSM region and inventory generation
- `TF_STATE_BUCKET` as the SSM transfer bucket (passed to inventory generator `--ssm-bucket`)

## Job Behavior

Pipeline stage order:
- `tofu_check`
- `ansible_check`
- `tofu_plan`
- `tofu_apply`
- `ansible_inventory`
- `ansible_smoke`
- `ansible_base`
- `ansible_runtime`
- `ansible_bootstrap`
- `tofu_destroy`

- `tofu:fmt_validate`: MR + main
- `ansible:lint_inventory`: MR + main (playbook syntax checks)
- `tofu:plan`: MR + main
- `tofu:apply`: manual on main
- `tofu:destroy`: manual on main and only if `DESTROY_CONFIRM=yes`
- `ansible:inventory`: manual on main (requires `tofu:apply` artifacts)
- `ansible:smoke`: manual on main
- `ansible:base`: manual on main
- `ansible:runtime`: manual on main
- `ansible:bootstrap`: manual on main

Ansible jobs are intentionally manual-gated and serially ordered.

## Destroy Safety Gate

To run destroy job, set pipeline variable:

- `DESTROY_CONFIRM=yes`

Without this variable, destroy job remains unavailable.

## Troubleshooting

- If jobs run on shared runners instead of local runner, check runner assignment/tags.
- If OpenTofu image fails with shell errors, ensure CI image entrypoint is overridden (already done in `.gitlab/ci/opentofu.yml`).
- If backend init fails, verify bucket/table names and AWS credentials in CI variables.
- If Ansible jobs fail with `TargetNotConnected`, instances may not be SSM-online yet after apply; wait and retry.
- If Ansible jobs fail during module transfer, confirm CI identity has S3 access to `TF_STATE_BUCKET`.
- If inventory generation fails, ensure `tofu:apply` completed and produced `infra/outputs.json` artifact.
