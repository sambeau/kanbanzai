# Post-P9 Feedback Analysis and Recommendations

| Field     | Value                                    |
|-----------|------------------------------------------|
| Type      | Research / Feedback Analysis              |
| Date      | 2026-03-28T15:31:17Z                     |
| Scope     | P6 through P9 (cumulative)               |
| Source    | Human reviewer feedback, retro signals, review artifacts |
| Status    | Draft                                    |

---

## 1. Purpose

This document collates feedback received after the P9 (MCP Discoverability and Reliability) implementation and review cycle, analyses it for recurring themes and systemic patterns, and proposes concrete recommendations for future work. The feedback covers not just P9 in isolation but the cumulative experience of five completed plans (P6–P9 plus Kanbanzai 2.0) reviewed in sequence.

---

## 2. Feedback Sources

| Source | Nature |
|--------|--------|
| Human reviewer commentary (post-P9) | Structured verbal feedback covering what worked, friction points, and change proposals |
| `review-P9-mcp-discoverability.md` | Formal plan-level review (5 minor documentation issues found and fixed) |
| `review-P8-decompose-reliability.md` | Formal plan-level review (pass, 1 minor deviation) |
| `review-P7-developer-experience.md` | Formal plan-level review (2 critical, 1 significant, multiple minor findings) |
| Retrospective signals (KE-01KMS0EE97M2P et al.) | 3 structured retro signals from P6 task completions |

---

## 3. What Worked Well

Three strengths were consistently highlighted across the feedback.

### 3.1 Specification Quality Drives Review Efficiency

> "Having per-feature acceptance criteria tables made the review systematic. I could literally go criterion by criterion: check the code, check the test, move on."

The P9 spec (`work/spec/mcp-discoverability-and-reliability.md`) with its per-feature acceptance criteria tables was singled out as the ideal review experience. This validates the specification-first workflow — when specs are well-structured, review becomes mechanical rather than exploratory. The same pattern held for P8, where the reviewer could verify precondition gates directly against the spec's numbered steps.

**Implication:** The spec format is working. Acceptance criteria tables should be a mandatory spec convention, not just a good habit.

### 3.2 Project Structure Is Navigable

> "I found every implementation file quickly. The `internal/mcp/` package is well-organised — one file per tool, test files colocated, clear naming."

The one-file-per-tool convention in `internal/mcp/`, the service/MCP layer separation, and colocated test files were all praised. This is a direct payoff from architectural decisions made early in the project. No action needed — maintain the convention.

### 3.3 Test Quality Provides Confidence

> "The nudge tests in particular — 9 tests covering every trigger and suppression condition — gave me immediate confidence."

Comprehensive test matrices (nudge tests, annotation canary tests, refresh scenario tests) let the reviewer trust coverage without having to enumerate edge cases independently. The retro signals from P6 corroborated this: the `advance: true` lifecycle transitions "worked perfectly" because the test coverage made the behaviour verifiable.

**Implication:** Table-driven tests with full trigger/suppression matrices are the project standard for any behavioural feature. This is already the convention; it should be documented as an explicit expectation.

---

## 4. Friction Analysis

Five friction points were identified. They cluster into two systemic themes: **the review workflow gap** and **documentation drift**.

### 4.1 Theme: Review Workflow Gap

Three of the five friction points relate to the same root cause — the Kanbanzai workflow system has no structured path for plan-level reviews.

#### 4.1.1 No Plan-Level Review Procedure

> "There's a code-review SKILL but it's oriented toward feature-level development reviews, not 'review an entire completed plan'. I was making up the procedure as I went."

The `.skills/code-review.md` SKILL covers per-feature review during the `reviewing` lifecycle gate. But reviewing a completed plan — checking that AGENTS.md is updated, that Scope Guard mentions it, that spec status is Approved, that all features are done — is a different activity with different inputs and different checklists. No procedure exists for this.

**Impact:** The reviewer improvised effectively, but improvisation doesn't scale and doesn't capture institutional knowledge. Each plan-level review rediscovers the same checklist from scratch.

#### 4.1.2 Reviews Happen Outside the Workflow System

> "I didn't use any Kanbanzai tools. No `status`, no `entity`, no `next`. The workflow system didn't 'pull' me into a structured path."

The reviewer went straight to files and grep, bypassing the workflow system entirely. This means:
- No context assembly (the reviewer had to discover scope manually)
- No nudge system (Nudge 2 would have fired but never got the chance)
- No retro signal contribution (the irony of which the reviewer noted explicitly)
- No structured record of review findings in the entity model

This is the strongest finding in the feedback. P9 built nudges specifically to catch missing retro signals, but because the review happened outside the entity lifecycle, the nudges were structurally unable to fire. The feedback channel the reviewer used — unstructured conversation — is exactly what Kanbanzai is designed to replace.

**Impact:** Critical workflow gap. Reviews are a known lifecycle gate but are not yet first-class workflow participants.

#### 4.1.3 Plan Scope Discovery Requires Detective Work

> "The plan is called `P9-mcp-discoverability` but no files are named with 'P9' in them. I found the scope through `git log` and grepping for 'discoverability'."

A `status(id: "P9-mcp-discoverability")` call would have provided everything instantly — feature list, task status, associated documents, attention items. But the reviewer didn't think to use it because they were in "review mode", not "workflow mode". This is a symptom of friction 4.1.2: when the system doesn't pull you in, you don't think to use it.

**Impact:** Minor in isolation (the information was findable), but symptomatic of the broader review workflow gap.

### 4.2 Theme: Documentation Drift

Two friction points relate to documentation becoming stale without detection.

#### 4.2.1 Stale Tool References Survive Multiple Plans

> "The 1.0 tool names in the SKILL file survived through Kanbanzai 2.0, P6, P7, P8, and P9 without being caught."

The `.skills/code-review.md` file contained references to `doc_record_submit` and `batch_import_documents` — tools that were removed in Kanbanzai 2.0. These stale references persisted through five subsequent plans. Every agent that read the SKILL file during that period received instructions referencing tools that don't exist.

This was caught by a human reviewer, not by any automated check. The P7 review found similar staleness in AGENTS.md (missing P7 project status). The P9 review found AGENTS.md missing P9 and the spec still in Draft status.

**Impact:** Significant. Stale agent-facing documentation silently degrades agent performance. The problem is systematic — every plan completion creates a documentation update obligation that nothing enforces.

#### 4.2.2 No Automated Documentation Currency Check

> "There's no health check or automated lint that catches 'SKILL file references tools that don't exist.'"

The health check system (`internal/health/`) validates entity integrity, lifecycle consistency, and referential integrity. But it has no awareness of documentation content — it doesn't know that `.skills/code-review.md` references tool names, let alone whether those tool names are current. The nudges added in P9 address retro signal contribution, which is a different category of drift.

**Impact:** Without automated detection, documentation staleness is discovered only by human review — which happens infrequently and unpredictably.

---

## 5. Corroborating Evidence

The feedback is not isolated. It corroborates signals from previous work:

| Signal | Source | Corroborates |
|--------|--------|--------------|
| Stale MCP binary wasted verification time; no built-in version check | P6 retro (KE-01KMS0EE97M2P) | Theme 4.2 (drift detection). P7 solved this specific case with `server_info`, but the general pattern persists. |
| P7 review found `.gitignore` gap and incomplete Makefile | `review-P7-developer-experience.md` | Theme 4.2 (documentation/config drift surviving across plans) |
| P8 review found minor spec deviation that wasn't caught during implementation | `review-P8-decompose-reliability.md` | Theme 4.1 (review procedures catch things that implementation misses) |
| Skills Content review found fabricated lifecycles and stale 1.0 tool names | P6 retro (worked-well signal) | Theme 4.2 (stale documentation in agent-facing files) |

---

## 6. Recommendations

Three recommendations, ordered by impact. Each addresses one or both systemic themes.

### R1: Plan-Level Review SKILL

**Addresses:** Theme 4.1 (review workflow gap)
**Effort:** Small (documentation only)
**Priority:** High

Create `.skills/plan-review.md` — a SKILL document that defines the procedure for reviewing a completed plan. Minimum contents:

1. **Entry point:** Call `status(id: "<plan-id>")` to get the full plan dashboard — feature list, task status, associated documents, attention items.
2. **Scope verification:** Confirm all features are in terminal state. Check for any `needs-rework` or `blocked` items.
3. **Spec conformance:** For each feature, read the spec acceptance criteria table and verify against implementation.
4. **Documentation currency:** Check that AGENTS.md Project Status and Scope Guard sections mention the plan. Check that all spec documents are in Approved status.
5. **Cross-cutting checks:** Run `go test -race ./...`. Check `health()` output for new warnings.
6. **Retro contribution:** Before finishing, contribute retro signals via `finish` or `knowledge(action: contribute)`.
7. **Report output:** Write findings to `work/reviews/review-<plan-slug>.md`.

This captures the procedure the reviewer improvised and makes it repeatable. It also naturally routes the reviewer through the workflow system (step 1 uses `status`, step 6 uses `finish` or `knowledge`), which addresses friction 4.1.2.

### R2: Review Tasks as First-Class Workflow Entities

**Addresses:** Theme 4.1 (review workflow gap)
**Effort:** Medium (design + implementation)
**Priority:** Medium

Route plan-level reviews through the entity lifecycle:

- When a plan's implementation is complete, create a review task (or transition the plan to a `reviewing` state).
- The review task is claimable via `next`, which provides context assembly — spec references, file lists, knowledge entries.
- Completing the review task via `finish` triggers nudges for retro signal contribution.
- The review report becomes a document record linked to the plan entity.

This closes the loop that the feedback identified: reviews happen outside the workflow system because there's no entity representing the review. Making the review a task means the workflow system can "pull" the reviewer in, assemble context, and capture signals — exactly as it does for implementation work.

**Design consideration:** This interacts with the existing feature-level `reviewing` lifecycle state. A plan-level review is conceptually different (it reviews the aggregate, not individual features). The design should clarify the relationship — whether plan review subsumes feature review or is additive.

### R3: Documentation Currency Health Check

**Addresses:** Theme 4.2 (documentation drift)
**Effort:** Medium (implementation)
**Priority:** Medium

Add a health check category that detects stale references in agent-facing documentation. Two tiers:

**Tier 1 — Tool name validation (concrete, automatable):**
- Scan `.skills/*.md` and `AGENTS.md` for patterns matching known tool name formats (e.g., backtick-wrapped identifiers that look like tool calls).
- Compare against the registered 2.0 tool set (available from the MCP server's tool registry).
- Flag any referenced tool name that isn't in the registry.

This would have caught the stale `doc_record_submit` and `batch_import_documents` references immediately after Kanbanzai 2.0 shipped.

**Tier 2 — Plan completion documentation checklist (procedural):**
- When a plan transitions to a terminal state, verify that AGENTS.md Project Status mentions it.
- Verify that AGENTS.md Scope Guard lists it as complete.
- Verify that all associated spec documents are in Approved status.

This would have caught the P7, P8, and P9 documentation gaps that were found only by human review.

**Implementation note:** Tier 1 requires access to the tool registry from the health check system, which currently operates on entity state only. This is a new dependency that needs design consideration. Tier 2 is simpler — it operates on entity and document state that the health system already has access to.

---

## 7. Non-Recommendations

Two items from the feedback are noted but not recommended for immediate action.

### Acceptance Criteria Tables as a Mandatory Spec Convention

The feedback praised the P9 spec's per-feature acceptance criteria tables. Making this mandatory is appealing but premature — the current spec format has evolved organically and different features benefit from different structures. Document the pattern as a strong recommendation in the bootstrap workflow rather than enforcing it mechanically.

### Retro Signal Auto-Capture from Unstructured Channels

The feedback noted the irony of providing retro signals in conversation rather than through `finish`. Attempting to auto-capture signals from chat would add significant complexity for marginal benefit. The better fix is R2 (route reviews through the entity lifecycle), which makes the structured channel the natural path rather than trying to capture from the unstructured one.

---

## 8. Summary

| # | Recommendation | Theme | Effort | Priority |
|---|----------------|-------|--------|----------|
| R1 | Plan-level review SKILL | Review workflow gap | Small | High |
| R2 | Review tasks as first-class entities | Review workflow gap | Medium | Medium |
| R3 | Documentation currency health check | Documentation drift | Medium | Medium |

The feedback tells a clear story: the implementation workflow is mature and working well (specs, structure, tests, lifecycle gates), but the review workflow has a structural gap. Reviews happen outside the system because the system doesn't model them. This means the system's own feedback mechanisms (nudges, retro signals, context assembly) can't help during the activity that most needs them.

R1 is the immediate fix — a checklist that routes reviewers through existing tools. R2 is the structural fix — making reviews first-class workflow participants. R3 addresses the secondary theme of documentation drift, which reviews currently catch manually.

None of these recommendations require new phases or architectural changes. R1 is pure documentation. R2 and R3 extend existing systems (entity lifecycle and health checks respectively) in directions they were designed to grow.