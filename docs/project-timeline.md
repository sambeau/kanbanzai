# Project Timeline

This document records the delivery history of the Kanbanzai project. For current project state, use `status()` or check the Scope Guard section in `AGENTS.md`.

---

## Phase 1: Workflow Kernel

**Status:** Complete

The repository contains design documents, specifications, planning documents, research, and working implementation code. All Phase 1 acceptance criteria are met, all audit bugs (B1–B8) are fixed, and all tests pass with race detector enabled.

**Binding contract:** `work/spec/phase-1-specification.md`

---

## Phase 2a: Entity Model Evolution

**Status:** Complete

All Phase 2a acceptance criteria are met. Entity model evolution, document intelligence, and migration completed.

**Binding contract:** `work/spec/phase-2-specification.md`  
**Progress tracking:** `work/plan/phase-2a-progress.md`

---

## Phase 2b: Context Profiles and Knowledge Management

**Status:** Complete

All Phase 2b acceptance criteria are met. Context profiles, knowledge management, and user identity implemented.

**Binding contract:** `work/spec/phase-2b-specification.md`  
**Progress tracking:** `work/plan/phase-2b-progress.md`

---

## Phase 3: Git Integration

**Status:** Complete

All Phase 3 acceptance criteria (§20.1–§20.12) are met, all audit remediation items (R1–R17) are resolved, automatic worktree creation on task/bug transition is implemented, and all tests pass with race detector enabled.

**Binding contract:** `work/spec/phase-3-specification.md`  
**Progress tracking:** `work/plan/phase-3-progress.md`

---

## Phase 4a: Orchestration Foundation

**Status:** Complete

All Phase 4a acceptance criteria are met: estimation tools, work queue, dispatch/complete task, human checkpoints, and orchestration health dashboard are implemented, and all tests pass with race detector enabled.

**Binding contract:** `work/spec/phase-4a-specification.md`

---

## Phase 4b: Feature Decomposition and Review

**Status:** Complete

All Phase 4b acceptance criteria (§16.1–§16.7) are met: feature decomposition and review, automatic dependency unblocking, worker review with rework lifecycle, conflict domain analysis with work queue integration, vertical slice guidance, incidents and RCA, and Phase 1 document store removal are implemented, and all tests pass with race detector enabled.

**Binding contract:** `work/spec/phase-4b-specification.md`

---

## Kanbanzai 2.0: MCP Tool Surface Redesign

**Status:** Complete

The 2.0 work replaced 97 entity-centric 1.0 MCP tools with 22 workflow-oriented tools in 7 feature groups. It was organised into 11 implementation tracks (A–K):

- **A** — Feature group framework
- **B** — Resource-oriented pattern + side-effect reporting
- **C** — Batch operations
- **D** — `status` dashboard
- **E** — `finish`
- **F** — `next`
- **G** — `handoff`
- **H** — `entity` (consolidated entity CRUD)
- **I** — `doc` (consolidated document operations)
- **J** — Feature group tools: `decompose`, `estimate`, `conflict`, `knowledge`, `profile`, `worktree`, `merge`, `pr`, `branch`, `cleanup`, `doc_intel`, `incident`, `checkpoint`
- **K** — 1.0 tool removal (all legacy tools removed, CLI updated, integration test passing)

The 2.0 MCP server registers exactly 22 tools across 7 groups (`core`, `planning`, `knowledge`, `git`, `documents`, `incidents`, `checkpoints`); group membership is controlled by `mcp.preset` / `mcp.groups` in `.kbz/config.yaml`.

**Binding contract:** `work/spec/kanbanzai-2.0-specification.md`  
**Implementation plan:** `work/plan/P4-kanbanzai-2.0-implementation-plan.md`

---

## P6: Workflow Quality & Code Review

**Status:** Complete

Two phases of work:

**Phase 1** addressed the top friction points from the Kanbanzai 2.0 retrospective:
- **(A) Smart lifecycle transitions** — `advance: true` on the `entity` tool's `transition` action walks a feature through multiple lifecycle stages, checking document prerequisites at each gate; `ValidNextStates` is exposed and all lifecycle error messages now include valid transitions
- **(B) Entity state consistency** — health checks added for terminal features with non-terminal children, early-state features with all-terminal children, and active worktrees whose branch is already merged
- **(C) Entity query and update fixes** — `entity(action: "list")` parent filter now works for both features (via `parent`) and tasks (via `parent_feature`), and `entity(action: "update")` accepts `depends_on` for task entities

**Phase 2** added a full code review workflow:
- **(D)** `reviewing` and `needs-rework` feature lifecycle states with the `developing → done` shortcut removed
- **(E)** Reviewer context profile (`.kbz/context/roles/reviewer.yaml`) and code review SKILL (`.skills/code-review.md`) with per-dimension review guidance and full orchestration procedure
- **(F)** Review orchestration pattern validated end-to-end at single-feature and multi-feature scale, with human checkpoint integration
- **(G)** `AGENTS.md`, `work/bootstrap/bootstrap-workflow.md`, and the quality gates policy updated to reflect the new review gate and SKILL reference

**Design:** `work/design/smart-lifecycle-transitions.md`  
**Plan:** `work/plan/P6-workflow-quality-and-review-plan.md`

---

## P7: Developer Experience

**Status:** Complete

Three coordinated improvements:

- **(A) `server_info` MCP tool** — reports build metadata, binary path, install record, and `in_sync` status; `make install` now writes `.kbz/last-install.yaml` via `kbz install-record write --by makefile`, enabling single-call server currency checks
- **(B) Human-friendly ID display** — entity IDs use break-hyphen format (`FEAT-01J3K-7MXP3RT5`) in all tool responses; `entity_ref` combines display ID and slug for at-a-glance identification; label context shown in parentheses when present
- **(C) Review naming and folder conventions** — review files go in `work/reviews/` with `review-{plan-or-feature}-{slug}.md` naming; `bootstrap-workflow.md` document placement table updated

**Specs:** `work/spec/server-info-tool.md`, `work/spec/human-friendly-id-display.md`, `work/spec/review-naming-and-folder-conventions.md`

---

## P8: decompose Reliability

**Status:** Complete

Two fixes to silent-failure modes in `decompose propose` that caused structurally plausible but wrong task breakdowns when a spec was not ready:

- **(A) AGENTS.md decompose precondition rule** — Stage 5 now requires agents to confirm the spec document record is `approved` and optionally run `index_repository` before calling `decompose propose`
- **(B) Service-level precondition gates** — added to `decompose propose` to detect and surface spec-not-ready conditions rather than proceeding silently

**Design:** `work/design/decompose-reliability.md`  
**Specs:** `work/spec/agents-md-decompose-rule.md`, `work/spec/decompose-precondition-gates.md`

---

## P9: MCP Discoverability and Reliability

**Status:** Complete

Six coordinated improvements to MCP tool usability:

- **(A) Tool annotations** — all 22 tools have all four MCP annotation fields (`readOnlyHint`, `destructiveHint`, `idempotentHint`, `openWorldHint`) set explicitly, with a canary test (`annotations_test.go`) that fails if any future tool is added without them
- **(B) Tool titles** — all 22 tools have human-readable `title` annotations for client UI display
- **(C) Improved tool descriptions** — seven tools (`status`, `entity`, `next`, `finish`, `doc`, `knowledge`, `retro`) have descriptions rewritten to guide agent behaviour: when to use each tool, what it replaces, and available actions
- **(D) Response nudges** — `finish` now emits an informational `nudge` field when a feature completes with no retrospective signals (Nudge 1) or when a task completes with a summary but no knowledge or retrospective contribution (Nudge 2)
- **(E) `doc(action: refresh)`** — new action to recompute a document's content hash in place, optionally demoting an approved document back to draft
- **(F) `doc(action: chain)`** — wires the existing `SupersessionChain()` service method to expose the full version history of a document via MCP

**Design:** `work/design/mcp-discoverability-and-reliability.md`  
**Spec:** `work/spec/mcp-discoverability-and-reliability.md`

---

## P11: Fresh Install Experience

**Status:** Complete

Four coordinated improvements to the out-of-box experience for new kanbanzai projects:

- **(A) MCP server connection** — `kbz init` now writes `.mcp.json` (and `.zed/settings.json` when `.zed/` already exists) with version-aware conflict logic; new `--skip-mcp` flag; `kanbanzai-getting-started` skill updated with a self-identifying description and a Preflight Check section
- **(B) Embedded review skills** — two new skills installed by `kbz init`: `kanbanzai-review` (full code review procedure with five evaluation dimensions, structured output format, and orchestration sequence) and `kanbanzai-plan-review` (plan-level review procedure with scope verification, feature completion checks, spec conformance, and retrospective contribution); `kanbanzai-workflow` updated to describe `reviewing` and `needs-rework` states; `kanbanzai-documents` updated with the new 8-row document type table; doc-currency health checker updated to scan `.agents/skills/kanbanzai-*/SKILL.md` instead of `.skills/*.md`
- **(C) Default context roles** — `kbz init` installs `base.yaml` (scaffold, never overwritten) and `reviewer.yaml` (kanbanzai-managed, version-aware updates) into `.kbz/context/roles/`; new `--skip-roles` flag; `--update-skills` extended to also update managed role files
- **(D) Standard document layout** — `DefaultDocumentRoots()` updated to eight directories (`work/design`, `work/spec`, `work/plan`, `work/dev`, `work/research`, `work/report`, `work/review`, `work/retro`); two new document types (`plan`, `retrospective`) added to the model; `InferDocType()` updated with new cases; `work/README.md` created on init with a directory map for humans and agents

**Design:** `work/design/fresh-install-experience.md`

---

## P10: Review Workflow and Documentation Currency

**Status:** Complete

Four coordinated improvements addressing the review workflow gap and documentation drift identified in post-P9 feedback:

- **(A) Plan-level review SKILL** (`.skills/plan-review.md`) — a 7-step procedure for reviewing completed plans, routing reviewers through `status`, `entity list`, and `health` tools, with structured report format and retro contribution requirements
- **(B) Plan review lifecycle** — plans gain a mandatory `reviewing` state between `active` and `done`, mirroring features; `active → done` shortcut removed, `reviewing → active` rework path added
- **(C) Documentation currency health check** — two-tier checker detecting stale tool names in `.skills/*.md` and `AGENTS.md` (Tier 1), and verifying `done` plans are mentioned in AGENTS.md Scope Guard with all spec documents approved (Tier 2)
- **(D) Plan document naming convention** — `P{N}-` filename prefix for plan documents (P4+), convention documented in `bootstrap-workflow.md`, legacy P1–P3 documents exempt

**Plan:** `work/plan/P10-review-and-doc-currency-plan.md`  
**Specs:** `work/spec/plan-review-lifecycle.md`, `work/spec/doc-currency-health-check.md`
