---
name: explore-tasks
description: Scan repository documentation, inspect completed vs open tasks from docs/roadmap.md and runbooks, and present a dynamic, curated selection of open tasks for the user to select.
---

# Explore Tasks & Roadmap Backlog

Use this skill whenever the user triggers `/explore-tasks` or asks to inspect current progress, explore available roadmap tasks, or select the next task to work on.

## Instructions for the Agent

When this skill is activated, do **not** use static or hardcoded task lists. Instead, perform dynamic inspection of the repository documentation on the fly:

### Step 1: Dynamically Read the Documentation
1. Read the canonical backlog in [docs/roadmap.md](file:///home/martin/Dokumente/git/gitops-example/docs/roadmap.md).
2. Check [Agents.md](file:///home/martin/Dokumente/git/gitops-example/Agents.md) or relevant runbooks under `docs/` (such as `docs/postgres-runbook.md`, `docs/openbao-runbook.md`, `docs/kafka-runbook.md`, `docs/ceph-runbook.md`, `docs/gateway-runbook.md`, `docs/kyverno-runbook.md`) if extra context on recent status changes is needed.

### Step 2: Extract Current State & Open Tasks
Dynamically parse what you find in `docs/roadmap.md`:
- **Current Phase Status**: Extract the status overview across all phases (e.g. Phase 1 through Phase 6+).
- **Implemented Work (Done)**: Identify what baselines and tasks have already been marked complete.
- **Open / Remaining Tasks**: Extract all remaining task blocks (identified by `TASK-...` IDs), including:
  - Task ID (e.g., `TASK-P3-02`, `TASK-P4-02`)
  - Title and Phase
  - Priority (High / Medium / Low)
  - Objective
  - Affected Files & Components
  - Acceptance Criteria

### Step 3: Present Dynamic Task Selection Menu
Present the findings clearly to the user:
1. **Status Overview**: A brief summary of completed phases vs active phases.
2. **Open Tasks Table**: A structured markdown table containing all active open tasks extracted from the docs:
   - Task ID
   - Phase & Priority
   - Title & Core Objective
   - Key Target Files
3. **Interactive Selection**:
   - Invite the user to pick any Task ID (or multiple IDs) to immediately proceed with creating an implementation plan or executing it.
   - Offer the option to request a deep dive / architectural explanation on any specific task before deciding.
