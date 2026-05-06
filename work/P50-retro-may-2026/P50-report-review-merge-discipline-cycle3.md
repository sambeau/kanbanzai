# Review Report: Merge Discipline (F4) — Cycle 3

| Field | Value |
|-------|-------|
| Feature | FEAT-01KQTWFY52EG1 |
| Reviewer | reviewer-conformance |
| Date | 2026-05-06 |
| Verdict | approved_with_followups |

## Findings Resolution

### BF-5: Transition state machine unenforced

**Status:** RESOLVED (worktree)

`IsValidFeatureTransition` is called in `entityTransitionAction` before `UpdateStatus` (worktree `internal/mcp/entity_tool.go` L721-726). The call guards the transition path:

```internal/mcp/entity_tool.go
		if entityType == "feature" && fromStatus != "" && !override {
			if !model.IsValidFeatureTransition(model.FeatureStatus(fromStatus), model.FeatureStatus(newStatus)) {
				return entityTransitionError(entitySvc, entityType, entityID, newStatus,
					fmt.Errorf("invalid transition %q → %q", fromStatus, newStatus)), nil
			}
		}
```

`FeatureValidTransitions` includes the new merging/verifying stages (worktree `internal/model/entities.go` L763-784):

- `FeatureStatusReviewing → FeatureStatusMerging` ✓
- `FeatureStatusMerging → FeatureStatusVerifying` ✓
- `FeatureStatusVerifying → FeatureStatusDone` ✓
- `FeatureStatusVerifying → FeatureStatusNeedsRework` ✓

Invalid transitions such as `cancelled → developing` are rejected because `FeatureStatusCancelled` has no outgoing transitions. Test assertions at worktree `internal/model/entities_test.go` L824-878 confirm all valid and invalid transitions, including `merging→done` (rejected — must go through verifying) and `verifying→merging` (rejected — backward).

**Evidence trace:** AC-001, AC-002, and the `reviewing→merging`/`merging→verifying`/`verifying→done` paths are all exercised by `TestMergeVerifyDone_LifecycleTransitions` and `TestFeatureValidTransitions`.

### BF-8: Agents skill missing merge prompt

**Status:** RESOLVED (worktree)

Post-Review Merge 5-step section added to both copies:

- `.agents/skills/kanbanzai-agents/SKILL.md` — "Post-Review Merge" section at L415-469 (worktree)
- `internal/kbzinit/skills/agents/SKILL.md` — "Post-Review Merge" section at L415-469 (worktree)

The section contains the five required steps: (1) verify all tasks terminal, (2) transition to `merging`, (3) merge check and execute, (4) verify merge ancestry, (5) build, test, and transition to `done` or `needs-rework`.

The dual-write rule (REQ-009/AC-009) is satisfied: both files contain identical Post-Review Merge content. The `orchestrate-review` SKILL.md merge prompt (at `.kbz/skills/orchestrate-review/SKILL.md` L286-360) is already present on main and is more detailed (per-spec AC-007).

### BF-9: Missing needs-rework→developing transition

**Status:** RESOLVED (worktree)

`FeatureStatusDeveloping: true` added to `FeatureStatusNeedsRework` in `FeatureValidTransitions` (worktree `internal/model/entities.go` L779-784):

```internal/model/entities.go
	FeatureStatusNeedsRework: {
		FeatureStatusDeveloping: true,
		FeatureStatusReviewing:  true,
		FeatureStatusSuperseded: true,
		FeatureStatusCancelled:  true,
	},
```

Test assertion at worktree `internal/model/entities_test.go` L845-849 confirms:

```internal/model/entities_test.go
	// needs-rework→developing is valid
	if !model.IsValidFeatureTransition(model.FeatureStatusNeedsRework, model.FeatureStatusDeveloping) {
		t.Error("needs-rework→developing should be a valid transition")
	}
```

The `validate/allowedTransitions` table in `internal/validate/lifecycle.go` already had `FeatureStatusDeveloping` in `FeatureStatusNeedsRework` on main (L165-172).

## Remaining Findings

### RF-1: `validate/allowedTransitions` missing merging/verifying stages (non-blocking)

The `allowedTransitions` table in `internal/validate/lifecycle.go` (both main and worktree) does not include `FeatureStatusMerging` or `FeatureStatusVerifying` entries. `FeatureStatusReviewing` does not list `FeatureStatusMerging` as a valid next state:

```internal/validate/lifecycle.go
		string(model.FeatureStatusReviewing): {
			string(model.FeatureStatusDone):        {},
			string(model.FeatureStatusNeedsRework): {},
			string(model.FeatureStatusSuperseded):  {},
			string(model.FeatureStatusCancelled):   {},
		},
```

This means `CanTransition(EntityFeature, "reviewing", "merging")` returns `false`. The primary enforcement path (`IsValidFeatureTransition` in `entity_tool.go`) is correct, so actual transitions are properly gated. The `allowedTransitions` table is used by `CanTransition`, `ValidNextStates`, and `NextStates` — any code path relying on those functions will miss the merging/verifying stages.

**Recommendation:** Add `FeatureStatusMerging` and `FeatureStatusVerifying` entries to `allowedTransitions` in `validate/lifecycle.go`, mirroring the `FeatureValidTransitions` map. Not blocking for merge because the primary enforcement path is correct.

### RF-2: AC-003/AC-004 — no auto-transition to needs-rework on verification failure (non-blocking)

The spec requires automatic transition to `needs-rework` when build or tests fail:

> AC-003: Given a feature in `verifying`, when `go build ./...` fails, then the feature transitions to `needs-rework` with the build error in the transition reason.
> AC-004: Given a feature in `verifying`, when `go test ./...` fails, then the feature transitions to `needs-rework` with the test failure output in the transition reason.

The implementation deliberately does NOT auto-transition to `needs-rework`. Instead it leaves the feature in `verifying` and surfaces verification output as warnings:

```internal/mcp/merge_tool.go
					} else {
						warnings = append(warnings, fmt.Sprintf("verification failed for %s after merge: build/tests did not pass. Feature left in verifying — revert or fix and retry.", entityID))
						warnings = append(warnings, fmt.Sprintf("verification output:\n%s", verifyResult.Output))
						// Do NOT auto-transition to needs-rework — leave in verifying so
						// the orchestrator can inspect and decide. The orchestrator should
						// investigate, fix, and re-run entity(action: transition, to: done)
						// or entity(action: transition, to: needs-rework) as appropriate.
					}
```

The rationale (orchestrator should inspect pre-existing test failures before deciding) is sound given 28 known pre-existing test failures, but the spec and implementation are out of alignment.

**Recommendation:** Either (a) update AC-003/AC-004 to reflect the current behaviour, or (b) implement the auto-transition with a waiver-file mechanism for known failures. The current behaviour is acceptable for merge — the feature doesn't reach `done` on failure, which is the critical invariant.

### RF-3: Main branch not yet updated with all resolutions (expected)

All three BF resolutions exist only in the worktree (`FEAT-01KQTWFY52EG1-merge-discipline`). The main branch already contains the supporting infrastructure (constants `FeatureStatusMerging`/`FeatureStatusVerifying`, `merge_tool.go` with `executeMerge`+`runVerifyingStage`, `stage-bindings.yaml`, `orchestrate-review` SKILL.md merge prompt), but is missing the `FeatureValidTransitions` entries, the `entity_tool.go` `IsValidFeatureTransition` call, and the `kanbanzai-agents` SKILL.md merge prompt. This is expected for an unmerged feature — the merge itself will resolve this.

## Verdict

**Approved with follow-ups.** All three blocking findings from Cycle 2 are resolved in the feature worktree. The `reviewing → merging → verifying → done` lifecycle chain is complete in `FeatureValidTransitions`, enforced by `IsValidFeatureTransition` in `entity_tool.go`, and documented in both skill files with proper dual-write. The implementation correctly verifies merge ancestry before marking worktrees merged and runs build+tests before reaching `done`.

RF-1 (validate/allowedTransitions gap) and RF-2 (spec-implementation alignment on auto-transition) are non-blocking follow-ups. RF-3 is self-resolving on merge.
