---
# kanbanzai-managed: true
# kanbanzai-version: dev
name: update-docs
description:
  expert: "Documentation maintenance and update skill producing consistent,
    current documentation aligned with implementation changes during the
    documenting stage"
  natural: "Update project documentation to reflect what was built, fix
    stale content, and keep docs accurate and navigable"
triggers:
  - update documentation
  - fix stale docs
  - update the docs to reflect changes
  - documentation needs updating
  - refresh project documentation
roles: [documenter]
stage: documenting
constraint_level: medium
---

## Vocabulary

- **content currency** — whether documentation accurately reflects the current state of the system; stale content has low currency
- **cross-reference integrity** — the property that all links, references, and pointers between documents resolve to valid, current targets
- **documentation drift** — the divergence between what documentation describes and what the system actually does, accumulating over time as code changes outpace doc updates
- **link rot** — broken references caused by targets being moved, renamed, or deleted without updating the documents that point to them
- **information architecture** — the organisation of documentation into a navigable structure where readers can find what they need without reading everything
- **documentation debt** — the accumulated backlog of documentation that is missing, outdated, or misleading, increasing onboarding cost and error rates
- **content audit** — a systematic review of existing documentation to identify stale, missing, or incorrect content
- **documentation coverage** — the proportion of system behaviour, APIs, and workflows that have corresponding up-to-date documentation
- **audience** — the intended reader of a document, determining the level of detail, assumed knowledge, and terminology used
- **changelog entry** — a concise record of what changed, when, and why, enabling readers to understand the evolution of a system without reading diffs
- **style consistency** — uniform formatting, tone, heading conventions, and terminology across all documents in a project
- **progressive disclosure** — organising documentation so that readers encounter overview information first and can drill into detail on demand
- **doc registration** — recording a document with `doc(action: register)` so the workflow system tracks its status, ownership, and content hash
- **content hash** — the fingerprint the system uses to detect drift between the registered version and the current file contents
- **doc refresh** — calling `doc(action: refresh)` after editing a registered document so the system updates its content hash and detects changes
- **supersession** — replacing an outdated document with a new version using `doc(action: supersede)`, preserving the old document as historical record

## Anti-Patterns

### Drive-By Update
- **Detect:** A single sentence or paragraph is changed in isolation without checking whether the same concept appears elsewhere in the documentation
- **BECAUSE:** Partial updates create internal contradictions — one document says the old behaviour, another says the new behaviour, and readers cannot tell which is authoritative
- **Resolve:** Search for related mentions across all documentation before committing an update; update every occurrence or add cross-references

### Code-Comment Duplication
- **Detect:** Documentation restates what inline code comments already explain, or vice versa
- **BECAUSE:** Duplicated explanations diverge over time because maintainers update one copy and miss the other, producing contradictory guidance
- **Resolve:** Documentation should explain *why* and *how to use*; code comments should explain *why this implementation*. Each has a distinct purpose — do not duplicate between them

### Stale Screenshot Syndrome
- **Detect:** Documentation contains screenshots, output samples, or example responses that no longer match the current system behaviour
- **BECAUSE:** Visual or concrete examples that contradict reality erode trust in all documentation — readers cannot distinguish which parts are current and which are stale
- **Resolve:** Re-capture examples from the current system, or replace fragile screenshots with textual descriptions that are easier to maintain

### Missing Registration
- **Detect:** A document is created or significantly updated without calling `doc(action: register)` or `doc(action: refresh)`
- **BECAUSE:** Unregistered or un-refreshed documents are invisible to the workflow system — they will not appear in health checks, gap analysis, or approval workflows
- **Resolve:** Register new documents immediately after creation; refresh existing documents after any substantive edit

### Audience Mixing
- **Detect:** A single document addresses both end users and developers, switching between usage instructions and implementation details
- **BECAUSE:** Mixed-audience documents force every reader to skip irrelevant sections, increasing cognitive load and making it harder to maintain either perspective coherently
- **Resolve:** Split into separate documents with clear audience declarations, or use clearly labelled sections that readers can skip

### Implicit Prerequisites
- **Detect:** Documentation assumes the reader has completed setup steps or has knowledge that is not referenced or linked
- **BECAUSE:** Readers who lack the assumed context follow instructions incorrectly or give up, producing support requests that the documentation was meant to prevent
- **Resolve:** Link to prerequisite documentation explicitly, or include a "Prerequisites" section listing what the reader needs before starting

### Changelog Neglect
- **Detect:** Behaviour-visible changes are made without a corresponding changelog entry
- **BECAUSE:** Without a changelog, users and developers must read diffs to understand what changed — changelogs are the primary mechanism for communicating evolution
- **Resolve:** Add a changelog entry for every change that affects behaviour, APIs, or configuration

## Checklist

Copy this checklist and track your progress:

- [ ] Identified all documents affected by the implementation changes
- [ ] Audited affected documents for content currency
- [ ] Checked cross-reference integrity (links still resolve)
- [ ] Updated all stale content to reflect current behaviour
- [ ] Verified style consistency with surrounding documentation
- [ ] Registered new documents with `doc(action: register)`
- [ ] Refreshed edited documents with `doc(action: refresh)`
- [ ] Added changelog entries for behaviour-visible changes
- [ ] Verified no audience mixing in updated documents

## Procedure

### Step 1: Identify Scope

1. Review the implementation changes (completed tasks, modified files, new features).
2. List every document that describes affected behaviour, APIs, or workflows.
3. IF the scope of documentation changes is unclear → STOP. Ask which documents need updating and what changed. Do not guess which docs are affected.
4. Check `doc(action: list)` for registered documents related to the affected features.

### Step 2: Audit Current State

1. Read each affected document fully.
2. For each document, note content that no longer matches the implementation.
3. Check all cross-references and links for validity.
4. Identify missing documentation — new features or behaviours with no corresponding docs.
5. IF a document describes behaviour that contradicts the implementation → flag for update.
6. IF a document references a design or spec that has been superseded → update the reference.

### Step 3: Update Content

1. Update stale content to reflect the current system behaviour.
2. Add new sections or documents for features that lack documentation.
3. Fix broken cross-references.
4. Maintain style consistency with the surrounding documentation (heading levels, tone, formatting).
5. IF a document needs to be replaced entirely rather than patched → create a new document and supersede the old one.
6. Add changelog entries for every behaviour-visible change.

### Step 4: Register and Refresh

1. For new documents: call `doc(action: register)` with the correct type and owner.
2. For updated documents: call `doc(action: refresh)` to update the content hash.
3. Commit the documentation changes together with any registration updates.

### Step 5: Validate

1. Re-read each updated document to verify accuracy.
2. Verify all cross-references resolve to valid targets.
3. Verify no content contradicts the current implementation.
4. IF validation reveals additional stale content → fix it → re-validate.

## Output Format

Documentation updates do not follow a single rigid template because they modify existing documents in place. The deliverable is:

```
## Documentation Update Summary

### Documents Updated
- `path/to/document.md` — description of what changed and why

### Documents Created
- `path/to/new-document.md` — what this document covers, registered as type X

### Documents Superseded
- `path/to/old-document.md` → superseded by `path/to/new-document.md`

### Cross-Reference Fixes
- Fixed link in `path/to/doc.md` pointing to moved target

### Changelog Entries
- Entry added to `CHANGELOG.md` or relevant changelog for [change description]

### Remaining Gaps
- [Any documentation gaps identified but not addressed, with reason]
```

## Examples

### BAD: Isolated Fix Without Context Check

> Updated `README.md` to change the CLI flag from `--verbose` to `--debug`.

**WHY BAD:** Only one file was checked. The flag name likely appears in the getting-started guide, the configuration reference, and inline help text. Changing one occurrence without searching for others creates contradictory documentation.

### BAD: Update Without Registration

> Created `docs/new-feature.md` explaining the new caching layer.
> Updated `docs/architecture.md` to mention the cache component.

**WHY BAD:** The new document was not registered with `doc(action: register)`, making it invisible to the workflow system. The edited document was not refreshed with `doc(action: refresh)`, so the system still has the old content hash.

### GOOD: Systematic Update with Full Traceability

> ## Documentation Update Summary
>
> ### Documents Updated
> - `docs/architecture.md` — added cache component to system diagram
>   description; refreshed with `doc(action: refresh)`
> - `docs/configuration.md` — added `cache_ttl` and `cache_backend`
>   configuration options with defaults and valid values
> - `docs/troubleshooting.md` — added "Cache miss rate too high" entry
>
> ### Documents Created
> - `docs/caching.md` — caching layer overview, configuration, and
>   monitoring; registered as type `design` with owner FEAT-042
>
> ### Cross-Reference Fixes
> - Fixed link in `docs/architecture.md` to `docs/storage.md` (file
>   was renamed from `docs/persistence.md` in previous PR)
>
> ### Changelog Entries
> - Added entry: "Added read-through cache layer behind EntityReader
>   interface; new configuration options `cache_ttl`, `cache_backend`"
>
> ### Remaining Gaps
> - Runbook for cache failure scenarios deferred — requires ops team input

**WHY GOOD:** Every affected document was identified and updated. New documents were registered. Edited documents were refreshed. Cross-reference integrity was checked and a stale link was caught. A changelog entry was added. Remaining gaps are explicitly acknowledged rather than silently ignored.

## Evaluation Criteria

1. Were all documents affected by the implementation changes identified and updated? Weight: required.
2. Were new documents registered and edited documents refreshed with the workflow system? Weight: required.
3. Do all cross-references and links resolve to valid, current targets? Weight: high.
4. Is updated content accurate and consistent with the current implementation? Weight: required.
5. Were changelog entries added for behaviour-visible changes? Weight: high.
6. Is style (tone, formatting, heading conventions) consistent with surrounding documentation? Weight: medium.
7. Are remaining documentation gaps explicitly acknowledged? Weight: medium.

## Questions This Skill Answers

- How do I update documentation after implementing a feature?
- What documents need updating when code changes?
- How do I find and fix stale documentation?
- What is the process for registering and refreshing documents after edits?
- How do I handle documentation for superseded features?
- When should I create a new document vs. update an existing one?
- How do I check cross-reference integrity across documentation?
- What belongs in a changelog entry?