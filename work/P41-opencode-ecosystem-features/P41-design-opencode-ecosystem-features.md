# Plan: OpenCode Ecosystem Features

**Plan ID:** P41-opencode-ecosystem-features  
**Status:** Shaping  
**Source:** [Competitive Analysis: OpenCode Plugin Ecosystem vs. Kanbanzai](../research/competitive-analysis-openagent-ecosystem.md)

## Summary

Adopt the most valuable features from the OpenCode plugin ecosystem (oh-my-openagent, micode, opencode-background-agents, Portal) into Kanbanzai. Three design efforts, two small enhancements, and five deferred items.

## Dependency Structure

```
Plan: "OpenCode ecosystem features" (P41)
│
├── Sub-plan A: Hash-Anchored Edit Tool
│   Standalone. Zero dependencies. Ready to design now.
│   Source: §6.1 of the competitive analysis report
│
├── Sub-plan B: Fast-Track Architecture
│   Standalone. No dependencies on A or C. Can start in parallel with A.
│   Phased: spec validator → plan validator → review gate validator → risk tiers
│   Source: §11 of the competitive analysis report
│   Side effect: deprioritizes web UI (§6.7) if successful
│
├── Sub-plan C: Model Routing & Agent Launcher
│   Start with feasibility design only. Do not commit to build until A and B are stable.
│   Unlocks: auto-compaction (§6.6), thinking-level control, true Ralph Loop (§6.8)
│   Source: §6.5, §7.1 of the competitive analysis report
│
├── Enhancement: Wisdom Forwarding
│   Small, standalone. Modify handoff to auto-surface sibling task knowledge.
│   Source: §6.4
│
└── Enhancement: Elicitation Checklist
    Small, standalone. Adopt Prometheus-style checklist into spec-author skill.
    Source: §6.3
```

## Sub-Plan Sequencing

| Order | Sub-plan | Can start | Depends on | Estimated effort |
|-------|----------|-----------|------------|------------------|
| 1 | A: Hash-Anchored Edits | Immediately | Nothing | Medium (new MCP tool) |
| 2 | Wisdom Forwarding | Immediately | Nothing | Small (handoff enhancement) |
| 3 | Elicitation Checklist | Immediately | Nothing | Small (skill update) |
| 4 | B: Fast-Track Architecture | Immediately (parallel with A) | Nothing | Medium-Large (3 roles, 3 skills, config, pipeline) |
| 5 | C: Feasibility Design | After A and B stable | Nothing (design-only phase) | Small (design document) |
| 6 | C: Implementation | After feasibility approved | C feasibility design | Large (provider integration, agent runtime) |

A and B can proceed in parallel — they touch different parts of the system (edit tools vs. stage gates). The two small enhancements can be done at any point. C is intentionally deferred: start the design to capture thinking while it's fresh, but don't build until A and B prove the pattern.

## Deferred Items

These are intentionally NOT sub-plans. They're captured here so the intention is recorded, but they'll be revisited based on outcomes:

| Item | Deferral condition | Revisit trigger |
|------|-------------------|-----------------|
| Web UI (§6.7) | Deprioritized if Design B eliminates most human gates | After fast-track deployment — measure residual human touchpoints |
| Auto-Compaction (§6.6) | Requires Design C | After model routing is implemented |
| Ralph Loop — full (§6.8) | Requires Design C | After model routing is implemented |
| LSP Integration (§6.9) | Lower priority than edit tool improvements | After Design A is stable |
| Mindmodel Convention Layer (§6.10) | Unclear gap — existing mechanisms may suffice | If convention drift becomes a documented problem |

## Decisions

- **DEC-01KQP-VGR1QM4R:** Web UI deferred pending fast-track outcomes. The primary use cases for a web UI (checkpoint response, document approval) are human gates. If fast-track automates them away, the value proposition shrinks to progress monitoring alone, which `status` already covers.
- **DEC-01KQP-VHJ9DG2B:** Model routing starts with feasibility design, not implementation. Model routing is the largest architectural change. The separate-server alternative should be evaluated before committing. Don't build until A and B are stable.

## Plan Lifecycle

- **Shaping:** The competitive analysis report serves as the shaping artifact — it identifies what to build, what depends on what, and what to defer.
- **Ready:** Advance when sub-plan decomposition is agreed and Design A is approved.
- **Active:** Sub-plans are spawned and closed independently. The parent plan remains active as long as any sub-plan is in flight.
- **Done:** When all sub-plans are closed or explicitly deferred. Deferred items are recorded as decisions, not as incomplete work.

## Related Documents

- [Competitive Analysis: OpenCode Plugin Ecosystem vs. Kanbanzai](../research/competitive-analysis-openagent-ecosystem.md) — the shaping artifact
- [OpenCode Ecosystem Evaluation (Independent)](../_project/research-opencode-ecosystem-evaluation.md) — companion evaluation
- [Prompt Engineering Guide](../../refs/prompt-engineering-guide.md) — referenced in compaction artifact design
