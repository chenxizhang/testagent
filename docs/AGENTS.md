# Multi-Agent Development Pipeline

## Overview

This project uses **three GitHub AI coding agents** working together in a continuous, automated loop to build the `mgc` CLI tool. No human intervention is required during development.

## Agents and Roles

| Agent | GitHub Login | Node ID | Primary Role |
|-------|-------------|---------|-------------|
| **GitHub Copilot** | `Copilot` | `BOT_kgDOC9w8XQ` | Developer — Feature implementation |
| **Claude** | `Claude` | `BOT_kgDODnPHJg` | Architect / Tester — Design, review, tests |
| **Codex** | `Codex` | `BOT_kgDODnSAjQ` | Developer — Parallel features, tooling |

## How the Pipeline Works

```
PM Orchestrator (hourly cron)
         │
         ▼
  Check project phase
  ┌──────────────────────────┐
  │  Phase N issues open?    │
  │  → Monitor / remind      │
  │                          │
  │  Phase N complete?       │
  │  → Create Phase N+1 tasks│
  │                          │
  │  No phases started?      │
  │  → Create Phase 1 tasks  │
  └──────────────────────────┘
         │
         ▼
  Create GitHub Issues → Assign to Agents
  ┌───────────┬──────────────┬──────────────┐
  │  Copilot  │    Claude    │    Codex     │
  │  picks up │  picks up    │  picks up    │
  │  issue    │  issue       │  issue       │
  │     ↓     │     ↓        │     ↓        │
  │ Creates   │  Creates     │  Creates     │
  │   branch  │   branch     │   branch     │
  │   + PR    │   + PR       │   + PR       │
  └───────────┴──────────────┴──────────────┘
         │
         ▼
  auto-merge-agent-prs.yml
  → Enable auto-merge on PR
  → Request Copilot code review
         │
         ▼
  architect-review.yml
  → Post review request to @claude
  → Claude reviews for architecture quality
         │
         ▼
  review-feedback-loop.yml
  → If review has suggestions → ping agent to fix
  → After 3 iterations OR no issues → proceed
         │
         ▼
  build-test.yml (CI)
  → go build ./...
  → go test ./... -race
  → Cross-compile check
         │
         ▼
  Auto-merge when CI passes
         │
         ▼
  auto-release.yml
  → Detect version change in main.go
  → Run tests
  → Build cross-platform binaries
  → Create GitHub Release with artifacts
```

## Project Phases

The project is divided into 6 phases, tracked via GitHub issue labels (`phase-1` through `phase-6`):

| Phase | Name | Primary Agent(s) |
|-------|------|-----------------|
| 1 | Architecture & Foundation | Claude (arch), Copilot (init), Codex (CI) |
| 2 | Authentication & Graph Client | Copilot (client), Codex (auth), Claude (tests) |
| 3 | Users & Groups Commands | Copilot (users), Codex (groups), Claude (tests) |
| 4 | Mail & Calendar + v0.1.0 | Copilot (mail), Codex (calendar), Claude (tests+release) |
| 5 | Advanced Features & Review | Copilot (JMESPath), Codex (files), Claude (arch review) |
| 6 | Polish, Docs & v1.0.0 | Copilot (completion), Codex (README), Claude (v1.0.0) |

## Workflow Files

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| `pm-orchestrator.yml` | Hourly + manual | Creates tasks, monitors progress, pings stalled agents |
| `auto-merge-agent-prs.yml` | PR opened | Enable auto-merge on agent PRs |
| `architect-review.yml` | PR opened/ready | Request Claude to review architecture quality |
| `review-feedback-loop.yml` | PR review submitted | Handle Copilot review feedback, ping agent to fix |
| `build-test.yml` | Push + PR | Build and test Go code, cross-compile |
| `auto-release.yml` | Push to main | Detect version bump, build binaries, create release |
| `task-scheduler.yml` | Daily + manual | Legacy daily task creator (news digest) |

## PM Orchestrator Logic

The PM orchestrator (`pm-orchestrator.yml`) is the brain of the system:

### Phase Detection Algorithm
```
for phase in 1..6:
  open_count  = issues with label "phase-N" AND state=open
  closed_count = issues with label "phase-N" AND state=closed

  if open_count > 0:
    → phase is ACTIVE → monitor it
    → check for stalled issues (no update > 24h)
    → break

  elif closed_count > 0:
    → phase is COMPLETE

if no active phase found:
  next_phase = last_completed_phase + 1
  → create tasks for next_phase
```

### Task Creation
Tasks are defined in `.github/project/mgc-project.json`. Each task includes:
- Title (used for deduplication)
- Detailed body with requirements and acceptance criteria
- Assigned agent (Copilot / Claude / Codex)
- Labels

### Stalled Issue Detection
If an issue has no updates for > 24 hours, the PM posts a reminder comment to ping the assigned agent.

## Review Feedback Loop

The `review-feedback-loop.yml` handles the code review cycle:

1. Agent creates PR → Copilot auto-reviews
2. If review has inline suggestions → comment `@copilot/claude/codex please fix...`
3. Agent pushes fix → review runs again
4. After 3 iterations OR no suggestions → auto-merge proceeds

Maximum iterations: 3 (to prevent infinite loops)

## Release Process

Releases are fully automated:

1. An agent updates `Version = "X.Y.Z"` in `mgc/main.go`
2. The change merges to `main`
3. `auto-release.yml` detects the new version (tag doesn't exist yet)
4. Runs all tests → if passing, builds cross-platform binaries
5. Creates git tag `vX.Y.Z`
6. Creates GitHub Release with binaries and changelog

Planned releases: v0.1.0 (Phase 4), v1.0.0 (Phase 6)

## Manual Override

You can manually trigger the PM orchestrator:

```bash
# Start a specific phase
gh workflow run pm-orchestrator.yml \
  -f force_phase=2

# Dry run to see what would be created
gh workflow run pm-orchestrator.yml \
  -f dry_run=true
```

## Agent Assignment API

Agents are assigned to issues via GitHub's GraphQL API (REST API silently ignores bot users):

```bash
gh api graphql -f query="
  mutation {
    addAssigneesToAssignable(input: {
      assignableId: \"<issue-node-id>\"
      assigneeIds: [\"BOT_kgDOC9w8XQ\"]  # Copilot
    }) {
      assignable {
        ... on Issue {
          assignees(first: 5) { nodes { login } }
        }
      }
    }
  }"
```

Agent bot node IDs:
- Copilot: `BOT_kgDOC9w8XQ`
- Claude: `BOT_kgDODnPHJg`
- Codex: `BOT_kgDODnSAjQ`
