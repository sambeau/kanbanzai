# Design: Fast-Track Architecture

**Plan ID:** P43-fast-track-architecture  
**Parent Plan:** [P41: OpenCode Ecosystem Features](../P41-opencode-ecosystem-features/P41-design-opencode-ecosystem-features.md)  
**Status:** Shaping  
**Source:** [P41 Competitive Analysis](../P41-opencode-ecosystem-features/P41-research-competitive-analysis.md) §11

## Overview

Replace mechanical human gates with automated evidence-backed validation. After a design is approved, downstream artifacts (spec, dev-plan) are structurally derivable — a human typing "LGTM" isn't adding value that an automated check can't provide. The fast-track system introduces validator roles that check structural completeness and traceability, risk-tiered automation levels, and an auto-approval pipeline.

This is the most distinctive Kanbanzai innovation from the ecosystem research — no competitor has anything comparable. It uses Kanbanzai's existing strengths (entity hierarchy, document intelligence, spawn_agent) rather than requiring new capabilities.

## Goals and Non-Goals

**Goals:**
- Replace mechanical human gates (spec approval, dev-plan approval, review verdict) with automated validation
- Catch document quality issues before implementation — cheaper than catching them during review
- Support risk-tiered automation: retro fixes can skip all gates, critical features keep all gates human
- Validators always run in fresh sessions to avoid context degradation
- Escalate to human only when automation can't resolve an issue (uncertainty, cycle limit, scope drift)

**Non-Goals:**
- Not replacing the design gate — architectural judgment remains human
- Not evaluating correctness of requirements — only structural completeness and traceability
- Not replacing the specialist review panel — review-gate-validator audits the review process, not the code
- Not removing human override — humans can always bypass any gate

## Design

### Three Validator Roles

#### Spec Validator

**Role:** `spec-validator` (inherits from `base`, NOT from `spec-author`)  
**Identity:** "Senior requirements quality auditor. Verify that specifications are complete, testable, and traceable to their parent design. Do not evaluate whether requirements are *correct* — only whether they are *well-formed* and *complete*."

**Validation checks (10 total, 5 blocking):**

| # | Check | Severity |
|---|-------|----------|
| S1 | All required sections present | Blocking |
| S2 | Overview references parent design | Blocking |
| S3 | Every requirement has unique REQ-ID | Blocking |
| S4 | Every REQ-ID appears in Verification Plan | Blocking |
| S5 | Every acceptance criterion is testable | Blocking |
| S6 | Acceptance criteria use checkbox format | Non-blocking |
| S7 | No requirement is a disguised implementation instruction | Non-blocking |
| S8 | Scope states in-scope AND out-of-scope | Non-blocking |
| S9 | No orphaned requirements (not in design) | Non-blocking |
| S10 | Non-functional requirements have measurable thresholds | Blocking |

**Key anti-patterns:** Assumed Traceability, Content Judgment, Hallucinated Completeness.

#### Plan Validator

**Role:** `plan-validator` (inherits from `base`, NOT from `architect`)  
**Identity:** "Senior implementation plan auditor. Verify that dev-plans are complete, well-decomposed, and fully traceable to their parent specification. Do not evaluate whether task ordering is *optimal* — only whether it is *valid* and *complete*."

**Validation checks (12 total, 6 blocking):**

| # | Check | Severity |
|---|-------|----------|
| D1 | All required sections present | Blocking |
| D2 | Scope references parent specification | Blocking |
| D3 | Every task references a spec requirement | Blocking |
| D4 | Every spec requirement covered by ≥1 task | Blocking |
| D5 | No scope drift (tasks not in spec) | Blocking |
| D6 | Dependency graph is acyclic | Blocking |
| D7 | No monolithic tasks (>3 files or >1 AC) | Non-blocking |
| D8 | Independent tasks marked parallelisable | Non-blocking |
| D9 | Verification maps every AC to producing task | Blocking |
| D10 | Risk assessment is non-empty | Non-blocking |
| D11 | File paths in deliverables exist or noted as new | Non-blocking |
| D12 | Non-functional requirements addressed by tasks | Non-blocking |

**Key anti-patterns:** Phantom Traceability, Architectural Second-Guessing, Unverified File References.

#### Review Gate Validator

**Role:** `review-gate-validator` (inherits from `reviewer`)  
**Identity:** "Senior review quality auditor. Verify that a completed review is thorough, evidence-backed, and suitable for auto-approval. Do not re-review the code — audit the review process itself."

**Validation checks (8 total, 5 blocking):**

| # | Check | Severity |
|---|-------|----------|
| R1 | Every reviewer produced structured output with evidence | Blocking |
| R2 | No rubber-stamp reviews (zero findings, no evidence) | Blocking |
| R3 | No severity inflation (>40% blocking) | Non-blocking |
| R4 | Every blocking finding cites a spec requirement | Blocking |
| R5 | Aggregate verdict consistent with per-dimension outcomes | Blocking |
| R6 | Deduplication pass was run | Non-blocking |
| R7 | Every acceptance criterion covered by ≥1 reviewer | Blocking |
| R8 | Reviewer selection was adaptive to change scope | Non-blocking |

**Key anti-patterns:** Re-Reviewing, Rubber-Stamp Acceptance.

### Risk Tiers

```yaml
fast_track:
  enabled: true
  default_tier: feature
  tiers:
    retro_fix:       # zero human gates
    bug_fix:         # spec human-gated, rest auto
    feature:         # design human-gated, rest auto
    critical:        # all gates human
```

| Tier | Design | Spec | Dev-Plan | Review | Max cycles |
|------|--------|------|----------|--------|------------|
| `retro_fix` | Auto | Auto | Auto | Auto | 3 |
| `bug_fix` | Auto | Human | Auto | Auto | 2 |
| `feature` | Human | Auto | Auto | Auto | 2 |
| `critical` | Human | Human | Human | Human | 0 |

### Tier Inference

When a feature is created, its tier is:
1. Explicitly set on the entity, or
2. Inferred from context: retro signal → `retro_fix`, bug entity → `bug_fix`, tag `critical`/`security` → `critical`, else → `default_tier`

### Validation Pipeline

```
Spec author completes spec
  → doc(action: "register")
  → fast-track check: tier allows auto?
    → YES: spawn_agent(role: spec-validator, skill: validate-spec)
      → pass → doc(action: "approve")  ← automatic
      → pass_with_notes → approve + attach notes
      → fail → escalate to human
    → NO: present to human (current behavior)
```

### Session Management

Validators MUST run in fresh sessions:
1. **Clean context:** Validator receives only the document under validation, the parent document, and the validation checklist. Never the conversation that produced the document.
2. **Spawn, don't continue:** Always via `spawn_agent` with fresh context.
3. **Output reduction:** Validator output reduces to verdict + N findings + evidence score. Full output offloaded to document record.
4. **Cycle tracking:** `max_auto_cycles` cap. When reached, mandatory human escalation.

### Failure Mode Guards

| Failure Mode | Mitigation |
|-------------|------------|
| Validator deadlock (fix → validate → new issue → loop) | `max_auto_cycles` cap |
| Context saturation (validator in author's session) | Always fresh sessions via `spawn_agent` |
| Validator sycophancy (approves too easily) | "Hallucinated Completeness" anti-pattern; every pass requires evidence |
| Drift accumulation (small allowances compound) | Validators cross-reference upstream, not just immediate parent |
| Orchestrator context bloat | Post-completion summarization; full output in document record |

## Alternatives Considered

### Per-feature vs. per-batch tiering

**Per-feature:** Finer control. A batch can contain one critical feature and three standard features. Each feature gets its own tier.

**Per-batch:** Simpler. One tier for the whole batch. But mixed-risk batches force either over-gating (critical tier for everything) or under-gating (feature tier for critical work).

**Decision:** Per-feature tiering. The batch-reviewing stage may need independent tier logic — that's an open question.

### Validator inheritance

**Inherit from reviewer:** spec-validator and plan-validator would share reviewer vocabulary and anti-patterns. But reviewers evaluate code; validators evaluate documents. Different identity, different cognitive profile.

**Inherit from base:** Clean identity. Validators are auditors, not reviewers.

**Decision:** spec-validator and plan-validator inherit from `base`. review-gate-validator inherits from `reviewer` — it audits review quality, which IS a kind of review.

## Dependencies

- Uses existing Kanbanzai capabilities: `doc_intel` (traceability checks), `spawn_agent` (fresh sessions), `conflict` (file overlap analysis)
- New: three roles, three skills, extended structural validators, fast-track config schema, auto-approval pipeline in stage bindings
- No dependency on P42 (hash-anchored edits) or P43-C (model routing)

## Open Questions

1. What's the validator evidence threshold for auto-approval? Run validators alongside human gates for N features; compare findings. If validators catch everything humans catch, the threshold can be lower.
2. Should fast-track tier be per-feature or per-batch? Per-feature gives finer control. The batch-reviewing stage may need independent tier logic.
3. How does fast-track interact with `override`? Humans can always override any gate. Fast-track reduces the need for overrides — the escape hatch remains.
4. Can a validator's pass/fail decision be appealed? Yes — human override. Validator findings become advisory, not binding.
5. Should validators use a different model than authors/reviewers? Validators are compliance-audit cognitive profile. If model routing (P43-C) is adopted, validators would be a distinct category. Until then, same model with near-zero temperature and different role prompt.
