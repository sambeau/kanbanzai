# Amendment 1 to: Deterministic Workflow Controller

**Status:** Draft
**Date:** 2026-05-13
**Owner:** P44-model-routing-agent-launcher
**Amends:** `P44-design-deterministic-workflow-controller.md`
**Theme:** Hygiene track — promote the workflow-hygiene fixes from Phase 2 into Phase 1, formalise the Definition of Done, introduce a Definition of Ready, and surface infrastructure health in `status`.

---

## 1. Motivation

The canonical P44 design correctly identifies a deterministic verifier (§5.3) as the structural fix for unmerged worktrees, ignored tests, forced merges, and bypassed close-out checks. However, the verifier is currently scheduled for Phase 2, behind a 1–3 month controller migration that depends on `internal/durable`, `internal/controller`, and `internal/policy`.

The deterministic checks themselves do **not** depend on that substrate. They are small Go functions that can be wired into the existing `merge(action: "check")` and `entity(action: "transition")` paths today. Deferring them stretches the bleeding by 2–4 months, which is the period during which the project has been losing work to bad merges and stale worktrees.

Three additional gaps in the canonical design surfaced during review:

1. The DoD exists only as a table inside a design document. There is no governable artefact (no policy file, no MCP surface) for "the agreed DoD" that can be reviewed, amended, and version-controlled as a first-class concern.
2. There is no Definition of Ready — no symmetric pre-edit gate. The supervisor can claim a task and start editing while the worktree is dirty, the branch has diverged from `main`, dependencies are incomplete, or the codebase index is stale.
3. `status` is silent on the metrics that would let a human notice the failure modes the project keeps hitting: stale worktrees, failing tests, unmerged feature branches, stale binaries, and bugs that closed without a review record.

This amendment promotes the deterministic verifier into Phase 1, adds DoD and DoR as governable policy artefacts, and extends `status` with the missing infrastructure health surface.

---

## 2. Scope of amendment

This amendment **adds** a Phase 1 sub-track ("Hygiene Track") to the canonical design. It does not change Phase 2 or Phase 3 architecture. It does not change Decisions 1, 2, 4, 5, 7, or 9. It refines Decision 3 (two-layered verifier) and Decision 6 (audit log) by sequencing them earlier.

---

## 3. New deliverables (Phase 1, Hygiene Track)

| # | Deliverable | Package / path | Notes |
|---|-------------|----------------|-------|
| H1 | Declarative DoD policy | `.kbz/policy/dod.yaml` + `internal/policy/dod.go` | One policy file, three entity-type entries (`feature`, `bug`, `retro-fix`); Go check functions register against named checks declared in YAML |
| H2 | Declarative DoR policy | `.kbz/policy/dor.yaml` + `internal/policy/dor.go` | Symmetric; runs at `next(id)` claim time and at every editing-tool invocation |
| H3 | Deterministic verifier (initial check set) | `internal/verifier/deterministic.go` | Implements the eight checks from canonical §5.3.1; called from existing `merge` and `entity transition` handlers |
| H4 | `dod` and `dor` MCP actions | New tool or new actions on existing `policy` tool | Read-only `get`, `list`, `evaluate` against an entity ID; lets humans inspect current rules and dry-run them |
| H5 | Editing-tool guardrails | `kanbanzai_edit_file`, `write_file`, commit hooks | Reject edits without `entity_id` for non-infrastructure roles; reject commits authored on `main` during a feature/bug stage |
| H6 | `status` infrastructure health | `internal/mcp/status_tool.go` | New sections: `tests`, `worktrees`, `binary`, `unmerged_work`, `bugs_without_review` |
| H7 | LLM verifier in non-blocking shadow mode | `internal/verifier/llm.go` (skeleton) | Records verdicts to audit log; cannot fail the gate in Phase 1; promoted to blocking in Phase 2 |
| H8 | Audit log (minimal) | `internal/audit` | Earlier-than-canonical landing for the dispatch/gate/verdict events the verifier needs to record; full schema as canonical §5.5 |

H1, H2, H3, H4, H6 are non-negotiable for Phase 1 exit. H5 is a hardening step that depends on H2. H7 and H8 are convenience: H7 lets the verifier learn before it bites; H8 provides the evidence trail the drift detector will consume in Phase 2.

---

## 4. Definition of Done — initial agreed set

The eight checks from canonical §5.3.1, transcribed into the proposed `.kbz/policy/dod.yaml`. This is the **agreed starting baseline**; future amendments can add, remove, or adjust required status per check.

```yaml
# .kbz/policy/dod.yaml
version: 1
entity_types:
  feature:
    checks:
      - name: git_status_clean
        required: true
      - name: branch_merged_to_main
        required: true
      - name: tests_pass
        required: true
      - name: doc_records_present
        required: true
        params:
          required_types: [specification, design, dev-plan]
      - name: worktree_removed
        required: true
      - name: all_tasks_terminal
        required: true
      - name: review_reports_present
        required: true
        params:
          min_dimensions: 1
      - name: incident_links_resolved
        required: false
  bug:
    checks:
      - name: git_status_clean
        required: true
      - name: branch_merged_to_main
        required: true
      - name: tests_pass
        required: true
      - name: worktree_removed
        required: true
      - name: review_reports_present
        required: true
        params:
          min_dimensions: 1
  retro-fix:
    checks:
      - name: git_status_clean
        required: true
      - name: branch_merged_to_main
        required: true
      - name: tests_pass
        required: true
      - name: worktree_removed
        required: true
```

**Closes failure mode:** P56-style "bug pipeline doesn't follow the DoD" by ensuring the bug pipeline runs the **same** check function, against entries in the **same** policy file, as features. There is no separate bug DoD code path.

**Governance:** changes to `dod.yaml` are PRs like any other code change. The `dod` MCP action lets humans see the current ruleset without grep'ing the repo.

---

## 5. Definition of Ready — initial agreed set

```yaml
# .kbz/policy/dor.yaml
version: 1
gates:
  task_claim:                     # runs at next(id) before transition ready→active
    checks:
      - name: worktree_exists
        required: true
      - name: worktree_status_clean
        required: true
      - name: branch_up_to_date_with_main
        required: true
      - name: dependencies_done
        required: true
      - name: role_skill_resolvable
        required: true
      - name: parent_feature_artefacts_approved
        required: true
        params:
          required_types: [specification, design, dev-plan]
      - name: codebase_index_fresh
        required: false           # advisory in Phase 1; promote to required in Phase 2
        params:
          max_age_hours: 24
  edit:                           # runs at every kanbanzai_edit_file / write_file call
    checks:
      - name: entity_id_present
        required: true
        params:
          exempt_roles: [base]
      - name: not_on_main
        required: true
        params:
          exempt_roles: [base]
      - name: server_binary_in_sync
        required: false           # advisory; warn-only
```

**Closes failure modes:**
- "AI Agents… assume they are the only working agent, so don't use worktrees" → `entity_id_present` + `worktree_exists`.
- "Hash-based edits and codebase-memory-mcp went missing without anyone noticing" → `codebase_index_fresh` (advisory in Phase 1; promotion to required in Phase 2 makes the staleness visible the moment it occurs).
- "Stashing hides state from parallel agents" → `worktree_status_clean` enforces commit-or-discard before claim.
- "Forced merges overwrite good code" → `branch_up_to_date_with_main` makes the divergence visible at claim time, not at merge time.

---

## 6. `status` infrastructure health (extension)

Extend the existing `status` MCP tool to add the following sections to its no-arg output. Per-entity `status(id)` adds an entity-scoped view of the same data.

### 6.1 `tests` section

Read from the existing `internal/teststatus` store; do **not** re-run tests.

```
tests:
  last_run_at: 2026-05-13T08:42:11Z
  last_run_age: 2h17m
  status: failing                   # passing | failing | stale | unknown
  failing_packages:
    - internal/verifier
    - internal/controller
  total_packages: 142
  failing_count: 2
```

Triggers an attention item if `status != passing` or `last_run_age > 24h`.

### 6.2 `worktrees` section

```
worktrees:
  active: 7
  stale: 2                          # > 7 days, no commits
  unmerged_complete: 1              # all tasks done, branch not merged to main
  diverged_from_main: 3             # branch behind main by > 50 commits
  orphaned_records: 0               # worktree record exists but directory absent
```

Each non-zero count produces an attention item with the offending entity IDs.

### 6.3 `binary` section

Read from `server_info` (already exposes `in_sync`, `version`, `git_sha`, `built_at`).

```
binary:
  version: 0.7.3
  git_sha: abc1234
  in_sync: false                    # running binary != installed binary
  age: 6d4h
  warning: "Running binary is 6 days older than HEAD; rebuild and reinstall."
```

`in_sync: false` produces a high-severity attention item. This is the explicit fix for the user-requested "add stale binary to status" — the data is already in `server_info`; this surfaces it in the routine `status` view so humans see it without having to ask.

### 6.4 `unmerged_work` section

```
unmerged_work:
  features_done_unmerged: 0         # entity status=done but branch not in main
  bugs_done_unmerged: 0
  bugs_closed_without_review: 1     # bug closed, no review-report doc exists
```

These are the lagging indicators that catch the exact "code was developed but not merged" failure mode the project has been hit by.

### 6.5 Attention-item priority order

When the no-arg `status` is called, attention items appear in this order:
1. `binary.in_sync == false` (will silently corrupt everything else)
2. `tests.status == failing`
3. `unmerged_work.features_done_unmerged > 0`
4. `worktrees.unmerged_complete > 0`
5. `bugs_closed_without_review > 0`
6. `worktrees.stale > 0`
7. Existing per-entity attention items

---

## 7. Refinement to Decision 3 (two-layered verifier)

**Original:** Two-layered; deterministic always runs first; LLM verifier always runs second; both can fail the gate.

**Refined for Phase 1:** Deterministic layer ships in Phase 1 and **can fail the gate**. LLM verifier ships in Phase 1 in **non-blocking shadow mode**: it produces a verdict, the verdict is recorded in the audit log and surfaced via `status`, but only the deterministic layer can block the transition. Phase 2 promotes the LLM verifier to blocking once audit-log evidence shows its judgement is calibrated.

**Rationale:** the project's track record on LLM judgement enforcement is poor. Letting the LLM verifier observe-and-record before it can block gives empirical evidence to anchor the promotion decision, and avoids reproducing the failure mode of "LLM was supposed to enforce this and silently passed."

**Cost:** marginal. The LLM verifier code is the same; only its return value's effect on the gate is gated by a flag.

---

## 8. Sequencing inside Phase 1

The hygiene track interleaves with the canonical Phase 1 deliverables. Suggested order:

1. **Week 1–2:** H6 (`status` extensions). No new policy infrastructure; pure read-side. Immediate visibility win.
2. **Week 2–3:** H1 + H3 (DoD policy file + deterministic verifier checks). Wired into existing `merge(action: "check")` only — does not yet block transitions, runs in parallel with current behaviour, results compared in audit log.
3. **Week 3–4:** H8 (minimal audit log) — needed by H3 for evidence trail.
4. **Week 4–5:** flip H3 from observe-only to blocking on `merge` and on `entity transition status: done`.
5. **Week 5–6:** H2 (DoR policy + checks at `next(id)` claim time).
6. **Week 6:** H5 (editing-tool guardrails — depends on H2).
7. **Week 6–7:** H4 (`dod` and `dor` MCP actions for human inspection).
8. **Week 7+:** H7 (LLM verifier shadow mode) — depends on H8.

This sequence delivers visibility (H6) in week 2, deterministic enforcement (H3) blocking by week 5, and the symmetric pre-edit gate (H2 + H5) by week 6. None of it depends on `internal/controller`, `internal/durable`, or the policy facade migration; those remain Phase 2.

---

## 9. Phase 1 exit criteria (amended)

In addition to the canonical Phase 1 exit criteria, add:

- `.kbz/policy/dod.yaml` exists, is approved, and is the source of truth for DoD checks (no DoD logic hardcoded in Go outside the registered check functions).
- `.kbz/policy/dor.yaml` exists, is approved, and is enforced at `next(id)` and at every editing-tool invocation.
- The deterministic verifier blocks `merge(execute)` and `entity(transition status: done)` for `feature`, `bug`, and `retro-fix` entity types when any required check fails.
- `status` (no-arg) surfaces tests, worktrees, binary sync state, and unmerged-work counts; binary out-of-sync produces a high-severity attention item.
- At least three features and one bug have closed under the new gates.
- Zero closes via `override` in the cohort, OR every override has a recorded `override_reason` and a corresponding entry in the audit log.

---

## 10. Risks introduced by this amendment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Deterministic checks block legitimate work because of edge cases (e.g. an unrelated dirty file in the worktree) | Medium | Medium | Two-week observe-only mode (Week 2–4) before flipping to blocking; record every would-have-blocked event in audit log and triage |
| DoR `branch_up_to_date_with_main` blocks too aggressively for long-running features | Medium | Low | Configurable threshold in `dor.yaml`; advisory by default in Phase 1 |
| Editing-tool guardrails (H5) break the supervisor's ability to do legitimate infrastructure work | Medium | Medium | `exempt_roles: [base]` escape hatch; revisit in Phase 2 with finer-grained policy |
| `codebase_index_fresh` check produces false positives if codebase-memory-mcp is unavailable | Medium | Low | Advisory only in Phase 1 |
| Hygiene-track work delays the canonical Phase 1 deliverables | Low | Medium | The two tracks are largely independent; H1/H3/H6 share no code with `dispatch_task`/eval-harness/`spawn_agent` removal |

---

## 11. Out of scope (explicitly)

- Stage controllers, `internal/durable`, `internal/policy` facade migration — remain Phase 2 per canonical design.
- LLM verifier as a blocking gate — remains Phase 2.
- Multi-provider routing — remains Phase 3.
- DoD/DoR policy versioning, migration, or hot-reload — Phase 2 concern; Phase 1 reloads on server restart.
- Per-role tool-permission enforcement at the MCP transport layer — remains Phase 2.

---

## 12. Cross-references

- Amends: `P44-design-deterministic-workflow-controller.md` (canonical design)
- Reuses: §5.3.1 deterministic check inventory; §5.5 audit-log schema
- Refines: Decision 3 (sequencing); Decision 6 (sequencing)
- Related: P53-infrastructure-hygiene (overlapping concerns, currently in `idea` status — this amendment likely subsumes it; recommend cancellation if approved)
- Related: P56-bug-lifecycle-hardening (the bug-DoD gap this amendment closes structurally rather than with a one-off bug code path)

---

*End of amendment.*
