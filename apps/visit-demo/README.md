# Visit Demo

Event-driven visit counter demo:

- `visit-web` (code folder `visit-ui`): frontend with a button and live counter
- `visit-gateway`: HTTP API that publishes visit events to Kafka and serves visit count
- `visit-processor`: Kafka consumer that writes to Postgres with per-pod rate limit (`RATE_LIMIT_PER_SEC`)

Deployment ownership model:

- service charts: `charts/visit-ui`, `charts/visit-gateway`, `charts/visit-processor`
- Flux `HelmRelease` resources: `kubernetes/apps/dev/visit-web/` and `kubernetes/apps/dev/visit-processing/`
- GitHub Actions build pipeline: `.github/workflows/apps-build-publish.yml`

## Project layout

- `visit-ui/`: TypeScript SSR frontend (Express + React server rendering)
- `visit-gateway/`: Go API producer service
- `visit-processor/`: Go consumer/writer service

## Local checks

```bash
cd apps/visit-demo
go test ./visit-gateway/...
go test ./visit-processor/...
cd visit-ui && npm ci && npm run check
```

## Container image expectations

Kubernetes manifests reference GHCR repositories:

- `ghcr.io/martin3398/visit-ui`
- `ghcr.io/martin3398/visit-gateway`
- `ghcr.io/martin3398/visit-processor`

Deployment tags are managed automatically by Flux image automation using timestamp tags (`YYYYMMDDHHmmSS-<8-char-git-sha>`).

Replace these with your real registry/image names before deployment.
