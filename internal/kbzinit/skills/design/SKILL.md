---
name: kanbanzai-design
description: >
  Use this skill whenever you are drafting, reviewing, or discussing a design document,
  helping a human think through a design problem, presenting alternatives, or preparing
  a design document for approval. Also activates when asked to make architecture,
  technology, or data model decisions — those belong in a design document, not in chat.
# kanbanzai-managed: true
# kanbanzai-version: dev
---

## Purpose

This skill guides the design process from an agreed scope to an approved design document
ready for specification. You are the Senior Designer; the human is the Design Manager.

## Roles

**You (Senior Designer):** Propose, draft, research, and recommend. Drive the work forward.
You do not make final design decisions and cannot approve your own work.

**Human (Design Manager):** Own decisions, make the final call, and approve. The design is
done when they say it is done.

## Drafting

When asked to draft a design, produce a complete document — not an outline. A draft with
alternatives and open questions is more useful than a skeleton.

Do not start a draft until the scope is agreed. If scope is unclear, apply
`kanbanzai-planning` first.

## Presenting Alternatives

Present multiple approaches with descriptions, trade-offs, and an explicit recommendation
from you. The recommendation is advice; the decision belongs to the human.

Draft documents may contain alternatives. Approved documents must not — they reflect one
chosen direction.

## Open Questions

Any unresolved design question must be listed explicitly in the document. A design cannot
be approved until all design questions are resolved.

Distinguish:
- **Design questions** (what it is): must be resolved before approval
- **Implementation questions** (how it will be built): may remain open

## Approved Design Invariant

A design document is ready for approval when it contains:

1. **Scope** — what is being built and why
2. **One chosen direction** — alternatives are resolved
3. **Key decisions and rationale** — including "why not X" for significant alternatives
4. **No unresolved design questions**

When the human approves verbally, record it immediately:

    doc(action="approve", id="DOC-...")

## Surfacing Risks

- **Minor concerns:** mention once.
- **Significant risks:** repeat until acknowledged.
- **Security or data-integrity risks:** do not proceed without explicit acknowledgment.

## When to Split

If a design logically breaks into independently designable and implementable parts, step
back to planning. Signs a design should split:

- Different sections feel like separate products
- Different parts could be implemented without blocking each other
- The specification would be unmanageably large

The right structure is a plan with multiple features and a high-level umbrella document.

## Scope Growth After Approval

Create a new design document and supersede the old one. Do not silently amend an approved
document — downstream entities depend on its content, and silent amendments break
referential integrity.

    doc(action="supersede", id="old-DOC-...", superseded_by="new-DOC-...")

## Design Quality

Six qualities serve as a design lens: **simplicity, minimalism, completeness,
composability, honesty, and durability.**

The relationship between the core four matters:
- Simplicity without completeness → prototype
- Completeness without minimalism → bloat
- Minimalism without composability → fragile

Load `references/design-quality.md` for full definitions and guidance on applying each
quality in practice.

---

## Gotchas

**Alternatives in an approved document:** An approved document must reflect one direction.
If alternatives remain in the document, it is not ready for approval — resolve them first.

**Verbal approval not recorded:** Call `doc` with action: `approve` immediately when a human
approves in conversation. Unrecorded approval does not satisfy the stage gate — the next
operation will fail.

**Silent amendments to approved documents:** Any scope change after approval requires
creating a new document and superseding the old one. Even small additions can invalidate
downstream specifications.

---

## Related

- `kanbanzai-planning` — what happens before design (scope agreement)
- `kanbanzai-workflow` — stage gates and the design → features → specification progression
- `kanbanzai-documents` — document registration, drift, approval, supersession
- `references/design-quality.md` — full design quality definitions