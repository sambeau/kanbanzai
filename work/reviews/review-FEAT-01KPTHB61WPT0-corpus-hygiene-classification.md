# Review: Corpus Hygiene and Classification Pipeline
**Feature:** FEAT-01KPTHB61WPT0
**Branch:** feature/FEAT-01KPTHB61WPT0-corpus-hygiene-classification
**Spec:** work/spec/doc-intel-corpus-hygiene.md
**Date:** 2026-04-22

---

```
Review Unit: corpus-hygiene-classification
Files:
  - .agents/skills/kanbanzai-getting-started/SKILL.md
  - .agents/skills/kanbanzai-documents/SKILL.md
  - .kbz/skills/orchestrate-review/SKILL.md
  - internal/kbzinit/skills/getting-started/SKILL.md
  - internal/kbzinit/skills/documents/SKILL.md
Spec: work/spec/doc-intel-corpus-hygiene.md
Reviewer Role: reviewer-conformance + reviewer-quality

Overall: approved

Dimensions:
  spec_conformance: pass_with_notes
    Evidence:
      - AC FR-001: Session Start Checklist contains "Corpus integrity check" item with
        doc(action:"audit") reference.
        (.agents/skills/kanbanzai-getting-started/SKILL.md, checklist bullet 3)
      - AC FR-002: Unregistered-file branch specifies doc(action:"import", path:"work").
        (.agents/skills/kanbanzai-getting-started/SKILL.md, checklist + expanded section)
      - AC FR-003: Stale-record branch specifies doc(action:"delete", id:"DOC-xxx") per record.
        (.agents/skills/kanbanzai-getting-started/SKILL.md, checklist + expanded section)
      - AC FR-004: Post-registration classification pass requirement present, cross-referencing
        Classification (Layer 3) section in kanbanzai-documents.
        (.agents/skills/kanbanzai-getting-started/SKILL.md, "Corpus integrity check" section)
      - AC FR-005: Rationale statement present in both inline checklist item and expanded section.
        (.agents/skills/kanbanzai-getting-started/SKILL.md)
      - AC FR-006: "New-Project Onboarding" section present.
        (.agents/skills/kanbanzai-getting-started/SKILL.md)
      - AC FR-007: All five onboarding steps present in correct order
        (configure → import → audit → classify → validate).
        (.agents/skills/kanbanzai-getting-started/SKILL.md, "New-Project Onboarding" > Steps)
      - AC FR-008: "5–10 minutes per document" and "4–8 hours for 50 documents" estimates present.
        (.agents/skills/kanbanzai-getting-started/SKILL.md, "Time estimate" note)
      - AC FR-009: "Existing-Project Adoption" section present.
        (.agents/skills/kanbanzai-getting-started/SKILL.md)
      - AC FR-010: All four adoption steps present in correct order
        (find → decide → configure → register-and-classify).
        (.agents/skills/kanbanzai-getting-started/SKILL.md, "Existing-Project Adoption" > Steps)
      - AC FR-011: Key principle "negative result means not addressed rather than not registered"
        present verbatim.
        (.agents/skills/kanbanzai-getting-started/SKILL.md, "Key principle")
      - AC FR-012: Classification (Layer 3) section positioned immediately after Registration
        and before Drift and Refresh (Approval Workflow intervenes, but classification is before
        Drift and Refresh as specified).
        (.agents/skills/kanbanzai-documents/SKILL.md)
      - AC FR-013: Section titled "Classification (Layer 3)"; string "Batch Classification
        Protocol" absent from the skill.
        (.agents/skills/kanbanzai-documents/SKILL.md)
      - AC FR-014: Opening imperative "After registering a document, classify it immediately
        if you have the document content in context. Do not defer classification to a batch run."
        present verbatim.
        (.agents/skills/kanbanzai-documents/SKILL.md)
      - AC FR-015: classification_nudge mandate with MUST language present.
        (.agents/skills/kanbanzai-documents/SKILL.md)
      - AC FR-016: Rationale about deferred documents accumulating as backlog never fully
        cleared present.
        (.agents/skills/kanbanzai-documents/SKILL.md)
      - AC FR-017: Classification checkbox added to Document Creation Checklist:
        "Classified the document with doc_intel(action: classify) if content was in context".
        (.agents/skills/kanbanzai-documents/SKILL.md)
      - AC FR-018: Step 1b present, calls doc_intel(action:"pending") filtered to feature
        under review.
        (.kbz/skills/orchestrate-review/SKILL.md, L130-148)
      - AC FR-019: Three-step per-document procedure (guide → read → classify) present
        inside 1b.
        (.kbz/skills/orchestrate-review/SKILL.md, L133-138)
      - AC FR-020: "Classification is NOT a blocking prerequisite. If context budget is
        exhausted, MUST proceed with reviewing anyway." present.
        (.kbz/skills/orchestrate-review/SKILL.md, L140-141)
      - AC FR-021: Priority order "specification → design → dev-plan" explicit in 1b.
        (.kbz/skills/orchestrate-review/SKILL.md, L132)
      - AC FR-022: Rationale about reviewer sub-agents, role-based search, and structural
        fallback present.
        (.kbz/skills/orchestrate-review/SKILL.md, L143-147)
      - AC FR-023: Checklist item "Classified unclassified feature documents (or confirmed
        context budget insufficient)" present.
        (.kbz/skills/orchestrate-review/SKILL.md, L106)
      - NFR-002: getting-started 0.3.0 → 0.4.0, documents 0.3.0 → 0.4.0. Both incremented.
        kbzinit copies carry matching 0.4.0 version.
      - NFR-004: Corpus integrity check at checklist slot 3, after "Store check" (slot 2) and
        before "Read project context" / "Check the work queue" (slots 4–5). Positioning correct.
      - NFR-005: All original classification-section content preserved: "When to run this
        protocol", "Classification-on-registration convention", priority ordering table,
        step-by-step procedure, anti-patterns. New framing (FR-014/015/016) prepended only.
      - NFR-006: Numbering sequence is 1a (status check) → 1b (new) → 2 (locate spec) →
        3 (identify files) → 4 (stop rule). Downstream steps unaffected.
      - kbzinit sync: internal/kbzinit/skills/getting-started/SKILL.md and
        internal/kbzinit/skills/documents/SKILL.md are content-identical to their
        .agents/skills/ counterparts, including version 0.4.0.
    Findings:
      - [non-blocking] Spec-gap: NFR-001 states "All changes MUST be confined to the three
        specified skill files" but the spec's In Scope section and NFR-001 make no mention of
        internal/kbzinit/ embedded copies. The implementation correctly updated them (required
        for runtime correctness — embedded copies must stay in sync with .agents/skills/
        originals). The spec, not the implementation, is deficient.
        (spec: NFR-001, location: work/spec/doc-intel-corpus-hygiene.md)
      - [non-blocking] Spec-gap: NFR-002 mandates "the front matter version field in each
        modified skill MUST be incremented." The .kbz/skills/orchestrate-review/SKILL.md uses
        a different YAML front matter schema (name / description / triggers / roles / stage /
        constraint_level) with no metadata.version field — in the original or in the worktree.
        The spec failed to account for this schema difference. The correct remediation (adding
        a metadata.version field to orchestrate-review) is minimal and could be applied as a
        follow-up, but the omission is not a defect in the implementation given the spec gap.
        (spec: NFR-002, location: .kbz/skills/orchestrate-review/SKILL.md)

  implementation_quality: pass
    Evidence:
      - Corpus integrity check sequencing: inline checklist item and expanded "Corpus integrity
        check" section are internally consistent. The conditional branching (unregistered files
        vs stale records) is clearly expressed and actionable.
        (.agents/skills/kanbanzai-getting-started/SKILL.md)
      - Classification obligation reframing: imperative opening, nudge mandate, and rationale
        are unambiguous and mutually reinforcing. The new framing does not contradict the
        preserved step-by-step procedure — it prepends mandatory context that the procedure
        then implements.
        (.agents/skills/kanbanzai-documents/SKILL.md)
      - Orchestrate-review 1b: non-blocking clause uses appropriately imperative "MUST proceed"
        language. Priority order (specification → design → dev-plan) matches the ordering
        table established in kanbanzai-documents, ensuring cross-skill consistency.
        (.kbz/skills/orchestrate-review/SKILL.md)
      - kbzinit copies: confirmed identical to .agents/skills/ originals; no sync drift
        introduced.
        (internal/kbzinit/skills/*)
    Findings:
      (none)

Finding Summary:
  Blocking: 0
  Non-blocking: 2
  Total: 2
```

---

## Supplementary Notes

### Spec-gap: NFR-001 kbzinit scope omission

The `internal/kbzinit/` directory holds embedded copies of agent skills compiled into the
binary for `kanbanzai init`. Any change to a canonical skill file must also update the
corresponding embedded copy or the two will diverge at the next `kanbanzai init` execution.
The implementation correctly kept them in sync. Future specifications that modify agent
skills should explicitly name the kbzinit embedded copies in their In Scope section and
NFR-001-equivalent constraint.

### Spec-gap: NFR-002 orchestrate-review version schema

`kanbanzai-getting-started` and `kanbanzai-documents` use a `metadata.version` field.
`orchestrate-review` uses a different YAML front matter schema with no `metadata` block.
The spec mandated version increment uniformly across all three skills without noting this
schema difference. The minimal follow-up fix would be to add `metadata:\n  version: "0.1.0"`
(or equivalent) to `orchestrate-review/SKILL.md`'s front matter, establishing a versioning
baseline for future modifications.