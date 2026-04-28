# Specification: Corpus Hygiene and Classification Pipeline

**Document ID:** (assigned on registration)
**Feature:** FEAT-01KPTHB61WPT0 — Corpus hygiene and classification pipeline
**Status:** Draft
**Design source:** `work/design/doc-intel-adoption-design.md` §4 (Fix 2) and §5 (Fix 3)

---

## Overview

This specification covers two coordinated changes to the Kanbanzai agent skill files: a mandatory corpus integrity check integrated into the session-start workflow (`kanbanzai-getting-started/SKILL.md`), and repositioning of Layer 3 classification as an immediate obligation on document registration rather than a deferred batch task (`kanbanzai-documents/SKILL.md`), together with a classification sub-step added to the review orchestration prerequisite phase (`orchestrate-review/SKILL.md`). Together these changes ensure that (a) the document corpus is structurally complete before any design work begins, and (b) classification is treated as a first-class, non-deferrable step in every document lifecycle event, so that concept search and role-based retrieval remain reliable across the full project history.

---

## Scope

### In scope

- Changes to `.agents/skills/kanbanzai-getting-started/SKILL.md`:
  - A mandatory corpus integrity check step added to the Session Start Checklist.
  - A new-project onboarding procedure section.
  - An existing-project adoption procedure section.
- Changes to `.agents/skills/kanbanzai-documents/SKILL.md`:
  - Relocation of the "Batch Classification Protocol" section to immediately after the Registration section (before Drift and Refresh).
  - Renaming the section from "Batch Classification Protocol" to "Classification (Layer 3)".
  - Reframing the section opening to make classification an immediate obligation.
  - Addition of the `classification_nudge` mandate.
  - Addition of a classification checkbox to the Document Creation Checklist.
- Changes to `.kbz/skills/orchestrate-review/SKILL.md`:
  - Addition of sub-step 1b to Step 1 (Verify prerequisites): classify unclassified feature documents before dispatching reviewers.
  - Addition of a classification entry to the orchestrate-review checklist.

### Out of scope

- Changes to `write-design/SKILL.md` or any design-stage skill.
- Changes to `implement-task/SKILL.md` or `orchestrate-development/SKILL.md`.
- Changes to `kanbanzai-agents/SKILL.md`.
- Go server code changes (no new MCP tool endpoints, no schema changes).
- Knowledge retrieval mandates at implementation time (Fix 4).
- Access instrumentation (Fix 5).
- Plan close-out knowledge curation (Fix 6).
- Changes to document templates.
- Changes to `reviewer-conformance/SKILL.md` or any other reviewer skill beyond `orchestrate-review`.

---

## Functional Requirements

### Getting-started: corpus integrity check

**FR-001** The `kanbanzai-getting-started/SKILL.md` Session Start Checklist MUST include a "Corpus integrity check" item that requires the agent to call `doc(action: "audit")` and review its output before proceeding to claim a task.

**FR-002** The getting-started skill MUST specify that if the audit output shows files on disk that are not registered, the agent MUST call `doc(action: "import", path: "work")` to register all unregistered documents in configured roots.

**FR-003** The getting-started skill MUST specify that if the audit output shows registered document records whose files are missing from disk, the agent MUST call `doc(action: "delete", id: "DOC-xxx")` for each such stale record.

**FR-004** The getting-started skill MUST specify that after any batch registration triggered by the integrity check, the agent MUST run a classification pass on the newly registered documents before proceeding (referencing the Classification (Layer 3) section in `kanbanzai-documents`).

**FR-005** The getting-started skill MUST include a rationale statement explaining that an incomplete corpus produces false negatives in design searches, and that a designer who finds no results cannot distinguish "not addressed" from "not registered."

### Getting-started: new-project onboarding

**FR-006** The getting-started skill MUST include a new-project onboarding procedure that applies when Kanbanzai is initialised on a project for the first time (no prior `.kbz/` directory).

**FR-007** The new-project onboarding procedure MUST include the following steps in order:
1. Configure document roots in `.kbz/config.yaml`.
2. Run `doc(action: "import", path: "<each-root>")` for each configured root.
3. Verify with `doc(action: "audit")` targeting 0 unregistered files.
4. Run batch classification prioritised: specifications first, designs second, dev-plans third, research and reports last.
5. Validate with `doc_intel(action: "find", role: "decision")` — if results are returned, the concept registry is populated.

**FR-008** The new-project onboarding procedure MUST include a time estimate of approximately 5–10 minutes per document for classification, with a note that a 50-document corpus requires roughly 4–8 hours of agent time for full classification.

### Getting-started: existing-project adoption

**FR-009** The getting-started skill MUST include an existing-project adoption procedure that applies when Kanbanzai is added to a project that already has documentation outside standard `work/` directories.

**FR-010** The existing-project adoption procedure MUST include the following steps in order:
1. Audit the repository for markdown files not covered by configured roots using `find . -name "*.md" | grep -v ".kbz"`.
2. Decide which documents belong in the corpus (design decisions, specifications, architectural rationale, requirements).
3. Add additional roots to `.kbz/config.yaml` as needed.
4. Register and classify per the new-project onboarding procedure.

**FR-011** The existing-project adoption procedure MUST include the key principle that the corpus must be complete enough that a negative search result means "this has not been addressed" rather than "this might have been addressed in an unregistered document."

### Documents: classification section relocation and reframing

**FR-012** The `kanbanzai-documents/SKILL.md` classification section MUST be positioned immediately after the Registration section and before the Drift and Refresh section.

**FR-013** The classification section MUST be titled "Classification (Layer 3)" — the title "Batch Classification Protocol" MUST NOT appear.

**FR-014** The classification section MUST open with the imperative: "After registering a document, classify it immediately if you have the document content in context. Do not defer classification to a batch run."

**FR-015** The classification section MUST state that the `doc register` response includes a `classification_nudge` and that agents MUST follow it before moving to the next task.

**FR-016** The classification section MUST include a rationale statement that Layer 3 classification is what enables concept search, role-based retrieval, and semantic guides, and that documents deferred to batch runs accumulate as a growing backlog that is never fully cleared.

### Documents: creation checklist update

**FR-017** The Document Creation Checklist in `kanbanzai-documents/SKILL.md` MUST include the following checkbox item: `[ ] Classified the document with doc_intel(action: "classify") if content was in context`.

### Orchestrate-review: classification prerequisite

**FR-018** The `orchestrate-review/SKILL.md` Step 1 (Verify prerequisites) MUST include a sub-step 1b that directs the orchestrator to call `doc_intel(action: "pending")` and filter results for documents owned by the feature under review.

**FR-019** Sub-step 1b MUST specify that for each unclassified document identified, the orchestrator MUST:
1. Call `doc_intel(action: "guide", id: "DOC-xxx")` to obtain the outline and content hash.
2. Read the sections needed to understand the content.
3. Call `doc_intel(action: "classify", id: "DOC-xxx", content_hash: "...", ...)` to submit classifications.

**FR-020** Sub-step 1b MUST specify that classification is NOT a blocking prerequisite: if the context budget is exhausted, the orchestrator MUST proceed with reviewing rather than stopping.

**FR-021** Sub-step 1b MUST specify the classification priority order for the review context: specification → design → dev-plan.

**FR-022** Sub-step 1b MUST include a rationale statement that reviewer sub-agents use `doc_intel` to navigate, that Layer 3 classification enables role-based search and produces richer guides, and that an unclassified corpus forces reviewers to fall back to structural navigation only.

**FR-023** The `orchestrate-review/SKILL.md` checklist MUST include the following item: `[ ] Classified unclassified feature documents (or confirmed context budget insufficient)`.

---

## Non-Functional Requirements

**NFR-001** All changes MUST be confined to the three specified skill files. No other files in the repository MUST be modified by this feature.

**NFR-002** All modified skill files MUST remain valid YAML front matter followed by Markdown content. The front matter `version` field in each modified skill MUST be incremented.

**NFR-003** No new MCP tool calls, endpoints, or server-side changes are introduced. All requirements are expressed as procedure changes to skill documents and MUST be implementable using only existing `doc`, `doc_intel`, and shell tools.

**NFR-004** The new corpus integrity check step MUST be positioned in the Session Start Checklist such that it executes after the store check (commit orphaned `.kbz/` files) and before the "Check the work queue" step.

**NFR-005** The classification section relocation in `kanbanzai-documents/SKILL.md` MUST preserve all existing content in the section (priority ordering, step-by-step procedure, anti-patterns), supplemented only by the new framing required by FR-014, FR-015, and FR-016.

**NFR-006** The orchestrate-review sub-step 1b MUST be inserted between the existing steps 1a (confirm feature is in `reviewing` status) and the existing step that locates specification documents, without altering the numbering logic of subsequent prerequisite steps beyond renumbering as needed.

---

## Acceptance Criteria

- [ ] **FR-001** — `kanbanzai-getting-started/SKILL.md` Session Start Checklist contains a "Corpus integrity check" item that references `doc(action: "audit")`.
- [ ] **FR-002** — Getting-started skill specifies calling `doc(action: "import", path: "work")` when audit shows unregistered on-disk files.
- [ ] **FR-003** — Getting-started skill specifies calling `doc(action: "delete")` for each stale record when audit shows registered files missing from disk.
- [ ] **FR-004** — Getting-started skill specifies running a classification pass after any batch registration triggered by the integrity check.
- [ ] **FR-005** — Getting-started skill includes a rationale statement about false negatives from an incomplete corpus.
- [ ] **FR-006** — Getting-started skill contains a new-project onboarding procedure section.
- [ ] **FR-007** — New-project onboarding procedure lists all five steps in the correct order (configure roots → import → audit → classify → validate).
- [ ] **FR-008** — New-project onboarding procedure includes the 5–10 minutes per document and 4–8 hours for 50 documents estimates.
- [ ] **FR-009** — Getting-started skill contains an existing-project adoption procedure section.
- [ ] **FR-010** — Existing-project adoption procedure lists all four steps in the correct order (find → decide → configure → register-and-classify).
- [ ] **FR-011** — Existing-project adoption procedure states the key principle about negative results meaning "not addressed" vs "not registered."
- [ ] **FR-012** — In `kanbanzai-documents/SKILL.md`, the classification section appears immediately after Registration and before Drift and Refresh.
- [ ] **FR-013** — Classification section is titled "Classification (Layer 3)" and the old title "Batch Classification Protocol" does not appear in the skill.
- [ ] **FR-014** — Classification section opens with the specified imperative about classifying immediately after registration.
- [ ] **FR-015** — Classification section states that the `classification_nudge` in the `doc register` response MUST be followed before moving to the next task.
- [ ] **FR-016** — Classification section includes the rationale about deferred documents accumulating as a backlog that is never fully cleared.
- [ ] **FR-017** — Document Creation Checklist in `kanbanzai-documents/SKILL.md` includes the classification checkbox item.
- [ ] **FR-018** — `orchestrate-review/SKILL.md` Step 1 contains sub-step 1b calling `doc_intel(action: "pending")` filtered to the feature under review.
- [ ] **FR-019** — Sub-step 1b specifies the three-step per-document procedure (guide → read → classify).
- [ ] **FR-020** — Sub-step 1b explicitly states that classification is NOT a blocking prerequisite and that the orchestrator MUST proceed if context budget is exhausted.
- [ ] **FR-021** — Sub-step 1b specifies the classification priority order: specification → design → dev-plan.
- [ ] **FR-022** — Sub-step 1b includes the rationale about reviewer sub-agents and semantic context.
- [ ] **FR-023** — `orchestrate-review/SKILL.md` checklist contains the classification item with the "or confirmed context budget insufficient" qualifier.

---

## Dependencies and Assumptions

**DEP-001** The design document `work/design/doc-intel-adoption-design.md` MUST be approved before implementation begins. All requirements in this specification are derived from §4 and §5 of that document.

**DEP-002** The `doc(action: "audit")` tool MUST return a list distinguishing between (a) files on disk not registered, and (b) registered records whose files are missing. The integrity check procedure depends on this output structure.

**DEP-003** The `doc(action: "import")` tool MUST be idempotent — already-registered documents are skipped. The getting-started integrity check relies on this to avoid double-registration.

**DEP-004** The `doc register` response MUST include a `classification_nudge` field. The FR-015 requirement that agents follow this nudge is only meaningful if the field is present in the response.

**DEP-005** The `doc_intel(action: "pending")` tool MUST support filtering by owning feature or document owner so that orchestrate-review sub-step 1b can scope the pending list to the feature under review.

**ASM-001** The three skill files modified by this feature are the authoritative source of agent behaviour for their respective contexts. Changes to skill files take effect for all subsequent agent sessions without any deployment step.

**ASM-002** Agents following the getting-started skill read the Session Start Checklist top-to-bottom and execute steps in order. Positioning the corpus integrity check after the store check and before the work queue check (NFR-004) is therefore sufficient to ensure it runs before task selection.

**ASM-003** The context budget for classification in sub-step 1b of orchestrate-review is expected to be limited. The non-blocking clause (FR-020) is required because classification should not prevent review from proceeding; it is a best-effort improvement to corpus quality.

**ASM-004** "Layer 3 classification" refers exclusively to the `doc_intel(action: "classify")` call that assigns semantic roles to document sections. It does not refer to document type assignment or document record metadata.
```

Now let me register this document with the doc system.