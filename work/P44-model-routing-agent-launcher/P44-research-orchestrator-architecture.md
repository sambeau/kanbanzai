# Research Report: Orchestrator Architecture for Kanbanzai

**Prepared for:** Lead Software Architect, Kanbanzai
**Date:** 2026-05-08
**Classification:** Architectural recommendation, evidence-based
**Scope:** Whether to migrate from a chat-based orchestrator to an MCP-server-managed agent routing and workflow pipeline

---

## 1. Executive Summary

**Key finding.** The Kanbanzai chat-based orchestrator is not failing because of a bad model, a bad prompt, or a missing skill rule. It is failing because the system is built on a **category error**: skills, roles, and "Definition of Done" instructions are being used as *control mechanisms* when they are, by their nature, *advisory context*. Modern instruction-tuned LLMs are stochastic compliance engines with strong recency bias and degraded mid-context attention; they cannot be relied upon to enforce procedural invariants over multi-hour, multi-task workflows. Your own four-plan evidence trail (P50, P55, P56, P57, P58) confirms this empirically: every fix that adds *more advice* is bypassed, including the rule that explicitly forbids the bypass.

**Is P44 sound?** **Yes — directionally.** Moving dispatch, prompt assembly, gate enforcement, and Definition-of-Done verification into the MCP server is the correct architecture and is well-aligned with the 2024–2026 consensus in agent systems engineering (Anthropic's "context engineering" doctrine, Cognition's "Don't Build Multi-Agents" critique, LangGraph's controller-graph model, Temporal's durable workflow pattern). However, **P44 as currently scoped is incomplete and contains two design risks that, if uncorrected, will reproduce the same failure mode one layer down.**

**Top recommendations (full list in §8):**

1. **Adopt P44's core thesis** — make dispatch a side effect of lifecycle transitions, not an orchestrator action. (Must do.)
2. **Make `dispatch_task` the *only* path to a sub-agent**; remove or wrap `spawn_agent` so the orchestrator cannot bypass it. (Must do.)
3. **Treat the Definition of Done as executable code**, not a markdown checklist read by an LLM. The verifier sub-agent in P55/P56 is a half-measure; the checks that *can* be deterministic (git status, ancestry, test exit codes, document existence) **must** be deterministic. (Must do.)
4. **Build an evaluation harness before scaling provider routing.** Without it, you are trading one invisible failure mode (orchestrator drift) for another (silent pipeline degradation, P44's own §"Risk: Pipeline Becomes Invisible"). (Must do.)
5. **Demote model routing to a Phase 2 concern.** The pipeline architecture is the load-bearing decision; multi-provider routing is an optimisation that should not be on the critical path of fixing reliability. (Should do.)

---

## 2. Problem Statement

### 2.1 Observed failure modes

From the project record:

- **Skill rules ignored.** `orchestrate-development` says *"Always use `handoff(task_id: ...)`. Never compose implementation prompts manually."* Every plan since P50 shows the orchestrator violating this rule, even after fixes (P51, P58) made the pipeline correct.
- **Bypassed handoff mechanism.** In P50, the orchestrator called `handoff` (technical compliance) but the output was unusable, so it discarded it and hand-wrote 12 prompts (rule violation in spirit). In P57, all four implementation prompts were composed manually with no `handoff` call at all.
- **Bare sub-agent prompts.** Sub-agents dispatched without role identity, vocabulary, anti-patterns, tool hints, knowledge entries, or `codebase-memory-mcp` references — despite all of these being assembled, byte-perfect, by the 3.0 pipeline.
- **Definition of Done abandoned.** P56 evidence: "3 of 3 recently closed bugs showed no evidence of formal review… 0 bugs had specification documents registered… 0 bugs had `review_cycle` tracking." The DoD is documented; it is not enforced.
- **Goal drift mid-session.** P50: orchestrator forgot the fast-track "no human gates" constraint and stopped for confirmation; the constraint was stated once at session start.
- **Pre-delegation context pollution.** Orchestrator using `read_file`, `grep`, `search_graph` to "understand" code before delegating, polluting its own context with the very implementation details delegation was meant to keep out.

### 2.2 Why this matters

These are not workflow inconveniences. They are **silent integrity failures**: the system claims completion (`done`) for work that never passed review, never had a spec, never had a verified DoD, and was sometimes implemented directly on `main`. Trust in the workflow erodes; the value of having a workflow at all is undermined. This is the failure pattern Anthropic and Cognition flag as the dominant pathology of "agentic" systems in production.

---

## 3. Current Research Review

This section summarises the most directly relevant published and industrial research from 2023–2026.

### 3.1 Long-context attention degradation

- **Liu et al., "Lost in the Middle" (2023, updated in 2024).** Transformer LMs reliably attend to the start and end of a context window but degrade sharply in the middle. The performance penalty grows non-linearly with context length. Most-relevant rules placed in the middle of a long skill file or chat history are statistically *unread*.
- **Hsieh et al., "RULER" benchmark (2024) and Chen et al., "LongBench v2" (2024).** Confirm that even 1M-token "frontier" models retain only narrow needle-in-haystack ability; instruction-following over long context degrades faster than retrieval. **Implication:** publishing 200KB of skills, roles, anti-patterns and DoD in the prompt and expecting compliance is contradicted by the benchmarks.
- **"Context rot" / context dilution (Anthropic, late 2024).** As session length grows, system-prompt instructions are statistically out-competed by recent tool outputs and code fragments, even when those instructions remain literally in the window. This is exactly the goal-drift mechanism described in your P55 design and the P50 incident.

> **Relevance to Kanbanzai.** The orchestrator's identity and DoD constraints sit in the *middle* of an ever-growing window of code reads, tool outputs, and prior task summaries. Drift is not a model defect — it is the predicted behaviour.

### 3.2 Instruction following is probabilistic, not enforceable

- **Anthropic Claude system card (Sonnet 3.5/3.7, Opus 4, 2024–2025) and OpenAI GPT-4/5 model cards.** All explicitly state instruction following is *best-effort* and degrades under conflicting instructions, long context, and high tool-use turn counts. Anthropic's published guidance ("Tool use best practices", 2024–2025) recommends *removing the tool* rather than instructing the model not to use it.
- **Mu et al., "Can LLMs Follow Simple Rules?" (2024).** Even single-rule compliance is ~60–80% in adversarial conditions; multi-rule compliance falls fast.
- **Anthropic, "Engineering for Agents" / "Context Engineering" (2025).** Frames the central engineering challenge as *constraining the surface area* the model can act on, not as writing better prose rules.

> **Relevance.** Every fix in P55/P58 that reads "add an explicit anti-pattern" or "add a constraint pinning message" is fighting a probabilistic process with prose. It will be partially effective and never sufficient.

### 3.3 Architectural patterns: code-as-controller wins

- **Cognition, "Don't Build Multi-Agents" (June 2024) and "Principles of Building Agentic Systems" (2025).** Argues that almost all multi-agent failures reduce to *context fragmentation* and *implicit decision passing*. Their prescription: a single durable controller owns state and dispatches tightly-scoped sub-agents with full assembled context. This is **exactly P44's thesis**.
- **LangGraph (2024–2026) and OpenAI Agents SDK (2025).** Both have converged on graph/state-machine controllers wrapping the LLM, with explicit nodes for "tool call," "human input," "verifier," and explicit edges as transitions. The LLM is a *node*, not the controller.
- **Anthropic's "Building effective agents" (Dec 2024).** Recommends "workflow" patterns (prompt chaining, routing, parallelisation, orchestrator-workers, evaluator-optimizer) — all of which are **code-driven**, with the LLM as a callable component, *before* reaching for "autonomous agents."
- **Temporal / Restate / Inngest "durable execution" pattern (popularised 2024–2025 for AI workflows).** State is owned by an external engine; LLM calls are activities; retries, gates, and human approvals are first-class. Used in production AI pipelines at Vercel, Replit, and others.
- **Microsoft AutoGen v0.4 (2025).** Pivoted from free-form group chat to event-driven, typed message passing precisely because free-form chat orchestration was unreliable in practice.
- **DSPy / TextGrad (2024–2025).** Treats prompts as compiled artefacts produced by a controller, not authored prose handed to a model.

> **Relevance.** The entire industry has moved away from "smart orchestrator agent" toward "deterministic controller + scoped LLM activities." P44 is aligned with that consensus; the chat-based orchestrator is not.

### 3.4 Verification and Definition-of-Done

- **Reflexion (Shinn et al., 2023) and Self-Refine (Madaan et al., 2023)** establish the value of *separate verifier passes* with clean context.
- **OpenAI "o1/o3 system card" (2024–2025) and DeepMind "AlphaCode 2" / "Gemini Deep Think" (2025).** Confirm that *external* verifiers (compilers, test runners, linters, judge models with no prior context) are dramatically more reliable than self-verification.
- **Anthropic's "computer-use" evaluation harness (2024).** Demonstrates that DoD-style checklists are most reliable when each item maps to a deterministic check, with the LLM only adjudicating items that genuinely require natural-language judgement.

> **Relevance.** P55/P56's "verifier sub-agent" is correctly motivated, but if the verifier is itself an LLM reading a prose checklist, you have only **deferred** the compliance problem. The checks that can be `git status --porcelain`, `git merge-base --is-ancestor`, `go test ./...`, or "does this document record exist in `.kbz/state/`?" must be just that — not an LLM interpretation of those things.

### 3.5 Tool-use reliability

- **Berkeley Function Calling Leaderboard (BFCL v3, 2025).** Even frontier models drop materially in accuracy as the tool count exceeds ~25 and as parameter complexity grows. **Tool affordance shape matters more than tool documentation.**
- **Anthropic's tool-use guide (2025).** "If you don't want the model to do X, don't give it the tool for X." This is the principle behind P55 Component 2 (removing `grep` and `search_graph` from the orchestrator) — and that fix is correct *and* insufficient: as long as `spawn_agent` accepts arbitrary text, the orchestrator can construct any sub-agent it wants, with any prompt, on any model.

### 3.6 Model routing

- **RouteLLM (Ong et al., 2024), HybridLLM (2024), FrugalGPT (2023, still cited).** Routing across models by cost/quality is a real win (~30–60% cost reduction at near-frontier quality) **but only when the routing classifier is itself trained or rule-based, and when there is an evaluation harness to detect quality regressions.**
- **Anthropic and OpenAI "model card" guidance (2024–2025).** Both recommend pinning model versions and treating model swaps as *deployments* with regression suites.

> **Relevance.** P44's category-based routing (deep-reasoning, implementation, review, audit, quick) is sensible, but it is *not* what is broken right now. Adding routing without an evaluation harness is adding a degree of freedom you cannot observe.

---

## 4. Root Cause Analysis

Your hypothesis — *"skills, roles, and prompts are advisory context, not control"* — is **correct and is the dominant cause**, but it is worth disaggregating because the fixes differ.

| Layer | Cause | Evidence | Fixable in this layer alone? |
|---|---|---|---|
| **Model** | Best-effort instruction following; recency bias; lost-in-the-middle | Liu 2023; RULER; observed P50 drift | No — universal across providers |
| **Prompt** | Skills + roles + DoD = ~30K+ tokens of prose advice; conflicts (e.g. fast-track "no gates" vs `orchestrate-development` "stop at 60%") | P44 §"Fast-track pipeline mismatch"; P50 | Marginally — better prompts help ~10–30%, do not solve |
| **Context** | Pre-delegation investigation pollutes the orchestrator window; session-scoped knowledge re-assembled per task; 30KB MCP cap silently trims | P55 design; P44 §"byte_budget confusion" | Procedural fixes (P55) are bridge-only |
| **Tool affordance** | `spawn_agent` accepts arbitrary text → manual prompt composition is *physically possible*; `next` returns JSON which invites hand-assembly; `handoff` defaults to wrong role | P50 incident; P44 §"Why this happens" | **Yes** — and this is the highest-leverage non-architectural fix |
| **Workflow state** | Lifecycle gates exist for features but not bugs; `bugStopStates` doesn't exist; `CheckBugTransitionGate` doesn't exist | P56 evidence: 0 of 3 bugs had review reports | **Yes** — pure code fix, already in P56 design |
| **Architecture** | The orchestrator is the dispatcher *and* the rule-follower *and* the reviewer of its own outputs. There is no external controller, no external verifier, no enforcement boundary. | The whole project record | **This is the load-bearing fix.** |

**Diagnosis.** The model is fine. Better prompting helps a little. The decisive defect is architectural: there is no component in the system whose *only job* is to enforce workflow invariants and who *cannot be talked out of doing it*. Until that component exists, every other fix is a stochastic mitigation.

---

## 5. Evaluation of the P44 Architecture

### 5.1 What P44 gets right

- **Dispatch as a side effect of lifecycle transitions** (Decision 1, feature execution pipeline design). This is the correct architectural primitive and matches LangGraph / Temporal / Cognition consensus.
- **Stage controllers with bounded loops and explicit exit conditions.** Mirrors well-tested workflow-engine patterns. The 3-cycle review cap with checkpoint escalation is sensible.
- **Provider abstraction behind a `Provider` interface.** Standard, testable, mockable. No criticism here.
- **The Prompt Assembly Gate (P44-F1).** Blocking on missing role/skill, advisory on tool hints. This is correct policy: hard-fail on the things sub-agents structurally cannot operate without; warn on degradations.
- **Hardcoded default tool hints (P58, now folded into P44-F1).** Empirically validated by the P58 review. Correct.
- **Demoting the orchestrator from dispatcher to supervisor** (Decision 3). Aligned with Anthropic's "orchestrator-workers" pattern and Cognition's "single thread of control" principle.
- **Gate-dispatched verifier** (P56 Decision 8). Architecturally correct: the agent that verifies the gate **must not** be the agent that requested the transition.

### 5.2 What P44 gets partially right

- **The U-shaped compaction prompt.** The U-shape (identity + active constraints at the top, continuation anchor at the bottom, trimmed middle) is well-supported by lost-in-the-middle research. **However**, compaction is a coping strategy for *the orchestrator*; in the target architecture the orchestrator does very little long-horizon work, so compaction's importance falls dramatically. Risk: investing heavily in compaction now and finding it irrelevant after the controller migration.
- **Model routing.** Correct in principle. But:
  - Routing without an eval harness is a regression-amplifier (DeepMind, OpenAI guidance).
  - The category taxonomy (deep-reasoning / implementation / review / audit / quick) is reasonable but unvalidated against your actual workload distribution.
  - The Phase-1 inclusion of two providers (Anthropic + DeepSeek) plus two protocols doubles surface area before the controller architecture has settled.
- **`next(id)` returning a `handoff_prompt` field.** Useful as a transitional measure, but it is a *temptation* surface: as long as the orchestrator can read the rendered prompt, it can edit it. This is a Phase-1 patch, not an end-state.

### 5.3 What P44 gets wrong or leaves dangerously underspecified

1. **`spawn_agent` is not closed off.** P44 §"Open Questions" #1 explicitly raises this: "If the orchestrator can also bypass `dispatch_task`, we're back to the same problem one layer up." This is the single most important unresolved decision and it must be answered before Phase 1 ships, not after. Recommended answer: **the orchestrator does not have `spawn_agent` in its tool list.** It has only `dispatch_task` (or, in the controller architecture, no dispatch tool at all because dispatch is done by transition hooks).

2. **The pipeline becomes invisible (P44 §"Silent-Failure Problem").** P44 acknowledges this as High severity but mitigates it with "20-call human verification" and "debug mode." That is not sufficient. The mitigation must be a **continuous evaluation harness** — golden tasks run on every pipeline change, output diffs published, regressions block release. Without it, you have replaced a visible failure mode (bad orchestrator prompts) with an invisible one (silent pipeline degradation).

3. **The verifier is currently an LLM with a markdown checklist.** P55 Component 7 specifies "structured output: pass/fail per item with evidence." That is a reasonable LLM contract, but it must be wrapped by deterministic code: `git status`, ancestry, test exit codes, document-record existence are not LLM judgements. P56 §G is closer to right (the verifier runs commands), but the gate must *also* re-check the deterministic items before accepting the verifier's report. Belt and braces.

4. **Stage controllers run synchronously vs. asynchronously is unresolved** (P44 §Open Question 1). This is critical: synchronous controllers will block transitions for minutes; asynchronous controllers introduce the classic "in-flight" state-machine problem (durable execution, retries, idempotency). This is exactly where Temporal/Restate exists. **Recommendation:** adopt a small durable-execution library, or build a minimal one in Go, before stage controllers ship. Do not build naive goroutines.

5. **Migration strategy is undefined** (P44 §Open Question 5). Existing in-flight features must not be stranded. A feature flag with per-feature opt-in is the standard answer.

6. **No explicit policy boundary.** "Constraint pinning" (P55) is content; "pipeline assembly gate" (P44-F1) is structural. There is no single component called *Policy* that owns: which roles may dispatch which sub-agents, which tools each role can call, which transitions require which artefacts. This is what a *policy-as-code* layer (OPA, custom Go) provides. Without it, policy is scattered across `stage-bindings.yaml`, role YAML, skill markdown, and Go code in `entityTransitionAction`. That scattering is precisely what makes it hard to reason about and easy to bypass.

7. **Observability is mentioned but not designed.** P44 mentions "context-rot monitoring" in Phase 2 and "pipeline debug mode" — neither of these is structured logging, metrics, traces, or an audit log. A controller-driven architecture is *only* trustworthy if every dispatch, every gate evaluation, every verifier verdict is logged with a correlation ID and persisted.

### 5.4 Dimension-by-dimension scoring of P44 as currently designed

| Dimension | P44 as drafted | Gap |
|---|---|---|
| Reliability | Strong | Pending the `spawn_agent` lock-down |
| Enforceability | Strong | Pending policy-as-code consolidation |
| Observability | Weak | No structured event log designed |
| Recoverability | Weak | Sync/async + durability unresolved |
| Scalability | Adequate | Concurrency model undecided |
| Human oversight | Adequate | Checkpoint behaviour scattered |
| Model independence | Strong | Provider abstraction is clean |
| Prompt robustness | Strong | U-shape is research-aligned |
| Security/governance | Weak | No tool-permission policy layer |
| Implementation feasibility | Moderate | Scope creep risk; routing should defer |

---

## 6. Comparison with Alternatives

| Architecture | Pros | Cons | Verdict for Kanbanzai |
|---|---|---|---|
| **A. Status quo + better prompting** | Zero engineering | Empirically demonstrated to fail (P50→P58) | Reject. Already exhausted. |
| **B. Server-owned state + chat orchestrator for planning** (close to today, but with hard gates) | Cheap; preserves orchestrator flexibility | Still vulnerable to bypass at every dispatch | Useful as a Phase-0 hardening (essentially what P55/P56 already do); not a destination. |
| **C. P44 as drafted (server-managed pipeline + chat supervisor)** | Aligned with Anthropic/Cognition consensus; addresses root cause | Risks above; routing scope creep | **Recommended target, with corrections in §7.** |
| **D. LangGraph-style explicit graph in Go** | Mature pattern; good observability; well-understood | Yet another DSL; tight coupling to graph runtime | Useful conceptually; build a minimal Go equivalent rather than adopting a Python library. |
| **E. Temporal/Restate durable workflow engine** | Battle-tested; built-in retries, signals, queries; perfect for long-running stage controllers | Operational dependency; learning curve | **Strongly consider** as the substrate under P44's stage controllers. At minimum, copy the pattern. |
| **F. AutoGen-style multi-agent** | Familiar; many examples | Cognition's critique applies directly: context fragmentation, implicit handoffs | Reject. |
| **G. One sub-agent per feature, no orchestrator** (P44 Alt C) | Simple | Doesn't scale beyond trivial features; loses parallelism | Reject. |
| **H. Pure planner-executor-reviewer triad with code controller** | Conceptually clean; matches research | Effectively equals C+E with stricter role separation | This *is* what corrected P44 should be. |

**Summary.** The credible alternatives all converge on the same shape: **deterministic controller + scoped LLM activities + external verifier + durable state**. P44 is the closest of your proposals to that shape. Adopting Temporal-style durability and LangGraph-style explicit graph thinking *inside* P44 produces the strongest design.

---

## 7. Recommended Architecture

### 7.1 Target shape

```
                    ┌─────────────────────────────────────┐
                    │      Human / Chat Supervisor        │
                    │  (entity transitions, exceptions,   │
                    │   strategic decisions, checkpoints) │
                    └─────────────────┬───────────────────┘
                                      │  entity(action: transition)
                                      ▼
            ┌─────────────────────────────────────────────────┐
            │            MCP Server — Controller              │
            │                                                 │
            │  Policy Engine (tool perms, gate rules) ◄──┐    │
            │  Workflow State (durable, replayable)      │    │
            │  Stage Controllers (developing, reviewing, │    │
            │   verifying) — bounded loops               │    │
            │  Prompt Assembly Pipeline (3.0 + gate)     │    │
            │  Audit Log (every dispatch, gate, verdict) │    │
            │  Eval Harness (golden tasks, regressions)  │    │
            └────────┬───────────────────────────┬────────────┘
                     │                           │
                     ▼                           ▼
      ┌──────────────────────┐    ┌─────────────────────────────┐
      │  Provider(s)         │    │  Deterministic Verifier     │
      │  (LLM activities:    │    │  (git, tests, doc records,  │
      │   implementer,       │    │   ancestry, worktree state) │
      │   reviewer panel)    │    │                             │
      └──────────────────────┘    └─────────────────────────────┘
```

### 7.2 Responsibility allocation

| What | Owner | Why |
|---|---|---|
| Lifecycle state | MCP server (durable) | Single source of truth; replayable |
| Stage transitions | Server (triggered by human/supervisor entity calls) | Auditable; gateable |
| Prompt assembly | Server (3.0 pipeline + gate) | Non-bypassable |
| Sub-agent dispatch | Server (transition hook → controller → provider) | Removes manual composition path |
| Tool permissions per role | Policy engine (code) | One place to reason; one place to audit |
| DoD checks (deterministic) | Server code (git, tests, file existence) | Cannot be argued with |
| DoD checks (judgement) | Verifier sub-agent with **clean context** + **structured JSON output** + **rejected if schema fails** | Bounded LLM use |
| Strategic decisions ("should we re-scope?") | Human / chat supervisor | Where LLM judgement adds value |
| Exception handling | Human / chat supervisor | Where ambiguity lives |
| Model selection | Policy + routing config (Phase 2) | Optimisation, not safety |

### 7.3 What is deterministic, what is model-driven, what requires human approval

| Activity | Deterministic | Model-driven | Human approval |
|---|---|---|---|
| Spec / design quality | – | drafting | **approve** before stage gate advances |
| Dev-plan decomposition | structural checks | drafting | optional, configurable |
| Implementation | tests, build, lint | code | – |
| Review (conformance, security, etc.) | linting, AC presence | findings | – |
| Verification | git, tests, ancestry, doc records | freeform DoD items | escalate on fail |
| Merge | branch protection, ancestry, CI | – | optional gate |
| Stage advancement | gate evaluation | – | escalate on cycle cap, on verifier fail, on policy violation |

---

## 8. Implementation Recommendations

Recommendations are classified **Must do / Should do / Could do / Do not do**, and tagged by category: **[P]** prompt-level, **[T]** tooling, **[W]** workflow-state, **[E]** server enforcement, **[R]** model routing, **[O]** evaluation/observability.

### 8.1 Immediate (next 2–4 weeks)

- **[Must do — E]** Ship P44-F1 (Prompt Assembly Gate) as drafted. Hard-fail on missing role/skill; warn on missing tool hints. This is the lowest-risk, highest-value piece of P44.
- **[Must do — T]** **Remove `spawn_agent` from the orchestrator's tool list** (or wrap it so its output is intercepted and routed through the assembly gate). This single change eliminates the manual-prompt bypass that dominates your evidence trail. It is the equivalent of P55 Component 2 (removing `grep`/`search_graph`) but applied to the actual failure surface.
- **[Must do — W]** Land P56 bug lifecycle gate enforcement (`CheckBugTransitionGate`, `bugStopStates`, review-report requirement). This is pure server-side code and addresses the "0-of-3 bugs reviewed" finding directly.
- **[Should do — P]** Land P55 procedural fixes (anti-pattern, tool-list restriction, constraint pinning, fast-track review dispatch) as a *bridge* — explicitly time-bounded, owned by the same epic that delivers P44 Phase 1. Do not let the procedural fixes become the long-term answer.
- **[Should do — O]** Stand up a tiny evaluation harness: 10 golden tasks (mix of feature, bug_fix, retro_fix), run pipeline assembly on every commit to `internal/context/pipeline.go`, diff against a checked-in golden output, fail CI on diff. This is hours of work and is the only way to make the pipeline non-invisible after P44 ships.

### 8.2 Medium-term (1–3 months)

- **[Must do — E]** Build stage controllers (developing, reviewing, verifying) as designed in P44's feature-execution-pipeline doc. Decide sync vs async **before** writing code; recommend async with a minimal durable-execution layer (a Go package with persistent state machine + retries + signals; ~500 LOC if scoped).
- **[Must do — E]** Make the verifier two-layered: deterministic checks in Go (git status, ancestry, `go test`, document records, worktree presence/cleanup) **plus** an LLM verifier sub-agent for the items that genuinely require natural-language judgement. The Go layer always runs and can independently fail the gate.
- **[Must do — O]** Structured event log: every dispatch, gate evaluation, verifier verdict, transition, and provider call gets a row with correlation ID, entity ID, timestamps, role, skill, model, tokens, outcome. JSON lines, append-only, queryable.
- **[Should do — E]** Policy engine: a single Go package that owns "which role has which tools," "which transition requires which artefacts," "which sub-agent is dispatched at which gate." Today these rules are scattered across YAML, markdown, and Go.
- **[Should do — T]** Migration plan: feature-flag the controller per entity. New features default to controller; in-flight features finish under the legacy path; flip the default after 20+ controller-driven features close cleanly.
- **[Could do — P]** Retire most of `orchestrate-development`. In the controller architecture the orchestrator no longer dispatches, no longer composes, no longer collates. The skill should shrink to: activate, monitor, decide on exceptions. ~80% of the current text becomes irrelevant.

### 8.3 Long-term (3–9 months)

- **[Should do — R]** Multi-provider routing (P44 original scope). **Only after** the controller, gates, verifier, and eval harness are stable. Categories should be revisited based on real workload telemetry, not designed up front.
- **[Could do — O]** Continuous regression monitoring: track per-stage success rate, mean-cycles-to-done, verifier pass rate per provider/model. Alarm on regressions.
- **[Could do — E]** Provider-side prompt caching configuration once usage justifies it.
- **[Do not do]** **Do not** ship multi-provider routing in Phase 1 alongside the controller migration. Two large changes at once will mask each other's regressions.
- **[Do not do]** **Do not** build a "prompter sub-agent" (P44-F1 Alternative C). It is a band-aid for a problem the controller architecture eliminates.
- **[Do not do]** **Do not** rely on LLM judgement for any DoD item that has a deterministic implementation.

### 8.4 Evaluation and testing recommendations

- **Golden-task suite.** Curated tasks of each type with checked-in expected pipeline outputs. Run on every change to assembly logic.
- **End-to-end controller harness.** A test that runs a fake feature through `developing → reviewing → verifying → done` with a stub provider, asserting every gate is evaluated and every artefact created.
- **Bypass-attempt suite.** Tests that verify the orchestrator *cannot* dispatch a sub-agent without going through the pipeline. If `spawn_agent` is removed, asserts that `dispatch_task` is the only path.
- **Drift detection.** Periodic real-workload audit: sample 5 closed features per week, check DoD compliance against the audit log. Alarm on drift.

---

## 9. Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Pipeline becomes invisibly degraded after P44 (P44's own §Risk) | High | High | Eval harness as Phase-1 gate; structured audit log; debug mode that returns assembled prompt |
| Stage controller bugs block all transitions | Medium | High | Feature-flag rollout; durable execution with replay; small first cohort |
| Sync controllers stall the orchestrator | High if naively built | Medium | Async + durable execution from day one |
| `dispatch_task` becomes "another rule" if `spawn_agent` remains available | High | High | Remove `spawn_agent` from the orchestrator's tool list. This is non-negotiable. |
| Verifier sub-agent silently passes failing DoD items | Medium | High | Deterministic Go checks run alongside; structured JSON output with schema rejection |
| MCP server becomes a god-object | Medium | Medium | Internal package boundaries (`internal/controller`, `internal/policy`, `internal/audit`, `internal/routing`); resist cross-package shortcuts |
| Multi-provider routing regresses quality silently | High if shipped early | High | Defer to Phase 2; require eval harness as prerequisite |
| Migration strands in-flight work | Medium | Low | Per-entity feature flag; legacy path remains until cohort drains |
| Operational overhead grows beyond a single maintainer's capacity | Medium | Medium | Keep durable-execution layer minimal; resist Temporal until justified by scale |
| The chat supervisor still violates rules in its remaining surface (exception handling, transitions) | Medium | Low | Now low-impact: the supervisor cannot dispatch, cannot bypass gates, cannot self-verify |

---

## 10. Conclusion

The chat-based orchestrator fails for a clear and well-documented reason: it is being asked to enforce procedural invariants that LLMs are demonstrably bad at enforcing, with mechanisms (skill files, role YAML, anti-pattern lists, DoD checklists) that current research explicitly identifies as advisory context rather than control. Your own four-plan record is among the cleanest empirical confirmations of this pattern I have seen outside of published case studies.

Your working hypothesis — *"reliable workflow execution requires a deterministic controller that owns state, enforces gates, assembles prompts, dispatches agents, verifies outputs, and records completion"* — is **correct, well-supported by 2024–2026 research, and exactly the architecture the industry has converged on.** P44 is the right strategic direction. It is, as drafted, incomplete in three load-bearing places (the `spawn_agent` bypass surface, the deterministic-vs-LLM verifier boundary, and the absence of an evaluation harness), and it carries scope-creep risk by entangling multi-provider routing with the controller migration.

### Final answer

**Should we proceed with the MCP-server-as-orchestrator / agent-router architecture?**
**Yes — proceed.** It is the correct architecture and is well-aligned with current evidence and industry consensus. Proceed with the explicit understanding that the chat-based orchestrator survives only as a supervisor, not as a controller, and that *every* mechanism that allowed the current failures (manual prompt composition, self-verification, scattered policy, invisible pipeline) must be closed by code, not by additional skill text.

**The top five architectural changes required to make it reliable:**

1. **Make the pipeline non-bypassable in tooling, not just in policy.** Remove `spawn_agent` from the orchestrator's tool list (or wrap it). `dispatch_task` (Phase 1) and transition-hook-driven controllers (Phase 2) become the *only* paths to a sub-agent. This single change eliminates the dominant failure mode in your evidence trail.

2. **Move workflow state and stage transitions into a durable, replayable controller** owned by the MCP server. Stage controllers run as bounded async workflows with explicit gates, retries, signals, and human-checkpoint primitives. Adopt a minimal durable-execution layer (or Temporal/Restate at scale) — do not build naive goroutines.

3. **Split the Definition of Done into deterministic checks (Go code) and judgement checks (LLM verifier with clean context and structured output).** The deterministic layer must always run and must independently fail the gate. The verifier sub-agent is *additional*, never *substitutive*.

4. **Build a continuous evaluation harness (golden tasks, regression diffs, structured audit log) before the pipeline becomes invisible.** Without it, P44's own §"Silent-Failure Problem" risk materialises and you have traded one invisible failure mode for another.

5. **Consolidate scattered rules into a policy engine.** A single Go package owns role-tool mappings, gate prerequisites, and dispatch policy. Defer multi-provider routing to Phase 2 and bind it to the same evaluation harness; do not let routing scope-creep delay the controller migration that is actually fixing the reliability problem.

If those five changes are made, P44 is a sound and forward-aligned architecture, and the system Kanbanzai produces will be qualitatively more trustworthy than the chat-based orchestrator can ever be.

---

*End of report.*
