# Kubernetes Layout

This directory holds the cluster desired state.

## Structure

- `clusters/`: Flux cluster entrypoints and staged Kustomizations
- `infrastructure/`: platform add-ons and shared infrastructure components
- `apps/`: app-specific policies and overlays

## Main Flow

- Cluster/bootstrap flow: `docs/deployment-pipeline-runbook.md`
- Platform details: `docs/phase1-infra-runbook.md`
- App/runtime details: `docs/visit-demo-runbook.md`

## Notes

- `clusters/dev/` is the active environment.
- Long-lived resources should be managed through Flux and committed to Git.
