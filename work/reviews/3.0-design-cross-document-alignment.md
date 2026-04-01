# Cross-Document Alignment Report: Kanbanzai 3.0 Design Documents

| Field | Value |
|-------|-------|
| Date | 2025-07-29 |
| Status | Review |
| Author | Design Review Agent |
| Documents Reviewed | `work/design/kanbanzai-3.0-workflow-and-tooling.md`, `work/design/skills-system-redesign.md` |
| Related | `work/design/orchestration-recommendations.md`, `work/research/agent-orchestration-research.md` |

---

## Purpose

This report documents the results of a cross-document alignment review between the two primary Kanbanzai 3.0 design documents:

1. **Skills System Redesign** (`work/design/skills-system-redesign.md`) — defines roles, skills, vocabulary, anti-patterns, the binding registry, and the context assembly model.
2. **Workflow Engine and MCP Tooling** (`work/design/kanbanzai-3.0-workflow-and-tooling.md`) — defines stage gate enforcement, MCP tool surface quality, lifecycle-aware context assembly, and observability.

The two documents were written to cover complementary concerns, with the skills redesign answering "what do agents know and how are they shaped?" and the workflow doc answering "what does the system enforce, how do tools behave, and what context do agents receive?" The **binding registry** is the declared integration surface between them.

This review found **3 problematic overlaps** (contradictions or duplications that will cause confusion if not resolved), **4 benign overlaps** (reasonable cross-referencing that needs a clear canonical owner), and **2 joint design concerns** (items that span both documents and need coordinated design).

No issues were found that require discarding work from either document. All resolutions preserve the content — they involve moving canonical ownership, adding cross-references, and consolidating duplicated content into one authoritative location.

---

## Table of Contents

1. [Problematic Overlaps](#1-problematic-overlaps)
   1.1 [Tool Filtering: Hard vs Soft (Contradiction)](#11-tool-filtering-hard-vs-soft-contradiction)
   1.2 [Document Templates: Who Defines Them, Where Do They Live?](#12-document-templates-who-defines-them-where-do-they-live)
   1.3 [Context Assembly Pipeline: Two Descriptions](#13-context-assembly-pipeline-two-descriptions)
2. [Benign Overlaps](#2-benign-overlaps)
   2.1 [Effort Budget Values (Duplicated Content)](#21-effort-budget-values-duplicated-content)
   2.2 [Decomposition Validation Checks (Duplicated Checks)](#22-decomposition-validation-checks-duplicated-checks)
   2.3 [Stage Gate Claims in Skills Doc DD-16 (Cross-Document Requirement)](#23-stage-gate-claims-in-skills-doc-dd-16-cross-document-requirement)
   2.4 [Observability Metrics (Incomplete Alignment)](#24-observability-metrics-incomplete-alignment)
3. [Joint Design Concerns](#3-joint-design-concerns)
   3.1 [Template Content + Structural Gate Checks](#31-template-content--structural-gate-checks)
   3.2 [The Full Assembly Pipeline](#32-the-full-assembly-pipeline)
4. [Recommended Actions Summary](#4-recommended-actions-summary)
5. [Research Integrity Check](#5-research-integrity-check)
   5.1 [Source Document Traceability](#51-source-document-traceability)
   5.2 [Alignment Report Recommendation Integrity](#52-alignment-report-recommendation-integrity)

---

## 1. Problematic Overlaps

These require resolution before either document can be considered ready for specification. If left unresolved, implementers will encounter contradictory instructions.

### 1.1 Tool Filtering: Hard vs Soft (Contradiction)

**Severity: High — direct contradiction between the two documents.**

#### What the Skills Doc Says

The skills redesign treats tool filtering as **hard exclusion**. Each role declares a `tools` field:

```
# Skills doc §3.1 — Role definition example
tools:
  - entity
  - doc_intel
  - knowledge
  - read_file
  - grep
  - search_graph
```

The assembly pipeline (§6.1, step 7) says:

> "Filter MCP tool definitions — Include only tools listed in the role's `tools` field (orchestrator needs: entity, handoff, next, status, spawn_agent, finish, ...) (does NOT need: edit_file, terminal, diagnostics, ...)"

The assembled context description (§3.4) explicitly states that tool definitions for tools NOT in the role's `tools` list are excluded from the assembled context. DD-8 codifies this: "Every loaded tool definition consumes attention budget. A security reviewer does not need `decompose`. Filtering reduces context size and focuses attention on relevant tools."

#### What the Workflow Doc Says

The workflow doc explicitly chose **soft filtering** (§9.2):

> "The binding registry declares per-stage tool subsets. For 3.0, the context assembly pipeline reads these and includes a 'tools you should use' list prominently in the assembled context. All tools remain available — this is guidance, not restriction."

Section 9.3 explains the rationale:

> "The research recommends that specification agents 'should have no access to implementation tools (terminal, file editing)' — a hard restriction (Research §3.3). Hard filtering (dynamically hiding tools from the MCP session) would be the purest implementation of this recommendation. However, it would require session-level tool registration and MCP protocol changes."

#### Analysis

The skills doc assumes hard filtering is possible and designs accordingly. The workflow doc explicitly considered hard filtering and rejected it for 3.0 due to claimed MCP protocol constraints, opting for soft filtering (guidance text in context).

**The research is unambiguous on this point.** Both research documents identify enforceable constraints as superior to advisory ones — this is one of the strongest findings across all sources reviewed:

- Orchestration research §2.2: "Every source that compares 'telling agents what to do' with 'preventing agents from doing the wrong thing' finds the latter wins decisively."
- Orchestration research §3.3: "The specification agent should have *no access* to implementation tools (terminal, file editing)."
- Orchestration research §4.2: Tool subsets per role are recommended as a high-impact ACI intervention.
- Skills research §3.2: Low freedom (exact constraints) is appropriate for fragile operations.
- SWE-agent: Interface design — including which tools are available — affects agent performance as much as model capability.

The workflow doc's rationale ("would require session-level tool registration and MCP protocol changes") is a pragmatic implementation concern, but the claimed protocol constraint is weaker than presented. The Kanbanzai MCP server already assembles different context per task, and the MCP protocol does not prevent a server from presenting a filtered tool list per session. This is an implementation effort question, not a protocol impossibility.

**The skills doc's hard filtering design (DD-8) is the research-aligned target.** The workflow doc's soft filtering is an acceptable *interim compromise* for the initial 3.0 release, but it should not be treated as the correct final answer. The alignment report must not instruct the skills doc to weaken its research-backed design intent.

#### Recommended Resolution

- **Canonical owner of the filtering mechanism decision:** Workflow doc §9 owns the *implementation timeline*. Skills doc DD-8 owns the *design intent*.
- **Design target:** Hard filtering — the role's `tools` field declares the enforced tool subset. This is the research-backed design and the skills doc should retain it as the target architecture.
- **3.0 implementation:** Soft filtering as a pragmatic stepping stone. The workflow doc §9.2's approach (guidance text in assembled context) is accepted for the initial release, with the explicit commitment to implement hard filtering once soft filtering's effectiveness is measured.
- **Action on Skills doc:** DD-8 retains its hard filtering design rationale unchanged. Add a single implementation note: "3.0 delivers soft filtering (guidance text) as a stepping stone; hard filtering is the target. See workflow doc §9.3 for the implementation timeline." The `tools` field, §6.1 step 7, and the §3.4 assembly description all retain their hard-filtering language as the design intent. They are not weakened to match the interim implementation.
- **Action on Workflow doc:** §9.2 should frame soft filtering as an interim measure: "For the initial 3.0 release, tool subsets are delivered as guidance text in the assembled context. Hard filtering (dynamically scoping the MCP tool list per session) is the design target (skills doc DD-8). The evaluation suite (§12) should track tool selection compliance; if soft filtering proves insufficient, hard filtering implementation should be prioritised." §9.3's trade-off analysis is retained but reframed from "rejection of hard filtering" to "deferral of hard filtering."
- **Content preserved:** The `tools` field content per role is unchanged. The role taxonomy's tool assignments remain the source data — first for guidance text (3.0), then for hard exclusion (future).
- **Research deviation acknowledged:** This resolution is a known deviation from the research recommendation for enforceable constraints (orchestration research §2.2, §4.2). The deviation is accepted for pragmatic reasons with a measured path to compliance. See §5 (Research Integrity Check) for the full accounting.

---

### 1.2 Document Templates: Who Defines Them, Where Do They Live?

**Severity: Medium — duplicated content with conflicting locations.**

#### What the Skills Doc Says

Each authoring skill carries a "gate-checkable template" (§5.1):

> "Each authoring skill template defines:
> 1. **Required sections** — the section names that must be present (5–8 sections, following DP-6). A specification template might require: Problem Statement, Requirements, Constraints, Acceptance Criteria, Verification Plan.
> 2. **Cross-reference requirements** — which other documents or entities must be cited.
> 3. **Acceptance criteria format** — how acceptance criteria are expressed."

The skills doc says validation scripts in `scripts/` check these at stage gates. DD-19 codifies this: "Each authoring skill defines required sections and cross-references. Validation scripts in `scripts/` check these at stage gates."

Templates live **inside each skill file** as part of the skill's output format.

#### What the Workflow Doc Says

Section 10.2 defines the same templates with specific section lists:

**Specification template (5 required sections):**
1. Problem Statement
2. Requirements
3. Constraints
4. Acceptance Criteria
5. Verification Plan

**Dev-plan template (5 required sections):**
1. Scope
2. Task Breakdown
3. Dependency Graph
4. Risk Assessment
5. Verification Approach

**Design document template (4 required sections):**
1. Problem and Motivation
2. Design
3. Alternatives Considered
4. Decisions

Section 10.3 says templates are delivered via the `handoff` assembly pipeline: "the `handoff` pipeline includes the template inline in the assembled context — not as a reference to a file, but as the actual section headings with brief guidance for each."

Section 10.4 designs automated structural checks at stage gates.

#### Analysis

Both documents define the specification template with **identical section names** (Problem Statement, Requirements, Constraints, Acceptance Criteria, Verification Plan). The skills doc leaves the other templates unenumerated; the workflow doc enumerates all three.

The fundamental question is: where does the template content live?

- Skills doc answer: In each skill file (as part of the skill's output format section).
- Workflow doc answer: In the assembly pipeline (injected during context assembly).

These aren't necessarily contradictory — the skill could define the template, and the pipeline could extract it from the skill during assembly. But neither document says that. Each presents itself as the primary location.

#### Recommended Resolution

- **Template content** belongs in the **skills redesign**, because:
  - Templates are authoring guidance — they shape what the agent writes, which is a skill concern.
  - Each skill carries its own vocabulary and anti-patterns; the template is part of the same context unit.
  - DD-19 explicitly makes this a design decision.
  - The skill's output format section is the natural home for "what does the output look like?"

- **Template enforcement** (structural checks at stage gates) belongs in the **workflow doc**, because:
  - Gate checking is MCP server logic, not skill content.
  - The workflow doc owns all stage gate behavior (§3).

- **Template delivery** is the assembly pipeline — templates are delivered because they're part of the skill that gets loaded during assembly. This is not a separate pipeline step.

- **Action on Skills doc:** The skills doc should enumerate the full template sections for all authoring skills (it currently only fully enumerates the spec template). This becomes the canonical source for template content.

- **Action on Workflow doc:** §10.2 should **remove the specific template section lists** and instead reference the skills doc: "Template sections are defined per authoring skill in the skills redesign (§5.1). See the `write-spec`, `write-design`, and `write-dev-plan` skills for the required sections." Section 10.3 should say "Templates are delivered as part of the skill's output format section during context assembly" rather than presenting them as a separate assembly-pipeline concern. Sections 10.4 (structural checks) and 10.5 (LLM-as-judge) remain in the workflow doc — they are enforcement mechanisms, not content.

- **Content preserved:** All template content is preserved. The specific section lists move to the skills doc (their natural home). The enforcement mechanisms stay in the workflow doc.

---

### 1.3 Context Assembly Pipeline: Two Descriptions

**Severity: Medium — duplicated pipeline descriptions that don't fully agree.**

#### What the Skills Doc Says

The skills doc describes the assembly pipeline in two places:

**§3.4 — Assembled Context:** A 5-step summary plus a 10-position attention-curve ordering table:

| Position | Content | Source | Attention |
|----------|---------|--------|-----------|
| 1 | Project identity and hard constraints | `base` role | **High** |
| 2 | Role identity | Selected role | **High** |
| 3 | Effort budget and orchestration pattern | Stage binding | **High** |
| 4 | Combined vocabulary payload | Role + Skill | **High** |
| 5 | Combined anti-pattern watchlist | Role + Skill | **Medium-High** |
| 6 | Skill procedure (numbered steps) | Selected skill | **Medium** |
| 7 | Output format + examples | Selected skill | **Rising** |
| 8 | Relevant knowledge entries | Knowledge system | **Rising** |
| 9 | Evaluation criteria | Selected skill | **High** |
| 10 | Retrieval anchors | Selected skill | **High** |

**§6.1 — The Assembly Pipeline:** A detailed 9-step operational pipeline:
1. Resolve task → parent feature → stage
2. Look up stage binding
3. Extract orchestration metadata
4. Resolve role with inheritance
5. Load skill
6. Surface relevant knowledge entries
7. Filter MCP tool definitions
8. Estimate token budget
9. Assemble in attention-curve order

#### What the Workflow Doc Says

**§7.5 — Implementation Approach:** An 8-step pipeline:
1. **Validate lifecycle state** (reject wrong-state assembly)
2. Determine the feature's current lifecycle stage
3. Look up the stage in the binding registry
4. Select the assembly strategy (stage-specific inclusions/exclusions)
5. Assemble context using stage-specific inclusions/exclusions
6. Insert orchestration pattern signal at top
7. Insert effort budget near top
8. Insert tool subset list near top

#### Analysis: Key Differences

| Aspect | Skills Doc | Workflow Doc |
|--------|-----------|-------------|
| Lifecycle state validation | Not mentioned | Step 1 — reject wrong-state assembly |
| Orchestration metadata extraction | Step 3 (single step) | Steps 6–8 (three separate insertions) |
| Token budget estimation | Step 8 (refuse if >60%) | Not mentioned |
| Stage-specific inclusion/exclusion | Not detailed | §7.3 — full table per stage |
| Attention-curve ordering | Full 10-position table (§3.4) | Not detailed (defers to skills) |
| Knowledge auto-surfacing | Step 6 + dedicated §6.3 | Not mentioned |

The two pipelines are **mostly complementary** — the skills doc focuses on *what goes in* (role, skill, vocabulary, attention ordering, token budgets) and the workflow doc focuses on *what varies by stage* (inclusion/exclusion, orchestration signals, lifecycle validation). But a reader consulting both will see three different pipeline descriptions (§3.4, §6.1, §7.5) and wonder which is authoritative.

#### Recommended Resolution

The assembly pipeline should have **one canonical description**:

- **Canonical owner:** Skills doc §6.1 — it is already the most detailed operational description.
- **Action on Skills doc §6.1:** Merge in the workflow doc's additions:
  - Add **lifecycle state validation** as step 0 or step 1 (from workflow §7.2).
  - Add the **stage-specific inclusion/exclusion table** as part of the assembly strategy selection (from workflow §7.3).
  - Clarify the orchestration pattern, effort budget, and tool subset insertions (from workflow §7.4, §8.4, §9.2).
- **Action on Workflow doc §7:** Shorten to a **requirements statement** for the pipeline. It should say: "The full assembly pipeline is defined in the skills redesign (§6.1). This document adds the following requirements to that pipeline:" followed by the three workflow-specific additions (state validation, stage-specific inclusion/exclusion table, orchestration pattern signalling). Remove the separate 8-step pipeline description.
- **Content preserved:** All content from both descriptions is preserved. The skills doc's pipeline gains the workflow doc's additions. The workflow doc retains its requirements (§7.2 lifecycle validation, §7.3 stage table, §7.4 orchestration signals) as authored content — just reframed as inputs to the canonical pipeline rather than a parallel pipeline description.

---

## 2. Benign Overlaps

These are cases where both documents touch the same topic but the overlap is manageable with clear canonical ownership and cross-references.

### 2.1 Effort Budget Values (Duplicated Content)

Both documents include the same per-stage effort budget values:

| Stage | Value (identical in both docs) |
|-------|-------------------------------|
| Designing | 5–15 tool calls |
| Specifying | 5–15 tool calls |
| Dev-planning | 5–10 tool calls |
| Developing (per task) | 10–50 tool calls |
| Reviewing (per dimension) | 5–10 tool calls |

The skills doc declares these in the binding registry (§3.3, `effort_budget` field per stage). The workflow doc repeats them in §8.3 as a reference table.

**Canonical owner:** Skills doc §3.3 — the binding registry is the source of truth for these values.

**Recommended action:** The workflow doc §8.3 should reference the binding registry instead of repeating the values: "These values come from the binding registry (skills redesign §3.3). The workflow doc's contribution is not the values themselves but how they're positioned and formatted in the assembled prompt (§8.2, §8.4)."

**Risk if not addressed:** Low. If someone updates effort budgets, they might update one document but not the other. A cross-reference prevents this.

---

### 2.2 Decomposition Validation Checks (Duplicated Checks)

Both documents list the same decomposition quality checks:

**Skills doc §5.2** (`decompose-feature` skill):
> "Do tasks have clear descriptions? Are dependencies declared? Are tasks sized for single-agent completion? Are there gaps (e.g., missing test tasks)?"

**Workflow doc §11.2** (decomposition quality validation in `decompose` tool):

| Check | Description | Severity |
|-------|-------------|----------|
| Description present | Every task has a non-empty summary | Error |
| Dependencies declared | If tasks reference each other, `depends_on` is populated | Warning |
| Single-agent sizing | No task description suggests multiple independent changes | Warning |
| Testing coverage | At least one task mentions testing or verification | Warning |
| No orphan tasks | Every task is reachable from the dependency graph root | Warning |

**Canonical owner:** Workflow doc §11 — these are tool-level validation checks in the `decompose` tool, not skill-level guidance. The skill's role is to carry vocabulary and anti-patterns for decomposition *quality*; the tool's role is to enforce structural *validity*.

**Recommended action:** The skills doc §5.2 should say: "The `decompose-feature` skill carries vocabulary and anti-patterns for decomposition quality. Tool-level validation checks (description present, dependencies declared, sizing, testing coverage) are defined in the workflow doc (§11)." This avoids both documents maintaining the same checklist independently.

**Risk if not addressed:** Medium. If a new validation check is added to the tool, the skill description would become inconsistent. A cross-reference prevents this.

---

### 2.3 Stage Gate Claims in Skills Doc DD-16 (Cross-Document Requirement)

Skills doc DD-16 says:

> "The `entity(action: 'transition')` tool rejects transitions with unmet prerequisites."

This is a design requirement stated in the skills doc about MCP server behavior that the workflow doc implements. DD-16 is derived directly from the research — Masters et al.'s hard constraints (ℋ), MetaGPT's SOPs with intermediate verification gates, and the orchestration research's #1 recommendation (§4.1: "the single highest-impact change because it converts the entire class of 'agents skip steps' failures from a quality problem into an impossibility"). It is not overreach; it is a cross-document requirement — the skills doc declares the requirement; the workflow doc owns the implementation.

**Canonical owner of the requirement:** Skills doc DD-16 (derived from research).
**Canonical owner of the enforcement mechanism:** Workflow doc §3.

**Recommended action:** Clarify the requirement/implementation split in DD-16: "Stage prerequisites are declared in the binding registry. The enforcement mechanism — how the `entity(action: 'transition')` tool checks and rejects transitions with unmet prerequisites — is designed in the workflow doc (§3). This design declares the prerequisite data and the requirement that transitions are rejected when prerequisites are unmet; that design implements the enforcement."

**Risk if not addressed:** Low. This is a clarity issue, not a contradiction.

---

### 2.4 Observability Metrics (Incomplete Alignment)

The skills doc §10.1 lists metrics to track and explicitly says a "companion observability design" is needed:

> "These are system-level instrumentation concerns... They should be addressed in a companion observability design."

The workflow doc §12 provides that companion — action pattern logging, stage-level metrics, and a small-sample evaluation suite. However, the metrics in the two documents don't fully align:

| Skills Doc §10.1 Metric | Workflow Doc §12 Coverage |
|---|---|
| First-attempt convention compliance | Not covered |
| Review finding specificity | Not covered |
| Stale-doc-caused errors | Not covered |
| Context assembly token utilisation | Not covered (but skills §6.2 handles this internally) |
| Review rubber-stamp rate | Related to "Review thoroughness" in §12.4 |
| Sub-agent dispatch per feature | Not covered |
| MAST failure mode incidents | Not covered |

The workflow doc introduces its own metrics that the skills doc didn't anticipate:

| Workflow Doc §12 Metric | Skills Doc Coverage |
|---|---|
| Gate failure rate | Not mentioned |
| Tool selection accuracy (stage compliance) | Not mentioned |
| Structural check pass rate | Not mentioned |
| Time per stage | Not mentioned |

**Recommended action:** The workflow doc §12.3 should incorporate the skills doc's metrics or explicitly note which metrics are tracked where. Some skills-specific metrics (convention compliance, finding specificity) may be better measured through the skill evaluation process (skills doc §9) than through system logging — in which case the workflow doc should say so and cross-reference.

**Risk if not addressed:** Low. The metrics aren't contradictory — they're complementary. But without alignment, some metrics will fall through the cracks because neither document takes ownership.

---

## 3. Joint Design Concerns

These are items that span both documents and would benefit from coordinated design — not because they're duplicated, but because changes to one affect the other.

### 3.1 Template Content + Structural Gate Checks

The template (skill concern) and the enforcement (workflow concern) are **tightly coupled**. If the spec template gains a sixth required section, both the skill's output format and the workflow doc's structural check must update. Having them in separate documents risks drift over time.

**Recommendation:** Define a **template schema** in the binding registry that both documents reference. This schema declares required section names per document type and serves as the single source of truth for both template content (used by skills) and gate checking (used by the workflow engine).

A possible structure within the existing binding registry format:

```
stage_bindings:
  specifying:
    # ...existing fields...
    document_template:
      required_sections:
        - "Problem Statement"
        - "Requirements"
        - "Constraints"
        - "Acceptance Criteria"
        - "Verification Plan"
      cross_references:
        - parent_design_document
      acceptance_criteria_format: "numbered-testable-assertions"
```

Both the skill (for output format guidance) and the gate check (for structural validation) read this structure. Changes to the template are made in one place and automatically affect both.

**Alternative:** If adding to the binding registry feels like overloading it, the template schemas could live in a dedicated file (e.g., `.kbz/templates/`) that both systems reference. The key requirement is: one source of truth for section names.

### 3.2 The Full Assembly Pipeline

The assembly pipeline is the primary integration surface between the two designs. It's where roles, skills, bindings, lifecycle validation, stage-specific context, effort budgets, tool subsets, and token budgets all converge.

Currently, the pipeline is described in three places:
1. Skills doc §3.4 (summary + attention-curve table)
2. Skills doc §6.1 (detailed 9-step pipeline)
3. Workflow doc §7.5 (8-step pipeline with different emphasis)

**Recommendation:** Consolidate into a single canonical pipeline in the skills doc §6.1, extended with the workflow doc's additions. The combined pipeline would look like:

| Step | Description | Source |
|------|-------------|--------|
| 0 | **Validate lifecycle state** — check feature is in correct state for this task. Reject with actionable error if not. | Workflow doc §7.2 |
| 1 | Resolve task → parent feature → feature lifecycle stage | Skills doc §6.1 |
| 2 | Look up stage binding | Skills doc §6.1 |
| 3 | Extract orchestration metadata (pattern, effort budget, prerequisites, max_review_cycles) | Skills doc §6.1 |
| 4 | Apply **stage-specific inclusion/exclusion strategy** — vary what context is included based on stage | Workflow doc §7.3 |
| 5 | Resolve role with inheritance | Skills doc §6.1 |
| 6 | Load skill (including output format template) | Skills doc §6.1 |
| 7 | Surface relevant knowledge entries (file paths, tags, recency) | Skills doc §6.1, §6.3 |
| 8 | Apply **tool subset** from role's `tools` field (3.0: guidance text; target: hard filtering — see §1.1) | Skills doc DD-8 (design), Workflow doc §9.2 (3.0 mechanism) |
| 9 | Estimate token budget — warn at 40%, refuse at 60% | Skills doc §6.2 |
| 10 | Assemble in attention-curve order (10-position table from §3.4) | Skills doc §3.4 |

The workflow doc §7 would then become a requirements statement: "The assembly pipeline (skills doc §6.1) must satisfy these stage-awareness requirements:" followed by §7.2 (lifecycle validation), §7.3 (inclusion/exclusion table), §7.4 (orchestration pattern signalling), and references to §8 (effort budget positioning) and §9 (tool subset guidance).

---

## 4. Recommended Actions Summary

| # | Issue | Severity | Canonical Owner | Action Required |
|---|-------|----------|-----------------|-----------------|
| 1 | Tool filtering: hard vs soft | **High** (contradiction) | Skills doc DD-8 (design target), Workflow doc §9 (3.0 mechanism) | Retain DD-8's hard filtering as design target. Workflow doc §9.2 reframes soft filtering as interim 3.0 stepping stone. Add implementation note to DD-8 acknowledging the phased approach. Track compliance via evaluation suite. |
| 2 | Document templates: duplicated content and location | **Medium** (duplication) | Skills doc §5.1 (content), Workflow doc §10.4-10.5 (enforcement) | Skills doc enumerates full template sections for all authoring skills. Workflow doc §10.2 removes section lists, references skills doc. Add template schema to binding registry. |
| 3 | Assembly pipeline: two descriptions | **Medium** (duplication) | Skills doc §6.1 | Skills doc §6.1 gains workflow doc's additions (state validation, stage table). Workflow doc §7 becomes a requirements statement referencing skills doc. |
| 4 | Effort budget values | **Low** (duplication) | Skills doc §3.3 (binding registry) | Workflow doc §8.3 references binding registry instead of repeating values. |
| 5 | Decomposition validation checks | **Low** (duplication) | Workflow doc §11 | Skills doc §5.2 references workflow doc §11 for tool-level checks. |
| 6 | Stage gate claims in DD-16 | **Low** (cross-document requirement) | Skills doc DD-16 (requirement), Workflow doc §3 (implementation) | Clarify requirement/implementation split. DD-16 states the research-derived requirement; workflow doc §3 implements the enforcement mechanism. |
| 7 | Observability metrics | **Low** (incomplete alignment) | Workflow doc §12 (system metrics), Skills doc §9 (skill evaluation metrics) | Workflow doc §12.3 incorporates or cross-references skills doc §10.1 metrics. |
| 8 | Template + gate coupling | **Design coordination** | Binding registry (shared) | Add `document_template` structure to binding registry as single source of truth for section names. |
| 9 | Assembly pipeline consolidation | **Design coordination** | Skills doc §6.1 (canonical pipeline) | Consolidate into one pipeline description. Workflow doc contributes requirements. |

### Suggested Sequence of Changes

The changes should be applied in this order to maintain consistency:

1. **Resolve the tool filtering contradiction first** (Action 1) — this is the only hard contradiction and affects the assembly pipeline design.
2. **Consolidate the assembly pipeline** (Actions 3, 9) — this is the biggest structural change and affects both documents.
3. **Move template content to skills doc** (Actions 2, 8) — depends on the pipeline being settled.
4. **Add cross-references for all benign overlaps** (Actions 4, 5, 6, 7) — these are small, safe changes.

---

## 5. Research Integrity Check

### 5.1 Source Document Traceability

As part of this review, the workflow and tooling document was checked against the underlying research (`work/research/agent-orchestration-research.md`). The following gaps were identified and have already been addressed in the current version of the workflow doc:

| Gap | Status | Section Added |
|-----|--------|--------------|
| Automated maker-checker review (structural checks + LLM-as-judge) | **Fixed** | Workflow §10.4, §10.5 |
| `handoff`/`next` lifecycle state validation (refuse wrong-state assembly) | **Fixed** | Workflow §7.2 |
| Evaluation test suite and stage-level metrics | **Fixed** | Workflow §12.3, §12.5 |
| Tool-testing agent methodology for ACI audit | **Fixed** | Workflow §5.6 |
| Hard tool restriction trade-off acknowledgement | **Fixed** | Workflow §9.3 |

No research misrepresentations were found. All specific claims, numbers, and attributions in both documents accurately trace to the research sources.

### 5.2 Alignment Report Recommendation Integrity

This alignment report's own recommendations were also checked against the research to ensure that resolving overlaps between the two design documents does not inadvertently dilute or override research findings. The research documents (`work/research/agent-skills-research.md` and `work/research/agent-orchestration-research.md`) are treated as canonical and authoritative — they are grounded in peer-reviewed academic research and should not be overridden by implementation convenience.

**Known deviations from research recommendations:**

| Recommendation | Research Position | This Report's Resolution | Deviation | Justification |
|---|---|---|---|---|
| Tool filtering (§1.1) | Hard filtering — enforceable tool subsets per role. "Every source that compares 'telling agents what to do' with 'preventing agents from doing the wrong thing' finds the latter wins decisively" (orchestration research §2.2). Specification agents "should have no access to implementation tools" (§3.3). SWE-agent: interface design affects performance as much as model capability. | Soft filtering for 3.0 as interim stepping stone; hard filtering retained as design target. | **Yes — pragmatic deferral.** | Implementation effort for 3.0 timeline. The design target (DD-8) is unchanged. Compliance will be measured via the evaluation suite (workflow §12). If soft filtering proves insufficient, hard filtering implementation is prioritised. This is a conscious, bounded compromise — not an abandonment of the research recommendation. |

**No other deviations were identified.** All other resolutions in this report (template ownership, pipeline consolidation, effort budget cross-references, decomposition check ownership, observability alignment) are consistent with the research recommendations.

**Integrity principle applied:** Where a resolution in this report conflicts with a research finding, the research finding is preserved as the design target and the resolution is framed as a phased implementation approach — not as a correction or rejection of the research. The skills design document's research-derived design decisions (DD-8, DD-16, DD-19, DD-20) are requirements, not suggestions, and this alignment report does not weaken them.

---

## Appendix: Document Section Cross-Reference

For convenience, here is a mapping of which sections in each document address the same topic:

| Topic | Skills Doc Section | Workflow Doc Section | Canonical Owner |
|-------|-------------------|---------------------|-----------------|
| Binding registry schema | §3.3 | §3.5 (as gate source) | Skills doc |
| Stage gate prerequisites (data) | §3.3 `prerequisites` per binding | §3.3 (gate prerequisite table) | Skills doc (data), Workflow doc (enforcement) |
| Stage gate enforcement (mechanism) | DD-16 (requirement) | §3 (full design) | Skills doc (requirement), Workflow doc (implementation) |
| Effort budget values | §3.3 `effort_budget` per binding | §8.3 (reference table) | Skills doc |
| Effort budget positioning/formatting | — | §8.2, §8.4 | Workflow doc |
| Orchestration pattern (data) | §3.3 `orchestration` per binding | §7.4 (signalling mechanism) | Skills doc (data), Workflow doc (delivery) |
| Tool subsets per role (data) | §3.1 `tools` field | §9.4 (subsets by stage table) | Skills doc |
| Tool filtering mechanism | §6.1 step 7, DD-8 (design target: hard filtering) | §9.2, §9.3 (3.0: soft filtering as stepping stone) | Skills doc (design target), Workflow doc (3.0 mechanism) |
| Document templates (content) | §5.1 (gate-checkable templates) | §10.2 (section lists) | Skills doc |
| Document template enforcement | DD-19 (scripts in `scripts/`) | §10.4 (structural checks) | Workflow doc |
| LLM-as-judge quality evaluation | — | §10.5 | Workflow doc |
| Assembly pipeline (full description) | §3.4 (summary), §6.1 (detailed) | §7.5 (parallel description) | Skills doc §6.1 |
| Lifecycle state validation in assembly | — | §7.2 | Workflow doc (requirement), Skills doc (implementation in pipeline) |
| Stage-specific inclusion/exclusion | — | §7.3 (full table) | Workflow doc |
| Token budget management | §6.2 | — | Skills doc |
| Knowledge auto-surfacing | §6.3 | — | Skills doc |
| Review-rework iteration cap (value) | §3.3 `max_review_cycles: 3` | §4.3 (cap mechanism) | Skills doc (value), Workflow doc (mechanism) |
| Review cycle tracking on entity | — | §4.2 | Workflow doc |
| Decomposition validation checks | §5.2 (prose list) | §11.2 (structured table) | Workflow doc |
| Decomposition skill vocabulary/anti-patterns | §5.2 | — | Skills doc |
| ACI tool description audit | — | §5 | Workflow doc |
| Actionable error messages | — | §6 | Workflow doc |
| Action pattern logging | — | §12 | Workflow doc |
| Observability metrics | §10.1 (list + scope note) | §12.3, §12.4 | Workflow doc (with cross-ref to skills §10.1) |
| Evaluation test suite | — | §12.5 | Workflow doc |
| Context compaction guidance | §5.2 (in orchestrate-development skill) | §13 (convention) | Skills doc (skill content), Workflow doc (convention) |