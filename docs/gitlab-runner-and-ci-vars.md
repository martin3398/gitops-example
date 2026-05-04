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
- `TF_STATE_KEY` = `gitops-showcase/dev/infra.tfstate`
- `TF_LOCK_TABLE` = `gitops-showcase-tofu-locks`
- `GHCR_USERNAME` = `<github-username>`
- `GHCR_TOKEN` = `<github-token-with-package-write>`

Notes:
- `GHCR_OWNER` is optional; if not set, CI defaults it to `GHCR_USERNAME`.
- Set `GHCR_OWNER` only when publishing under a different org/user namespace.

For private GHCR images, also create Kubernetes secrets:
- workload pull secret `ghcr-pull` in `visit-edge` and `visit-processing`
- Flux image-reflector secret `ghcr-registry` in `flux-system`

See `docs/ghcr-setup.md` for exact commands.

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

Flux bootstrap in Ansible uses:
- `FLUX_GIT_SSH_PRIVATE_KEY_B64` (base64-encoded deploy key private key used by Flux `GitRepository/dev-repo` and image automation Git commits; requires write access to `main`)
- `FLUX_GIT_KNOWN_HOSTS_B64` (base64-encoded GitLab SSH known_hosts line)

Encoding examples:

```bash
base64 -w0 flux-gitlab
printf 'gitlab.com ssh-ed25519 <host-key>' | base64 -w0
```

Flux image automation for app updates uses the same Flux Git credential (`flux-system` secret via `GitRepository/dev-repo`).
No additional CI variables are required for app image-tag Git commits.

Environment naming:
- current environment is `dev`
- state key default is `gitops-showcase/dev/infra.tfstate`
- Flux entrypoint path is `kubernetes/flux/clusters/dev/`

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
- `ansible_flux`
- `tofu_destroy`

- `tofu:fmt_validate`: MR + main
- `ansible:lint_inventory`: MR + main (playbook syntax checks)
- `tofu:plan`: MR + main
- `tofu:apply`: manual on main (single provision gate)
- `tofu:destroy`: manual on main and only if `DESTROY_CONFIRM=yes`
- `ansible:inventory`: automatic on main after `tofu:apply` (requires `tofu:apply` artifacts)
- `ansible:smoke`: automatic on main after `ansible:inventory`
- `ansible:base`: automatic on main after `ansible:smoke`
- `ansible:runtime`: automatic on main after `ansible:base`
- `ansible:bootstrap`: automatic on main after `ansible:runtime`
- `ansible:flux_bootstrap`: automatic on main after `ansible:bootstrap`
- `apps:visit-ui:test`, `apps:visit-gateway:test`, `apps:visit-processor:test`: MR + main when visit-demo or chart files change
- `apps:visit-ui:build`, `apps:visit-gateway:build`, `apps:visit-processor:build`: automatic on main when visit-demo or chart files change

Ansible jobs are intentionally serially ordered and run automatically after the provision gate.

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
- If Flux bootstrap fails with source auth errors, verify `FLUX_GIT_SSH_PRIVATE_KEY_B64` and `FLUX_GIT_KNOWN_HOSTS_B64` decode to valid values.
- If Flux image automation fails to push updates, verify the Git credential in `flux-system` has write access to `main`.
