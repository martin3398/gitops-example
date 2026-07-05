# Visit Demo

Event-driven visit counter demo.

## Layout

- `visit-ui/`: frontend with SSR + hydration
- `visit-gateway/`: HTTP API service
- `visit-processor/`: Kafka consumer / Postgres writer
- `visit-loadgen/`: load generator

## Deployment

- service charts: `charts/visit-ui`, `charts/visit-gateway`, `charts/visit-processor`, `charts/visit-loadgen`
- Flux `HelmRelease` resources: `kubernetes/apps/base/visit-web/`, `kubernetes/apps/base/visit-processing/`, and `kubernetes/apps/base/visit-loadgen/`
- Flux app stages: `kubernetes/clusters/dev/apps/`
- GitHub Actions build pipeline: `.github/workflows/apps-build-publish.yml`

## Image Flow

- GitHub Actions builds and pushes immutable GHCR tags.
- Flux `ImageRepository`, `ImagePolicy`, and `ImageUpdateAutomation` keep deployed tags current.

## Local checks

```bash
cd apps/visit-demo
go test ./visit-gateway/...
go test ./visit-processor/...
go test ./visit-loadgen/...
cd visit-ui && npm ci && npm run check
```

## Container images

Kubernetes manifests reference GHCR repositories in this checkout:

- `ghcr.io/martin3398/visit-ui`
- `ghcr.io/martin3398/visit-gateway`
- `ghcr.io/martin3398/visit-processor`

Deployment tags are managed automatically by Flux image automation using timestamp tags (`YYYYMMDDHHmmSS-<8-char-git-sha>`).

See `docs/visit-demo-runbook.md` for runtime, API, and load-generator behavior.
