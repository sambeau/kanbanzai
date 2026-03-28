# Bootstrap Workflow

- Status: active
- Purpose: define the simplified workflow process used to build Kanbanzai before the tool exists
- Date: 2026-03-18
- Related:
  - `work/design/product-instance-boundary.md`
  - `work/design/document-centric-interface.md`
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
| **Plans** | Described in planning documents (`work/plan/`) |
| **Features** | Described in planning documents and the implementation plan |
| **Tasks** | Described in the implementation plan's work breakdown |
| **Bugs** | Filed as issues or tracked in planning documents |
| **Decisions** | Made in design documents and conversation; indexed by agents in `work/plan/phase-1-decision-log.md` |

Decisions are made through the natural design process — in conversation between the human designer and AI agents, and in design documents. Agents are responsible for extracting decisions from these sources and recording them in the decision log. The human designer does not need to write decision records directly.

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

### Now available via the tool

These features have been implemented in Phase 1, Phase 2a, and Phase 2b. They can be used during bootstrap-workflow through the MCP server or CLI, though manual alternatives remain acceptable:

- MCP operations for creating/updating entities (Phase 1 + Phase 2a)
- Automated lifecycle state machine enforcement (Phase 1)
- YAML state files under `.kbz/state/` (Phase 1)
- Automated health checks and validation (Phase 1 + Phase 2a)
- ID allocation via `id_allocate` (Phase 1)
- Document scaffolding via `doc_scaffold` (Phase 1)
- Knowledge management: contribute, retrieve, and lifecycle management via MCP and CLI (Phase 2b)
- Context profiles and inheritance resolution via MCP and CLI (Phase 2b)
- Context assembly for agent sessions via MCP and CLI (Phase 2b)
- Batch document import via MCP and CLI (Phase 2b)
- Agent capabilities: link suggestions, duplicate detection, extraction guides via MCP (Phase 2b)
- User identity auto-resolution from git config / local config (Phase 2b)

### Still deferred

These depend on tooling not yet implemented:

- Merge gates tied to workflow state
- Multi-agent orchestration and delegation

## 5. Bootstrap Conventions

### Document-centric working

Per `work/design/document-centric-interface.md`, the human interface to the workflow is documents and chat. During bootstrap this means:

- Humans write and review design documents and make decisions in conversation.
- Agents normalise documents on ingest — cleaning language, tightening prose, improving structure — and present the result for human approval before the document becomes canonical.
- Approved documents are returned unchanged on retrieval.
- The formality gradient applies: early documents (proposals, draft designs) are informal prose; later documents (specifications, plans) are formal and precise.

### Decision records

Decisions are made in design documents and in conversation. Agents extract and record them in `work/plan/phase-1-decision-log.md` as an internal tracking artifact. The human designer does not need to write decision records directly — the agent maintains the log.

The format established in the decision log is already close to what kbz-workflow will use and can be migrated directly.

### Document types during bootstrap

The document types defined in `work/design/document-centric-interface.md` apply during bootstrap in simplified form:

| Document type | Bootstrap practice |
|---|---|
| **Proposal** | Written by the human designer, usually pasted into chat |
| **Draft design** | Created collaboratively in chat; stored in `work/design/` when substantive |
| **Design** | Distilled by the agent from drafts and conversation; stored in `work/design/` |
| **Specification** | Created by the agent from approved designs; stored in `work/spec/` |
| **Implementation plan** | Created by the agent from specifications; stored in `work/plan/` |
| **Research report** | Created by agents; stored in `work/research/` |

### Document placement

- Design documents go in `work/design/`.
- Spec documents go in `work/spec/`.
- Planning documents go in `work/plan/`.
- Research reports go in `work/research/`.
- Review reports go in `work/reviews/` — review reports produced by the formal `reviewing` lifecycle gate; one file per reviewed feature or bug.
- General-purpose reports go in `work/reports/` — retrospectives, friction analyses, audit findings, research outputs, progress reports. Does **not** include review lifecycle artifacts (those go in `work/reviews/`).
- Bootstrap-specific process documents go in `work/bootstrap/`.

### Document registration

When you create a new document in `work/`, you must register it with the kanbanzai system. Documents are the human interface, but the system needs metadata records to track status, ownership, and lifecycle.

**Standard workflow:**

1. **Create the markdown file** in the appropriate `work/` subdirectory
2. **Immediately register it** using the MCP tool:
   ```
   doc_record_submit(
     path="work/design/my-document.md",
     type="design",
     title="Human-Readable Title",
     created_by="agent-name"
   )
   ```
3. **Commit both together** — the markdown file and the generated document record in `.kbz/state/documents/`

**For multiple documents (batch import):**

After creating several documents, use the batch import tool:

```
batch_import_documents(
  path="work/plan",
  default_type="dev-plan",
  created_by="agent-name"
)
```

The batch import tool is **idempotent** — it skips already-registered documents and only imports new ones. This makes it safe to run repeatedly as a safety check.

**Type mapping:**

| Location | Type | Notes |
|----------|------|-------|
| `work/design/` | `design` | Design documents, architecture, policies |
| `work/spec/` | `specification` | Formal specifications with acceptance criteria |
| `work/plan/` | `dev-plan` | Implementation plans, decision logs, progress tracking |
| `work/research/` | `research` | Research reports, analysis, exploration |
| `work/reviews/` | `report` | Review reports produced by the formal `reviewing` lifecycle gate; one file per reviewed feature or bug |
| `work/reports/` | `report` | General-purpose reports: retrospectives, friction analyses, audit findings, research outputs, progress reports |

**Why this matters:**

Unregistered documents are invisible to document intelligence, entity extraction, approval workflow, and health checks. Forgetting to register a document breaks the document-centric interface model.

**Safety check before major commits:**

```
batch_import_documents(path="work")
```

This will catch any documents you forgot to register individually.

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

## 6. Workflow Stage Gates

The proper workflow progression is:

**planning → design → features → spec → dev-plan → tasks → developing → reviewing → done**

You must not skip stages. Each stage has a human approval gate.

### Stage 1: Planning (Human-Led)

**What happens:** Human identifies a need or opportunity. Discussion about *whether* to do something and *what* the high-level goal is.

**Output:** Rough consensus that work should proceed to design.

**Agent role:** Answer questions, surface related context, but do not make architectural decisions.

### Stage 2: Design (Human-Led, Agent-Assisted)

**What happens:** Human writes or approves a design document in `work/design/` that describes:
- What the feature/system is
- Why it exists
- How it fits the vision
- Key architectural decisions and tradeoffs

**Output:** An approved design document in `work/design/`.

**Gate:** Design document must exist and be approved before proceeding.

**Agent role:**
- Draft design documents when asked
- Surface design alternatives and tradeoffs
- **DO NOT make technical architecture decisions without human approval**
- **DO NOT create planning documents with embedded design decisions**

**Rules for agents:**
- Before making technology choices (frameworks, libraries, protocols) → stop and ask for design document approval
- Before defining API structures or data models → stop and ask for design document approval
- Before deciding on system boundaries or deployment models → stop and ask for design document approval
- If you find yourself using invented ID patterns like "P5-DES-xxx" → stop, you are making design decisions without approval

### Stage 3: Features (Agent-Assisted)

**What happens:** With an approved design in hand, create a Plan entity (if needed) and extract Feature entities from the design document.

**Output:** Plan entity and Feature entities in `.kbz/state/` (or described in planning documents during bootstrap).

**Gate:** Design document must be approved before creating Plan or Feature entities.

**Agent role:**
- Create Plan entity using `create_plan` (ensure prefix is registered)
- Extract features from approved design using document intelligence
- Create Feature entities with `create_feature` or describe them in planning documents

### Stage 4: Specification (Human-Led, Agent-Assisted)

**What happens:** Write detailed acceptance criteria for each feature. Specifications are binding contracts.

**Output:** Specification document in `work/spec/`.

**Gate:** Features must exist before writing specification.

**Agent role:**
- Draft specification documents when asked
- Ensure acceptance criteria map to features
- Ensure specification is testable and complete

### Stage 5: Dev Plan & Tasks (Agent-Assisted)

**What happens:** Decompose features into tasks, define dependencies, estimate.

**Output:** Dev plan document and Task entities.

**Gate:** Specification must exist and be approved before decomposition.

**Agent role:**
- Use `decompose_feature` to propose task breakdown (when available)
- Create Task entities after human reviews proposal
- Record decisions in decision log

### Stage 6: Implementation and Review (Agent-Driven)

**What happens:** Execute tasks through the `developing` state. When implementation is complete, the feature transitions to `reviewing` for a mandatory code review pass before it can transition to `done`.

**Gates:**
- Tasks must exist before implementation begins.
- **Code review is a mandatory feature lifecycle gate.** A feature must pass through the `reviewing` state before it can transition to `done`. There is no shortcut from `developing` directly to `done`.

**Agent role:** Execute tasks as designed. When implementation is complete, follow the review orchestration procedure in `.skills/code-review.md` — the canonical source for review expectations, per-dimension guidance, finding classification rules, and the full orchestration procedure.

### Emergency Brake

**If you are about to:**
- Write a document in `work/plan/` that contains "Decision:", "Architecture:", "Technology Choice:", or similar design content
- Create entities (Plan, Feature, Task) without an approved design document
- Use decision ID formats that don't exist in the system
- Make technology or architecture choices without human approval

**Then STOP and ask the human:**
- "Should we write a design document for this first?"
- "Is there an approved design that covers this decision?"
- "Which design document should I reference for this work?"

## 7. What Bootstrap-Workflow Is Not

- It is **not a permanent process**. It exists only until the kbz tool can take over.
- It is **not a lesser process**. Manual stewardship during bootstrap is a designed property, not a failing.
- It is **not an excuse to skip discipline**. The behavioural rules (commit policy, review policy, agent protocol) apply in full. Only the automated enforcement is deferred.
- It is **not part of the reusable product**. This document describes how we work on this project right now. It is not a template for future users of Kanbanzai.

## 8. Migration Path

When the kbz tool reaches sufficient maturity (per P1-DEC-015, still to be decided):

1. Existing design documents are ingested into the system; the system extracts and indexes decisions, requirements, and entity relationships. (Batch import is available via `kbz import` or the `doc_import` MCP operation.)
2. Existing decisions in the decision log are reconciled with the decisions extracted from design documents and migrated to canonical Decision records.
3. Plans, features, and tasks described in planning documents are created as canonical records. (The Epic→Plan migration tool is available via `migrate_phase2`.)
4. Batch document import via `kbz import` can ingest multiple documents at once, accelerating migration of existing project materials.
5. The bootstrap-workflow process is gradually replaced by kbz-workflow.
6. This document is archived or removed.

The goal is not a big-bang migration. It is a gradual transition where bootstrap-workflow shrinks as kbz-workflow grows, until bootstrap-workflow is no longer needed.