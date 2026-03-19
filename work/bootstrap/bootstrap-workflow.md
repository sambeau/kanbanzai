# Bootstrap Workflow

- Status: active
- Purpose: define the simplified workflow process used to build Kanbanzai before the tool exists
- Date: 2026-03-18
- Related:
  - `work/design/product-instance-boundary.md`
  - `work/spec/phase-1-specification.md` §22
  - `work/plan/phase-1-decision-log.md` P1-DEC-005, P1-DEC-015

---

## 1. What This Document Is

This document defines **bootstrap-workflow** — the process we follow right now to build Kanbanzai.

It exists because of a fundamental duality: we are building a workflow tool before the workflow tool exists. That creates two distinct workflows that must not be confused:

- **kbz-workflow** — the workflow process the Kanbanzai tool will implement and enforce. Described in `work/design/` and `work/spec/`. This is what we are *building*.
- **bootstrap-workflow** — the simplified process we use right now to build Kanbanzai. Described in this document. This is what we *follow*.

Bootstrap-workflow is a minimal viable near-subset of kbz-workflow. Everything in it should be compatible with the future tool — just simpler, manual, and without automated enforcement.

## 2. Why the Distinction Matters

Without clear separation:

- Agents apply kbz-workflow rules that depend on tooling that doesn't exist yet (MCP operations, automated validation, health checks).
- Humans and agents confuse design documents (describing what the tool will do) with operational instructions (describing what to do now).
- Bootstrap-specific simplifications leak into the tool's design, or kbz-workflow complexity creeps into current practice.

The rule is simple: **when working on this project, follow bootstrap-workflow. When designing or implementing the tool, refer to kbz-workflow.**

## 3. What We Track Today

Bootstrap-workflow tracks the same five entity types as kbz-workflow, but without formal YAML state files, MCP operations, or automated lifecycle enforcement.

| Entity | How we track it today |
|---|---|
| **Epics** | Described in planning documents (`work/plan/`) |
| **Features** | Described in planning documents and the implementation plan |
| **Tasks** | Described in the implementation plan's work breakdown |
| **Bugs** | Filed as issues or tracked in planning documents |
| **Decisions** | Recorded in `work/plan/phase-1-decision-log.md` |

The decision log is the most mature bootstrap artifact — it already follows a structured format that closely mirrors what kbz-workflow will use.

## 4. Which Policies Apply Now

### Fully applicable during bootstrap

These policies describe human and agent behaviour, not tool automation. Follow them now.

**Git commit policy** (`work/design/git-commit-policy.md`):
- Commit message format: `<type>(<object-id>): <summary>`
- Commit type vocabulary: `feat`, `fix`, `docs`, `test`, `refactor`, `workflow`, `decision`, `chore`
- One coherent change per commit
- No force-pushing shared branches, no vague checkpoint commits, no unrelated files
- Object IDs in commit messages are best-effort — use descriptive scope when no formal ID exists (e.g., `docs(bootstrap): ...`)

**Quality gates** (`work/design/quality-gates-and-review-policy.md`):
- Multi-dimensional review (specification conformance, implementation quality, test adequacy, documentation currency, workflow integrity)
- Structured review output when reviewing work
- Agents may propose approval but must not claim human approval occurred

**Agent interaction protocol** (`work/design/agent-interaction-protocol.md`):
- Treat human input as intake, not canonical truth
- Normalise before committing to documents
- Do not fabricate important facts (severity, approval scope, rationale)
- Ask for clarification when ambiguous
- Show normalised results before important commits
- Be explicit about uncertainty
- Respect phase scope

### Partially applicable — follow the intent, not the mechanism

- **"Use the workflow system for canonical changes"** — the intent (don't bypass process) applies. The mechanism (MCP operations) doesn't exist yet. During bootstrap, canonical changes go through reviewed commits.
- **"Preserve traceability"** — use best-effort references to decision IDs, document sections, and descriptive labels. Formal entity IDs don't exist yet.

### Deferred until the tool exists

These depend on kbz tooling and are not part of bootstrap-workflow:

- MCP operations for creating/updating entities
- Automated lifecycle state machine enforcement
- YAML state files under `.kbz/state/`
- Automated health checks and validation
- ID allocation via `id_allocate`
- Document scaffolding via `doc_scaffold`
- Merge gates tied to workflow state
- Multi-agent orchestration and delegation

## 5. Bootstrap Conventions

### Decision records

Continue using the format established in `work/plan/phase-1-decision-log.md`. This format is already close to what kbz-workflow will use and can be migrated directly.

### Document changes

- Design changes go in `work/design/`.
- Spec changes go in `work/spec/`.
- Planning changes go in `work/plan/`.
- Bootstrap-specific process documents go in `work/bootstrap/`.

### Commits during bootstrap

When no formal entity ID exists, use a descriptive scope:

- `docs(design-basis): clarify normalisation rules`
- `decision(P1-DEC-009): ratify minimum required fields`
- `docs(bootstrap): add bootstrap workflow document`

When formal IDs exist (post-tool), switch to them.

### Review during bootstrap

- Human review is the primary gate. The tool cannot enforce review yet.
- Agents should apply the quality gate dimensions when asked to review work.
- Review results are communicated in conversation or commit messages, not stored as workflow records.

## 6. What Bootstrap-Workflow Is Not

- It is **not a permanent process**. It exists only until the kbz tool can take over.
- It is **not a lesser process**. Manual stewardship during bootstrap is a designed property, not a failing.
- It is **not an excuse to skip discipline**. The behavioural rules (commit policy, review policy, agent protocol) apply in full. Only the automated enforcement is deferred.
- It is **not part of the reusable product**. This document describes how we work on this project right now. It is not a template for future users of Kanbanzai.

## 7. Migration Path

When the kbz tool reaches sufficient maturity (per P1-DEC-015, still to be decided):

1. Existing decisions in the decision log are migrated to canonical Decision records.
2. Epics, features, and tasks described in planning documents are created as canonical records.
3. The bootstrap-workflow process is gradually replaced by kbz-workflow.
4. This document is archived or removed.

The goal is not a big-bang migration. It is a gradual transition where bootstrap-workflow shrinks as kbz-workflow grows, until bootstrap-workflow is no longer needed.