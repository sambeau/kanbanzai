# Context Compaction for AI Agent Orchestration: Summary & Implementation Guide

## For Senior Product Managers

### The problem

Kanbanzai's orchestrator agent manages sub-agents through structured workflow stages. Over a long session, the orchestrator's context window fills up — conversation history, tool outputs, and reasoning chains accumulate until the model runs out of working memory. When that happens, the agent's performance degrades: it loses track of what's been done, makes inconsistent decisions, or the session simply crashes.

### The question we needed to answer

When the orchestrator's context fills up, we need to compact it — extract the essential state and hand it to a fresh session so work can continue. There are two ways to do this:

1. **Summary-based:** Write a prose narrative of "what happened so far" (like meeting minutes for a human)
2. **State-based:** Write a structured snapshot of "where to resume from" (like a program counter + register state for a CPU)

Which one produces better outcomes for continued work? And does the research support a specific structure for this handoff artefact?

### The answer in one slide

**Build state-based, not summary-based.** When an agent hands off to itself in a fresh session, it doesn't need to know the journey — it needs to know exactly where to continue. A structured artefact that captures active tasks, pending decisions, and references to stored knowledge outperforms a prose summary. Position the most critical information at the beginning and end (because models pay most attention there). Keep it short — every token in the handoff is a token not available for actual work.

### What this means for the roadmap

- **Today (no new infrastructure):** We can build the compaction template and wire it into the orchestration procedure. The orchestrator manually triggers compaction when it notices context pressure.
- **After model routing (P41):** Automated triggers — the system detects when context is 60% full (warning) and 80% full (compact now) using token counts from the API.
- **Deferred:** Training custom models to do their own compaction. Research shows this works well (Microsoft's Memento: 2-3× memory reduction) but requires infrastructure we don't have yet.

### What we're not doing

We are not building a summarisation pipeline. We are not training custom models (yet). We are not building complex heuristic scoring for "when to compact." The evidence supports a simpler approach that works within our current architecture.

---

## For Software Architects & Engineers

### The core decision: state-based vs. summary-based compaction

The research report evaluated four approaches to context compaction across seven dimensions. Here's the comparison that matters:

| | State-Based (U-shaped) | Summary-Based | Learnable (Memento) | Retrieval-Anchored |
|---|---|---|---|---|
| **Continuation accuracy** | Medium-High | Low-Medium | High | Medium-High |
| **Token efficiency** | High (5-10× compression) | Medium (2-3×) | Very High (6×) | Very High |
| **Implementation complexity** | Low | Very Low | Very High | Medium |
| **Model-agnostic** | Yes | Yes | No (per-model SFT) | Yes |
| **Can we build it now?** | Yes, procedurally | Yes | No | Yes, partially |

**The recommendation is state-based compaction with retrieval anchoring** — the two rightmost columns that we can implement today — combined.

### Why summary-based loses

Prose summaries have three failure modes that make them unsuitable for agent handoff:

1. **Detail erosion.** Summaries flatten distinctions. "Reviewed three tasks, two passed, one needs rework" loses which tasks, what the rework is, and why. The fresh session has to rediscover this through tool calls.
2. **Recency bias.** Summaries overweight the last few turns of conversation. If the most recent activity was fixing a minor bug, the summary over-represents that and under-represents the architectural decision made 30 turns ago that constrains all subsequent work.
3. **No structured retrieval path.** A prose summary says "we consulted knowledge entry KE-01KXYZ about error handling." A state-based artefact says `KE-01KXYZ` as a machine-readable reference the fresh session can pass directly to `knowledge(action: "get")`.

### The U-shaped compaction artefact template

This is the concrete output. When the orchestrator compacts, it produces this:

```markdown
# Compaction Artefact: [Feature/Batch ID] — [Timestamp]

## Identity & Routing
[Brief: role, active feature, current phase — kept under 50 tokens total]
Vocabulary: [15-20 domain terms active in this session]

## Active Constraints
- NEVER [X] BECAUSE [Y]
- ALWAYS [A] BECAUSE [B]
- File ownership boundaries: [file → task mapping]
- Dependency locks: [task A blocks task B until …]

## Active State

### Done (since last compaction or session start)
| Task ID | Summary | Verdict |
|---------|---------|---------|
| TASK-042 | Implement gate validation | Passed |
| TASK-043 | Fix edge case in status tool | Passed |

### In Flight
| Task ID | Status | Current blocker |
|---------|--------|-----------------|
| TASK-044 | Active (sub-agent running) | Waiting for completion |

### Ready (next dispatch candidates)
| Task ID | Priority | Depends on |
|---------|----------|------------|
| TASK-045 | P1 | TASK-044 |
| TASK-046 | P2 | — |

## Active Decisions
[Only decisions still constraining current work. Not historical decisions whose effects are already captured in code or knowledge entries.]

- DEC-007: Use hash-anchored edit validation pattern → applies to all file modification tools
- DEC-012: Sub-agent parallelism capped at 4 for this batch → file ownership constraints

## Surfaced Knowledge
[KE-IDs to query in the fresh session. The fresh session calls knowledge(action: "get") for each.]

| KE-ID | Topic | Why relevant |
|-------|-------|-------------|
| KE-01KN5CXMBWSXE | edit_file worktree limitation | Affects TASK-045 implementation |
| KE-01KQ7TKTJ7YVB | Shell escaping in worktrees | Affects TASK-046 implementation |

## Continuation Anchor
Resume from: Phase 3 (Task Dispatch)
Next action: Check TASK-044 status → if done, dispatch TASK-045 + TASK-046 in parallel
Expected effort: 3-5 tool calls before first dispatch
```

### Why this ordering?

The sections are ordered for the U-shaped attention curve (Liu et al. 2024, TACL — primary evidence):

| Position | Attention | Section | Rationale |
|----------|-----------|---------|-----------|
| **Top** | **High** | Identity + Vocabulary | Routing signal — determines which knowledge clusters activate |
| **Near top** | **High** | Active Constraints | Hard rules must survive; peak attention ensures compliance |
| **Middle** | Lower | Active State | Tabular data survives attention degradation better than prose |
| **Near bottom** | Rising | Active Decisions + Surfaced Knowledge | Recency bias helps recall |
| **Bottom** | **High** | Continuation Anchor | End-of-context attention peak; this is what the agent acts on first |

### What gets discarded — and why

The compaction artefact explicitly **omits**:

| Discarded | Why it's safe to drop |
|-----------|----------------------|
| Task completion details | The task's `done` status + verdict captures the outcome. Implementation details don't constrain future work. |
| Historical reasoning chains | If the reasoning led to a decision, the decision is preserved. If it led to knowledge, the KE-ID is preserved. The chain itself is dead weight. |
| Conversation structure | "You asked me to…", "I then responded…" — this is scaffolding for the old session, not useful state for the new one. |
| Raw tool outputs | Once a tool result has been acted on, the raw output is noise. |
| Failed attempts whose conclusions are in knowledge entries | If KE-01KXYZ records "heredoc syntax fails in sh shell; use python3 for file creation in worktrees," the 40 turns of debugging that led to that conclusion don't need to be preserved. |

### Knowledge graph anchoring: the biggest token saver

This is the highest-confidence recommendation in the report. Kanbanzai already has KE-IDs (knowledge entry identifiers). Instead of inlining knowledge content into the compaction artefact, we inline **references** that the fresh session resolves on demand.

Token math:
- **Inlining a knowledge entry:** 200-800 tokens (entry content)
- **Referencing a KE-ID:** ~15 tokens (`| KE-01KN5CXMBWSXE | edit_file worktree limitation | Affects TASK-045 |`)
- **Saving per entry:** 185-785 tokens

For a typical session that surfaces 5-10 knowledge entries, this saves 1,000-8,000 tokens — roughly 10-40% of a 20,000-token compaction budget. Those tokens go to active state and constraints instead.

The fresh session runs `knowledge(action: "get", id: "KE-01KN5CXMBWSXE")` for each reference. This is a single tool call that returns the full entry.

### Compaction trigger strategy

Two-phase approach, usable at different stages of infrastructure maturity:

**Today (procedural — no infrastructure change):**
The orchestrator estimates context pressure at each Phase boundary (after task dispatch, after review synthesis). It checks: "Am I approximately 60%+ through my context window?" If yes, it writes the compaction artefact and instructs the human to start a fresh session. This is already partially described in Kanbanzai's Phase 5 orchestration procedure.

**After model routing (P41 dependency):**
The system tracks actual token counts from API response metadata (`usage.input_tokens`). Triggers:
- **60% utilisation → soft warning.** "Context pressure building. Complete current task dispatch cycle, then prepare for compaction."
- **80% utilisation → hard trigger.** Compaction happens at the next task boundary (never mid-task).
- **90%+ → emergency compaction.** Finish current tool call, compact immediately.

The graduated approach (soft → hard → emergency) is used by both Claude Code and LeRiM. It avoids the failure mode of a single threshold: compacting too early wastes capacity; compacting too late risks mid-task truncation.

### Evaluation: how we'll know if it works

Three metrics, tracked as a compound score:

1. **Task completion rate.** After compaction, does the fresh session complete the work that was in-flight? Binary yes/no + time-to-completion. If compaction regularly causes task failure or significant rework, the artefacts are dropping critical state.

2. **Decision consistency.** Compare the pre-compaction plan (which tasks to dispatch next, which sub-agent roles to assign) against post-compaction actions. High divergence means the fresh session is making different choices — either because it's missing context or because the compaction artefact is ambiguous.

3. **Token efficiency.** `(Tasks completed post-compaction) / (Compaction artefact tokens + retrieval overhead tokens)`. This catches the failure mode where compaction artefacts grow so large they consume more tokens than they save.

**Implementation:** Log pre-compaction "intent" (the orchestrator already states its plan in Phase 2/3). After compaction, compare the fresh session's first dispatch decisions against that intent. Track all three metrics per compaction event; optimise for the compound, not any single one.

The research report found no standardised benchmark for compaction quality. These metrics are our own design, based on evaluation patterns from Anthropic's multi-agent research system and the failure modes identified in the literature.

### What we're deferring: learnable compaction

Microsoft Research's Memento (April 2026) demonstrates that models can be trained to compact their own context — 2-3× memory reduction with accuracy largely preserved. It works by teaching the model to segment its reasoning into blocks, produce dense "mementos" at block boundaries, and reason forward from the compressed state.

This is promising but not our current path. Memento requires:
- Training data generation pipeline (228K annotated traces for their release)
- Per-model SFT fine-tuning (three-stage curriculum, ~30K examples per model)
- Custom inference infrastructure (vLLM fork with block masking)
- RL fine-tuning to close remaining accuracy gaps

All of this assumes we control the inference stack — which we don't, and won't until model routing (P41) is complete. Even then, Memento was tested on math/code reasoning, not agent orchestration. The transfer is plausible but unverified.

**What we can steal from Memento now:** The finding that structured, schema-driven compaction outperforms freeform approaches. Our U-shaped template is essentially Memento's "memento" concept applied at the session level rather than the reasoning-block level — a structured compression of state designed for machine consumption, not human reading.

### Implementation sequence

```
Phase 1 (now, no dependencies)
├── Design the U-shaped compaction template (extend prompt engineering guide patterns)
├── Add "context pressure check" step at each Phase boundary in orchestration procedure
├── Wire entity(state) tool calls to populate Active State section automatically
└── Implement KE-ID resolution step at session start (query all referenced KE-IDs)

Phase 2 (after P41 model routing)
├── Replace manual estimation with token-count-based triggers (60%/80%/90%)
├── Automate compaction artefact generation (orchestrator writes at trigger threshold)
└── Begin logging compaction metrics (task completion, decision consistency, token efficiency)

Phase 3 (evaluation-driven iteration)
├── Review first 20 compaction events against metrics
├── Tune template: which sections grow, which shrink, what's missing
├── A/B test U-shaped ordering vs. flat ordering (head-to-head, same state, different layout)
└── Decide whether to invest in per-model tuning for frequently-used models

Phase 4 (deferred — revisit after Phase 2 is stable)
└── Evaluate Memento/learnable compaction for agent trajectories
```

### Risks and mitigations

| Risk | Severity | Mitigation |
|------|----------|------------|
| U-shaped ordering provides no benefit over flat structure | Low | The cost of ordering is zero (same content, different sequence). A/B test in Phase 3 to confirm. |
| Fresh session fails to query referenced KE-IDs | Medium | Add mandatory step at session start: "Query all KE-IDs in compaction artefact before any other work." |
| Compaction artefact grows too large (defeats the purpose) | Medium | Hard cap: artefact must be under 25% of context window. Token budgets per section. Enforced in template. |
| Procedural triggers miss context exhaustion (orchestrator doesn't notice until too late) | Medium | Add context check at every Phase boundary, not just after task completion. If the orchestrator crashes, the compaction artefact is lost — the human restarts from the last saved artefact. |
| Cross-model handoff degrades quality (artefact written by Claude Opus, read by a different model) | Unknown | Start with same-model handoff (Claude → Claude). Test cross-model only after same-model is stable. |

### Key references (for further reading)

| What to read | Why |
|-------------|-----|
| `refs/prompt-engineering-guide.md` | The U-shaped template pattern we're extending. Already in the repo. |
| `work/research/context-compaction-strategy.md` | The full research report — 20+ sources, detailed methodology, confidence gradings. |
| Anthropic, "Effective Context Engineering for AI Agents" (Sep 2025) | The primary industry reference on compaction strategy. |
| Liu et al., "Lost in the Middle" (TACL 2024) | Why we order sections the way we do. |
| Microsoft Research, "Memento" (Apr 2026) | What learnable compaction looks like — our deferred target. |

---

## One-page checklist for implementation

- [ ] Define compaction artefact template (extend from prompt engineering guide)
- [ ] Add `knowledge(action: "get")` resolution step to session start procedure
- [ ] Add "context pressure check" at each Phase boundary in `orchestrate-development` skill
- [ ] Wire `entity(action: "get")` and `status()` tool calls to auto-populate Active State section
- [ ] Hard cap: artefact ≤ 25% of context window; token budgets per section
- [ ] Log pre-compaction intent + post-compaction decisions for consistency metric
- [ ] Run first 20 compaction events with human review before enabling automated triggers
- [ ] A/B test U-shaped vs. flat ordering (Phase 3)
- [ ] Revisit Memento/learnable approaches after model routing is stable (Phase 4)
