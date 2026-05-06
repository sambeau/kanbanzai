# Guide: Remediation Ownership Model

| Field    | Value                                                     |
|----------|-----------------------------------------------------------|
| Plan     | P54-review-remediation-workflow                           |
| Type     | guide                                                     |
| Status   | Draft                                                     |
| Author   | sambeau (orchestrator)                                    |
| Date     | 2026-05-06                                                |

---

## Overview

When a review fails, blocking findings must be placed under the correct ownership scope for remediation. Placing remediation at the wrong scope creates entity-management confusion and lifecycle ambiguity (FR-04). This guide provides the decision tree, criteria, and entity-creation steps for each of the three ownership models.

---

## Decision Tree

```
                            ┌─────────────────────────┐
                            │  Failed review received  │
                            │  with blocking findings  │
                            └────────────┬────────────┘
                                         │
                                         ▼
                            ┌─────────────────────────┐
                            │  Are ALL findings        │
                            │  scoped to a SINGLE     │
                            │  feature entity?         │
                            └────────────┬────────────┘
                                         │
                         ┌───────────────┴───────────────┐
                         │                               │
                         ▼ YES                           ▼ NO
              ┌─────────────────────┐      ┌─────────────────────────┐
              │ SINGLE-FEATURE      │      │  Do findings span        │
              │ MODEL               │      │  MULTIPLE features       │
              │                     │      │  within ONE batch?       │
              │ See §Single-Feature │      └────────────┬────────────┘
              └─────────────────────┘                   │
                                         ┌──────────────┴──────────────┐
                                         │                             │
                                         ▼ YES                         ▼ NO
                              ┌─────────────────────┐    ┌─────────────────────┐
                              │ BATCH-LEVEL         │    │ CROSS-CUTTING       │
                              │ MODEL               │    │ MODEL               │
                              │                     │    │                     │
                              │ See §Batch-Level    │    │ See §Cross-Cutting  │
                              └─────────────────────┘    └─────────────────────┘
```

---

## Single-Feature Model

### When to use

- **ALL** blocking findings are scoped to exactly one feature entity.
- The fix does not require changes to other features, shared infrastructure, or workflow skills.
- Example: a feature's implementation is missing test coverage for its own acceptance criteria.

### Decision criteria

| Criterion | Check |
|-----------|-------|
| All findings reference the same feature ID | `entity(action: "get", id: "<FEAT-xxx>")` confirms scope |
| No findings reference batch-level or cross-cutting concerns | Review each finding's "Affected entity" field |
| Fix does not require changes to shared packages used by other features | Check file paths in findings against other active worktrees |

### Entity-creation steps

```text
# 1. Transition the feature to needs-rework (if not already)
entity(action: "transition", id: "<FEAT-xxx>", status: "needs-rework")

# 2. Create remediation tasks under the feature
decompose(action: "propose", feature_id: "<FEAT-xxx>")
decompose(action: "review", feature_id: "<FEAT-xxx>", proposal: <proposal>)
decompose(action: "apply", feature_id: "<FEAT-xxx>", proposal: <proposal>)

# 3. Register the remediation dev-plan under the feature
doc(action: "register", path: "work/<feature-slug>/<plan-id>-dev-plan-remediation-<slug>.md",
    type: "dev-plan", title: "Remediation Dev-Plan: <summary>", owner: "<FEAT-xxx>")
```

### Close-out

```text
# After all remediation tasks are done and re-review report is approved:
entity(action: "transition", id: "<FEAT-xxx>", status: "reviewing")
# Proceed through standard review → merge → done lifecycle
```

### Example (from P50)

Finding BF-3 ("state_modified code exists only on main, not in worktree") was scoped to a single feature (F3 of B1-p51-exec). Remediation was a direct commit to main via the worktree for that feature. No other features were affected.

---

## Batch-Level Model

### When to use

- Blocking findings span **two or more features** within the same batch.
- The findings are about the batch's aggregate delivery, not a single feature.
- Example: a batch conformance review finds that multiple features lack verification evidence or have interleaved worktree changes.

### Decision criteria

| Criterion | Check |
|-----------|-------|
| Findings reference two or more features in the same batch | `entity(action: "get", id: "<BATCH-xxx>")` confirms child features |
| No findings require a new plan (e.g., reusable workflow gap) | Review finding descriptions for cross-batch scope |
| Fixes are confined to the batch's features | Check affected file paths |

### Entity-creation steps

```text
# 1. Transition each affected feature to needs-rework
entity(action: "transition", id: "<FEAT-aaa>", status: "needs-rework")
entity(action: "transition", id: "<FEAT-bbb>", status: "needs-rework")
# ... for each affected feature

# 2. Create per-feature remediation tasks
# For each feature:
decompose(action: "propose", feature_id: "<FEAT-xxx>")
decompose(action: "review", feature_id: "<FEAT-xxx>", proposal: <proposal>)
decompose(action: "apply", feature_id: "<FEAT-xxx>", proposal: <proposal>)

# 3. Register the remediation dev-plan under the batch
doc(action: "register", path: "work/<batch-slug>/<plan-id>-dev-plan-remediation-<slug>.md",
    type: "dev-plan", title: "Remediation Dev-Plan: <summary>", owner: "<BATCH-xxx>")
```

### Close-out

```text
# After all per-feature remediation tasks are done:
# Transition each feature back through review → merge → done
# When all features are done, the batch can advance to reviewing
```

### Example (from P50)

The B1-p51-exec batch conformance review produced 10 blocking findings (BF-1 through BF-10) spanning 4 features. The remediation dev-plan was registered under the batch. Per-feature tasks were created under each affected feature (F2, F3, F4, F5). Re-review verified each finding was resolved per feature.

---

## Cross-Cutting Model

### When to use

- Blocking findings span **multiple batches** or represent a **reusable workflow gap**.
- The fix is not confined to a single feature or batch — it requires a new plan with its own design, specification, and implementation.
- Example: a review finds that the orchestrator skill is missing a required checklist item, affecting all future batches that use it.

### Decision criteria

| Criterion | Check |
|-----------|-------|
| Findings reference entities in two or more batches | Check finding scopes across batch boundaries |
| The fix would benefit from its own design/spec cycle | Is the gap a reusable workflow improvement? |
| Fixing it under a single feature would be scope creep | Would the fix change behavior for other batches? |

### Entity-creation steps

```text
# 1. Create a new plan entity
entity(action: "create", type: "strategic-plan",
       id: "P<n>-<slug>",
       name: "Remediation: <summary>",
       summary: "<description of the cross-cutting gap and what the plan will deliver>",
       status: "active")

# 2. Create a batch under the plan
entity(action: "create", type: "batch",
       id: "B<n>-<slug>",
       name: "Remediation Batch: <summary>",
       parent: "P<n>-<slug>")

# 3. Create features under the batch (one per affected area)
entity(action: "create", type: "feature",
       id: "FEAT-<xxx>",
       name: "<feature name>",
       parent: "B<n>-<slug>",
       summary: "<description>")

# 4. Register the remediation dev-plan under the new plan
doc(action: "register", path: "work/<plan-slug>/<plan-id>-dev-plan-remediation-<slug>.md",
    type: "dev-plan", title: "Remediation Dev-Plan: <summary>", owner: "P<n>-<slug>")
```

### Close-out

```text
# The new plan follows its own lifecycle independently:
# design → spec → dev-plan → implement → review → merge → done
# It is not coupled to the original batch's lifecycle.
```

### Example (from P54)

P54 (this plan) is itself an example of the cross-cutting model. P50's batch conformance review identified a reusable workflow gap: there was no standardized bridge from a failed review to executable remediation. Rather than fixing this under P50 or P51, a new plan (P54) was created with its own design, specification, and implementation. The original finding (the manual remediation process used during P50) became the motivating example, not the scope boundary.

---

## Comparison Table

| Aspect | Single-Feature | Batch-Level | Cross-Cutting |
|--------|---------------|-------------|---------------|
| **Scope** | One feature | Multiple features in one batch | Multiple batches or reusable gap |
| **Dev-plan owner** | Feature | Batch | New plan |
| **Tasks under** | Original feature | Each affected feature | New feature(s) under new batch |
| **Entity transitions** | Feature → needs-rework | Each feature → needs-rework | New plan → active; new batch → active |
| **Close-out** | Feature → reviewing → done | Per-feature → done; batch → reviewing | New plan follows full lifecycle |
| **Re-review report owner** | Feature or batch | Batch | New plan |
| **P50 example** | BF-3 (F3 direct commit) | BF-1 through BF-10 (B1-p51-exec) | P54 (this plan) |

---

## Edge Cases

### Finding spans a feature and a shared package

If a finding affects a feature (FEAT-xxx) AND a shared package (e.g., `internal/service/`) used by other features:

1. Check whether the shared-package change is truly required or can be scoped to the feature.
2. If the shared change is unavoidable, escalate to batch-level model — the change affects multiple features.
3. Record the shared-package risk in the dev-plan's Risk Assessment section.

### Feature is already done/merged

If a finding references a feature that is already `done`:

1. The feature lifecycle does not support `done → needs-rework` (see KE on lifecycle-model).
2. Options:
   - Create a new feature under the same batch for the remediation (preferred).
   - Supersede the original feature with a new one (if the fix is substantial).
   - For trivial fixes, commit directly and note in the re-review report (use sparingly).

### Finding references a deferred plan (e.g., P53)

If a finding says "this should use P53 tooling" but P53 hasn't shipped:

1. Record P53 unavailability as a Risk in the dev-plan.
2. Use the manual fallback steps from the [Review Remediation Procedure](P54-report-review-remediation-procedure.md) §Phase 2.
3. Defer the P53-dependent portion of the finding with a "revisit when P53 ships" rationale.
4. Do not block remediation on an unavailable dependency.

---

## See Also

- [Review Remediation Procedure](P54-report-review-remediation-procedure.md)
- [P54 Specification: Review Remediation Workflow](P54-spec-review-remediation-workflow.md) — FR-04
- [Remediation Dev-Plan Template](P54-template-remediation-dev-plan.md)
- [Re-Review Report Template](P54-template-re-review-report.md)
