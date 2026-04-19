# GitLab Runner and CI Variables

This document describes the local Docker Compose runner setup and required CI variables for the OpenTofu pipeline.

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

## Job Behavior

- `tofu:fmt_validate`: MR + main
- `tofu:plan`: MR + main
- `tofu:apply`: manual on main
- `tofu:destroy`: manual on main and only if `DESTROY_CONFIRM=yes`

## Destroy Safety Gate

To run destroy job, set pipeline variable:

- `DESTROY_CONFIRM=yes`

Without this variable, destroy job remains unavailable.

## Troubleshooting

- If jobs run on shared runners instead of local runner, check runner assignment/tags.
- If OpenTofu image fails with shell errors, ensure CI image entrypoint is overridden (already done in `.gitlab/ci/opentofu.yml`).
- If backend init fails, verify bucket/table names and AWS credentials in CI variables.
