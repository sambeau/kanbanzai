# Design: Lifecycle Gate Hardening (P31)

| Field  | Value                     |
|--------|---------------------------|
| Date   | 2026-04-23                |
| Status | Draft                     |
| Author | Claude Sonnet 4.6         |
| Plan   | P31-lifecycle-gate-hardening |

---

## Related Work

### Prior documents consulted

| Document | Type | Relevance |
|----------|------|-----------|
| `work/design/workflow-design-basis.md` §7.1 | design | Canonical source of truth: YAML files + SQLite cache; lifecycle as enforced workflow |
| `work/design/kanbanzai-3.0-workflow-and-tooling-v2.md` §3.4, §3.5, Q6 | design | Gate override mechanism design; override audit trail; binding registry as gate source |
| `work/design/quality-gates-and-review-policy.md` §2 | design | Core principle: "No unit of work should be treated as complete merely because code exists" |
| `work/research/p28-issues-investigation.md` §2.3 | research | Cluster C root cause: merge gate allows override bypass; status dashboard does not surface orphans |
| `work/reports/retro-p28-doc-intel-polish-workflow-reliability.md` | report | Issue 6: FEAT-01KPVDDYQQS1Y orphaned in reviewing with no review report; visible only at plan close |

### Constraining decisions

- **P17 §3.4 Gate Override:** Overrides are permitted and audit-logged, but the design
  notes they should be used for "exceptional circumstances." The current implementation
  allows `override: true` to bypass the review report existence check for `reviewing →
  done` without any enforcement distinction between gate types.
- **P17 §3.5 / WP-5:** Gates are driven by the binding registry. Any new gate condition
  should follow the same registry-driven pattern rather than being hardcoded into the
  merge tool handler.
- **Quality Gates Policy §2:** Completion requires quality review against a documented
  standard. A feature in `reviewing` status that merged without a review report violates
  this principle — the review step occurred but left no artifact.

### How this design extends prior work

The override mechanism (P17 §3.4) was designed for situations where a gate condition
cannot be met for legitimate reasons (e.g. an external blocker). This design does not
remove override capability; it adds a second, narrower enforcement point: the absence of
a review report for a `reviewing`-stage feature is not an exceptional circumstance — it
is evidence that the reviewing step was skipped. The design makes this condition visible
before merge and as a standing dashboard warning, closing the gap without restricting
legitimate use of `override`.

---

## Overview

Lifecycle gate hardening closes two gaps in the `reviewing` stage enforcement: a missing
review report at merge time is not blocked (and can be silently bypassed with
`override: true`), and features orphaned in `reviewing` with no report are invisible to
the status dashboard. This plan adds a non-bypassable merge gate for `reviewing`-stage
features without a registered review report, and a standing dashboard warning for
features in that state.

## Goals and Non-Goals

**Goals:**
- Prevent merging a `reviewing`-stage feature that has no registered review report document.
- Make the gate non-bypassable with `override: true` for this specific condition.
- Surface features stuck in `reviewing` with no registered report as a `warning`-severity attention item in the `status` dashboard (project, plan, and feature scope).
- Provide a clear, actionable error message when the gate fires.

**Non-Goals:**
- Does not change the override mechanism for any other gate condition — all existing gates remain bypassable as before.
- Does not require the review report to be in `approved` status — a `draft` report satisfies the gate.
- Does not retroactively block already-merged features with no report (they appear as dashboard warnings only).
- Does not change the reviewing stage's `human_gate: true` binding or the reviewing workflow itself.
- Does not address the state persistence performance issues (P29) or handoff skill assembly issues (P30).

## Dependencies

- `internal/gate/` — gate result type must gain a `Bypassable bool` field; `ReviewReportExistsGate` must be added.
- `internal/mcp/merge_tool.go` (or equivalent merge gate handler) — must check `Bypassable` before accepting `override: true`.
- `internal/mcp/status_tool.go` (or equivalent) and the attention item assembler — must add `OrphanedReviewingFeatureCheck`.
- `internal/service/document.go` (or `docRecordSvc`) — `List()` filtered by owner + type must be available to both the gate and the status check; no new methods anticipated.
- No dependency on P29 or P30. Can be implemented and shipped independently.

## Problem and Motivation

When a feature reaches the `reviewing` lifecycle stage, it is expected to have a review
report registered in the document store before merging. The review report is the artifact
that makes the quality gate meaningful: it records what was checked, what was found, and
whether the feature meets the specification.

Two failure modes exist today:

**Failure mode 1 — Silent merge via override.** The `merge` tool's pre-merge gate checks
whether required lifecycle conditions are met, but the `override: true` parameter bypasses
all gate conditions uniformly. A developer or agent can set `override: true` to force a
merge of a `reviewing`-stage feature with no review report. This occurred in P28 with
FEAT-01KPVDDYQQS1Y, which was merged at commit `1ecf036` in `reviewing` state with no
review report, using `override: true`. The override was not blocked, flagged, or logged
differently from a legitimate override.

**Failure mode 2 — Invisible orphan.** After the above merge, the feature's status was
left at `reviewing` in the entity store (the merge did not advance the lifecycle). This
produced an entity that was visually `reviewing` but already merged — an orphaned state
that was invisible to both orchestrators until the second session attempted to close P28
and discovered the inconsistency. The `status` project dashboard does not include a check
for features in `reviewing` with no registered review report, so the orphan generated no
warning.

Together these failures mean the reviewing stage can be effectively bypassed without any
signal: no blocked merge, no dashboard warning, no audit distinction.

If nothing changes: every plan risks an invisible lifecycle orphan. The pattern has
recurred in P26 and P28. The reviewing stage's quality guarantee degrades to a convention
rather than an enforced workflow step.

---

## Design

### Component overview

This design introduces changes at two layers:

1. **Merge gate enforcement** — a new non-overridable gate condition on the merge tool
   for `reviewing`-stage features.
2. **Status dashboard warning** — a new attention item surfaced by the `status` tool
   for features stuck in `reviewing` without a registered review report.

These two components are independent and can be implemented and tested separately. They
address different points in the failure timeline: the gate fires at merge time (preventing
the orphan), the dashboard warning fires at any `status` call (surfacing orphans that
already exist).

---

### Component 1: Merge gate — review report existence check

**Trigger:** `merge(action: execute, entity_id: FEAT-xxx)` when the feature's current
lifecycle status is `reviewing`.

**Condition:** At least one document of type `report` must be registered in the document
store with `owner` matching the feature ID and `status = approved` (or `status = draft`
as an acceptable fallback — see Alternatives Considered).

**Current gate architecture:** The merge tool delegates gate evaluation to a `GateRouter`
(`internal/mcp/merge_tool.go` → `internal/gate/`). The gate router evaluates conditions
from the binding registry and calls service-level checks. New structural gate conditions
follow this path.

**New gate condition:** Add `ReviewReportExistsGate` to the gate evaluation chain for
`reviewing`-stage features. This gate:
- Calls `docRecordSvc.List()` filtered by owner = feature ID, type = `report`.
- Returns `gate.Blocked` if the result set is empty.
- Returns `gate.Pass` if at least one report document exists (any status).

**Override behaviour:** This gate condition is **not bypassable by `override: true`**
when the feature is in `reviewing` status. The rationale: if a feature has genuinely been
reviewed, producing a report is a 1-minute operation. The absence of a report is not a
legitimate blocker — it is evidence of a skipped step. Contrast with a legitimate override
scenario: a dependency gate blocked by an external service that cannot be unblocked in
reasonable time.

To implement this distinction, the gate result carries a `Bypassable bool` field. The
merge tool's override logic checks this field before accepting the override. Existing
gate conditions remain bypassable; only `ReviewReportExistsGate` sets `Bypassable: false`.

**Error message:** When blocked, the merge tool returns:

```
Cannot merge FEAT-xxx: feature is in 'reviewing' status but no review report is
registered.

To resolve:
  1. Run the review: handoff(task_id: ...) with role: reviewer-conformance (and
     other reviewer roles per the developing stage binding).
  2. Register the report: doc(action: register, type: report, owner: FEAT-xxx, ...).
  3. Retry merge(action: execute, entity_id: FEAT-xxx).

If the feature was reviewed but the report was not registered, register it now.
This gate cannot be bypassed with override: true — a report must exist.
```

---

### Component 2: Status dashboard — orphaned reviewing warning

**Trigger:** Any `status()` or `status(id: P-xxx)` call.

**Condition:** A feature with `status = reviewing` that has no registered document of
type `report` with `owner` matching the feature ID.

**Current dashboard architecture:** The `status` tool assembles attention items from a
set of check functions. Each check function scans relevant entities and returns a list of
`AttentionItem` structs with a severity level (warning, info, error).

**New attention item:** Add `OrphanedReviewingFeatureCheck` to the attention item
assembly. This check:
- Lists all features with `status = reviewing`.
- For each, calls `docRecordSvc.List()` filtered by owner = feature ID, type = `report`.
- If no report document exists, emits an `AttentionItem` with:
  - `severity: warning`
  - `entity_id: FEAT-xxx`
  - `message: "Feature FEAT-xxx (slug) is in 'reviewing' status with no registered review report"`

**Scope:** Project-level `status()` and plan-scoped `status(id: P-xxx)` both run this
check. Feature-scoped `status(id: FEAT-xxx)` runs it for the single feature only.

**Performance:** The check calls `docRecordSvc.List()` once per `reviewing` feature. At
typical project scale (1–3 features in reviewing at any time) this is a negligible cost.
The check is skipped if there are zero `reviewing` features.

---

### Data flow

```
merge(execute, FEAT-xxx)
  → GateRouter.Evaluate(feature)
      → ReviewReportExistsGate.Check(featureID, docRecordSvc)
          → docRecordSvc.List(owner=featureID, type=report)
          → [] → Blocked{Bypassable: false}
          → [report] → Pass
      → if Blocked && !result.Bypassable → return error (ignore override flag)
      → if Blocked && result.Bypassable && override=true → proceed with audit log

status() / status(P-xxx)
  → AttentionItemAssembler.Run()
      → OrphanedReviewingFeatureCheck(features, docRecordSvc)
          → for each reviewing feature: docRecordSvc.List(owner=f.ID, type=report)
          → emit AttentionItem{severity: warning} for each with no report
```

---

### Failure modes and handling

| Failure mode | Handling |
|---|---|
| `docRecordSvc.List()` returns an error | Gate returns `Pass` (fail-open). The merge proceeds; a log warning is emitted. Fail-open is preferred over a broken merge tool. |
| Feature has a report in `draft` status (not yet approved) | Gate returns `Pass`. A draft report is evidence that the reviewing step was performed. Requiring approval before merge would create a chicken-and-egg problem for the first review cycle. |
| `status()` call with docRecordSvc unavailable | Check is skipped silently. Attention items are best-effort; missing one does not break the dashboard. |
| Feature already merged but status left at `reviewing` (pre-existing orphan) | The dashboard warning fires on every `status()` call for such features, making them visible. No retroactive gate action is taken on already-merged features. |

---

## Alternatives Considered

### Alternative A: Make the review report gate bypassable (status quo + visibility only)

Add the review report existence check to the merge gate but keep it bypassable with
`override: true`, matching the behaviour of all existing gates. Add the dashboard warning.

**What it makes easier:** No change to the override mechanism. Consistent gate semantics.
**What it makes harder:** Reproduces the exact failure mode from P28. An agent under time
pressure will set `override: true` and the gate is bypassed again. The dashboard warning
helps surface orphans after the fact but does not prevent them.
**Rejected because:** The failure mode is not "the gate is hard to satisfy" — it is "the
gate was silently bypassed." Making the gate bypassable leaves the bypass path open.

### Alternative B: Block the transition `reviewing → done` at the entity level (not the merge tool)

Add a lifecycle transition guard in `UpdateStatus` that prevents `reviewing → done` if no
report exists, rather than checking at merge time.

**What it makes easier:** Gate fires on any `reviewing → done` transition, not just
`merge`. Closes the gap even if merge is bypassed.
**What it makes harder:** `UpdateStatus` does not have access to `docRecordSvc` today —
it would need to be injected. This couples the entity service to the document service,
creating a dependency direction that does not currently exist and that the architecture
(separation of concerns, `internal/service/` as a peer set) does not support cleanly.
**Rejected because:** Introducing a cross-service dependency in the entity mutation path
for this use case is a larger architectural change than warranted. The merge tool is the
intended control surface for lifecycle advancement; the gate at merge time is the correct
enforcement point.

### Alternative C: Require an approved (not just registered) review report

Change the gate condition to require `status = approved` rather than merely the existence
of a document of type `report`.

**What it makes easier:** Stronger guarantee — the report was reviewed and approved, not
just filed.
**What it makes harder:** Creates a blocking dependency on human approval of the report
before any merge can proceed. The reviewing stage already has `human_gate: true` in the
binding — the human approval step exists. Requiring approval at the document level as well
doubles the approval ceremony with no additional quality benefit.
**Rejected because:** The purpose of the gate is to ensure the reviewing step was
performed, not to enforce a second approval layer. A filed report (draft or approved) is
sufficient evidence.

### Alternative D: Do nothing (status quo)

Accept that the reviewing lifecycle can be silently bypassed and document the risk.

**What it makes easier:** No implementation work.
**What it makes harder:** The pattern has recurred in P26 and P28. With no structural
change, it will recur again. The quality-gates policy ("no unit of work should be treated
as complete merely because code exists") is violated on each recurrence.
**Rejected because:** Recurrence across two plans with no fix makes this a structural
gap, not a one-off error.

---

## Decisions

### Decision 1: Non-bypassable gate for missing review report

- **Decision:** The `ReviewReportExistsGate` is not bypassable with `override: true` when
  the feature is in `reviewing` status.
- **Context:** All existing merge gate conditions are bypassable with `override: true`
  (P17 §3.4). The question is whether this new condition should follow the same pattern.
- **Rationale:** Filing a review report is a minimal, always-achievable action (it takes
  under a minute if the review was actually performed). There is no legitimate scenario in
  which a reviewed feature cannot have a report registered before merging. The absence of
  a report is not a blocker — it is evidence of a skipped step. Making the gate bypassable
  would reproduce the exact failure from P28.
- **Consequences:** Agents and developers cannot merge a `reviewing`-stage feature without
  a report. If a feature was reviewed but no report was filed, the operator must file a
  (possibly minimal) report before proceeding. This is a new hard requirement on the merge
  path for `reviewing`-stage features only.

### Decision 2: Gate is fail-open on document service errors

- **Decision:** If `docRecordSvc.List()` returns an error during gate evaluation, the gate
  returns `Pass` and logs a warning, rather than blocking the merge.
- **Context:** The merge tool must remain functional even if the document index is
  temporarily unavailable or corrupted. A broken merge tool is a more serious operational
  problem than a missed gate check.
- **Rationale:** Fail-open on infrastructure errors is the correct policy for a
  non-safety-critical enforcement point. The dashboard warning provides a second signal.
  A document service error is an unusual condition that should be investigated separately.
- **Consequences:** If the document index is corrupted or unavailable, the gate is
  effectively disabled. This is an acceptable risk given the low probability of the
  condition and the availability of the dashboard warning as a backup signal.

### Decision 3: Dashboard check is a warning, not an error

- **Decision:** The `OrphanedReviewingFeatureCheck` emits `severity: warning`, not
  `severity: error`.
- **Context:** An entity in `reviewing` with no report may be mid-review (the review is
  in progress but the report has not been filed yet). Emitting an error would create
  false-positive noise for active reviews.
- **Rationale:** A warning is appropriate for a condition that may be transient (review
  in progress) or may indicate a problem (review bypassed). An error severity is reserved
  for conditions that definitely indicate a broken state. The gate provides the hard
  enforcement at merge time; the dashboard provides the soft signal during normal
  operation.
- **Consequences:** Operators must notice warnings and investigate. In practice, a
  `reviewing` feature with no report that has been in that state for more than one session
  is almost certainly an orphan, not an active review. A future enhancement could escalate
  the severity based on age.

### Decision 4: `Bypassable bool` field added to gate result type

- **Decision:** The `gate.Result` type gains a `Bypassable bool` field. The merge tool
  checks this field when `override: true` is set. Existing gate conditions default to
  `Bypassable: true` (preserving current behaviour). Only `ReviewReportExistsGate` sets
  `Bypassable: false`.
- **Context:** The override mechanism (P17 §3.4) is used correctly in many legitimate
  scenarios. Removing override entirely would break existing workflows. A per-gate
  bypassability flag is the minimum change required to distinguish this gate from others.
- **Rationale:** This is the narrowest possible change to the gate type that achieves the
  required distinction. It is backward-compatible: all existing gate conditions continue
  to work as before. The flag is opt-in for non-bypassability.
- **Consequences:** Future gates can also set `Bypassable: false` when warranted. The
  `Bypassable` field should be documented in the gate interface as the standard mechanism
  for non-overridable conditions.