## Overview

P28 addresses ten specific gaps surfaced by two post-P27 evidence sources: the doc-intel Layer 3 classification pilot (339 documents, April 2026) and the P27 implementation retrospective. The gaps span three layers — skill documentation, doc-intel tool responses, and workflow infrastructure — and are addressed in three sequential sprints ordered by risk and dependency.

## Goals and Non-Goals

**Goals**
- Document the concise-output instruction and classify atomicity guarantee in the classification skill
- Add a "Resume plan" checklist to the getting-started and workflow skills
- Enrich `doc_intel guide` and `pending` responses with taxonomy, section counts, and suggested classifications
- Reduce the `doc register → classify` workflow from three tool calls to two
- Audit MCP parameter structs for missing `json:` tags
- Add a direct `proposed → active` plan lifecycle transition (conditioned on in-flight features)
- Make `decompose(action: apply)` register an auto-approved skeleton dev-plan document
- Fix `worktree(action: create)` timeouts under large worktree counts

**Non-Goals**
- Changes to the Layer 3 classification data model or storage format
- Server-side enforcement of classification after registration (classification remains a convention)
- Automatic or server-triggered lifecycle transitions based on feature state
- New MCP tool actions beyond the changes described above
- Changes to any skill file not listed in the Dependencies section

| Field  | Value                                              |
|--------|----------------------------------------------------|
| Date   | 2026-04-22                                         |
| Status | approved |
| Author | architect (Claude Sonnet 4.6)                      |
| Plan   | P28-doc-intel-polish-workflow-reliability          |

---

## Related Work

### Prior documents consulted

| Document | Type | Relevance |
|---|---|---|
| `PROJECT/design-entity-structure-and-document-pipeline` | Design | Canonical plan lifecycle state machine. §11.1 defines `proposed → designing → active → done` and the rule that plans do not auto-transition based on Feature state. Directly constrains the lifecycle gap fix. |
| `FEAT-01KN83QN0VAFG/design-lifecycle-integrity` | Design | Gate and override pattern for lifecycle transitions. Decisions D1–D3 establish that gates apply to all terminal transitions, overrides are preserved and logged, and the override path is the correct escape hatch for legitimate exceptions. |
| `FEAT-01KPNNYYXQSYW/specification-doc-intel-batch-classification` | Specification | Defines the existing `classification_nudge` field on `doc register` responses and the `guide → classify` workflow. The nudge already exists; pilot §5.4 is an enhancement to it, not a new feature. |
| `FEAT-01KPTHB61WPT0/specification-doc-intel-corpus-hygiene` | Specification | Already added corpus integrity check and classification-on-registration content to `kanbanzai-getting-started/SKILL.md` and `kanbanzai-documents/SKILL.md`. The skill documentation feature (Sprint 0) must not duplicate this work. |

### Constraining decisions

| Decision | Source | Constraint on this design |
|---|---|---|
| Plan lifecycle: `proposed → designing → active → done` (§11.1) | `design-entity-structure-and-document-pipeline` | A direct `proposed → active` transition would bypass the designing gate. The design must justify any such bypass clearly or adopt a documentation-only workaround. |
| Plans do not auto-transition based on Feature state (§11.1) | `design-entity-structure-and-document-pipeline` | The lifecycle gap is a known deliberate trade-off, not an oversight. The fix must work within this constraint or explicitly supersede it. |
| Override path is preserved and logged (D3) | `design-lifecycle-integrity` | Gate overrides are the sanctioned escape hatch. Any lifecycle fix that removes override logging would conflict with D3. |
| `classification_nudge` is a static string template with ID interpolation only (FR-004, NFR-001) | `spec-doc-intel-batch-classification` | The nudge is currently a short string. Embedding `content_hash` and section outline changes the payload materially; the new design must justify the size impact. |

### Open questions from prior work

The corpus-hygiene spec (FEAT-01KPTHB61WPT0) added corpus integrity content to `kanbanzai-getting-started` but does not add a "resume plan" procedure. The lifecycle gap is therefore unaddressed in prior skill documentation. This design fills that gap.

---

## Problem and Motivation

Two sources of evidence — the doc-intel Layer 3 classification pilot (339 documents, April 2026) and the P27 implementation retrospective — surfaced ten specific gaps in the Kanbanzai system. Left unaddressed, these gaps will recur on every plan that uses the decompose workflow, every cross-session plan resume, and every doc-intel classification operation.

The gaps fall into three categories:

**Skill documentation gaps (no code change required)**

The classification skill file lacks explicit guidance for two patterns discovered in the pilot: the "concise output" instruction that prevents token-budget failures in bulk operations, and the atomicity guarantee that makes `classify` calls safely retriable after a partial-batch failure. Separately, neither `kanbanzai-getting-started` nor `kanbanzai-workflow` documents what an agent must do at the start of a session when resuming a plan that is already mid-lifecycle — leaving four invisible blocking steps to be discovered sequentially under time pressure.

**Doc-intel tool friction (server-side enhancements)**

Three enhancements to the `guide` and `pending` responses would materially reduce the token cost and failure rate of classification operations. Currently, agents have no way to know a document's section count before reading it, forcing trial-and-error batch sizing that produced ~7 failed batches in the pilot. The `guide` response does not include the role taxonomy, so agents working from a cold context window are exposed to invalid-role failures. The `doc register` nudge does not include the `content_hash` or section outline, forcing an extra `guide` round-trip for every document registered. Additionally, the `Classification` struct was found to have only `yaml:` tags, causing silent deserialization failures on `json.Unmarshal`; an audit is needed to check for the same issue in other parameter structs.

**Workflow infrastructure defects (state machine and tool fixes)**

Three defects block legitimate workflows. First, the plan lifecycle state machine has no path from `proposed` directly to `active`, forcing agents resuming a mid-lifecycle plan to step through `designing` and apply a gate override — producing a misleading audit trail. Second, `decompose(action: apply)` creates tasks but does not register a dev-plan document, so every decompose-based feature requires a gate override at the `dev-planning → developing` boundary — affecting every plan that uses the decompose workflow. Third, `worktree(action: create)` times out on repositories with many existing worktrees (~34+), creating a hard blocker at plan start when multiple features need worktrees before dispatch.

If nothing changes: every cross-session plan resume incurs four invisible setup steps; every decompose-based feature needs a manual gate override; bulk classification runs will continue to have ~20% batch failure rates; and worktree creation will be unreliable at plan scale.

---

## Design

The plan is structured as three sequential sprints, ordered by risk and dependency. Each sprint is independently releasable.

### Sprint 0 — Skill documentation updates

Pure markdown changes to three skill files. No code changes, no gate overrides, no deployment required.

**Classify skill: concise-output instruction and atomicity guarantee**

Two additions to the classification skill (`.agents/skills/kanbanzai-documents/SKILL.md` and the bulk-classification procedure):

1. A "Concise output" instruction in the sub-agent prompt template: agents must suppress per-document commentary and report only final counts. This is the single change that eliminated batch failures in Waves 4–6 of the pilot.
2. An explicit atomicity statement: `classify` calls commit to the persistent index as they succeed. After any batch failure, `doc_intel(action: "pending")` is the authoritative ground truth; re-dispatching without checking `pending` first wastes tokens on no-ops.

These additions complement the content already added by FEAT-01KPTHB61WPT0 (corpus hygiene). They must be inserted into the existing section structure, not replace it.

**Getting-started and workflow skills: "Resume plan" checklist**

A new "Resuming an in-flight plan" section added to both `kanbanzai-getting-started/SKILL.md` and `kanbanzai-workflow/SKILL.md`. The checklist covers, in order:

1. Run `git status` and commit any orphaned `.kbz/` working-tree changes.
2. Check the plan's lifecycle state; if `proposed`, step through `designing` (create a lightweight design doc or register an existing one) and then advance — or apply a `proposed → active` override once the lifecycle fix in Sprint 2 lands.
3. For each feature, check whether a dev-plan doc is registered; if not (decompose workflow), apply the gate override — or skip this step once Sprint 2 lands.
4. Confirm a worktree exists for each in-flight feature; create any missing ones, falling back to `terminal` + manual wiring if `worktree(action: create)` times out.

The checklist is framed as a transitional procedure: steps 2 and 3 explicitly note that they become unnecessary once the corresponding Sprint 2 fixes are merged.

### Sprint 1 — Doc-intel tool enhancements

Five changes to the `doc_intel` and `doc` tool implementations. These are additive — no existing response fields are removed or renamed.

**§5.1 Section count in `pending` response**

`doc_intel(action: "pending")` currently returns a flat list of document IDs. Each entry gains a `section_count` integer (already stored in the Layer 1 index — this is a read-only addition to the response struct). This allows agents to right-size batches from a single call.

**§5.2 Taxonomy in `guide` response**

`doc_intel(action: "guide")` gains a `taxonomy` block:

```
"taxonomy": {
  "roles": ["requirement","decision","rationale","constraint","assumption",
            "risk","question","definition","example","alternative","narrative"],
  "confidence": ["high","medium","low"]
}
```

This block is derived from the Go constants in `internal/docint/types.go` at compile time. The `guide` response becomes self-contained: an agent can correctly classify any document using only the information returned by `guide`, without having read the skill file. Future taxonomy changes automatically propagate to `guide` responses.

**§5.3 Suggested classifications in `guide` response**

`guide` gains an optional `suggested_classifications` array for sections where the heading pattern gives an unambiguous high-confidence role assignment (e.g. "Acceptance Criteria" → `requirement`, "Alternatives Considered" → `alternative`, "Glossary" → `definition`). The match is performed server-side against a static heading-pattern table. Only `high`-confidence suggestions are included; sections without a strong heading match are omitted.

Agents treat suggestions as pre-populated defaults: they submit the suggestions directly and override only where the content warrants a different role. This reduces the `read_file` calls needed for well-structured documents.

**§5.4 `content_hash` and outline in register nudge**

The `classification_nudge` returned by `doc(action: "register")` currently contains a short instructional string. It gains two additional fields: `content_hash` (the hash the `classify` call requires) and `outline` (the section tree from Layer 1, identical to what `guide` returns). The `guide` call in the `register → guide → classify` workflow becomes optional when the agent has the document content in context at registration time, shrinking the workflow to `register → classify`.

The nudge remains a separate field alongside the existing `document` object (consistent with FR-004 from the batch-classification spec). The additional fields make the payload larger, but the tradeoff is justified: the `guide` round-trip costs one full MCP call, and the registration path is already context-heavy.

**§5.5 MCP parameter struct JSON tag audit**

The `Classification` and `ConceptIntroEntry` structs were found to have only `yaml:` tags, causing silent deserialization failures when decoded via `json.Unmarshal`. A one-time audit of all MCP parameter structs that are decoded from JSON string parameters (pattern: `req.RequireString` + `json.Unmarshal`) ensures all exported fields have explicit `json:` tags. A Go test is added to assert this invariant for the identified structs, preventing future drift.

### Sprint 2 — Workflow infrastructure fixes

Three targeted fixes to the state machine and tool layer.

**Issue 1: Plan lifecycle `proposed → active`**

A direct `proposed → active` transition is added to the plan state machine. This transition is only valid when at least one of the plan's features is in a post-designing lifecycle state (i.e., the plan demonstrably has in-flight work). If no features are in-flight, the transition is rejected with a descriptive error that directs the agent to use the standard `proposed → designing` path.

This is a deliberate exception to the §11.1 rule that plans always progress through `designing`. The exception is justified because the `designing` state represents intent, not a point-in-time gate: when features are already mid-lifecycle, the design phase has implicitly occurred. The override is logged in the entity's audit trail with a system-generated rationale.

*Alternative considered:* documentation-only workaround (document the step-through-designing procedure). Rejected: the workaround produces misleading override records that appear as legitimate gate violations in audit trails. A first-class transition is cleaner and does not require ongoing skill-file maintenance.

**Issue 2: `decompose` registers dev-plan at apply time**

When `decompose(action: apply)` creates tasks, it also registers a skeleton dev-plan document on the owning feature and marks it auto-approved. The skeleton document records the feature ID, the date of decompose-apply, and a table of the created tasks with their summaries. It is stored at `work/dev-plan/<feature-slug>-decomposed.md` and follows the standard dev-plan document type.

The `dev-planning → developing` gate check finds this document and passes without an override. The document serves as an audit artifact recording that the decompose workflow was used.

*Alternative considered:* a `decomposed: true` flag on the feature that relaxes the gate. Rejected: the flag adds complexity to the gate logic and produces no artifact. The skeleton document approach is simpler, produces a useful record, and does not require a new feature flag.

**Issue 3: Worktree creation timeout**

Three changes:

1. **Profile and fix the root cause.** Investigate whether the timeout originates in `git worktree list` serialisation, `.git/worktrees/*/lock` contention, or the MCP tool's internal timeout. The fix is targeted at the identified cause.
2. **Retry with backoff.** `worktree(action: create)` retries up to three times with 2-second backoff on timeout or lock errors before failing.
3. **Document the fallback.** The `worktree` tool description is updated to explicitly document the `terminal` + manual-wiring fallback pattern so agents have a sanctioned escape hatch when the tool fails.

---

## Alternatives Considered

### Alternative A: Merge Sprint 0 into Sprints 1 and 2

Distribute the skill documentation updates across the two code-change sprints, so each sprint delivers both code and the documentation that covers for it in the interim.

**Rejected.** The skill documentation updates are independently useful and carry zero risk. Blocking them on code changes (which require spec, review, and merge cycles) delays the most immediately valuable fixes. Sprint 0 should be independent.

### Alternative B: Fix the plan lifecycle gap with documentation only

Instead of adding a `proposed → active` state machine transition, document the step-through-designing workaround in the getting-started skill.

**Rejected as the primary fix**, though the workaround is included in Sprint 0 as a transitional measure. A documentation-only fix means the misleading override audit trail persists indefinitely, and every agent must know the workaround. A first-class transition is cleaner. Sprint 0 documents the workaround; Sprint 2 makes it unnecessary.

### Alternative C: Add a `decomposed: true` feature flag instead of a skeleton dev-plan

Set a flag on the feature entity when `decompose(action: apply)` is called, and relax the `dev-planning → developing` gate when the flag is set.

**Rejected.** A flag adds complexity to the gate evaluation logic, is invisible to the document audit trail, and requires a new concept ("decomposed feature") in the data model. A skeleton dev-plan is simpler, consistent with existing patterns, and produces an artifact that correctly records what happened.

### Alternative D: Make the `pending` response include full section outlines

Instead of just a section count (§5.1), return the full section tree in the `pending` response so agents can plan batches with full visibility.

**Rejected.** For a corpus of 339+ documents, returning full outlines in a single `pending` call would produce a very large payload. Section count (or a small/medium/large bucket) is sufficient for batch planning and proportionate to the problem.

### Alternative E: Do all doc-intel enhancements in one feature

Combine §5.1–§5.5 into a single feature rather than splitting into "guide enrichment" and "register workflow".

**Rejected.** §5.1–§5.3 all touch the `guide`/`pending` response path. §5.4–§5.5 touch the `doc register` response and the struct audit, which is a different code path and carries different risk. Keeping them as separate features allows them to be reviewed and merged independently, reducing blast radius.

---

## Decisions

**D1: Sprint ordering is fixed — documentation before tools before infrastructure**

- **Decision:** Sprint 0 (skill docs) must be merged before Sprint 1 begins; Sprint 1 must be merged before Sprint 2 begins.
- **Context:** Sprint 0 documents workarounds for Issues 1 and 2. Sprint 2 fixes those issues. If Sprint 2 merges before Sprint 0, agents operating between sprints have no documented workaround.
- **Rationale:** Each sprint's skill documentation is the safety net for the sprint that follows it. Strict ordering ensures agents always have documented procedures for current system state.
- **Consequences:** Sprint 2 cannot be parallelised with Sprint 1. Total calendar time is longer than a parallel approach, but operational risk is lower.

**D2: `proposed → active` transition requires in-flight features as a precondition**

- **Decision:** The direct `proposed → active` transition is only valid when the plan has at least one feature in a post-designing lifecycle state.
- **Context:** The §11.1 decision establishes that `designing` is a required plan phase. A completely unrestricted `proposed → active` bypass would make the designing phase optional for all plans.
- **Rationale:** The precondition preserves the intent of §11.1 — plans must have a design phase — while allowing the legitimate cross-session resume case. A plan with no in-flight features has no justification for bypassing the designing gate.
- **Consequences:** Plans with no features, or whose features are all still at `proposed`, cannot use the shortcut. This is the intended behaviour.

**D3: Skeleton dev-plan from `decompose apply` is auto-approved**

- **Decision:** The skeleton dev-plan registered by `decompose(action: apply)` is auto-approved at creation time, not left in `draft` status.
- **Context:** The gate check at `dev-planning → developing` requires an *approved* dev-plan doc. A draft skeleton would not satisfy the gate.
- **Rationale:** The skeleton's purpose is to satisfy the gate on behalf of the decompose workflow. Requiring a separate approval step would reintroduce the override requirement the feature is designed to eliminate.
- **Consequences:** Skeleton dev-plans appear as approved documents without a human approval event. This is a deliberate trade-off: the human implicitly approved the plan by calling `decompose(action: apply)`.

**D4: Taxonomy in `guide` response is derived from Go constants, not hardcoded**

- **Decision:** The `taxonomy` block in the `guide` response is populated from the `FragmentRole` constants defined in `internal/docint/types.go`, not from a separate hardcoded list.
- **Context:** A hardcoded list in the response-building code would drift from the taxonomy constants whenever new roles are added.
- **Rationale:** Single source of truth. The taxonomy constants are already the authoritative definition; the `guide` response should reflect them directly.
- **Consequences:** Any addition to `FragmentRole` automatically appears in `guide` responses. Removal of a role causes existing classifications using that role to fail validation — but that is the correct behaviour.

**D5: `suggested_classifications` in `guide` are high-confidence only**

- **Decision:** Only heading-pattern matches with `high` confidence are included in `suggested_classifications`. Medium and low confidence matches are omitted.
- **Context:** Suggested classifications are intended to reduce agent effort for unambiguous cases, not to pre-populate guesses that the agent must then second-guess.
- **Rationale:** A wrong suggestion costs more than no suggestion — the agent must read the content to evaluate and correct it. High-confidence matches (e.g. "Acceptance Criteria" → `requirement`) are reliably correct and require no verification. Low/medium matches introduce noise.
- **Consequences:** The coverage of suggested classifications is partial. Agents still need to read content for sections with non-obvious roles. This is the intended behaviour.

---

## Dependencies

| Dependency | Type | Notes |
|---|---|---|
| `internal/docint/types.go` — `FragmentRole` constants | Read | D4: taxonomy in guide response derived from these constants |
| `internal/docint/` — `guide` and `pending` action handlers | Write | Sprint 1: §5.1, §5.2, §5.3 changes |
| `internal/doc/` — `doc register` response builder | Write | Sprint 1: §5.4 nudge enhancement |
| Plan state machine — `internal/` lifecycle transition logic | Write | Sprint 2: Issue 1, `proposed → active` transition |
| `decompose` action handler | Write | Sprint 2: Issue 2, skeleton dev-plan registration |
| `worktree` tool implementation | Write | Sprint 2: Issue 3, retry/backoff |
| `.agents/skills/kanbanzai-documents/SKILL.md` | Write | Sprint 0: concise-output, atomicity; must not duplicate FEAT-01KPTHB61WPT0 content |
| `.agents/skills/kanbanzai-getting-started/SKILL.md` | Write | Sprint 0: resume-plan checklist; must not duplicate FEAT-01KPTHB61WPT0 content |
| `.agents/skills/kanbanzai-workflow/SKILL.md` | Write | Sprint 0: resume-plan checklist |
| FEAT-01KPTHB61WPT0 (corpus hygiene) | Predecessor | Sprint 0 skill edits must complement, not conflict with, this feature's changes |