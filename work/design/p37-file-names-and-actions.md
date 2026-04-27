# P37 File Names and Actions — Design

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-04-27T12:34:44Z           |
| Status | Draft                          |
| Author | sambeau                        |
| Plan   | P37-file-names-and-actions     |

## Related Work

### Prior documents consulted

| Document | Type | Relationship |
|----------|------|-------------|
| `work/design/document-centric-interface.md` | design | Defines the document type taxonomy and explicitly leaves folder structure (§12.2) and art files (§12.1) as open questions — this design resolves both |
| `work/plan/phase-1-decision-log.md` (P1-DEC-006) | decision | Established `.kbz/` as instance root and defined the `{TYPE}-{TSID13}-{slug}.yaml` entity filename format — this design extends that convention to human-facing work documents |
| `work/plan/phase-1-decision-log.md` (P1-DEC-021) | decision | Chose compact TSID13 over full ULIDs, rejected hierarchical task IDs (`FEAT-xxx.Txxx`) — this design proposes plan-scoped feature IDs for human-facing use only, keeping TSID13 as canonical |
| `work/reviews/review-P7-developer-experience.md` | report | P7 review identified naming and folder inconsistencies as a delivered feature — this design supersedes that partial fix with a comprehensive standard |
| `work/reports/kanbanzai-2.0-workflow-retrospective.md` | report | Identified friction points around file organisation and findability |

### Decisions that constrain this design

1. **P1-DEC-006** — `.kbz/state/` owns entity state; `work/` owns human-authored content. This separation is preserved.
2. **P1-DEC-021** — TSID13 is the canonical ID format for features, tasks, bugs, and decisions. This design does not replace TSID13 — it layers a human-friendly alias on top for features only.
3. **Document-centric interface §3** — "The human interface is documents and chat." File naming must serve human browsing and AI agent reference equally.
4. **Document-centric interface §4** — Eight document types are recognised. This design formalises the mapping between types and filename prefixes.

### Open questions resolved by this design

- **§12.2 "Folder structure: by type or by feature?"** — Resolved: plan-first folders with type-prefixed filenames.
- **§12.1 "Art files, images, and document bundles"** — Partially resolved: per-plan `assets/` folder; detailed asset naming deferred until real assets are produced.

---

## Problem and Motivation

The Kanbanzai project has 430 files in `work/` spread across 18 folders. As the project transitions from medium to large scale and prepares for multi-person teams, several structural problems are blocking effective collaboration:

**Folder duplication.** Four folder pairs exist for the same purpose: `spec/` and `specs/`, `dev-plan/` and `dev-plans/`, `retro/` and `retros/`, `eval/` and `evaluation/`. A fifth folder (`dev/`) appears to be a deprecated alias for `dev-plan/`. AI agents create files in whichever variant they encounter first.

**Inconsistent naming.** At least seven different filename patterns coexist in the same folders:
- Topic-slug only: `lifecycle-integrity.md`
- Plan-prefix: `p24-ac-pattern-and-decompose.md`
- Feature-ID prefix: `feat-01kq2e0rb4p8a-plan-id-prefix-resolution.md`
- Redundant type prefix: `spec-default-roles.md` (inside `spec/`)
- Type + entity-ID + slug: `review-FEAT-01KMKRQSD1TKK-skills-content.md`
- Version prefix: `3.0-role-system.md`
- Unformatted with spaces: `PLAN vs DEV-PLAN proposal.txt`

Case is also inconsistent: `FEAT-01KM...` and `feat-01kpx...` appear in the same folder.

**Scattered context.** Documents for a single plan are spread across 7+ type folders. Finding "everything about P24" requires searching `design/`, `spec/`, `dev-plan/`, `reviews/`, `reports/`, `research/`, and `retro/`. This scattering worsens as plans grow in scope.

**Opaque feature IDs.** Plan IDs (`P24`) are small, sequential, and memorable. Feature IDs (`FEAT-01KMKRQRRX3CC`) are 18 characters of opaque TSID. They cannot be spoken aloud, remembered, or used naturally in conversation between team members. This is tolerable for a single developer using cut-and-paste, but blocks effective human-to-human communication in teams.

**Missing file operations.** There is no safe way to move or delete a work file without manually updating `.kbz/state/documents/` records. This makes reorganisation risky and discourages cleanup.

If nothing changes, these problems compound as the project grows. Every new file makes the existing mess slightly harder to navigate. Every new team member faces a steeper learning curve.

---

## Design

### 1. Plan-first folder organisation

Documents are organised by plan, not by type. Each plan gets a folder under `work/`:

```
work/
  P24-ac-pattern/
    P24-design-ac-pattern.md
    P24-spec-ac-pattern.md
    P24-dev-plan-ac-pattern.md
    P24-F1-spec-validator-rules.md
    P24-F2-design-error-reporting.md
    P24-review-ac-pattern.md
    P24-retro.md
    assets/
  P25-write-file/
    P25-design-write-file.md
    ...
  _project/
    research-ai-orchestration.md
    report-user-feedback.md
  templates/
    ...
```

**Rationale:** The most common workflow is "I'm working on P24, show me everything." Plan-first makes this `ls work/P24-*/`. Type-first requires searching 7+ folders. With 115 files in `spec/` alone, type folders are already hard to scan. Plan folders cap at ~5–15 files each, which is browsable.

**Plan folder naming:** `P{n}-{short-slug}/`. The slug is kept to 2–3 words maximum. During active development the plan number is well-known, but the slug helps when returning to old work or scanning a directory listing. The cost of a short slug is negligible; the benefit for orientation is real.

**The `_project/` folder:** Project-level documents that don't belong to any plan (general research, cross-cutting reports, project-wide retrospectives) live in `work/_project/`. The underscore prefix sorts it before numbered plan folders, making it easy to find. A project-level document uses the filename pattern `{type}-{slug}.md` without a plan ID.

**The `templates/` folder:** Document templates remain at `work/templates/`, outside the plan structure.

### 2. Plan-scoped feature IDs

Features receive a short, sequential ID scoped to their parent plan:

| Entity | Canonical ID | Human-facing ID | Example |
|--------|-------------|----------------|---------|
| Plan | `P{n}-{slug}` | `P24` | `P24-ac-pattern` |
| Feature | `FEAT-{TSID13}` | `P24-F3` | `FEAT-01KMKRQRRX3CC` → `P24-F3` |
| Task | `TASK-{TSID13}` | (not needed) | `TASK-01KM8JVTJ1ZC5` |

**How it works:**

- The canonical feature ID remains `FEAT-{TSID13}`. All internal state, cross-references, and storage filenames continue to use it. Nothing in `.kbz/state/` changes.
- Each feature gains a `display_id` field: `P{plan-number}-F{sequence}`. The sequence counter is per-plan, stored in the plan's state file.
- The display ID is used in filenames, conversation, commit messages, and anywhere humans interact with feature references.
- The system can resolve `P24-F3` to `FEAT-01KMKRQRRX3CC` and vice versa. Both forms are accepted as input in all tools.

**Allocation:** The sequence counter lives in `.kbz/state/plans/P{n}-{slug}.yaml` as a `next_feature_seq` field. Incrementing this counter is an atomic operation on a single file — Git's merge conflict resolution handles the rare case of two people creating features on the same plan simultaneously (the second merge gets a conflict on the counter, which is trivially resolved by incrementing again).

**Features without plans:** Every feature must belong to a plan. If a feature is genuinely standalone, it gets a single-feature plan. This is a healthy forcing function — it means every piece of work has a named scope, however small. The cost of creating a plan is one command.

**Why not user-scoped IDs (`F-sam-3`)?** User-scoped IDs embed identity into the identifier, which creates problems: what happens when sam leaves? When two people both want the prefix `s`? When a feature is reassigned? Plan-scoping avoids all of these because the scope is the work, not the person.

### 3. Canonical filename template

All work documents follow this template:

```
{plan-id}-{type}-{slug}.md
```

Where:
- `{plan-id}` is the plan ID: `P24`, `P25`, etc.
- `{type}` is the document type prefix (see §4)
- `{slug}` is a lowercase-kebab-case human description

For feature-scoped documents:

```
{plan-id}-{feature-seq}-{type}-{slug}.md
```

Where `{feature-seq}` is `F1`, `F2`, etc.

**Examples:**

| Document | Filename |
|----------|----------|
| Plan 24's design | `P24-design-ac-pattern.md` |
| Plan 24's specification | `P24-spec-ac-pattern.md` |
| Plan 24, Feature 3's specification | `P24-F3-spec-auth-flow.md` |
| Plan 24's retrospective | `P24-retro.md` |
| Plan 24, Feature 1's review | `P24-F1-review-validator-rules.md` |
| Project-level research | `research-ai-orchestration.md` |

**The slug is decoration.** The system identifies a document by `{plan-id}-{type}` (or `{plan-id}-{feature-seq}-{type}`) — the slug is ignored for lookup purposes. A human or AI agent can place anything in the slug position and the system will still resolve the document. This means:
- Renaming the slug (e.g. for clarity) does not break references
- The system enforces one document per (plan, type) or (plan, feature, type) tuple
- If two files match the same tuple but differ only in slug, that's a conflict to be resolved

**Character rules:**
- Filenames are lowercase throughout, including the plan ID and feature sequence: `p24-f3-spec-auth-flow.md`
- Exception: the plan and feature prefixes use uppercase `P` and `F` for visual distinction: `P24-F3-spec-auth-flow.md`
- Slugs use only `[a-z0-9-]`
- No spaces, no underscores, no uppercase in slugs

**No versions in filenames.** Git handles versioning. If a document is fundamentally rewritten (not edited), the `doc supersede` mechanism creates a new document record linked to the old one. Draft status is tracked in `.kbz/state/documents/`, not in the filename.

### 4. Standard document types

The following types are recognised, with their filename prefixes:

| Type | Prefix | Description |
|------|--------|-------------|
| design | `design-` | Architecture and design decisions |
| spec | `spec-` | Formal specifications with acceptance criteria |
| dev-plan | `dev-plan-` | Implementation plans and task breakdowns |
| review | `review-` | Formal review reports (code, plan, design) |
| report | `report-` | Internal analyses, evaluations, status reports |
| research | `research-` | External research, investigations, comparisons |
| retro | `retro-` | Retrospectives |
| proposal | `proposal-` | Early-stage proposals before formal design |

**Clarifications on scope:**
- `report` covers internal project analysis: evaluations, progress reports, architectural assessments, performance analyses. It is for documents generated *about* the project, by the project.
- `research` covers external investigation: technology comparisons, best-practice surveys, vendor evaluations, literature reviews. It is for documents that look *outward* to inform design decisions.
- `review` is reserved for formal reviews that follow the review workflow (conformance, quality, security, testing). A casual assessment is a `report`, not a `review`.
- `proposal` is for early-stage ideas that carry more questions than answers. A proposal becomes a `design` once the major decisions are resolved.

**Types not formalised:**
- `handoff` — currently exists as files in `plan/`. These become `report-` documents (a handoff is a project status report for context transfer).
- `evaluation` — becomes `report-` (an evaluation is an internal analysis report).
- `decision-log` — kept in plan state, not as a standalone document type.

### 5. Assets

Each plan folder may contain an `assets/` subfolder for non-Markdown files:

```
work/P24-ac-pattern/
  assets/
    wireframe-login.png
    icon-draft-v2.svg
```

Project-level assets live in `work/_project/assets/`.

**Naming within `assets/`:** File extensions do the heavy lifting for asset type identification. A naming convention within `assets/` is deferred until the project starts producing real assets. If subfolders are needed (e.g. `assets/art/`, `assets/audio/`), they can be added per-project without changing the design.

**Assets are not registered as documents.** The document registry (`.kbz/state/documents/`) tracks Markdown documents. Assets are referenced *from* documents but are not independently tracked. This keeps the document system simple and avoids creating registration overhead for every exported PNG.

### 6. File operations

Two new CLI/MCP operations are needed to maintain state consistency when humans or agents reorganise files:

#### 6.1 `kbz move`

Moves a work file to a new location, updating all document records that reference the old path.

**Usage patterns:**

```
kbz move work/design/old-name.md P24
```

Moves the file into `work/P24-{slug}/`, renaming it according to the filename template. The system infers the document type from the existing document record (or from the filename if unregistered) and constructs the canonical filename.

```
kbz move work/P24-{slug}/P24-design-foo.md P25
```

Re-parents a document from one plan to another. Updates the document record's owner field, moves the file, and renames it with the new plan prefix.

**How file vs ID ambiguity is resolved:** The first argument is always a file path (it contains `/` or `.`). The second argument is always a plan ID (it matches `P{n}` or `P{n}-{slug}`). There is no ambiguity because these patterns don't overlap.

**What `kbz move` does:**
1. Validates the source file exists and the target plan exists
2. Determines the document type (from document record or filename inference)
3. Constructs the target filename using the canonical template
4. Creates the target plan folder if it doesn't exist
5. Runs `git mv` to move the file (preserving Git history)
6. Updates the document record's `path` field
7. Updates the document record's `owner` field if the plan changed
8. Reports what it did

**What `kbz move` does not do:**
- It does not re-register documents. If the file has no document record, it moves the file but warns that no record was updated.
- It does not update references inside other documents. Cross-references are by entity ID, not by file path, so they remain valid.

#### 6.2 `kbz delete`

Removes a work file and its associated document record.

```
kbz delete work/P24-{slug}/P24-report-evaluation.md
```

**What `kbz delete` does:**
1. Validates the file exists
2. Finds the associated document record (if any)
3. Prompts for confirmation (unless `--force` is passed)
4. Runs `git rm` to delete the file
5. Removes the document record from `.kbz/state/documents/`
6. Reports what it did

**Safeguard:** Deleting an approved document requires `--force` and logs a warning. Approved documents are canonical — deleting one should be deliberate.

#### 6.3 `kbz move` for re-parenting entities

Re-parenting (changing which plan a feature belongs to) is a separate operation from file moves, though it may trigger file moves as a side-effect:

```
kbz move P24-F3 P25
```

When the first argument matches a feature display ID (`P{n}-F{n}`), this is an entity re-parent operation:
1. Updates the feature's parent plan in `.kbz/state/features/`
2. Allocates a new feature sequence number in the target plan (`P25-F{next}`)
3. Moves all of the feature's documents from the source plan folder to the target plan folder, renaming them with the new plan and feature prefix
4. Updates all document records with new paths and owner

This is a heavier operation than a simple file move. It changes entity relationships. It should prompt for confirmation.

### 7. Migration strategy

Adopting this system across 430 existing files requires a phased approach:

**Phase 1: Enforce naming on new files.** Immediately apply the filename template and plan-folder structure to all new documents. The `doc register` command validates filenames against the template and rejects non-conforming paths. Old files remain where they are.

**Phase 2: Build `kbz move` and `kbz delete`.** These are prerequisites for safe migration. They must be working and tested before bulk moves.

**Phase 3: Migrate existing files.** Write a migration script that:
1. Reads all document records from `.kbz/state/documents/`
2. Determines each document's plan, type, and current path
3. Generates `kbz move` commands for each file
4. Executes the moves in a single commit (atomic, reversible via `git revert`)

Files without document records are flagged for manual triage — they may be orphaned, deprecated, or unregistered.

**Phase 4: Remove empty old folders.** After migration, the old type-first folders (`design/`, `spec/`, etc.) should be empty. Remove them in a cleanup commit.

---

## Alternatives Considered

### Alternative A: Clean up type-first folders (status quo, improved)

Keep the current `work/design/`, `work/spec/`, etc. structure but:
- Deduplicate folders (merge `specs/` into `spec/`, etc.)
- Enforce the filename template within each type folder
- Standardise on singular folder names

**Trade-offs:**
- Easier to implement (no bulk migration needed for folder structure)
- Doesn't solve the scattering problem — a plan's documents remain spread across 7+ folders
- Doesn't scale: `spec/` already has 115 files and growing
- Type-in-filename becomes redundant when the folder already conveys type

**Rejected because:** It addresses symptoms (duplicate folders, inconsistent names) without fixing the structural cause (documents for a single plan are scattered). At 430 files and growing, this approach delays the harder reorganisation while the problem gets worse.

### Alternative B: Flat structure with no folders

Put all work documents in `work/` directly, relying entirely on the filename template for organisation:

```
work/
  P24-design-ac-pattern.md
  P24-spec-ac-pattern.md
  P25-design-write-file.md
  ...
```

**Trade-offs:**
- Maximally flat — no folder decisions to make
- Files sort naturally by plan ID
- A single `ls work/` shows everything
- At 430+ files, the listing is overwhelming
- No folder-level operations (`ls work/P24-*/` doesn't work)
- Assets have nowhere clean to live

**Rejected because:** A flat folder with hundreds of files is not browsable. The plan-folder approach gives the benefits of filename-based sorting while keeping individual folders manageable.

### Alternative C: Global sequential feature IDs

Use a global sequence counter for features: `F1`, `F2`, `F3`, independent of plans.

**Trade-offs:**
- Simpler than plan-scoped IDs (one counter, not one per plan)
- No coupling between feature ID and parent plan
- Requires a global counter — higher collision risk with multiple concurrent users
- Doesn't carry plan context: `F42` tells you nothing about which plan it belongs to
- Loses the natural grouping of `P24-F1`, `P24-F2`, `P24-F3`

**Rejected because:** Plan-scoped IDs are more informative (`P24-F3` immediately tells you the plan) and collision risk is lower (the counter is per-plan, so only concurrent feature creation on the same plan can collide — a rare event).

### Alternative D: Keep TSID feature IDs, add display formatting only

Don't introduce short feature IDs at all. Instead, improve display of existing TSIDs:
- Show `FEAT-01KMK-RQRRX3CC` (break-hyphens) in UIs
- Accept short prefixes: `FEAT-01KMK` resolves if unambiguous
- No new ID system to maintain

**Trade-offs:**
- Zero implementation cost for ID management
- No second identifier to maintain
- `FEAT-01KMK` is still not speakable or memorable
- Prefix resolution is fragile — ambiguity grows as more features are created
- Filenames remain long: `feat-01kmkrqrrx3cc-design-auth-flow.md`

**Rejected because:** The core problem is human communication in teams. "I'm working on P24-F3" is natural language. "I'm working on FEAT-01KMK" is not. The display formatting helps readability but doesn't solve speakability or memorability.

### Alternative E: User-scoped sequential IDs

Each developer gets a namespace: `sam-F1`, `sam-F2`, or single-letter: `Fs1`, `Fs2`.

**Trade-offs:**
- Distributed generation with no collisions
- Short and sequential within a user's namespace
- Embeds identity into the identifier — problematic when people leave or features are reassigned
- Two-letter collision (`sam` vs `sarah` both wanting `s`)
- The user prefix is noise when you're the only person on a feature
- Doesn't group by plan — loses the "P24-F1, P24-F2, P24-F3" affordance

**Rejected because:** Scoping by person creates fragile identifiers that break on team changes. Scoping by plan creates identifiers that group related work, which is more useful.

---

## Decisions

### D1: Plan-first folder organisation

**Decision:** Organise `work/` by plan, not by document type.

**Context:** The project has 430 files in 18 folders, with 4 duplicate folder pairs. Documents for a single plan are scattered across 7+ type folders. The largest type folder (`spec/`) has 115 files.

**Rationale:** The most common access pattern is "show me everything for the plan I'm working on." Plan-first makes this a single folder listing. Type information is carried in the filename prefix, so "show me all designs" is still answerable via `find work -name "*-design-*"`. Individual plan folders stay small (5–15 files), keeping them browsable as the project scales.

**Consequences:**
- All 430 existing files will need to be migrated (Phase 3 of the migration strategy)
- New AI agent instructions must specify plan folders, not type folders
- `doc register` must validate that files are in the correct plan folder
- Cross-plan queries ("all designs") require `find` or `grep` instead of `ls` — an acceptable trade-off
- The `_project/` folder handles documents that don't belong to any plan

### D2: Plan-scoped feature display IDs

**Decision:** Features receive a human-facing display ID in the form `P{n}-F{seq}`, allocated by a per-plan sequence counter. The canonical `FEAT-{TSID13}` ID is retained for all internal state.

**Context:** Plan IDs (`P24`) are small and memorable. Feature IDs (`FEAT-01KMKRQRRX3CC`) are 18-character opaque strings that cannot be spoken or remembered. Multi-person teams need speakable identifiers.

**Rationale:** Plan-scoping gives features the same human qualities as plans: small, sequential, contextual. The sequence counter is per-plan, minimising collision risk. The canonical TSID is preserved so that all internal referencing, storage, and distributed generation continue to work unchanged. The display ID is a layer on top, not a replacement.

**Consequences:**
- Plan state files gain a `next_feature_seq` counter field
- Feature state files gain a `display_id` field
- All tools must accept both `P24-F3` and `FEAT-01KMKRQRRX3CC` as input
- Display ID resolution must be fast (in-memory lookup from plan/feature state)
- Features must belong to a plan (standalone features get single-feature plans)
- Existing features need display IDs allocated during migration

### D3: Canonical filename template

**Decision:** Work documents follow the template `{PlanID}-{type}-{slug}.md` or `{PlanID}-{FeatureSeq}-{type}-{slug}.md`. The slug is human decoration; the system identifies documents by the structured prefix.

**Context:** Seven different naming patterns coexist in the project. Case, prefix style, and slug format are all inconsistent. AI agents generate filenames unpredictably without a strict template.

**Rationale:** A canonical template eliminates ambiguity for both humans and AI agents. Making the slug decorative (ignored for lookup) means humans can name files naturally without breaking references. The structured prefix (`P24-design-`, `P24-F3-spec-`) provides plan context, type context, and uniqueness.

**Consequences:**
- `doc register` enforces the template on new registrations
- Existing files must be renamed during migration
- The system must parse filenames to extract plan ID, optional feature sequence, and type
- Slug changes (renames for clarity) don't require document record updates — only the structured prefix matters for identity
- Templates and project-level documents have simpler patterns (no plan prefix)

### D4: Eight standard document types

**Decision:** The system recognises eight document types: design, spec, dev-plan, review, report, research, retro, and proposal. `review` is reserved for formal reviews. `report` covers internal analysis. `research` covers external investigation.

**Context:** The system currently tracks six types (specification, dev-plan, report, design, research, retrospective). The `report` type is overloaded — it covers review reports, configuration references, user docs, and general reports. `proposal` and `evaluation` exist as folders but not as formal types.

**Rationale:** Separating `review` from `report` clarifies intent: a review follows the formal review workflow with conformance/quality/security/testing dimensions; a report is an internal analysis document. Absorbing `evaluation` into `report` and adding `proposal` as a distinct type matches how documents are actually created and used. Eight types is enough to be precise without being bureaucratic.

**Consequences:**
- Existing review reports change type from `report` to `review`
- `evaluation` documents become `report` type
- `proposal` is added to the type registry
- The document-centric interface design (§4) gains an additional recognised type
- Document templates should be provided for each type

### D5: File operation tools

**Decision:** Introduce `kbz move` and `kbz delete` commands that maintain document record consistency. `kbz move` handles both file relocation and entity re-parenting.

**Context:** There is no safe way to move or delete a work file without manually updating `.kbz/state/documents/` records. This makes reorganisation risky and discourages cleanup.

**Rationale:** File operations that don't update state create orphaned records and broken paths. Wrapping `git mv` and `git rm` with state updates makes reorganisation safe. Combining file move and entity re-parent into one command (distinguished by argument pattern) keeps the interface small.

**Consequences:**
- New `move` and `delete` subcommands in the CLI
- New `move` and `delete` MCP tool actions (or extensions to existing tools)
- `git mv` and `git rm` are used under the hood to preserve Git history
- Approved document deletion requires `--force` as a safeguard
- The migration script (Phase 3) is built on top of `kbz move`

### D6: No configurability for structure

**Decision:** The folder structure, filename template, and document types are convention, not configuration. Only the plan ID prefix character and the work directory name are configurable.

**Context:** Configuration creates documentation burden, testing combinations, and "which settings were chosen?" confusion for new team members. AI agents must read config before every file operation if conventions are configurable.

**Rationale:** Opinionated defaults are a feature. Every project using Kanbanzai shares the same conventions, which means skills, documentation, and agent instructions are universal. The only things worth configuring are the plan prefix (for monorepos that need `BE-P`, `FE-P` namespacing) and the work directory name (some projects may prefer `docs/`).

**Consequences:**
- Simpler implementation — no settings parsing for structural decisions
- Skills and agent instructions can hard-code conventions
- Projects that need radically different organisation must fork or extend rather than configure
- Reduces the surface area for inconsistency across projects

---

## Open Questions

1. **Multiple documents of the same type per plan.** The one-document-per-(plan, type) constraint works for most cases, but a plan might legitimately have two research documents. Should the slug serve as a disambiguator in this case? If so, the system would identify documents by `{plan-id}-{type}-{slug}`, not just `{plan-id}-{type}`. This makes the slug load-bearing for identity, which contradicts the "slug is decoration" principle. A resolution: allow a numeric suffix (`P24-research-1-ai-orchestration.md`, `P24-research-2-naming-conventions.md`) where the number is part of the type identifier, not the slug.

2. **Retrospective file naming.** A plan typically has one retro, but it could have multiple (mid-plan check-in, final retro). Same resolution as above applies.

3. **What about `docs/` directory?** Some documents (getting-started, configuration-reference) currently live in `docs/` and are registered to features. Should published user documentation follow the same plan-folder convention, or does `docs/` remain separate as the public-facing documentation tree? Recommendation: `docs/` stays separate — it's the output of the system, not a working document. But this needs confirming.

4. **Meta-planning IDs.** If a layer above plans is added in the future, plan IDs might need a programme prefix (e.g. `A-P24` for programme A). The current design doesn't preclude this — it would be an extension to the plan ID format. Deferred.

5. **Asset naming conventions.** Detailed conventions for `assets/` subfolders and file naming are deferred until real assets are produced. The design accommodates assets without constraining their naming.