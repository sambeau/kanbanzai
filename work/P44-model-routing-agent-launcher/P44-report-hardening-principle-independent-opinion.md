# Report: Independent Opinion on Adopting the Hardening Principle for Kanbanzai

| Field | Value |
|---|---|
| Date | 2026-05-12T13:01:39Z |
| Status | Draft |
| Author | GPT-5.5 |
| Plan | P44-model-routing-agent-launcher |
| Related report | `work/P44-model-routing-agent-launcher/P44-research-hardening-principle.md` |

## Executive Summary

I recommend adopting the mantra as a core Kanbanzai design principle:

> Every fuzzy LLM step that must behave identically every time must eventually be replaced by a deterministic tool.

For Kanbanzai, the practical formulation is:

> Prompts are prototypes. Tools are production infrastructure.

Kanbanzai already points in this direction through MCP tools, schema-validated state, lifecycle gates, worktrees, document records, status dashboards, merge checks, and context assembly. The mantra would make that direction explicit: if a workflow rule matters enough that agents must follow it every time, it should not live only in prose. It should become a tool contract, gate, validator, generated context packet, health check, or deterministic workflow command.

For consumers, the lesson is that repeated agent mistakes should not be fixed indefinitely by stronger prompting. Instructions are useful for discovering and stabilising a workflow; once a step is mechanical and recurring, it should be hardened into a deterministic tool.

## Recommended Doctrine

Kanbanzai should treat LLMs as appropriate for:

- interpreting intent;
- drafting designs, specifications, and code;
- synthesising tradeoffs;
- reviewing where judgment is required;
- choosing which deterministic tool to call next.

Kanbanzai should not rely on LLMs for steps that require identical behaviour every time, including:

- lifecycle transition validity;
- document registration and approval gating;
- task claiming;
- worktree creation;
- context assembly;
- merge safety;
- branch cleanup;
- conflict detection;
- file write atomicity;
- acceptance-criteria parsing;
- review report presence;
- workflow-state consistency;
- role and skill routing.

The system doctrine should become:

1. Humans own intent: priorities, product direction, approvals, and tradeoffs.
2. LLMs own fuzzy reasoning: synthesis, drafting, code generation, and review judgment.
3. Tools own repeatable mechanics: state transitions, validation, routing, gates, writes, merges, and cleanup.
4. Any repeatedly violated instruction becomes a hardening candidate.

## Fundamental Changes Required

### 1. Introduce an explicit hardening lifecycle

Kanbanzai should have a first-class process for converting fuzzy workflow steps into deterministic tools:

1. Observe a rule written in a role, skill, prompt, checklist, or review comment.
2. Detect recurrence: agents forget it, misapply it, or humans repeatedly verify it manually.
3. Classify the step as human judgment, fuzzy LLM reasoning, advisory guidance, or deterministic mechanics.
4. Prototype the behaviour in prose while the workflow is still changing.
5. Harden stable mechanical behaviour into a tool, validator, gate, generated packet, schema, or health check.
6. Remove duplicated prompt prose and replace it with a short reference to the invariant or tool.
7. Measure whether the hardened path reduces failures, overrides, manual recovery, or review friction.

This reframes stronger prompting as a temporary discovery mechanism, not the final enforcement layer.

### 2. Add a canonical invariant catalog

Kanbanzai should generalise the existing direction of high-violation MCP rule invariants into a product-level invariant registry. Each invariant should include:

- stable code;
- owner package or tool;
- severity: advisory, warning, blocking, or non-bypassable;
- enforcement surface: tool, gate, health check, generated context, or role guidance;
- bypass policy;
- deterministic refusal shape;
- test coverage;
- consumer-facing explanation.

Candidate invariants include:

| Invariant | Correct enforcement |
|---|---|
| Do not implement unclaimed tasks | `next(id)` claim requirement and `finish` state checks |
| Do not work under unregistered entities | `next`, `handoff`, `worktree`, and scoped file tools refuse unknown IDs |
| Do not manually compose sub-agent prompts | server-managed `handoff` or future `dispatch_task` path |
| Do not skip approved specification or dev-plan gates | lifecycle transition gates |
| Do not merge without review artifact | merge gate |
| Do not silently partially edit files | atomic edit tool |
| Do not shell-read `.kbz/state` | tool descriptions, task-context warnings, and detectable telemetry where feasible |

The important shift is that every high-value workflow rule should have a machine-enforced home.

### 3. Replace fragile multi-step orchestration with composite deterministic tools

Kanbanzai exposes strong primitives today: `entity`, `doc`, `next`, `handoff`, `worktree`, `finish`, `pr`, `merge`, `status`, and `health`. The next hardening frontier is reducing sequences where the LLM must remember exact ordering.

High-value composite tools would include:

- `dispatch_task`;
- `start_feature_development`;
- `prepare_review`;
- `run_review_panel`;
- `close_out_feature`;
- `advance_with_gates`;
- `repair_workflow_state`;
- `publish_document`;
- `create_feature_from_approved_design`.

The LLM can still decide when to call these tools, but it should not hand-assemble fragile procedural sequences every time.

### 4. Treat document structure as data, not prose

Kanbanzai's documents are human-legible, Git-reviewable sources of persistent memory. Where document content drives workflow, the system should parse and validate it deterministically.

Examples:

| Document step | Fuzzy risk | Hardening target |
|---|---|---|
| Spec acceptance criteria | Agent misses or misparses criteria | deterministic AC parser and diagnostics |
| Dev-plan traceability | Agent creates untraceable tasks | traceability matrix validator |
| Review reports | Agent produces plausible but incomplete review | report schema and required evidence checks |
| Design approval | Human approves prose but system misses decisions | explicit decision records and structured extraction |
| Document registration | Agent forgets registration | unregistered-doc detection and approval refusal |
| Concept tagging | Agent skips enrichment | approval gate and guide suggestions |

The LLM should draft and interpret. The system should parse, validate, hash, register, and gate.

### 5. Make prompt-only rules visible technical debt

Kanbanzai should surface rules that affect correctness but are enforced only by prose. This could become a `health` category such as `prompt_only_invariant`, `advisory_invariant`, or `unhardened_workflow_rule`.

Examples:

| Rule | Current weak surface | Suggested hardening |
|---|---|---|
| Agents must classify documents after registration | skill prose | doc approval gate or registration warning |
| Agents must not dispatch without handoff | role guidance | remove raw dispatch tool or require `dispatch_task` |
| Reviews must cite requirements | review skill | report schema validator |
| Specs must have acceptance criteria | spec skill | spec validator |
| Worktrees must exist before edits | AGENTS/skill prose | scoped write tools refuse without worktree |

This would make Kanbanzai self-improving by showing where it still depends on agent obedience rather than tool contracts.

### 6. Strengthen consumer extension points

Consumers need a way to harden project-specific workflows without changing Kanbanzai core. Kanbanzai should provide clear extension points for:

- project-local scripts exposed as deterministic tools;
- project-local MCP tools;
- health check plugins;
- validation hooks;
- document schema templates;
- test command contracts;
- release workflow contracts;
- migration workflow contracts;
- deploy workflow contracts.

Consumer examples:

| Consumer workflow | Do not leave to LLM | Harden as |
|---|---|---|
| Database migration workflow | applying migrations, naming, rollback files | migration CLI/tool |
| Release process | version bump, changelog, tag, publish | release tool/checklist gate |
| API schema update | OpenAPI generation and client regeneration | deterministic generator |
| Test selection | exact package commands and flags | project test matrix tool |
| Deployment | command construction and environment selection | deploy script with dry-run |
| Docs publishing | paths, frontmatter, link validation | doc publishing tool |
| Feature flag cleanup | scattered reference removal | project-specific analyzer |

Consumer guidance should be simple: if your team writes "agents must always remember to..." more than once, you probably need a tool.

## Highest-Impact Areas of Change

### 1. Agent dispatch and prompt assembly

This is the highest-impact area because mistakes here cascade into every downstream task.

Current risks include manual context assembly, missed role/skill/stage bindings, incomplete sub-agent prompts, excessive context, and parallel-agent conflicts.

The hardening target is a server-owned `dispatch_task` path that combines task claim, worktree verification, conflict checks, role/skill hydration, and prompt assembly. Once this exists, orchestrators should not have direct raw sub-agent dispatch for implementation work.

### 2. Lifecycle gates and transitions

Lifecycle state is Kanbanzai's source of truth. Gates should become more explicit, more comprehensive, and more consistently non-bypassable where appropriate.

High-impact gates include:

- feature cannot develop without approved specification and dev-plan;
- task cannot finish unless active and committed;
- review cannot complete without structured report;
- merge cannot execute without terminal tasks and required artifacts;
- bug close-out cannot skip review or verification requirements;
- overrides require a reason and audit trail;
- some gates are non-bypassable.

### 3. Review evidence and merge readiness

Review contains fuzzy judgment, but the review process should be deterministic.

Keep LLM judgment for code quality, conformance interpretation, security reasoning, and test adequacy. Harden the required dimensions, report schema, severity taxonomy, finding format, requirement references, blocking classification, report registration, and merge-gate consumption of review status.

The ideal pattern is that LLM reviewers produce judgments while deterministic validators decide whether the review artifact is complete enough to satisfy the gate.

### 4. Specification and dev-plan validation

Specs and dev-plans are the bridge from human intent to agent execution. Any ambiguity here multiplies downstream.

Harden:

- required spec sections;
- acceptance criteria format;
- requirement IDs;
- design-to-spec-to-dev-plan-to-task traceability;
- task dependency graph validity;
- coverage of every AC by at least one task;
- task traceability back to requirements;
- absence of unresolved questions at approval;
- absence of open alternatives in binding specifications.

### 5. Document lifecycle and document intelligence

Kanbanzai's document system is a major differentiator, but documents are also a common source of fuzzy failure.

Harden:

- path generation;
- registration;
- content-hash approval;
- drift detection;
- supersession chains;
- classification preparation;
- required concept tagging where retrieval depends on it;
- stale document references;
- tool-name validation in docs;
- generated registry sections.

### 6. File, Git, and worktree safety

Anything that writes files or mutates Git state should be deterministic.

High-impact rules include:

- no entity-scoped edits without a worktree;
- atomic multi-edit behaviour;
- worktree creation before edits;
- branch health checks before merge;
- cleanup after merge;
- no hidden stash usage;
- commit required before `finish`;
- generated files updated together;
- no shell workarounds when a scoped tool exists.

### 7. Observability and retrospective hardening

The hardening principle depends on noticing where fuzzy steps are failing. Kanbanzai should track repeated workarounds, instruction violations, manual recovery, checkpoint categories, overrides, review rework cycles, unregistered documents, orphaned workflow state, stale binary incidents, and health warnings over time.

Every recurring retrospective theme should be classified as one of:

- human judgment;
- fuzzy LLM reasoning;
- deterministic hardening candidate.

The result should feed directly into a hardening backlog.

## What Should Remain Fuzzy

The principle should not become "replace agents with tools." The correct boundary is:

> If two different high-quality answers could both be acceptable, keep it fuzzy. If the same input must produce the same output, harden it.

The following should remain LLM-led or human-led:

- design synthesis;
- requirements drafting;
- ambiguity detection;
- code implementation;
- code review reasoning;
- architecture tradeoffs;
- research synthesis;
- human checkpoint framing.

## Suggested Adoption Plan

### Phase 1: Doctrine and vocabulary

Add the hardening principle to `README.md`, `AGENTS.md`, generated consumer install instructions, relevant workflow skills, and design/spec templates.

Introduce vocabulary for fuzzy step, deterministic step, hardening candidate, prompt-only invariant, tool-enforced invariant, advisory rule, blocking gate, and non-bypassable gate.

### Phase 2: Audit existing workflow rules

Inventory rules from `AGENTS.md`, `.agents/skills/`, `.kbz/skills/`, `.kbz/roles/`, tool descriptions, retrospective signals, health warnings, and repeated user corrections.

Classify each rule as human-owned, fuzzy LLM, advisory, deterministic hardening candidate, or already hardened.

### Phase 3: Harden top recurring failures

Start with rules that are high consequence, frequently violated, and mechanically checkable:

- dispatch only through server-assembled handoff;
- entity existence before work;
- worktree before edits;
- orphaned workflow state before task claim;
- review report before merge;
- spec/dev-plan traceability validation;
- document classification/approval completeness;
- finish/commit consistency.

### Phase 4: Publish a consumer hardening guide

Consumer documentation should include this checklist:

1. Run the workflow manually with agents first.
2. Notice repeated mechanical steps.
3. Ask whether the step should produce the same result every time.
4. If yes, make it a script, CLI, MCP tool, validator, or health check.
5. Keep the LLM as the caller, not the executor.
6. Add tests and logs.
7. Remove long prompt instructions once tool enforcement exists.

## Conclusion

The biggest conceptual shift is that Kanbanzai should stop treating prompt compliance as an acceptable enforcement layer for critical workflow behaviour.

Prompt instructions are useful for orientation, vocabulary, judgment criteria, role shaping, and temporary prototypes. But for critical behaviour, prompt instructions should be considered unhardened infrastructure.

The mature Kanbanzai loop should be:

1. Let the LLM try the workflow.
2. Watch where reliability depends on obedience.
3. Convert those parts into tools.
4. Leave only true judgment with the LLM.
5. Repeat.

This aligns strongly with Kanbanzai's existing direction. The system is already halfway there; adopting the mantra would make that direction explicit, measurable, and teachable to consumers.
