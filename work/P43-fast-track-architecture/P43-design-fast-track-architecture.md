# Design: Fast-Track Architecture

**Plan ID:** P43-fast-track-architecture  
**Parent Plan:** [P41: OpenCode Ecosystem Features](../P41-opencode-ecosystem-features/P41-design-opencode-ecosystem-features.md)  
**Status:** Shaping  
**Source:** [P41 Competitive Analysis](../P41-opencode-ecosystem-features/P41-research-competitive-analysis.md) §11

## Overview

Replace mechanical human gates with automated evidence-backed validation. After a design is approved, downstream artifacts (spec, dev-plan) are structurally derivable — a human typing "LGTM" isn't adding value that an automated check can't provide. The fast-track system introduces validator roles that check structural completeness and traceability, risk-tiered automation levels, and an auto-approval pipeline.

This is the most distinctive Kanbanzai innovation from the ecosystem research — no competitor has anything comparable. It uses Kanbanzai's existing strengths (entity hierarchy, document intelligence, spawn_agent) rather than requiring new capabilities.

**Architectural validation:** This design implements the "validation bottleneck" pattern identified by Google Research (Kim & Liu, 2026). Google found that centralized orchestration contained error amplification to 4.4× (vs. 17.2× for independent agents) because "the orchestrator acts as a validation bottleneck" — catching errors at each stage before they propagate. P43's validators at spec, plan, and review gates are exactly this pattern: catch document quality issues at the stage where they're cheapest to fix, preventing the 4.4× cost multiplier of catching them during implementation. See `research-agent-orchestration-research.md` §2.2.

## Goals and Non-Goals

**Goals:**
- Replace mechanical human gates (spec approval, dev-plan approval, review verdict) with automated validation
- Catch document quality issues before implementation — cheaper than catching them during review
- Support risk-tiered automation: retro fixes can skip all gates, critical features keep all gates human
- Validators always run in fresh sessions to avoid context degradation
- Escalate to human only when automation can't resolve an issue (uncertainty, cycle limit, scope drift)
- **Enforce validator verdicts at the state machine level** — blocking failures prevent stage advancement (not just advisory)

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
| D13 | Every task has an actionable description (≥50 words, states what it produces, what inputs it requires, what "done" means beyond the AC) | Blocking |

**Key anti-patterns:** Phantom Traceability, Architectural Second-Guessing, Unverified File References.

**Rationale for D13:** Task description quality is the single strongest predictor of workflow success. Research (Masters et al., 2025) found that "performance gains correlate almost linearly with the quality of the induced task graph" and Anthropic's multi-agent team found that "without detailed task descriptions, agents duplicate work, leave gaps, or fail to find necessary information." A plan that passes D1–D12 can still fail in execution if tasks are too vague for agents to execute independently. D13 ensures every task has sufficient detail: an objective, the inputs it needs, and what "done" means beyond the acceptance criterion line item.

**Validator rubrics:** Checks S5, S7, D7, D10, D13, R2, and R3 rely on LLM classification rather than structural pattern matching. These checks require concrete rubrics — not just pass/fail labels — to produce consistent verdicts across different validator runs. Each LLM-classification check must have:
1. A clear definition of what "pass" means with 2–3 positive examples
2. A clear definition of what "fail" means with 2–3 negative examples
3. A "borderline → escalate" pattern for ambiguous cases

Research (Anthropic, 2024/2025) found that "bad tool descriptions can send agents down completely wrong paths" and that iterating on tool descriptions produced a 40% decrease in task completion time. For validators, the "tool descriptions" are the check rubrics. Rubrics should be tested on 15–20 real Kanbanzai documents before the validation pipeline is built — following Anthropic's evaluation approach where "a set of about 20 queries representing real usage patterns" was sufficient to spot dramatic impacts.

Rubrics are maintained in `work/P43-fast-track-architecture/validator-rubrics/`:

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

### Enforceable Stage Transitions

Validator verdicts are **enforceable**, not advisory. When a validator fails a blocking check, the workflow state machine refuses to advance the feature to the next stage. This implements the research finding that "enforceable constraints beat advisory instructions" (MetaGPT SOPs with verification gates, Masters et al. hard constraints ℋ, Microsoft programmatic gates).

**Transition validator hooks** are added to stage bindings. Before a feature transitions:
- `specifying → dev-planning`: spec-validator must have passed all blocking checks (S1–S5, S10)
- `dev-planning → developing`: plan-validator must have passed all blocking checks (D1–D6, D9, D13)
- `reviewing → done`: review-gate-validator must have passed all blocking checks (R1–R2, R4–R5, R7)

If a blocking check failed, `entity(action: "transition")` returns an error with the validator findings. The feature remains in its current stage until the document is fixed and re-validated.

**Override:** Humans can always override via `entity(action: "transition", override: true, override_reason: "...")`. This is the escape hatch — validators enforce the common case, override handles the exceptions. Override usage is tracked as a metric: if a particular check is consistently overridden, its rubric needs refinement.

**Non-blocking checks** do not prevent stage advancement. They attach findings to the document record and are visible in `status` dashboards. Accumulation of non-blocking findings across multiple features triggers a quality review signal.

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
| `retro_fix` | Auto | Auto | Auto | Conditional | 3 |
| `bug_fix` | Auto | Human | Auto | Auto | 2 |
| `feature` | Human | Auto | Auto | Auto | 2 |
| `critical` | Human | Human | Human | Human | 0 |

**`retro_fix` review gate: conditional on change type.** The `retro_fix` tier automates all gates, but the review gate is conditional:
- **Implementation changes** (any file outside `work/`, `docs/`, `refs/`): A specialist review panel runs, and the review-gate-validator audits it. The validator must pass blocking checks R1–R2, R4–R5, R7 before merge.
- **Documentation-only changes** (`work/`, `docs/`, `refs/` only): The review gate is skipped entirely with an explicit check that no implementation files were modified.

This prevents the gap where a `retro_fix` with code changes would merge with no specialist scrutiny. The review panel provides the code audit; the review-gate-validator ensures the audit was thorough.

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
1. **Clean context:** Validator receives only the document under validation, the parent document, and the validation checklist (with rubrics). Never the conversation that produced the document.
2. **Dispatch, don't continue:** Always via the `dispatch_validator(role, skill, context)` abstraction. This uses `spawn_agent` today and will route through model routing dispatch when P44 is built (see Forward Compatibility below).
3. **Structured output pattern:** Validators produce two artifacts:
   - **Summary → orchestrator:** Verdict (pass/pass_with_notes/fail) + number of blocking/non-blocking findings + evidence score + reference to full report. The orchestrator never holds the full validator output in context.
   - **Full report → document store:** Detailed per-check analysis, evidence citations, uncertain findings. Written to `work/{feature}/reports/{validator}-{timestamp}.md` and registered via `doc(action: "register", type: "report")`.
   This implements the "subagent output to filesystem" pattern validated by Anthropic's multi-agent team: lightweight references to the orchestrator, full detail available for human audit.
4. **Cycle tracking:** `max_auto_cycles` cap. When reached, mandatory human escalation.

### Failure Mode Guards

| Failure Mode | Mitigation |
|-------------|------------|
| Validator deadlock (fix → validate → new issue → loop) | `max_auto_cycles` cap |
| Context saturation (validator in author's session) | Always fresh sessions via `spawn_agent` |
| Validator sycophancy (approves too easily) | "Hallucinated Completeness" anti-pattern; every pass requires evidence |
| Drift accumulation (small allowances compound) | Validators cross-reference upstream, not just immediate parent |
| Orchestrator context bloat | Post-completion summarization; full output in document record |

### Forward Compatibility with Model Routing

P43 is designed to work with `spawn_agent` today and transition to P44's model routing dispatch when available. The validator pipeline does not call `spawn_agent` directly — it calls `dispatch_validator(role, skill, context)`, a thin abstraction that:
- **Today:** delegates to `spawn_agent` with fresh context
- **When P44 is built:** routes through the model routing dispatch loop, which can apply the `audit` category (near-zero temperature, consistency-optimized model) automatically

This abstraction lives in the stage binding configuration, not in validator code. Validators don't know or care how they're dispatched. The abstraction boundary ensures P43 doesn't hardcode the current dispatch mechanism and makes the transition to model routing a configuration change, not a code change.

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

- Uses existing Kanbanzai capabilities: `doc_intel` (traceability checks), `spawn_agent` (fresh sessions), `conflict` (file overlap analysis), `entity` (state machine transitions)
- New: three roles, three skills, validator rubrics, extended structural validators, `transition_validator` hooks in stage bindings, fast-track config schema, auto-approval pipeline, `dispatch_validator` abstraction
- No dependency on P42 (hash-anchored edits) or P44 (model routing) — but includes forward-compatible `dispatch_validator` abstraction for future P44 integration

## Open Questions

1. What's the validator evidence threshold for auto-approval? Run validators alongside human gates for N features; compare findings. If validators catch everything humans catch, the threshold can be lower.
2. Should fast-track tier be per-feature or per-batch? Per-feature gives finer control. The batch-reviewing stage may need independent tier logic.
3. How does fast-track interact with `override`? Humans can always override any gate via `entity(action: "transition", override: true)`. Override usage is tracked as a metric — if a particular check is consistently overridden, its rubric needs refinement.
4. Can a validator's pass/fail decision be appealed? Yes — human override. Validator findings become advisory, not binding. The override metric tracks when this happens.
5. Should validators use a different model than authors/reviewers? Validators are compliance-audit cognitive profile. Research (Masters et al., 2025) shows that audit tasks value consistency over creative depth — GPT-5.4 at near-zero temperature matches this profile better than Claude Opus with extended thinking. Initial implementation uses same model with different temperature and role prompt. When P44 model routing is built, validators route through the `audit` category automatically via the `dispatch_validator` abstraction (see Forward Compatibility above). No code changes required in validators themselves.
