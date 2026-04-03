# Design: Documentation Editorial Pipeline

| Field | Value |
|-------|-------|
| Date | 2025-07-31 |
| Status | Draft |
| Author | Design Agent, with human review |
| Informed by | `refs/documentation-structure-guide.md`, `refs/technical-writing-guide.md`, `refs/humanising-ai-prose.md`, `refs/punctuation-guide.md` |
| Pipeline reference | `refs/documentation-pipeline.md` |
| Decisions | Open questions 1–4 resolved — see §7 |

---

## 1. Problem Statement

AI-generated documentation has two categories of problem that a single editing pass cannot solve:

1. **Structural problems.** Documents bury key points, ignore audience needs, and follow flat structures instead of the inverted pyramid. These are architectural issues that must be resolved before any prose editing begins.

2. **Prose problems.** AI produces text with a recognisable fingerprint: banned vocabulary ("delve", "leverage", "utilize"), formulaic sentence patterns ("not just X, but Y"), hedging, significance inflation, copula avoidance, em-dash overload, and rigid tricolon rhythm. A single "make it better" pass conflates structural editing, fact-checking, AI-artifact removal, and sentence craft — doing none of them well.

Real-world publishing solves this with a sequential pipeline of specialists: developmental editor, fact-checker, line editor, copy editor, proofreader. Each pass has a narrow focus, clear boundaries, and works at a different scale. The same principle applies to AI agents: narrow, well-defined tasks with explicit checklists produce dramatically better results than broad, multi-objective prompts.

### What happens if nothing changes

- Documents receive a single editorial pass that tries to do everything, producing inconsistent quality.
- AI artifacts survive into published documentation because no stage specifically hunts for them.
- Factual claims go unverified because the editor is busy fixing prose.
- Structural problems are patched at the sentence level instead of resolved architecturally.

---

## 2. Goals and Non-Goals

### Goals

- A five-stage editorial pipeline where each stage has a dedicated SKILL, a focused role, explicit boundaries, and a structured output format.
- Integration into the Kanbanzai workflow system as a lightweight lifecycle — less ceremony than the feature pipeline, more automation, fewer human gates.
- Clean slicing of the four existing styleguides so each stage draws from a non-overlapping subset of guidance.
- Roles and triggers that feel natural when invoked in conversation or by an orchestrator.

### Non-Goals

- Replacing the existing `update-docs` skill, which handles documentation currency (keeping existing docs in sync with code changes). The pipeline handles *new* document creation and *editorial refinement*.
- Diagram or illustration generation. This is a parallel concern flagged by the Edit stage, not a sequential stage in the text pipeline.
- Automated publication or deployment. The pipeline produces a polished document; what happens to it afterward is out of scope.

---

## 3. Design

### 3.1 The pipeline

A document passes through five stages. Each stage works at a smaller scale than the previous one. There is no point polishing a sentence the structural editor will cut, and no point fact-checking a section that might be reorganised.

```
Write  →  Edit  →  Check  →  Style  →  Copyedit
```

| Stage | Job | Scale |
|-------|-----|-------|
| **Write** | Produce a structured first draft following the inverted pyramid | Document |
| **Edit** | Verify and improve structure, scannability, tone gradient, audience fit | Section |
| **Check** | Verify every fact against the implementation, test code examples, flag hallucinations | Claim |
| **Style** | Hunt and eliminate AI writing patterns — banned words, clichés, filler, robotic structure | Paragraph / sentence |
| **Copyedit** | Polish sentences, enforce active voice, simplify punctuation, ensure consistency | Sentence / word |

The principle is **large to small**: structural decisions before content verification, content verification before prose cleanup, prose cleanup before sentence-level polish. Each stage trusts that the previous stage did its job.

### 3.2 Stage boundaries

The most important property of the pipeline is that each stage knows what it owns and what it must not touch. Boundary discipline prevents stages from drifting into each other's territory, which is the primary failure mode in multi-pass editing.

**Write** owns document purpose, structure, outline, audience assumptions, content, and the inverted pyramid at every level. It does not attempt to polish prose — imperfect sentences are fine at this stage.

**Edit** owns document-level and section-level structure: heading skeleton, inverted-pyramid compliance, scannability, tone gradient, and structural tells (inline-header lists, title-case headings, emoji bullets). It does not touch individual sentences, word choice, or punctuation. If it finds a factual issue, it flags it for the Check stage.

**Check** owns factual accuracy: source-of-truth compliance, code example verification, hallucination detection, and substance assessment (vague claims, significance inflation, promotional language). Its output is a list of classified findings, not a rewritten document. It does not restructure or restyle.

**Style** owns AI-specific artifacts: banned vocabulary, inflated adjectives, faux-insider phrases, staccato rhetoric, hedging, tricolon overuse, the rigid 1-2-3 paragraph formula, robotic transitions, elegant variation (thesaurus syndrome), and copula avoidance. It rewrites sentences to eliminate these patterns but does not restructure sections or change factual content.

**Copyedit** owns sentence clarity: active/passive voice, smothered verbs, sentence length, punctuation, capitalisation, abbreviations, parallel structure, consistency, and reading rhythm. It trusts that everything above the sentence level is already correct.

### 3.3 Styleguide slicing

Each stage draws from a different, non-overlapping subset of the four styleguides. No section appears in two SKILLs.

| Stage | Primary reference | Also draws from |
|-------|-------------------|-----------------|
| **Write** | `documentation-structure-guide.md` (full) | `technical-writing-guide.md` §1 (voice), §4 (word choice) |
| **Edit** | `documentation-structure-guide.md` §1, 4–6, 8 | `technical-writing-guide.md` §8, 10, 11; `humanising-ai-prose.md` §6 (structural tells) |
| **Check** | `humanising-ai-prose.md` §5 (content and substance) | `documentation-structure-guide.md` §3 (source of truth) |
| **Style** | `humanising-ai-prose.md` §1–3, 7, 9 | `technical-writing-guide.md` §4 (word choice) |
| **Copyedit** | `punctuation-guide.md` (full) | `technical-writing-guide.md` §2–3, 5–7; `humanising-ai-prose.md` §4 (punctuation tells) |

Each SKILL contains a distilled version of its relevant guidance inline (vocabulary, anti-patterns, checklist, procedure). The full guides remain available as Level 3 references when deeper detail is needed.

### 3.4 SKILLs

Each stage has a corresponding SKILL following the project's evidence-based format: vocabulary payload, named anti-patterns with Detect/BECAUSE/Resolve, checklist, numbered procedure, structured output format, BAD/GOOD examples, weighted evaluation criteria, and retrieval-anchor questions.

| Stage | SKILL | Lines | Key content |
|-------|-------|-------|-------------|
| Write | [`.kbz/skills/write-docs/SKILL.md`](../../.kbz/skills/write-docs/SKILL.md) | 260 | 16 vocabulary terms, 5 anti-patterns, 9-step procedure from purpose statement through fact verification to opening-last |
| Edit | [`.kbz/skills/edit-docs/SKILL.md`](../../.kbz/skills/edit-docs/SKILL.md) | 369 | 13 vocabulary terms, 6 anti-patterns, 10-step procedure from heading-skeleton extraction through structural-tell detection |
| Check | [`.kbz/skills/check-docs/SKILL.md`](../../.kbz/skills/check-docs/SKILL.md) | 317 | 12 vocabulary terms, 5 anti-patterns, 10-step procedure from claim enumeration through classification |
| Style | [`.kbz/skills/style-docs/SKILL.md`](../../.kbz/skills/style-docs/SKILL.md) | 372 | 15 vocabulary terms, 5 anti-patterns, 10-step procedure from read-first through banned-word scan to read-aloud test |
| Copyedit | [`.kbz/skills/copyedit-docs/SKILL.md`](../../.kbz/skills/copyedit-docs/SKILL.md) | 308 | 14 vocabulary terms, 5 anti-patterns, 10-step procedure from voice correction through punctuation to rhythm check |

All five SKILLs are under the 500-line limit. Their anti-pattern sets are disjoint — no two stages share an anti-pattern, reinforcing the boundary discipline.

### 3.5 Roles

Each stage maps to a role that defines the agent's identity and vocabulary. Following the project's conventions (DP-3: brief identities, real job titles, no flattery), each role is under 50 tokens of identity.

The Write stage uses the existing `documenter` role. The four editorial stages each get a new role:

| Stage | Role ID | Identity | Rationale |
|-------|---------|----------|-----------|
| Write | `documenter` | Senior technical writer | Existing role. Already carries documentation vocabulary and anti-patterns for drift, stale examples, assumed knowledge, and duplication. |
| Coordinate | `doc-pipeline-orchestrator` | AI content editor | The orchestrator for the full pipeline. "AI content editor" matches the emerging industry title for the person who shepherds AI-generated content through editorial stages — reviewing, coordinating, and deciding when a document is done. Carries vocabulary around stage sequencing, change-tracking, re-entry decisions, and checkpoint management. |
| Edit | `doc-editor` | Developmental editor | The publishing industry term for the person who evaluates and improves structure, organisation, and argument flow — distinct from line editing or copy editing. "Developmental" routes to structural-assessment knowledge. |
| Check | `doc-checker` | Technical fact-checker | Combines the QA specialist and fact-checker functions from the proposal. "Fact-checker" is the industry term; "technical" scopes it to implementation verification rather than journalistic fact-checking. |
| Style | `doc-stylist` | AI prose editor | A novel role with no direct publishing analogue. "AI prose editor" is honest about the job — this person specialises in recognising and eliminating machine-generated writing patterns. The vocabulary payload (banned words, fingerprint clusters, copula avoidance) does the real routing work. |
| Copyedit | `doc-copyeditor` | Senior copy editor | The standard publishing term. Copy editors fix grammar, punctuation, voice, and consistency at the sentence level. "Senior" follows the project convention for editorial roles. |

**Role vocabulary design.** Each role carries vocabulary terms specific to its editorial function. The `doc-editor` vocabulary includes "heading skeleton", "inverted pyramid compliance", "scannability". The `doc-stylist` vocabulary includes "AI fingerprint cluster", "banned word", "copula avoidance". The `doc-copyeditor` vocabulary includes "smothered verb", "em-dash hygiene", "serial comma". This vocabulary separation is what makes the roles useful — it routes each agent to the right region of its knowledge space.

**Trigger words.** Each SKILL defines natural-language triggers that activate it in conversation or orchestration. The triggers are designed to be distinct across stages (no two stages share a trigger phrase) and to match how a human would naturally request the work:

| Stage | Triggers |
|-------|----------|
| **Write** | "write documentation", "draft a document", "create a README", "write a getting-started guide", "write a manual", "author technical documentation" |
| **Edit** | "edit document structure", "review document organisation", "check document architecture", "developmental edit", "structural review of documentation" |
| **Check** | "fact-check documentation", "verify document accuracy", "check docs for hallucinations", "QA documentation", "validate technical claims" |
| **Style** | "humanise AI prose", "remove AI artifacts", "style-check documentation", "strip AI clichés", "clean up AI-generated text" |
| **Copyedit** | "copy edit documentation", "polish prose", "fix passive voice", "simplify sentences", "proofread documentation", "final edit" |

The triggers progress from creation language ("write", "draft", "create") through evaluation language ("edit", "review", "check", "verify") to refinement language ("humanise", "strip", "polish", "simplify", "proofread"). This mirrors the pipeline's large-to-small progression.

### 3.6 Workflow integration

The documentation pipeline integrates into the Kanbanzai workflow system but with a lighter touch than the feature lifecycle. The rationale: documentation editing is lower-risk than code changes, each stage's output is human-readable prose (not compiled artifacts), and the pipeline's sequential structure already provides quality gates at every boundary.

#### Stage binding: `doc-publishing`

A new stage binding covers the full pipeline as an orchestrated workflow:

```yaml
doc-publishing:
  description: "Running a document through the editorial pipeline"
  orchestration: pipeline-coordinator
  roles: [doc-pipeline-orchestrator]
  skills: [orchestrate-doc-pipeline]
  human_gate: false
  effort_budget: "5-15 tool calls per stage"
  sub_agents:
    roles: [documenter, doc-editor, doc-checker, doc-stylist, doc-copyeditor]
    skills: [write-docs, edit-docs, check-docs, style-docs, copyedit-docs]
    topology: sequential
  notes: >
    Pipeline coordinator — lighter than a full orchestrator. Sequential
    dispatch only, no parallelism. No mandatory human gates; advisory
    checkpoints after Edit and after Copyedit. The coordinator passes
    the document and the previous stage's changelog to each successive
    stage, handles re-entry when a later stage flags an earlier-stage
    problem, and collates all changelogs into a completion summary.
```

#### Automation over approval

The feature pipeline has human gates at specifying, dev-planning, and reviewing — three mandatory approval points. The documentation pipeline has **zero mandatory gates**. Instead it uses advisory checkpoints:

| Feature pipeline | Documentation pipeline | Rationale |
|-----------------|----------------------|-----------|
| Design → **human approval** → Spec | Write → Edit (automatic) | Structural editing is safe to automate. The Edit stage can only rearrange and flag — it cannot introduce incorrect content. |
| Spec → **human approval** → Dev plan | Edit → Check (automatic) | Fact-checking against implementation is deterministic. The Check stage flags issues; it does not rewrite. |
| Dev plan → **human approval** → Develop | Check → Style (automatic) | AI-artifact removal operates on a concrete checklist (banned words, patterns). Over-correction is guarded by the "What not to fix" list in the SKILL. |
| Review → **human approval** → Done | Style → Copyedit (automatic) | Sentence-level editing is low-risk. The Copyedit stage cannot change meaning — its anti-patterns explicitly guard against this. |

**Recommended human checkpoints** (not gates — the pipeline continues without them):

1. **After Edit** — structural decisions are expensive to undo. If the editor reorganised sections in a way you disagree with, this is the cheapest point to catch it.
2. **After Copyedit** — a final read-through of the finished product before publication.

The orchestrator can promote either checkpoint to a hard gate for high-stakes documents (public-facing documentation, API references) by setting `human_gate: true` on the relevant stage.

#### Individual stage invocation

The pipeline doesn't have to run end-to-end. Each stage can be invoked independently:

- "Fact-check this document" → triggers `check-docs` alone.
- "Style-check this README" → triggers `style-docs` alone.
- "Copy edit the spec" → triggers `copyedit-docs` alone.

This is useful when a document was written by a human and only needs specific editorial passes, or when re-running a single stage after addressing findings from a previous run.

#### Change tracking

Each stage outputs three things:

1. **The revised document** (or, for the Check stage, the original with annotations).
2. **A changelog** — what was changed and why.
3. **Flags** — issues outside this stage's scope that a previous or later stage should address.

The changelog is the key automation enabler. It lets the human reviewer scan the delta rather than re-reading the entire document, and it lets the orchestrator detect when a later stage is flagging problems that belong to an earlier stage (indicating a need for re-entry).

#### Re-entry rules

When a later stage discovers a problem that belongs to an earlier stage:

- **Check finds a structural problem** → flags it. If severe (e.g. entire section is misplaced), the orchestrator sends the document back to Edit.
- **Style finds a factual issue** → flags it. The orchestrator sends it to Check.
- **Any stage finds the document unsalvageable** → sends it back to Write.

Re-entry should be rare. Frequent re-entry indicates the Write stage needs better guidance or the source material needs improvement. The orchestrator logs re-entry events for retrospective analysis.

#### Idempotency

Every stage is designed to be safe to run twice. Running the copy editor on already-copyedited text should not degrade the output. This is a testable property: run each stage twice in succession and diff the results. The second pass should produce minimal or no changes.

---

## 4. Alternatives Considered

### 4.1 Three stages instead of five

Collapsing Style into Copyedit and Edit into Write would produce a simpler three-stage pipeline: Write → Check → Polish.

**Rejected because:** The Style and Copyedit stages target fundamentally different problems. Style hunts AI-specific artifacts (banned words, fingerprint clusters, copula avoidance) using a concrete trigger list. Copyedit handles traditional prose craft (voice, punctuation, rhythm). Combining them produces a SKILL with too many objectives — the n=19 cliff from the skills system research shows that adding rules past a threshold actively hurts compliance. Similarly, combining Write and Edit means the writer is simultaneously creating content and evaluating structure, which produces neither good content nor good structure.

### 4.2 Mandatory human gates at every stage

Placing a human approval requirement between each stage, mirroring the feature pipeline.

**Rejected because:** Documentation editing is lower-risk than code changes. Each stage's output is readable prose, not compiled artifacts that could break a build. The pipeline's sequential structure already provides quality boundaries. Mandatory gates at every stage would make the pipeline too slow for practical use — five approvals to publish a README. Advisory checkpoints at two points (after Edit, after Copyedit) provide sufficient human oversight without bottlenecking the workflow.

### 4.3 Parallel editorial stages

Running Check, Style, and Copyedit in parallel on the output of Edit, then merging.

**Rejected because:** The stages have ordering dependencies. The Style stage removes AI artifacts (banned words, filler phrases, formulaic structures), which changes the sentences. If Copyedit runs in parallel on the pre-Style text, its voice and punctuation fixes will be applied to sentences that Style is about to rewrite. Merging parallel edits to the same sentences is a conflict-resolution problem with no clean solution. Sequential execution avoids this entirely.

### 4.4 A single omnibus SKILL with all editorial guidance

One large "edit-docs" SKILL containing everything: structure, facts, AI artifacts, and prose.

**Rejected because:** This is the status quo, and it produces the problems described in the problem statement. A single pass trying to do everything does nothing well. The research basis for the skills system (Vaarta Analytics, 2026) shows that prompt accuracy at 19 requirements drops below accuracy at 5 requirements. Five focused SKILLs of 12–16 vocabulary terms each stay well within the effective range; one combined SKILL of 60+ terms would not.

---

## 5. Dependencies

### Existing infrastructure (no changes needed)

- **SKILL system.** The five SKILLs follow the existing evidence-based format and are already created.
- **Stage binding system.** The new `doc-publishing` binding follows the existing schema.
- **Role system.** New roles follow the existing YAML format with inheritance from `base`.
- **Orchestrator pattern.** The existing `orchestrator-workers` topology supports sequential sub-agent dispatch.

### New artifacts to create

| Artifact | Path | Status |
|----------|------|--------|
| Pipeline overview | `refs/documentation-pipeline.md` | ✅ Created |
| Write SKILL | `.kbz/skills/write-docs/SKILL.md` | ✅ Created |
| Edit SKILL | `.kbz/skills/edit-docs/SKILL.md` | ✅ Created |
| Check SKILL | `.kbz/skills/check-docs/SKILL.md` | ✅ Created |
| Style SKILL | `.kbz/skills/style-docs/SKILL.md` | ✅ Created |
| Copyedit SKILL | `.kbz/skills/copyedit-docs/SKILL.md` | ✅ Created |
| `doc-editor` role | `.kbz/roles/doc-editor.yaml` | ✅ Created |
| `doc-checker` role | `.kbz/roles/doc-checker.yaml` | ✅ Created |
| `doc-stylist` role | `.kbz/roles/doc-stylist.yaml` | ✅ Created |
| `doc-copyeditor` role | `.kbz/roles/doc-copyeditor.yaml` | ✅ Created |
| `doc-pipeline-orchestrator` role | `.kbz/roles/doc-pipeline-orchestrator.yaml` | ✅ Created |
| Orchestration SKILL | `.kbz/skills/orchestrate-doc-pipeline/SKILL.md` | ✅ Created |
| Stage binding entry | `.kbz/stage-bindings.yaml` | ✅ Added |

### Relationship to existing documentation skills

The pipeline complements rather than replaces the existing `update-docs` skill. `update-docs` handles documentation currency — keeping existing documents in sync with code changes. The pipeline handles editorial refinement of new or substantially revised documents. They serve different triggers and can coexist.

---

## 6. Implementation Status

### Phase 1: Roles — ✅ Complete

Five role YAML files created, each under 50 lines: `doc-editor` (developmental editor), `doc-checker` (technical fact-checker), `doc-stylist` (AI prose editor), `doc-copyeditor` (senior copy editor), and `doc-pipeline-orchestrator` (AI content editor).

### Phase 2: Stage binding and orchestration — ✅ Complete

The `doc-publishing` entry added to `.kbz/stage-bindings.yaml`. The `orchestrate-doc-pipeline` SKILL (240 lines) coordinates sequential dispatch of the five stages, handles change tracking, manages re-entry, and implements the advisory checkpoint protocol.

### Phase 3: Testing and calibration — next

Run real documents through the pipeline. Identify where stages over-reach their boundaries, where the banned-word lists need updating, where the fact-checker misses claims or flags false positives, and where the copyeditor changes meaning. Adjust SKILLs based on observed behaviour.

This phase is explicitly ongoing — editorial calibration is never finished.

---

## 7. Decisions (formerly Open Questions)

### 7.1 Orchestration scope → lightweight pipeline coordinator

**Decision:** A lightweight sequential coordinator, not a full orchestrator.

The feature pipeline orchestrator makes complex decisions — parallel dispatch, conflict detection, dependency ordering, saturation limits. The documentation pipeline has none of that. It is a straight line: stage 1 finishes, start stage 2.

The coordinator's responsibilities are:

- **Sequential dispatch** — pass the document and the previous stage's changelog to the next stage.
- **Re-entry logic** — when a stage flags an issue for an earlier stage, decide whether to loop back or continue and note it.
- **Advisory checkpoints** — after Edit and after Copyedit, offer the human a chance to review (but don't block).
- **Completion summary** — collate all five changelogs into a single report showing what happened at each stage.

No parallelism, no conflict detection, no saturation management. The role is `doc-pipeline-orchestrator` with identity "AI content editor" — matching the emerging industry title for the person who shepherds AI-generated content through editorial stages.

### 7.2 Document tracking → doc records, not entities

**Decision:** Use `doc(action: register)` for tracking. No full entity lifecycle.

The feature entity lifecycle (proposed → designing → specifying → developing → reviewing → done) does not map to editorial stages. Forcing "being fact-checked" and "being copy-edited" into a lifecycle designed for code development would require either awkward state names or a parallel lifecycle system.

What `doc(action: register)` already provides:

- Document status tracking (draft → approved → superseded).
- Ownership and content hashing for drift detection.
- Visibility via `doc(action: list)`.

What the pipeline coordinator adds on top:

- Per-stage changelogs showing what happened and why.
- A completion summary collating all stages.
- Flags for issues that need human attention.

This is sufficient for the current documentation volume. If documentation volume grows to the point where dozens of documents are in flight simultaneously, entity tracking could be reconsidered — but that scenario is unlikely given documentation tends to happen in distinct phases rather than continuously.

### 7.3 Feedback to the Write stage → yes, with guardrails

**Decision:** Build a human-reviewed proposal mechanism, not an automatic feedback loop.

When the Check stage consistently flags the same class of problem (e.g. vague performance claims appearing in 60% of recent documents), that pattern should feed back into the Write SKILL's anti-patterns. This closes the loop: downstream stages improve upstream quality over time.

**Mechanism:**

1. The pipeline coordinator already collates changelogs. After a natural cadence (every 10 documents, or quarterly), run a retrospective synthesis across accumulated Check-stage and Style-stage findings.
2. Identify recurring classifications. The existing `retro` tool can drive this.
3. Propose new anti-patterns for the Write SKILL, with BECAUSE clauses referencing the evidence.
4. A human reviews and accepts or rejects the proposal.

**Guardrails:**

- **No auto-insertion.** The feedback loop proposes anti-patterns; a human decides whether to accept them. This prevents overfitting to recent documents.
- **Generalise, don't overfit.** The lesson from "three documents had vague latency claims" is "always substantiate performance claims with measurements", not "never mention latency". Proposals must be the generalised version.
- **Cap anti-pattern count.** The Write SKILL stays at 5–8 anti-patterns (respecting the n=19 cliff from the skills research). Every addition must either replace or merge with an existing anti-pattern, not accumulate alongside it.
- **Distinguish source problems from writer problems.** A cluster of vague-claim findings might reflect thin source material (a vague design document) rather than a systemic Write-stage failure. The retrospective synthesis must distinguish these.

### 7.4 Selective pipeline → no, all stages for v1

**Decision:** Every document runs through all five stages. No stage skipping.

Even human-written documents benefit from the full pipeline. The Edit stage catches structural problems that humans create just as readily as AI does — burying the key point under background context is a universal authoring habit. The Style stage may find fewer AI artifacts in human text, but it still catches inflated adjectives and significance inflation. The Copyedit stage is where developer-authored documentation needs the most help.

Running all five stages also keeps the coordinator simple: no conditional logic for stage skipping, no validation of skip requests, no edge cases around what happens when you skip Check but Style finds a factual issue.

Individual stage invocation (§3.6) remains available for targeted re-runs — "re-run the copy edit after I've addressed the findings" — but the default pipeline always runs all five stages in order.