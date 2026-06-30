# Visit Demo Runbook

This runbook covers the current delivery and operations model for the visit demo app stack.

## Scope

- `visit-ui` (React Router SSR + hydration)
- `visit-gateway` (HTTP API + Kafka publish + count query)
- `visit-processor` (Kafka consume + Postgres write)
- `visit-loadgen` (continuous configurable load generator, paused by default)

## Runtime Architecture

- Browser requests `/` from `visit-ui`.
- `visit-ui` server-side route loader fetches initial count from gateway.
- Browser hydration reuses loader data and revalidates every 5 seconds.
- Queueing actions use direct browser API calls to `POST /api/v1/visit-events?count=N`.
- Count reads use `GET /api/v1/visits/count` through loader revalidation.
- Count reads also expose Kafka consumer lag for the `visit-processor-v1` consumer group.

## Ingress Routing Contract

- `/api` -> `visit-web-visit-gateway`
- `/` -> `visit-web-visit-web`

This keeps app and API on the same origin and avoids CORS complexity.

The intended local browser URL is `http://gitops.local` as part of the repo-wide local domain convention documented in the README. The app also keeps a hostless ingress fallback so the AWS ingress NLB DNS name can be used directly for the visit app.

## API Contract (Current)

- `POST /api/v1/visit-events?count=N`
  - `N` must be an integer `1..100`
  - Returns `202` with `{ data: { status, queued } }`
- `GET /api/v1/visits/count`
  - Returns `200` with `{ data: { count, queued, queue } }`
  - `count` is the processed Postgres visit count
  - `queued` is Kafka lag for topic `visits.requested` and consumer group `visit-processor-v1`
  - if Kafka lag lookup fails, `queued` is `null` and `queue.status` is `unavailable`

## Build and Runtime (visit-ui)

- Source: TypeScript only (`src/*.ts`, `src/*.tsx`)
- Server entry: `src/server.tsx`
- Browser entry: `src/client.tsx`
- Route modules: `src/routes/`
- Build output:
  - `dist/server.js` (Node SSR runtime)
  - `dist/client.bundle.js` (browser bundle)
  - `dist/template.html` and `dist/app.css` (static assets)

## End-to-End Verification

From a browser:

1. Open app root URL.
2. Confirm initial count renders.
3. Click `Register Visit` and confirm status message updates.
4. Click `Register Visit x10` and confirm status message updates.
5. Confirm count changes after periodic 5-second revalidation.
6. Click refresh icon and confirm spinner appears without layout shift.

From cluster checks:

```bash
kubectl -n visit-web get pods
kubectl -n visit-processing get pods
kubectl -n flux-system get kustomizations,helmreleases
```

Automated post-deploy verification is included in:

```bash
task pipeline:verify
```

The verification task checks visit app deployment readiness and exercises the gateway queue/count API path.

## Load Generator

The load generator runs as `visit-loadgen` in the `visit-loadgen` namespace. It is deployed continuously but defaults to `paused`, so it does not send traffic until enabled.

Modes:

- `paused` - sends no traffic; keeps the pod healthy and observable
- `random` - picks a random configured band and holds it for a random phase duration
- `fixed` - stays on the configured `fixedBand`

Default bands:

```text
idle=0:0.2:2
below=0.3:0.8:4
capacity=0.9:1.2:3
overload=2:3:2
burst=5:10:1
```

One `visit-processor` pod is intentionally rate-limited to `1 msg/s`, so `overload` and `burst` phases should grow Kafka lag while `idle` and `below` phases should let lag drain.

Enable stochastic load by changing the loadgen HelmRelease values:

```yaml
config:
  mode: random
```

Use a controlled band for focused testing:

```yaml
config:
  mode: fixed
  fixedBand: burst
```

Disable traffic without removing the deployment:

```yaml
config:
  mode: paused
```

Hard-disable the pod if needed:

```yaml
replicaCount: 0
```

Observe progress with:

```bash
kubectl -n visit-loadgen logs deploy/visit-loadgen -f
kubectl -n visit-processing logs deploy/visit-processing-visit-processor -f
curl http://gitops.local/api/v1/visits/count
```

## Troubleshooting

- `react/jsx-runtime` browser error:
  - stale or wrong client artifact is being served
  - ensure app serves `client.bundle.js`, then hard refresh browser
- Queue request fails:
  - verify ingress `/api` route and gateway pod health
- Count stuck at old value:
  - check route revalidation and gateway count endpoint
- Queued count unavailable:
  - verify Kafka is ready and the `visit-processor-v1` consumer group has committed offsets
- Load generator sends no traffic:
  - check `config.mode`; the default is `paused`
- Layout jump during refresh:
  - spinner should use fixed-size icon slot (no conditional text row)

## Next Scope (Roadmap Sync)

- OpenBao integration is implemented through External Secrets for app database credentials.
- Implement and tune HPA for `visit-processor` based on measured load behavior.
- Apply cluster-level authentication hardening (authn/authz and RBAC), excluding app-level auth features.
- Add Renovate to automate dependency and workflow update cadence.
