# Orchestration & Fast-Track Pipeline: Consolidated Roadmap

**Audience:** Head of Product, Senior Architects, Development Team  
**Date:** 2026-05-06  
**Source:** [Integrated Architectural Assessment](P41-assessment-orchestration-architecture-cross-reference.md)  
**Status:** Active — this roadmap reflects the assessment's findings applied to all plans

---

## Overview

This roadmap consolidates the six plans (P51, P52, P54, P44, P43, and the P41 strategy report) into a single execution sequence aligned with the Three Horizons strategy from the [Context Rot & Fast-Track Strategy Report](P41-report-context-rot-and-fast-track-strategy.md). It reflects all changes applied from the [Architectural Assessment](P41-assessment-orchestration-architecture-cross-reference.md) (May 2026).

---

## Horizon 0: Fix Fast-Track Now (This Week)

**Goal:** Eliminate the P50 failures — no implicit gates, no ghost work discovered mid-implementation, sub-agents receiving correct implementer context.

| # | Plan | What | Effort | Depends on | Status |
|---|---|---|---|---|---|
| 1 | **P51** | Handoff pipeline unification + sub-agent role routing fix + context-budget recalibration + `finish` limit documentation | ~1 day | — | Shaping |
| 2 | **P52** | Fast-track behavioural profile (3-phase: Session-Start Audit → Dispatch → Close-Out) with no-stop contract, P44 dispatch replacement note, procedural compaction trigger | ~1 day | P51 (companion) | Shaping |
| 3 | — | **Test:** Run P50 feature set with P51 + P52. Verify no implicit gates, no ghost work, correct sub-agent prompts | ~1 day | P51, P52 | Not started |

### P51 Deliverables (see [P51 Design](../P51-handoff-pipeline-unification/P51-design-handoff-pipeline-unification.md))

- Remove legacy 2.0 fallback from `handoff`
- Fix sub-agent role routing default (`implementer-go` not `orchestrator`)
- Recalibrate `DefaultContextWindowTokens` → 1,000,000 (Goal 6)
- Raise `assemblyDefaultBudget` (Goal 6)
- Document `finish` 500-char limit (Goal 7)
- Add topic-level detail to `trimmed` metadata
- 20 consecutive verified `handoff` calls across feature types (prerequisite for P44)

### P52 Deliverables (see [P52 Design](../P52-fast-track-orchestration/P52-design-fast-track-orchestration.md))

- Add fast-track behavioural profile to `orchestrate-development` skill
- No-stop contract preamble
- Phase 0: Session Start Audit (status, ghost-work, dirty-state, server_info)
- Phase 1: Dispatch with explicit `role: "implementer-go"` (P44 dispatch_task note)
- Phase 2: Close-Out with procedural compaction trigger
- Four anti-patterns: Implicit Gate, Ghost Work Discovery, State Ambiguity Drift, Milestone Pause
- Track implicit-gate frequency as metric during first 5 runs

---

## Horizon 1: Build the Strategic Architecture (This Quarter)

**Goal:** Deliver the model routing dispatch architecture that addresses context rot at the architectural level. Close the review-remediation gap.

| # | Plan | What | Effort | Depends on | Status |
|---|---|---|---|---|---|
| 4 | **P54** (doc-driven) | Review remediation workflow — documented workflow for failed review → remediation dev-plan → tasks → verification → re-review report | ~2 days | P51 | Shaping |
| 5 | **P44 Phase 1** | `dispatch_task` MCP tool + Anthropic & DeepSeek providers + 3 categories + session-scoped context + pipeline health assertions + debug mode + token-budget communication + production acceptance gate (20-run audit) | 4–6 weeks | P51 (tested), P43 (stability gate) | Feasibility design |
| 6 | **P54** (automated) | Automated finding extraction and remediation dev-plan generation via `dispatch_task(category: "deep-reasoning", action: "remediate")` | ~1 week | P44 Phase 1 | Not started |
| 7 | — | Procedural compaction deployment: U-shaped artefact template, KE-ID resolution, context pressure check in orchestration procedure | ~1 week | P44 Phase 1 (for template) | Not started |

### P44 Phase 1 Build Gate

P43 validators must meet stability threshold before P44 Phase 1 begins:
- spec-validator: ≥5 real spec documents, no false-positive blocking failures
- plan-validator: ≥5 real dev-plan documents, no false-positive blocking failures
- review-gate-validator: ≥5 real review reports, no false-positive blocking failures

### P44 Phase 1 Production Acceptance Gate

P44 must not enter production until:
- 20 consecutive `dispatch_task` calls across different feature types (retro_fix, bug_fix, feature) and categories (deep-reasoning, implementation, review)
- All 20 calls produce correct role/skill/knowledge context verified by human audit using pipeline debug mode

### P44 Phase 2 (after Phase 1 validated)

- OpenAI provider + tertiary fallback chains
- `audit` and `quick` categories
- Token budget enforcement (per-feature caps)
- Provider health checks
- **Context-rot monitoring instrumentation** (goal drift, utilisation, latency) — assigned here per assessment

### P44 Phase 3 (after Phase 2 stable)

- Auto-compaction at threshold with token-count-based graduated triggers
- True Ralph Loop (continuous execution with auto-compaction)
- Compaction evaluation: first 10 compactions with human oversight; procedural triggers validated before automated triggers
- Strict mode tool calling (DeepSeek Beta)

---

## Horizon 2: Validate and Expand (Ongoing)

**Goal:** Expand validator coverage, tune thresholds, monitor for context rot in production.

| # | Plan | What | Effort | Depends on | Status |
|---|---|---|---|---|---|
| 8 | **P43** | Expand validator coverage: design validator, threshold tuning, effectiveness metrics | Ongoing | — | Built (expanding) |
| 9 | **P44 Phase 2** | Context-rot monitoring instrumentation (goal drift, utilisation, latency) | Part of Phase 2 | P44 Phase 1 | Not started |
| 10 | — | Context-rot monitoring dashboard | TBD | P44 Phase 2 instrumentation | Not started |

---

## Capability Ownership Map

| Capability | Owner | Horizon | Notes |
|---|---|---|---|
| Pipeline correctness (single path, role routing) | P51 | 0 | Prerequisite for P44 |
| Fast-track orchestrator behaviour | P52 | 0 | Behavioural profile; dispatch mechanism changes in P44 |
| Context-budget recalibration | P51 (Goal 6) | 0 | Five-minute config change; assigned per assessment |
| `finish` summary limit documentation | P51 (Goal 7) | 0 | One-line tool description change |
| Review remediation workflow (document-driven) | P54 | 1 | Ships after P51 |
| Model routing dispatch architecture | P44 | 1 | Strategic architecture decision |
| Review remediation workflow (automated) | P54 | 1 | Gates on P44 Phase 1 |
| Automated validators (quality gates) | P43 | 2 | Already built; expanding |
| Context-rot monitoring instrumentation | P44 Phase 2 | 2 | Assigned per assessment |
| Token-budget communication to agents | P44 Phase 1 | 1 | Phase 1 AC, not deferred |
| Compaction system (procedural triggers) | P52 Phase 2 + P44 Phase 3a | 0/1 | Procedural triggers now; automated triggers in P44 Phase 3b |
| Compaction system (automated triggers) | P44 Phase 3b | 1 | After token tracking validated |
| Pipeline health assertions & debug mode | P44 Phase 1 | 1 | Mitigation for Risk 1 |
| Implicit-gate metric tracking | P52 (OQ 4) | 0 | Tracked during first 5 fast-track runs |

---

## Key Risks Tracked

| Risk | Severity | Owner | Mitigation Status |
|---|---|---|---|
| Pipeline becomes invisible (silent failure) | **High** | P44, P51 | P51 testing gate + P44 health assertions + debug mode + 20-run acceptance gate — designed, not yet implemented |
| P52 profile insufficient against implicit gates | Medium | P52, P44 | Metric tracking + P44 automation of mechanical audit checks — designed |
| Compaction artefact quality unproven | Medium | P44 | First 10 compactions with human oversight + dry-run mode — designed |
| Cross-model compaction quality unknown | Low | P44 Phase 3 | Deferred to Phase 3 |
| Large features still need compaction with dispatch-loop | Low | P44 Phase 3a | Procedural triggers available today |

---

## Sequencing Diagram

```
This Week (Horizon 0):     P51 ──┬── P52 ── Test
                                 │
This Quarter (Horizon 1):        ├── P54 (doc-driven)
                                 │
                                 ├── P43 stability gate check ── P44 Phase 1
                                 │                                    │
                                 │                                    ├── P54 (automated)
                                 │                                    ├── Procedural compaction
                                 │                                    └── P44 Phase 2 (monitoring instrumentation)
                                 │
Ongoing (Horizon 2):             ├── P43 expansion (design validator, tuning)
                                 └── P44 Phase 3 (auto-compaction, Ralph Loop)
```

---

## Related Documents

- [Context Rot & Fast-Track Strategy Report](P41-report-context-rot-and-fast-track-strategy.md)
- [Integrated Architectural Assessment](P41-assessment-orchestration-architecture-cross-reference.md)
- [P51 Design: Handoff Pipeline Unification](../P51-handoff-pipeline-unification/P51-design-handoff-pipeline-unification.md)
- [P52 Design: Fast-Track Orchestration Profile](../P52-fast-track-orchestration/P52-design-fast-track-orchestration.md)
- [P54 Design: Review Remediation Workflow](../P54-review-remediation-workflow/P54-design-review-remediation-workflow.md)
- [P44 Design: Model Routing & Agent Launcher](../P44-model-routing-agent-launcher/P44-design-model-routing-agent-launcher.md)
- [P43 Design: Fast-Track Architecture](../P43-fast-track-architecture/P43-design-fast-track-architecture.md)
- [P41 Plan: OpenCode Ecosystem Features](P41-design-opencode-ecosystem-features.md)
