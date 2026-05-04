# Visit Demo

Event-driven visit counter demo:

- `visit-ui`: frontend with a button and live counter
- `visit-gateway`: HTTP API that publishes visit events to Kafka and serves visit count
- `visit-processor`: Kafka consumer that writes to Postgres with per-pod rate limit (`RATE_LIMIT_PER_SEC`)

Deployment ownership model:

- service charts: `charts/visit-ui`, `charts/visit-gateway`, `charts/visit-processor`
- Flux `HelmRelease` resources: `kubernetes/apps/dev/visit-demo/`
- GitLab build pipeline: `.gitlab/ci/apps.yml`

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

Kubernetes manifests currently reference:

- `ghcr.io/example/visit-ui:v0.1.0`
- `ghcr.io/example/visit-gateway:v0.1.0`
- `ghcr.io/example/visit-processor:v0.1.0`

Replace these with your real registry/image names before deployment.
