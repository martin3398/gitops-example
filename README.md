# GitOps Example

AWS-based Kubernetes GitOps lab with infra, bootstrap, workloads, and operational runbooks.

## Included

- `OpenTofu` infrastructure for a 6-node RKE2 cluster
- `Ansible` host prep, cluster bootstrap, and Flux handoff
- `Flux`-managed platform and app delivery
- `GitHub Actions` CI/CD and GHCR image publishing
- Platform pieces: Gateway API edge, observability, OpenBao, Postgres, Kafka, Ceph, visit demo

## Done

- Phase 1 infra and cluster bootstrap are implemented.
- Phase 2 GitOps platform and workloads are implemented.
- Phase 5 RKE2 migration is implemented.
- Phase 6 Gateway API edge migration is implemented.
- Visit demo delivery and image automation are implemented.
- OpenBao, Postgres, Kafka, Ceph, observability, load generation, and autoscaling are implemented baselines.
- Loki distributed mode on Ceph object storage is implemented.

## Open

- See `docs/roadmap.md` for the detailed remaining phase work.

## Start Here

- `kubernetes/README.md`
- `infra/README.md`
- `ansible/README.md`
- `apps/visit-demo/README.md`

## Key Docs

- `docs/deployment-pipeline-runbook.md`
- `docs/github-actions-runbook.md`
- `docs/phase1-infra-runbook.md`
- `docs/roadmap.md`
- `docs/visit-demo-runbook.md`
- `docs/openbao-runbook.md`
- `docs/ceph-runbook.md`
- `docs/kafka-runbook.md`
- `docs/postgres-runbook.md`
- `docs/observability-runbook.md`
- `docs/scheduling-runbook.md`

## Conventions

- Terraform/OpenTofu manages cloud resources only.
- Ansible manages host setup and cluster bootstrap only.
- Flux manages all long-lived Kubernetes resources.
- Persistent changes should be committed to Git.
