# Visit Demo

Event-driven visit counter demo:

- `visit-web` (code folder `visit-ui`): frontend with a button and live counter
- `visit-gateway`: HTTP API that publishes visit events to Kafka and serves visit count
- `visit-processor`: Kafka consumer that writes to Postgres with per-pod rate limit (`RATE_LIMIT_PER_SEC`)
- `visit-loadgen`: always-running Kubernetes load generator for controlled queue pressure

Deployment ownership model:

- service charts: `charts/visit-ui`, `charts/visit-gateway`, `charts/visit-processor`, `charts/visit-loadgen`
- Flux `HelmRelease` resources: `kubernetes/apps/base/visit-web/` and `kubernetes/apps/base/visit-processing/`
- Flux app stages: `kubernetes/clusters/dev/apps/`
- GitHub Actions build pipeline: `.github/workflows/apps-build-publish.yml`

## Project layout

- `visit-ui/`: TypeScript SSR frontend (Express + React server rendering)
- `visit-gateway/`: Go API producer service
- `visit-processor/`: Go consumer/writer service
- `visit-loadgen/`: Go load generator service

## Local checks

```bash
cd apps/visit-demo
go test ./visit-gateway/...
go test ./visit-processor/...
go test ./visit-loadgen/...
cd visit-ui && npm ci && npm run check
```

## Load generator

`visit-loadgen` is deployed as a Kubernetes `Deployment` and defaults to `paused` mode, which sends no traffic.

Modes:

- `paused` - keep the pod running but send no traffic
- `random` - randomly switch between configured load bands
- `fixed` - stay on one configured band, set by `LOADGEN_FIXED_BAND`

The default random bands are calibrated around the current processor capacity of one `visit-processor` pod handling about `1 msg/s`.

Enable random load by changing the Helm values under `kubernetes/apps/base/visit-loadgen/helmrelease-visit-loadgen.yaml`:

```yaml
config:
  mode: random
```

The generator logs the active band, target rate, sent/failed messages, processed count, and Kafka queued count.

## Container image expectations

Kubernetes manifests reference GHCR repositories:

- `ghcr.io/martin3398/visit-ui`
- `ghcr.io/martin3398/visit-gateway`
- `ghcr.io/martin3398/visit-processor`

Deployment tags are managed automatically by Flux image automation using timestamp tags (`YYYYMMDDHHmmSS-<8-char-git-sha>`).

Replace these with your real registry/image names before deployment.
