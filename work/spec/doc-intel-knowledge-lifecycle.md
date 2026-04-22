# Specification: Knowledge Lifecycle Mandate

| Field         | Value                                                                            |
|---------------|----------------------------------------------------------------------------------|
| Feature       | FEAT-01KPTHB649DPK                                                               |
| Design source | `work/design/doc-intel-adoption-design.md` §6 (Fix 4) and §8 (Fix 6)            |
| Status        | draft                                                                            |
| Author        | Spec-author task                                                                 |

## Related Work

### Prior designs and specifications consulted

- [Design: Document Intelligence Adoption and Integration](../design/doc-intel-adoption-design.md) —
  this specification directly implements Fix 4 (§6) and Fix 6 (§8) of that design. All functional
  requirements are derived from §§6.3–6.5 and §8.2.

### Decisions that constrain this specification

- Knowledge retrieval steps are added to existing skill phases by inserting numbered sub-steps
  (e.g. step 1a, step 4a). The existing step numbering in each skill file MUST remain unchanged to
  preserve compatibility with references in other documents.

- The Phase 6 Close-Out knowledge curation pass in `orchestrate-development/SKILL.md` applies only
  to tier 2 entries via `knowledge(action: "list", status: "contributed", tier: 2)`. Tier 3
  entries are handled separately through promotion guidance, not confirmation, per the design's
  reasoning that tier 3 entries are self-pruning and confirming them directly bypasses the
  promotion signal.

- The plan review arm of Fix 6 (changes to the `review-plan` skill or `plan-review` workflow) is
  out of scope for this feature and MUST NOT be addressed here.

### How this specification extends prior work

This specification modifies three existing skill files. It does not create new skill files or
document templates, and it does not change the behaviour of any MCP tool. It extends the
procedural checklists and phase descriptions of the named skills with mandatory knowledge
retrieval and curation steps.

---

## Overview

This specification defines the requirements for closing the knowledge feedback loop in the
Kanbanzai workflow. As diagnosed in the design, knowledge entries are contributed after tasks but
never read before them — the `use_count` across all 59 existing entries is zero. This feature
mandates that implementer agents retrieve domain-relevant knowledge before writing any code, and
confirm or flag entries after completing their task. It mandates that orchestrators surface
confirmed knowledge to sub-agents at the start of a feature and perform a structured curation pass
at close-out. It mandates that the `kanbanzai-agents` skill document these obligations explicitly
in both the Task Lifecycle Checklist and the Context Assembly section. Together these changes
transform the knowledge base from an append-only log into a confidence-weighted, actively
consumed reference.

---

## Scope

### In scope

- Adding a mandatory knowledge retrieval step (step 1a) to Phase 1 of
  `.kbz/skills/implement-task/SKILL.md`
- Adding mandatory confirm/flag steps to Phase 4 of `.kbz/skills/implement-task/SKILL.md`
- Adding two checklist items to `.kbz/skills/implement-task/SKILL.md`
- Adding a BAD/GOOD example pair to `.kbz/skills/implement-task/SKILL.md` illustrating
  knowledge retrieval absence versus proactive retrieval
- Adding a mandatory knowledge retrieval step (step 1a) to Phase 1 of
  `.kbz/skills/orchestrate-development/SKILL.md`
- Adding a mandatory knowledge curation pass (step 4a) to Phase 6 Close-Out of
  `.kbz/skills/orchestrate-development/SKILL.md`
- Adding two checklist items to `.agents/skills/kanbanzai-agents/SKILL.md`
- Updating the Context Assembly section of `.agents/skills/kanbanzai-agents/SKILL.md` to mandate
  active knowledge querying after claiming a task

### Out of scope

- Changes to `write-design/SKILL.md` or any other design-stage skill (Fix 1 — separate feature)
- Corpus onboarding, integrity checks, or corpus completeness changes (Fix 2)
- Classification triggers, `doc_intel` changes, or batch classification (Fix 3)
- Access instrumentation, new fields on knowledge entries, or changes to `doc(action: "audit")`
  output (Fix 5)
- Changes to the `review-plan` skill or plan-review workflow (the plan review arm of Fix 6)
- Go server code changes of any kind
- Changes to any skill file not explicitly listed above

---

## Functional Requirements

### implement-task/SKILL.md

**FR-001** — `implement-task/SKILL.md` Phase 1 MUST contain a step, inserted after step 1 and
numbered 1a, that requires the agent to call `knowledge(action: "list")` with tags derived from
the task's feature area before writing any code.

**FR-002** — The Phase 1 step required by FR-001 MUST instruct the agent to review all entries
returned by the knowledge list call and to note any entries that describe known pitfalls for the
task's domain.

**FR-003** — The Phase 1 step required by FR-001 MUST include a BECAUSE rationale stating that
knowledge entries record hard-won discoveries from previous tasks and that an agent that skips
this step risks re-discovering the same problems from scratch.

**FR-004** — `implement-task/SKILL.md` Phase 4 MUST contain a step requiring the agent to call
`knowledge(action: "confirm")` for each knowledge entry that proved accurate during the task.

**FR-005** — `implement-task/SKILL.md` Phase 4 MUST contain a step requiring the agent to call
`knowledge(action: "flag")` for each knowledge entry that proved inaccurate during the task.

**FR-006** — The Phase 4 steps required by FR-004 and FR-005 MUST include a BECAUSE rationale
stating that confirmation is the mechanism by which the knowledge base self-curates, and that
unflagged inaccurate entries continue to mislead future agents indefinitely.

**FR-007** — The `implement-task/SKILL.md` Checklist MUST include an item with the exact text:
"Called knowledge list with domain-relevant tags before writing any code".

**FR-008** — The `implement-task/SKILL.md` Checklist MUST include an item with the exact text:
"Confirmed accurate and flagged inaccurate knowledge entries after task completion".

**FR-009** — `implement-task/SKILL.md` MUST include a BAD/GOOD example pair that contrasts an
agent skipping knowledge retrieval (and re-discovering a known issue from scratch) with an agent
performing proactive knowledge retrieval (and avoiding the known issue using an existing entry).

**FR-010** — The BAD example required by FR-009 MUST identify the absence of a knowledge list
call as the root cause of the failure, not the presence of the known issue itself.

**FR-011** — The GOOD example required by FR-009 MUST show the agent calling `knowledge list`
with domain-relevant tags, finding a relevant entry, and using that entry's content to avoid a
known pitfall before writing any implementation code.

### orchestrate-development/SKILL.md

**FR-012** — `orchestrate-development/SKILL.md` Phase 1 MUST contain a step, inserted after
step 1 and numbered 1a, that requires the orchestrator to call `knowledge(action: "list")` with
feature-area tags and `status: "confirmed"` after reading the dev-plan.

**FR-013** — The Phase 1 step required by FR-012 MUST require the orchestrator to surface
relevant knowledge entries to sub-agents by including them in `handoff` tool calls via the
`instructions` parameter.

**FR-014** — `orchestrate-development/SKILL.md` Phase 6 Close-Out MUST contain a step, inserted
after step 4 and numbered 4a, that requires the orchestrator to call `knowledge(action: "list")`
with `status: "contributed"` and `tier: 2` to retrieve all tier 2 knowledge entries contributed
during the feature's development.

**FR-015** — The Phase 6 Close-Out step required by FR-014 MUST require the orchestrator to
apply one of three dispositions to each returned tier 2 entry: call `knowledge(action: "confirm")`
for entries that proved accurate, call `knowledge(action: "flag")` for entries that proved
inaccurate, or call `knowledge(action: "retire")` for entries superseded by architectural
changes made in this plan.

**FR-016** — The Phase 6 Close-Out step required by FR-014 MUST provide guidance that tier 3
entries should not be confirmed but instead promoted to tier 2 via `knowledge(action: "promote")`
for those that proved valuable, because tier 3 entries are self-pruning and direct confirmation
would bypass the promotion signal.

**FR-017** — The Phase 6 Close-Out step required by FR-014 MUST identify the knowledge
confirmation pass as the primary knowledge curation mechanism for the feature.

### kanbanzai-agents/SKILL.md

**FR-018** — The `kanbanzai-agents/SKILL.md` Task Lifecycle Checklist MUST include an item
requiring the agent to call `knowledge list` with domain-relevant tags before starting
implementation, positioned after the existing "Read the assembled context" item.

**FR-019** — The `kanbanzai-agents/SKILL.md` Task Lifecycle Checklist MUST include an item
requiring the agent to confirm accurate knowledge entries and flag inaccurate ones after
completing the task, positioned after the existing task completion item.

**FR-020** — The `kanbanzai-agents/SKILL.md` Context Assembly section MUST mandate that after
calling `next(id)` to claim a task, agents MUST actively call `knowledge(action: "list")` with
task-relevant tags. The mandate MUST be stated as a requirement, not a suggestion or passive
description of system behaviour.

**FR-021** — The Context Assembly mandate required by FR-020 MUST explain that active querying
supplements the context packet's automatic knowledge surfacing because the automatic matching may
miss cross-cutting concerns and entries relevant to tasks that span multiple domains.

---

## Non-Functional Requirements

**NFR-001** — All changes are text-only edits to Markdown skill files. No Go server code changes
are required or permitted by this feature.

**NFR-002** — All additions MUST be backward compatible with the existing structure of each skill
file. Existing phases, checklist items, section headings, and step numbering MUST remain
unchanged except where this specification explicitly requires additions.

**NFR-003** — New steps and checklist items MUST use the tool invocation vocabulary established
in the design: `knowledge(action: "list")`, `knowledge(action: "confirm")`,
`knowledge(action: "flag")`, `knowledge(action: "retire")`, and `knowledge(action: "promote")`.
Synonyms or paraphrases for these operations MUST NOT be introduced.

**NFR-004** — The knowledge lifecycle requirements across all three files MUST be internally
consistent. The same obligation described in `implement-task/SKILL.md` (retrieve before coding,
confirm/flag after completing) MUST be reflected in `kanbanzai-agents/SKILL.md` using compatible
language. The orchestrator obligations described in `orchestrate-development/SKILL.md` MUST NOT
contradict the implementer obligations described in `implement-task/SKILL.md`.

---

## Acceptance Criteria

- [ ] **FR-001**: `implement-task/SKILL.md` Phase 1 contains a step 1a requiring
  `knowledge(action: "list")` with feature-area tags, inserted after the existing step 1.
- [ ] **FR-002**: The Phase 1 step 1a instructs the agent to review all returned entries and note
  domain-relevant pitfalls.
- [ ] **FR-003**: The Phase 1 step 1a includes a BECAUSE rationale about re-discovering known
  problems.
- [ ] **FR-004**: `implement-task/SKILL.md` Phase 4 contains a `knowledge(action: "confirm")`
  step for entries that proved accurate.
- [ ] **FR-005**: `implement-task/SKILL.md` Phase 4 contains a `knowledge(action: "flag")` step
  for entries that proved inaccurate.
- [ ] **FR-006**: The Phase 4 additions include a BECAUSE rationale about self-curation and
  indefinitely misleading entries.
- [ ] **FR-007**: `implement-task/SKILL.md` Checklist contains an item with the exact text
  "Called knowledge list with domain-relevant tags before writing any code".
- [ ] **FR-008**: `implement-task/SKILL.md` Checklist contains an item with the exact text
  "Confirmed accurate and flagged inaccurate knowledge entries after task completion".
- [ ] **FR-009**: `implement-task/SKILL.md` contains a BAD/GOOD example pair contrasting absent
  knowledge retrieval with proactive retrieval.
- [ ] **FR-010**: The BAD example identifies the missing knowledge list call as the root cause.
- [ ] **FR-011**: The GOOD example shows the agent finding and using a knowledge entry to avoid a
  known pitfall before implementation.
- [ ] **FR-012**: `orchestrate-development/SKILL.md` Phase 1 contains a step 1a requiring
  `knowledge(action: "list")` with feature-area tags and `status: "confirmed"`.
- [ ] **FR-013**: The Phase 1 step 1a requires surfacing relevant entries to sub-agents via
  `handoff` `instructions`.
- [ ] **FR-014**: `orchestrate-development/SKILL.md` Phase 6 Close-Out contains a step 4a
  requiring `knowledge(action: "list")` with `status: "contributed"` and `tier: 2`.
- [ ] **FR-015**: The Phase 6 step 4a requires confirm, flag, and retire dispositions for tier 2
  entries based on accuracy and relevance.
- [ ] **FR-016**: The Phase 6 step 4a provides guidance to use `knowledge(action: "promote")` for
  valuable tier 3 entries rather than confirming them.
- [ ] **FR-017**: The Phase 6 step 4a identifies the confirmation pass as the primary knowledge
  curation mechanism.
- [ ] **FR-018**: `kanbanzai-agents/SKILL.md` Task Lifecycle Checklist contains an item requiring
  knowledge list with domain-relevant tags before implementation, after the "Read assembled
  context" item.
- [ ] **FR-019**: `kanbanzai-agents/SKILL.md` Task Lifecycle Checklist contains an item requiring
  confirm/flag of knowledge entries after task completion.
- [ ] **FR-020**: `kanbanzai-agents/SKILL.md` Context Assembly section mandates active
  `knowledge(action: "list")` after `next(id)`, stated as a requirement.
- [ ] **FR-021**: The Context Assembly mandate explains that active querying finds entries that
  automatic surfacing may miss for cross-cutting concerns and multi-domain tasks.

---

## Dependencies and Assumptions

**DEP-001** — This feature depends on `work/design/doc-intel-adoption-design.md` being in
approved status. All functional requirements are derived exclusively from §6 (Fix 4) and §8
(Fix 6) of that document. If the design is revised before implementation, this specification
must be reviewed for consistency.

**DEP-002** — The `knowledge` tool's `list`, `confirm`, `flag`, `retire`, and `promote` actions
must be available and functional in the running MCP server before the modified skills can be
exercised. This feature adds no new tool actions; it only adds skill-file instructions that call
existing ones.

**DEP-003** — This feature does not depend on Fix 5 (access instrumentation). The
`knowledge(action: "list")` calls mandated here will function correctly regardless of whether the
`last_accessed_at` and `recent_use_count` fields introduced by Fix 5 are present. The two fixes
are independently deployable.

**ASM-001** — It is assumed that agents executing the modified skills have write access to the
knowledge store (confirm, flag, retire, promote operations) for the project in which they are
working.

**ASM-002** — It is assumed that the existing `implement-task/SKILL.md` Examples section can
accommodate an additional BAD/GOOD example pair without restructuring. The new pair is appended
after the existing examples.

**ASM-003** — It is assumed that the `orchestrate-development/SKILL.md` Checklist will be
updated to reflect the two new Phase 1 and Phase 6 obligations added by this specification,
consistent with how the checklist mirrors the procedure steps in that skill.