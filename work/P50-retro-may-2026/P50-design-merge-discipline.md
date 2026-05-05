# Design: Merge Discipline and Definition of Done

**Plan ID:** P50-retro-may-2026  
**Status:** Shaping  
**Source:** Session analysis 2026-05-05 — 30 unmerged branches discovered during P50 merge operation

## Overview

During the P50 merge operation, 30 branches were discovered that were not merged to main
despite their features being in `done` or `reviewing` status. Seventeen of those had worktree
records claiming `merged` status while `git merge-base --is-ancestor` showed otherwise. The
most impactful was P43 (fast-track architecture) — 18 tasks done, all tests passing, code
reviewed, but sitting on a branch for weeks while the main binary had no fast-track
implementation at all.

The root cause is twofold: there is no "merge" step in the definition of done, and there is
no post-merge verification to catch regressions. Features can reach `done` status with their
code still on a branch. Worktree records can report `merged` without the branch actually
being on main.

This design proposes two changes: add `merging` and `verifying` stages to the feature
lifecycle, and add a merge-discipline prompt to the workflow so agents are prompted to
merge after review passes.

## Goals and Non-Goals

**Goals:**
- Add `merging` and `verifying` lifecycle stages so a feature is not `done` until merged and verified
- Add a merge prompt to the workflow so agents are prompted to merge after review passes
- Fix the worktree `merged` status to require actual verification that the branch is on main
- Ensure zero test failures on main after merge (28 pre-existing failures exist as of 2026-05-05 and must be addressed or explicitly waived)

**Non-Goals:**
- Not fixing the 28 pre-existing test failures in this feature — they are tracked separately
- Not adding CI/CD integration — the verifying stage runs locally via `go build ./... && go test ./...`
- Not changing the `merge` tool's behaviour — it already works correctly when called

## Design

### Feature 5a: Merge and Verify lifecycle stages

Two new stages after `reviewing`:

```
reviewing → merging → verifying → done
```

**`merging`** — The feature has passed review and is ready to merge. The agent:
1. Calls `pr(action: "create")` if a PR doesn't already exist
2. Calls `merge(action: "check")` to verify merge gates
3. Calls `merge(action: "execute")` to merge to main
4. The feature advances to `verifying` automatically on successful merge

**`verifying`** — Post-merge verification on main:
1. `go build ./...` must pass with zero errors
2. `go test ./...` must pass with zero failures
3. If verification fails, the feature transitions to `needs-rework` with the failure output
4. If verification passes, the feature advances to `done`

**Worktree `merged` status fix:** After `merge(action: "execute")` succeeds, the system must
verify that the branch is actually on main (`git merge-base --is-ancestor <branch> main`)
before marking the worktree record as `merged`. If the branch is not on main, the worktree
remains `active` and the feature remains in `merging`.

### Feature 5b: Post-review merge prompt

After the review gate passes (all specialist reviewers produce reports, review-gate-validator
passes), the orchestrator skill must include a merge prompt:

> All review gates have passed. The feature is ready to merge.
> 1. Verify the PR exists: `pr(action: "status", entity_id: "...")`
> 2. Check merge gates: `merge(action: "check", entity_id: "...")`
> 3. Execute the merge: `merge(action: "execute", entity_id: "...")`
> 4. After merge, run `go build ./... && go test ./...` from the repository root
> 5. If all pass, transition the feature to `done`

This prompt is added to the `kanbanzai-agents` skill and the `orchestrate-review` skill.

### Merging stage gate enforcement

The new stage bindings:

```yaml
merging:
  description: "Merging reviewed code to main"
  orchestration: single-agent
  roles: [orchestrator]
  skills: [orchestrate-development]
  human_gate: false
  prerequisites:
    documents:
      - type: report
        status: approved
    tasks:
      all_terminal: true

verifying:
  description: "Post-merge build and test verification"
  orchestration: single-agent
  roles: [orchestrator]
  skills: [orchestrate-development]
  human_gate: false
  prerequisites:
    - merged_to_main: true
```

### What's Not Included

| Item | Reason |
|------|--------|
| Fixing pre-existing test failures | 28 failures predate P50. They should be fixed but are a separate concern. The `verifying` stage will catch regressions and prevent adding new failures. |
| CI/CD webhook integration | Future work. Local `go build/test` is sufficient for now. |
| Auto-merge | The agent still calls merge explicitly. Auto-merge is a future option. |

## Dependencies

- P43 fast-track must be merged to main (done — 2026-05-05 merge operation)
- The `merging` and `verifying` stage bindings must be added to `.kbz/stage-bindings.yaml`
- The `orchestrate-review` and `kanbanzai-agents` skills need the merge prompt added
- Dual-write rule applies to `internal/kbzinit/` copies of skill files

## Alternatives Considered

### Merge prompt in the `finish` tool vs. in the skill

The `finish` tool could include the merge prompt in its response after the last task is
completed. But `finish` doesn't know about review status — the orchestrator does. Putting
the prompt in the skill keeps it at the right layer.

**Decision:** Prompt in the skill, not the tool.

### Post-merge verification as a separate feature

The lifecycle changes and the merge prompt could be separate features. But they're tightly
coupled — the lifecycle provides the stage, the prompt drives the behaviour.

**Decision:** Single feature with two sub-concerns.

## Open Questions

1. Should the `verifying` stage gate be auto or human? Auto seems right — build/test is
   mechanical verification. But if tests fail and the feature goes to `needs-rework`, a human
   should see that.

2. Should the merge prompt also remind agents to delete the feature branch after merge?
   Probably yes — orphaned branches are how we got into this situation.

3. Should pre-existing test failures block the `verifying` stage? They shouldn't — the stage
   should check for *regressions* (new failures), not pre-existing ones. This requires a
   baseline of known failures or a waiver mechanism.
