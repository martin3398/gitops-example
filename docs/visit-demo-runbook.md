# Visit Demo Runbook

This runbook covers the current delivery and operations model for the visit demo app stack.

## Scope

- `visit-ui` (React Router SSR + hydration)
- `visit-gateway` (HTTP API + Kafka publish + count query)
- `visit-processor` (Kafka consume + Postgres write)

## Runtime Architecture

- Browser requests `/` from `visit-ui`.
- `visit-ui` server-side route loader fetches initial count from gateway.
- Browser hydration reuses loader data and revalidates every 5 seconds.
- Queueing actions use direct browser API calls to `POST /api/v1/visit-events?count=N`.
- Count reads use `GET /api/v1/visits/count` through loader revalidation.

## Ingress Routing Contract

- `/api` -> `visit-web-visit-gateway`
- `/` -> `visit-web-visit-web`

This keeps app and API on the same origin and avoids CORS complexity.

## API Contract (Current)

- `POST /api/v1/visit-events?count=N`
  - `N` must be an integer `1..100`
  - Returns `202` with `{ data: { status, queued } }`
- `GET /api/v1/visits/count`
  - Returns `200` with `{ data: { count } }`

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

## Troubleshooting

- `react/jsx-runtime` browser error:
  - stale or wrong client artifact is being served
  - ensure app serves `client.bundle.js`, then hard refresh browser
- Queue request fails:
  - verify ingress `/api` route and gateway pod health
- Count stuck at old value:
  - check route revalidation and gateway count endpoint
- Layout jump during refresh:
  - spinner should use fixed-size icon slot (no conditional text row)

## Next Scope (Roadmap Sync)

- Vault lab remains pending and tracked as next major platform lab.
- Add load generator scenarios to validate throughput and queue behavior under controlled traffic.
- Implement and tune HPA for `visit-processor` based on measured load behavior.
- Apply cluster-level authentication hardening (authn/authz and RBAC), excluding app-level auth features.
- Add Renovate to automate dependency and workflow update cadence.
