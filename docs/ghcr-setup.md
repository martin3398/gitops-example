# GHCR Setup (GitLab CI + Flux)

This runbook sets up private image build/push and deployment pulls for:

- `visit-web`
- `visit-gateway`
- `visit-processor`

## 1) Create GitHub token for GHCR push

Create a GitHub Personal Access Token (classic) with:

- `write:packages`
- `read:packages`
- `repo` (if repository/package visibility requires it)

## 2) Configure GitLab CI variables

In GitLab `Settings -> CI/CD -> Variables`, set:

- `GHCR_USERNAME` (GitHub username)
- `GHCR_TOKEN` (token from step 1)

Optional:
- `GHCR_OWNER` (only if image namespace differs from `GHCR_USERNAME`)

If `GHCR_OWNER` is not set, CI defaults it to `GHCR_USERNAME`.

These are used by `.gitlab/ci/apps.yml` build jobs.

## 3) Update image repositories in GitOps files

Replace `ghcr.io/example/...` with your real owner in:

- `kubernetes/apps/dev/visit-web/helmrelease-visit-web.yaml`
- `kubernetes/apps/dev/visit-web/helmrelease-visit-gateway.yaml`
- `kubernetes/apps/dev/visit-processing/helmrelease-visit-processor.yaml`
- `kubernetes/apps/dev/visit-web/imagerepository-visit-web.yaml`
- `kubernetes/apps/dev/visit-web/imagerepository-visit-gateway.yaml`
- `kubernetes/apps/dev/visit-processing/imagerepository-visit-processor.yaml`

## 4) Create pull secrets for workloads

Create `ghcr-pull` secret in both workload namespaces:

```bash
kubectl -n visit-edge create secret docker-registry ghcr-pull \
  --docker-server=ghcr.io \
  --docker-username="$GHCR_USERNAME" \
  --docker-password="$GHCR_TOKEN"

kubectl -n visit-processing create secret docker-registry ghcr-pull \
  --docker-server=ghcr.io \
  --docker-username="$GHCR_USERNAME" \
  --docker-password="$GHCR_TOKEN"
```

HelmRelease values already reference `imagePullSecrets: [{name: ghcr-pull}]`.

## 5) Create registry credentials for Flux image scanning

Create Docker auth secret in `flux-system` named `ghcr-registry`:

```bash
kubectl -n flux-system create secret docker-registry ghcr-registry \
  --docker-server=ghcr.io \
  --docker-username="$GHCR_USERNAME" \
  --docker-password="$GHCR_TOKEN"
```

`ImageRepository` objects already reference `spec.secretRef.name: ghcr-registry`.

### Optional: apply these via Ansible flux bootstrap

If you export `GHCR_USERNAME` and `GHCR_TOKEN` before running:

```bash
ansible-playbook -i ansible/inventories/dev/hosts.yml ansible/playbooks/flux-bootstrap.yml --tags flux
```

the role will create/update:

- `ghcr-pull` in `visit-edge` and `visit-processing`
- `ghcr-registry` in `flux-system`

## 6) Run first pipeline and verify

Push changes to `main` and verify:

- GitLab jobs: `apps:visit-ui:build`, `apps:visit-gateway:build`, `apps:visit-processor:build`
- Flux image objects:

```bash
kubectl -n flux-system get imagerepositories,imagepolicies,imageupdateautomations
```

- Workload status:

```bash
kubectl -n visit-edge get pods
kubectl -n visit-processing get pods
```

If pods show `ImagePullBackOff`, verify `ghcr-pull` in both namespaces.
