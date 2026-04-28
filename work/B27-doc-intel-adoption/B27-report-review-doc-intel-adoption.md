# Plan Review: Document Intelligence Adoption (P27)
**Plan:** P27-doc-intel-adoption
**Status at review:** reviewing → done
**Date:** 2026-04-22
**Reviewer:** reviewer-conformance

---

## 1. Verdict

**PASS WITH FINDINGS**

All four features reached `done` status. All six design fixes are delivered and
spec-conformant. No blocking findings were raised in any feature review. The test suite
(29 packages) passes cleanly on main after all four squash-merges. One post-merge
regression was identified and resolved in commit `ff5b940` in the same session.

Remaining findings are non-blocking follow-up items; none prevent advancement to `done`.

---

## 2. Feature Census

| Feature ID | Slug | Terminal Status | Verdict |
|---|---|---|---|
| FEAT-01KPTHB5Z1H7T | design-stage-consultation | done | ✅ |
| FEAT-01KPTHB61WPT0 | corpus-hygiene-classification | done | ✅ |
| FEAT-01KPTHB649DPK | knowledge-lifecycle-mandate | done | ✅ |
| FEAT-01KPTHB66Y8TM | doc-intel-instrumentation | done | ✅ |

All four features are in terminal (`done`) status. No feature is pending, blocked, or
in a non-terminal state.

---

## 3. Fix-to-Feature Traceability Matrix

| Fix | Design Section | Delivering Feature | Delivery Status |
|---|---|---|---|
| Fix 1: Design-stage corpus consultation mandate | §3 | FEAT-01KPTHB5Z1H7T | ✅ Delivered |
| Fix 2: Corpus completeness and session-start integrity check | §4 | FEAT-01KPTHB61WPT0 | ✅ Delivered |
| Fix 3: Classification as immediate obligation | §5 | FEAT-01KPTHB61WPT0 | ✅ Delivered |
| Fix 4: Mandatory knowledge retrieval at implementation time | §6 | FEAT-01KPTHB649DPK | ✅ Delivered |
| Fix 5: Access instrumentation | §7 | FEAT-01KPTHB66Y8TM | ✅ Delivered |
| Fix 6: Plan close-out knowledge curation | §8 | FEAT-01KPTHB649DPK | ✅ Delivered |

All six fixes from the design are accounted for. No fix was descoped or deferred.

### Fix-to-artefact mapping summary

**Fix 1** modified four artefacts: `write-design/SKILL.md` (Phase 0: Corpus Discovery),
`write-spec/SKILL.md` (cross-reference check), `work/templates/specification-prompt-template.md`
(Required Related Work section), and `.kbz/roles/reviewer-conformance.yaml` (blocking
anti-pattern entry).

**Fixes 2 and 3** modified three skill files: `kanbanzai-getting-started/SKILL.md`
(corpus integrity check, onboarding procedures), `kanbanzai-documents/SKILL.md`
(classification section relocated and reframed as immediate obligation), and
`orchestrate-review/SKILL.md` (sub-step 1b: classify before dispatching reviewers).
Corresponding `internal/kbzinit/` embedded copies were updated in the same commits.

**Fixes 4 and 6** modified three skill files: `implement-task/SKILL.md` (knowledge
retrieval in Phase 1, confirm/flag in Phase 4, BAD/GOOD example pair),
`orchestrate-development/SKILL.md` (knowledge retrieval in Phase 1, curation pass in
Phase 6 Close-Out), and `kanbanzai-agents/SKILL.md` (Task Lifecycle Checklist, Context
Assembly mandate). The `internal/kbzinit/` embedded copy for `kanbanzai-agents` was
updated.

**Fix 5** modified Go server code: `internal/docint/types.go` (new `SectionAccessInfo`
type, `AccessCount`/`LastAccessedAt` on `DocumentIndex`), `internal/service/intelligence.go`
(counter increments on `GetOutline`, `GetDocumentIndex`, `GetSection`, `FindByEntity`,
`FindByConcept`, `FindByRole`, `Search`), `internal/service/knowledge.go`
(`recent_use_count`, `last_accessed_at`, `sort:"recent"`), and `internal/service/doc_audit.go`
(Most Accessed Documents table in audit output).

---

## 4. Spec Conformance Summary

### FEAT-01KPTHB5Z1H7T — Design-Stage Corpus Consultation

**Source review:** `work/reviews/review-FEAT-01KPTHB5Z1H7T-design-stage-consultation.md`
**Overall review verdict:** `approved_with_followups`
**Spec conformance dimension:** pass
**Blocking findings:** 0

All 22 acceptance criteria (FR-001 through FR-022, NFR-001, NFR-002) passed. Phase 0
Corpus Discovery is present and correctly sequenced. The Related Work template with
Option A / Option B forms is implemented. The reviewer-conformance blocking check fires
on absent or placeholder Related Work sections with `verdict: REJECTED`. The
write-spec cross-reference check is present with all four prescribed steps and a BECAUSE
clause.

Two non-blocking findings:
- **FU-1** (quality): `write-design/SKILL.md` Output Format section still declares
  "exactly 4 required sections" — does not reflect the new Required Related Work section
  mandated by Phase 0 Step 0.5. An agent consulting Output Format as authoritative
  receives a contradictory signal.
- **FU-2** (spec-gap): `spec/doc-intel-design-stage-consultation.md` NFR-004 says
  "five steps of Phase 0" but parenthetically enumerates six items. The implementation
  is correct (unnumbered pre-check + five numbered steps 0.1–0.5); the spec wording
  is ambiguous.

### FEAT-01KPTHB61WPT0 — Corpus Hygiene and Classification Pipeline

**Source review:** `work/reviews/review-FEAT-01KPTHB61WPT0-corpus-hygiene-classification.md`
**Overall review verdict:** `approved`
**Spec conformance dimension:** pass_with_notes
**Blocking findings:** 0

All 23 acceptance criteria (FR-001 through FR-023, NFR-001 through NFR-006) passed.
Corpus integrity check is correctly sequenced in the Session Start Checklist. All
onboarding and adoption procedure steps are present in the correct order with correct
time estimates. Classification section is relocated, retitled, and reframed with
imperative opening, `classification_nudge` mandate, and rationale. Sub-step 1b in
`orchestrate-review` is non-blocking, priority-ordered, and includes the correct
three-step per-document procedure (guide → read → classify). Embedded kbzinit copies
verified content-identical at version 0.4.0.

Two non-blocking spec-gap findings:
- **NFR-001 gap:** Spec's In Scope section and NFR-001 do not mention the
  `internal/kbzinit/` embedded copies. Implementation correctly updated them; the spec
  omission is a gap for future maintainers.
- **NFR-002 gap:** `orchestrate-review/SKILL.md` uses a different YAML front matter
  schema with no `metadata.version` field. Spec mandated version increment uniformly
  without accounting for the schema difference.

### FEAT-01KPTHB649DPK — Knowledge Lifecycle Mandate

**Source review:** `work/reviews/review-FEAT-01KPTHB649DPK-knowledge-lifecycle-mandate.md`
**Overall review verdict:** `approved`
**Spec conformance dimension:** pass_with_notes
**Blocking findings:** 0

All 21 functional and non-functional requirements passed. Step 1a in `implement-task`
Phase 1 is present with `knowledge(action: "list")`, domain-tag instruction, and BECAUSE
rationale. Phase 4 confirm/flag steps are present with self-curation rationale. BAD/GOOD
example pair is present; BAD example correctly names the missing knowledge list call as
the root cause. Orchestrate-development Phase 1 step 1a surfaces confirmed entries to
sub-agents via `handoff` instructions. Phase 6 step 4a provides confirm/flag/retire
dispositions for tier 2 entries with promote guidance for tier 3. `kanbanzai-agents`
Context Assembly section mandates active querying as a requirement. Embedded kbzinit
copy matches the `.agents/` original.

Two non-blocking findings:
- **ASM-003 gap:** `orchestrate-development/SKILL.md` Checklist was not updated to
  mirror the new Phase 1 step 1a and Phase 6 step 4a obligations. ASM-003 asserted it
  would be updated, but no FR mandated it explicitly.
- **Style note:** BECAUSE rationale for Phase 4 confirm/flag (FR-006) is attached only
  to step 6 (flag); step 5 (confirm) has no standalone rationale sentence. The text
  semantically covers both, but a reader scanning step 5 in isolation will not see
  rationale.

### FEAT-01KPTHB66Y8TM — Doc Intel Access Instrumentation

**Source review:** `work/reviews/review-FEAT-01KPTHB66Y8TM-doc-intel-instrumentation.md`
**Overall review verdict:** `approved_with_followups`
**Spec conformance dimension:** pass_with_notes
**Blocking findings:** 0

All 21 functional requirements mapped to implementation and confirmed present. All counter
call sites verified (`Get`, `List`, `GetOutline`, `GetDocumentIndex`, `GetSection`,
`FindByEntity`, `FindByConcept`, `FindByRole`, `Search`). `sort:"recent"` and audit
Most Accessed table implemented per spec. Counter writes are lazy (background goroutines)
and errors are silently absorbed. All new YAML fields carry `omitempty` for backward
compatibility. One concern (CONCERN-TA-01: missing FR-016 Search counter test) was
resolved in commit `648e939` before the review was concluded.

Six remaining non-blocking notes:
- **NOTE-SC-02 / NOTE-IQ-01:** `FindByEntity` spawns a goroutine unconditionally before
  checking whether the result slice is non-empty, inconsistent with `FindByConcept` and
  `FindByRole` which guard with `if len(matches) > 0`. No spurious counter increments
  occur (the inner function is a no-op on empty input), but the pattern is inconsistent
  and incurs unnecessary goroutine allocation overhead.
- **NOTE-IQ-02:** Misleading comment in `TestKnowledgeService_Get_IncrementsMultipleTimes`
  implies a service re-creation step that does not occur.
- **NOTE-TA-02:** No MCP-level test for `sort:"recent"` parameter passthrough
  (`knowledge_tool_test.go` not in changed files).
- **NOTE-TA-03:** No MCP-level test asserting `most_accessed_table` key in the audit
  response (`doc_tool_test.go` audit tests do not pass an `IntelligenceService`).

---

## 5. Documentation Currency Check

### AGENTS.md

`AGENTS.md` does not require a mandatory update as a result of P27. The changes
delivered by this plan live in skill files (`.kbz/skills/`, `.agents/skills/`) and
role files (`.kbz/roles/`), which are the authoritative source of agent procedure and
are read directly by agents at task time. AGENTS.md correctly delegates procedural
mandates to skills via the stage-bindings and skills reference table.

One minor currency observation: the "Before Every Task — Required Checklist" section in
`AGENTS.md` lists git status, orphaned workflow state, and branch checks, but does not
mention the corpus integrity check now mandated by `kanbanzai-getting-started/SKILL.md`.
Since agents follow the getting-started skill at session start, the check will be executed
regardless; AGENTS.md does not need to duplicate it. This is an observation, not a
deficiency.

The dual-write rule in AGENTS.md ("Dual-write rule for skill changes") was followed by
all three features that modified `.agents/skills/` files.

### Spec document status fields

Two spec documents carry stale `Status: Draft` text in their document headers, despite
being in `approved` status in the doc record system:

- `work/spec/doc-intel-corpus-hygiene.md` — header reads `**Status:** Draft`
- `work/spec/doc-intel-access-instrumentation.md` — header reads `**Status:** Draft`

The other two specs (`doc-intel-design-stage-consultation.md`,
`doc-intel-knowledge-lifecycle.md`) correctly show `| Status | approved |` in their
metadata tables.

This is a minor documentation currency issue. The doc record system is authoritative;
the stale header fields are cosmetic. Recommend updating both headers to `approved` in a
follow-up pass.

### write-design/SKILL.md Output Format section

The Output Format section of `write-design/SKILL.md` (identified in FU-1 above) still
declares "The document then has exactly 4 required sections" and does not list Related
Work in the embedded document template. This is the highest-priority documentation
currency item because it creates a contradictory signal: an agent that reads Phase 0 is
required to write a Related Work section, but an agent that reads the Output Format is
told the document has exactly four required sections and sees no Related Work in the
template. This should be corrected in the first available follow-up.

### orchestrate-review/SKILL.md version field

`orchestrate-review/SKILL.md` uses a different YAML front matter schema from the
`kanbanzai-getting-started` and `kanbanzai-documents` skills and has no
`metadata.version` field. A versioning baseline should be established by adding a
`metadata:` block in a follow-up commit.

---

## 6. Cross-Cutting Check Results

| Check | Result |
|---|---|
| `go test ./...` (29 packages, post-merge on main) | ✅ All pass |
| `git status` on main | ✅ Clean working tree |
| Post-merge regression | ✅ Identified and fixed (commit `ff5b940`) |
| Feature entity statuses | ✅ All 4 features in `done` |
| Spec documents (doc record system) | ✅ All 4 specs approved |
| Feature review reports | ✅ All 4 approved, 0 blocking findings |
| Blocking findings across all feature reviews | ✅ 0 |

### Post-merge regression detail

After the corpus-hygiene feature (FEAT-01KPTHB61WPT0) squash-merged, the
`internal/kbzinit/` embedded skill copies for `kanbanzai-getting-started` and
`kanbanzai-documents` lost their `# kanbanzai-managed: true` and
`# kanbanzai-version: dev` comment markers. The feature had rewritten the YAML
frontmatter using key-value pairs rather than the comment form that `kbzinit`'s
`transformSkillContent` and `hasLine` functions require. Additionally, the
`Direct File Write to Workflow Roots` anti-pattern entry (needed by the
`TestP12_Integration_NewProject` integration test) was dropped from the embedded
getting-started skill.

Fix applied in commit `ff5b940` on 2026-04-22. Both embedded skill files restored to
correct format. All 29 test packages pass after the fix.

---

## 7. Scope Reductions and Deviations

No design fixes were descoped or deferred. The design's Phase 2 (access instrumentation)
and Phase 3 (validate and tighten) phasing was fully collapsed into the plan: all six
fixes were delivered in parallel features rather than sequentially.

One design-described item was adapted at specification time without deviation from intent:
the design (§3.5) proposed changes to "`reviewer-conformance/SKILL.md`" as an alternative
target. The spec for FEAT-01KPTHB5Z1H7T correctly identified that no
`reviewer-conformance/SKILL.md` file exists — the only reviewer-conformance artefact is
`.kbz/roles/reviewer-conformance.yaml` — and targeted that file instead. This is a
specification-time correction, not a deviation.

The plan review arm of Fix 6 (changes to the `review-plan` skill or plan-review workflow
at plan close-out) was explicitly placed out of scope for FEAT-01KPTHB649DPK. The design
(§8.2) described a confirmation pass "at plan close-out" which could be interpreted as a
`review-plan` skill change. The spec elected to cover only the `orchestrate-development`
Phase 6 arm. This is a valid scoping decision: the `orchestrate-development` arm covers
the primary mechanism; a plan-review skill addition remains as a future follow-up (see
§8 below).

---

## 8. Open Follow-Up Items

The following non-blocking findings from feature reviews are candidates for future work,
ordered by priority.

### High priority

**FU-P27-01** — Update `write-design/SKILL.md` Output Format section  
Source: FEAT-01KPTHB5Z1H7T review (FU-1)  
File: `.kbz/skills/write-design/SKILL.md` L193-194  
Change: Update "exactly 4 required sections" to "exactly 5 required sections"; add
Related Work as the first section in the embedded document template, with Option A /
Option B guidance matching `specification-prompt-template.md`. Without this, the Output
Format provides a contradictory signal to agents who consult it as the authoritative
design-document template.

**FU-P27-02** — Add knowledge checklist items to `orchestrate-development/SKILL.md`  
Source: FEAT-01KPTHB649DPK review (ASM-003 gap)  
File: `.kbz/skills/orchestrate-development/SKILL.md` checklist section (L98-115)  
Change: Add checklist items mirroring the new Phase 1 step 1a (knowledge retrieval) and
Phase 6 step 4a (curation pass) obligations.

**FU-P27-03** — Fix `FindByEntity` goroutine guard inconsistency  
Source: FEAT-01KPTHB66Y8TM review (NOTE-SC-02, NOTE-IQ-01)  
File: `internal/service/intelligence.go` `FindByEntity`  
Change: Wrap both `wg.Add(1)` / goroutine-spawn sites in `FindByEntity` with
`if len(matches) > 0` guards, mirroring `FindByConcept` and `FindByRole`. Eliminates
unnecessary goroutine allocation on empty result sets.

### Medium priority

**FU-P27-04** — Update stale `Status: Draft` headers in two spec files  
Source: Documentation currency check (§5)  
Files: `work/spec/doc-intel-corpus-hygiene.md`, `work/spec/doc-intel-access-instrumentation.md`  
Change: Update header status field to `approved` to match doc record system state.

**FU-P27-05** — Add `metadata.version` field to `orchestrate-review/SKILL.md`  
Source: FEAT-01KPTHB61WPT0 review (NFR-002 gap)  
File: `.kbz/skills/orchestrate-review/SKILL.md`  
Change: Add `metadata:\n  version: "0.1.0"` to YAML front matter to establish a
versioning baseline consistent with other skills.

**FU-P27-06** — Add plan-review arm to Fix 6 (plan close-out knowledge curation)  
Source: FEAT-01KPTHB649DPK scope decision, design §8.2  
File: `.kbz/skills/review-plan/SKILL.md` (or equivalent plan-review skill)  
Change: Add a knowledge curation pass to the plan review workflow (the arm that was
explicitly descoped from FEAT-01KPTHB649DPK).

### Low priority

**FU-P27-07** — Correct misleading comment in `knowledge_access_test.go`  
Source: FEAT-01KPTHB66Y8TM review (NOTE-IQ-02)  
File: `internal/service/knowledge_access_test.go`  
Change: Remove or correct comment in `TestKnowledgeService_Get_IncrementsMultipleTimes`
that falsely implies a service re-creation step.

**FU-P27-08** — Add MCP-level test for `sort:"recent"` parameter passthrough  
Source: FEAT-01KPTHB66Y8TM review (NOTE-TA-02)  
File: `internal/mcp/knowledge_tool_test.go`  
Change: Add test exercising `knowledgeListAction` with `sort:"recent"` to verify the
parameter is passed through to the service layer.

**FU-P27-09** — Add MCP-level test for `most_accessed_table` in audit response  
Source: FEAT-01KPTHB66Y8TM review (NOTE-TA-03)  
File: `internal/mcp/doc_tool_test.go`  
Change: Add end-to-end audit test with a seeded `IntelligenceService` asserting
`most_accessed_table` key in the response.

**FU-P27-10** — Correct NFR-004 wording in corpus-consultation spec  
Source: FEAT-01KPTHB5Z1H7T review (FU-2)  
File: `work/spec/doc-intel-design-stage-consultation.md` NFR-004  
Change: Clarify "five steps of Phase 0" to "a pre-discovery check followed by five
numbered steps (0.1–0.5)" to eliminate the ambiguity between the pre-check and the
numbered step count.

**FU-P27-11** — Add BECAUSE rationale to `implement-task` Phase 4 step 5 (confirm)  
Source: FEAT-01KPTHB649DPK review (style note)  
File: `.kbz/skills/implement-task/SKILL.md` Phase 4 step 5  
Change: Attach a brief rationale sentence to the confirm step directly, so agents
scanning step 5 in isolation see the reasoning without needing to continue to step 6.

---

## 9. Retrospective Observations

Three retrospective signals were contributed to the knowledge base during this review
(KE-01KPV0AKAGFH0, KE-01KPV0AR78BV0, KE-01KPV0AY0QT04). Summaries follow.

### What worked well

**Prescriptive design document.** The design (§3–§8) specified exact tool calls,
BECAUSE clause wording, and checklist item text verbatim. This left minimal
interpretation to implementing agents: the implementing agent could copy the
prescribed text from the design into the skill file with high confidence of
conformance. Feature reviews confirmed this — spec conformance passed on the first
review for all four features, with no re-work needed. This level of design prescriptiveness
is the right approach for skill-file change plans where correctness of wording matters.

**Clean feature decomposition.** Fixes 2 and 3 were bundled into one feature
(FEAT-01KPTHB61WPT0) because they modify overlapping skill files and share the same
conceptual theme (corpus completeness → classification as obligation). Fixes 4 and 6
were bundled into FEAT-01KPTHB649DPK for similar reasons. The instrumentation feature
(FEAT-01KPTHB66Y8TM) was cleanly isolated as the only Go code change. This decomposition
produced no cross-feature conflicts and allowed all four features to be implemented in
parallel.

**Regression caught in-session.** The kbzinit managed-marker regression was caught
during the review phase of the same session in which the merge occurred, rather than
surfacing as a production defect in a future session. The `TestP12_Integration_NewProject`
integration test provided the detection signal. This is the correct feedback loop:
integration tests that exercise the kbzinit binary should be part of every pre-merge
check for features that touch embedded skill files.

### What created friction

**kbzinit dual-write format trap.** The corpus-hygiene feature followed the
dual-write rule (updated both `.agents/skills/` and `internal/kbzinit/skills/`), but
the feature's frontmatter format change from comment-style markers to YAML key-value
pairs was not validated against `kbzinit`'s parser requirements. The dual-write rule
as stated in AGENTS.md is necessary but not sufficient: it mandates updating the
embedded copy, but not verifying that the copy retains the comment markers that
`transformSkillContent` and `hasLine` require. The fix took one commit, but it could
have been prevented by running the integration test suite against the worktree before
merging.

**Spec scope omission for kbzinit.** Two specs in this plan (corpus-hygiene, and
implicitly knowledge-lifecycle) did not list `internal/kbzinit/` copies in their In
Scope or NFR sections. This was caught during review as a spec-gap finding rather
than a blocking defect, but it reflects a recurring pattern: the dual-write rule lives
in AGENTS.md and is not enforced at specification time. Future specs that modify
`.agents/skills/kanbanzai-*/SKILL.md` files should be required to name the
corresponding `internal/kbzinit/skills/<name>/SKILL.md` copies in their scope section.

### What to change for future plans

1. **Run `go test -run TestP12_Integration_NewProject ./...` as a pre-merge gate for
   any feature that modifies files under `.agents/skills/kanbanzai-*/` or
   `internal/kbzinit/skills/`.** This integration test is the authoritative validator
   for kbzinit embedded skill correctness.

2. **Add a spec template note for skill-file changes:** when a spec lists
   `.agents/skills/kanbanzai-<name>/SKILL.md` in scope, it must also list
   `internal/kbzinit/skills/<name>/SKILL.md` in the same section. This should be
   documented in the spec template or in the `write-spec` skill.

3. **Include Output Format consistency as a checklist item in `write-design` review:**
   when Phase 0 or any other section mandates a new document section, the reviewer
   should verify that the Output Format template in the same skill is updated to
   reflect it. FU-P27-01 is the concrete gap from this plan.