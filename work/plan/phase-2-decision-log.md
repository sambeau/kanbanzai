# Phase 2 Decision Log

This document records architectural and scope decisions for Phase 2 of the Kanbanzai project.

It follows the same format as `work/plan/phase-1-decision-log.md`. Phase 1 decisions (P1-DEC-001 through P1-DEC-021) remain in that file. Phase 2 decisions begin at P2-DEC-001.

---

## Decision Register

---

## `P2-DEC-001: Knowledge lifecycle MVP scope for Phase 2b`

- Status: accepted
- Date: 2026-03-24
- Scope: phase-2b, knowledge-lifecycle
- Related:
  - `work/design/machine-context-design.md` §9
  - `work/spec/phase-2-specification.md` §19, §22.12
  - `work/plan/phase-2-scope.md` §9.3

### Decision

Phase 2b ships a minimum viable knowledge lifecycle. The full model from the machine-context design is split into two delivery tranches.

**Phase 2b delivers:**

- **Contribute and retrieve** — agents write knowledge entries via `context_contribute`, agents read them via scoped retrieval. This is the floor.
- **Deduplication on write** — exact topic match rejects; Jaccard word-set similarity > 0.65 flags near-duplicates with a pointer to the existing entry.
- **Status lifecycle** — entries follow: `contributed → confirmed → disputed → stale → retired`. At minimum, `contributed`, `confirmed`, and `retired` are functional. `disputed` and `stale` may be set manually or by agent feedback but do not require automated detection in Phase 2b.
- **Wilson confidence scoring** — each entry carries a confidence score computed from `(use_count, miss_count)` using Wilson score lower bound. New entries start at 0.5.
- **Usage reporting** — bundled with task completion. Default is "all entries were fine"; only negatives require detail. Reports update `use_count` and `miss_count`.
- **Tier-dependent confidence filtering** — context assembly uses minimum confidence thresholds per tier (Tier 1: no filter, Tier 2: > 0.3, Tier 3: > 0.5).

**Phase 3 delivers (deferred):**

- **Git anchoring and automatic staleness detection** — tying entries to file paths, flagging when files change. Without this, staleness is reactive: agents discover wrong knowledge via usage, report it, and confidence drops.
- **TTL-based automatic pruning** — tier-dependent TTLs with reset-on-use and prune-on-expiry. In Phase 2b, humans retire entries manually.
- **Automatic promotion triggers** — Tier 3 → Tier 2 promotion when `use_count ≥ 5, miss_count = 0`. In Phase 2b, humans promote manually.
- **Post-merge compaction** — detecting duplicates and contradictions across merged branches. In Phase 2b, duplicates from parallel branches are handled manually or by a coordinator agent.

### Rationale

The core value of Phase 2b is "agents can share and retrieve scoped knowledge." The quality and hygiene mechanisms (pruning, compaction, staleness detection) matter more as the knowledge store grows — they solve scaling problems that don't exist yet when the system is first adopted.

The MVP subset is the smallest set that closes the feedback loop: agents contribute knowledge, other agents consume it, usage data flows back, and confidence scores reflect actual reliability. Everything deferred to Phase 3 is an optimisation on top of that working loop.

Git anchoring is the most valuable deferral. Without it, staleness detection is reactive — an agent retrieves a stale entry, tries to use it, discovers it's wrong, and reports it. This costs one wasted attempt per stale entry. With git anchoring, the system catches staleness proactively. This is valuable but not load-bearing for the MVP.

### Alternatives Considered

- **Ship the full lifecycle in Phase 2b.** Rejected — post-merge compaction alone is a substantial subsystem (detection, auto-merge, conflict marking, scope-based rules). Bundling it with the base knowledge system risks making Phase 2b as large as Phase 2a.
- **Ship only contribute and retrieve, without confidence scoring.** Rejected — confidence scoring is what makes the system self-correcting. Without it, wrong knowledge persists indefinitely until a human notices. The Wilson score implementation is small (a single formula) and the usage reporting protocol is lightweight (~50–100 tokens per report).
- **Ship TTL pruning but defer confidence scoring.** Rejected — TTL pruning without confidence data is arbitrary (it prunes old entries regardless of quality). Confidence scoring without TTL pruning is safe (entries accumulate but don't rot silently). The right order is confidence first, then TTL.

### Consequences

- Phase 2b knowledge entries carry all lifecycle and retention fields from the design (`use_count`, `miss_count`, `confidence`, `ttl_days`, etc.) but `ttl_days` is informational only — no automated pruning acts on it.
- `git_anchors` is defined in the schema as an optional field but is not processed by any staleness detection system in Phase 2b.
- Manual retirement via an MCP tool is the primary mechanism for removing bad knowledge. Agents can also flag entries via usage reporting, which degrades confidence scores.
- Phase 3 scope is defined: git anchoring, TTL pruning, promotion triggers, post-merge compaction.

### Follow-up Needed

- Formalise the KnowledgeEntry schema as a spec-grade entity definition (P2-DEC-001 defines scope; schema formalisation is a separate task, see open question §9.1 in `phase-2-scope.md`)
- Define MCP tool names and request/response shapes for knowledge operations
- Define the usage reporting protocol (what fields are included in task completion reports)

---

## `P2-DEC-002: Context profile inheritance uses leaf-level replace`

- Status: accepted
- Date: 2026-03-24
- Scope: phase-2b, context-profiles
- Related:
  - `work/design/machine-context-design.md` §6.6
  - `work/spec/phase-2-specification.md` §19.1
  - `work/plan/phase-2-scope.md` §9.2

### Decision

Context profile inheritance uses **leaf-level replace** semantics, following the CSS cascade model.

When the system walks the inheritance chain (e.g., `base → developer → backend`), each profile's fields are layered in order. If a child profile defines a key that a parent also defines, the child's value **completely replaces** the parent's value at that key. There is no deep merge.

Specifically:

- **Scalar fields** (strings, numbers): child wins.
- **List fields** (e.g., `conventions`, `packages`): child's list replaces the parent's list entirely. Lists are not concatenated or deduplicated.
- **Map fields** (e.g., `architecture`): child's map replaces the parent's map at the top level. Individual keys within a map are not merged from parent and child.
- **Absent fields**: if the child does not define a field, the parent's value is inherited unchanged.

If a child profile wants the parent's conventions plus its own, it must include the parent's entries explicitly in its own list. This is deliberate — it makes the resolved profile predictable by reading a single file, without needing to mentally walk the inheritance chain.

### Rationale

Deep merge is appealing in theory but creates hard-to-debug behaviour. When two lists are concatenated from different ancestors, it becomes unclear where a specific entry came from, whether order matters, and how to remove an inherited entry. CSS solved this decades ago: specificity wins, and if you want to compose, you do it explicitly.

The target audience for profile authoring is humans writing short YAML files. A profile typically has 3–10 convention entries and a handful of architecture notes. Explicit inclusion of parent entries is not burdensome at this scale.

The system enforces only two structural rules: inheritance references must resolve, and the inheritance graph must not contain cycles. Everything else is the profile author's responsibility.

### Alternatives Considered

- **Deep merge with concatenation.** Lists from parent and child are concatenated. Rejected — produces surprising results when a child profile intends to narrow the parent's scope (e.g., replace a broad convention with a specific one). No clean way to "remove" an inherited entry.
- **Deep merge with explicit override markers.** Use `!replace` or `!override` annotations to control merge behaviour per field. Rejected — adds YAML complexity for marginal benefit. The profile files are small enough that explicit inclusion is simpler.
- **No inheritance.** Each profile is fully self-contained. Rejected — leads to massive duplication across profiles. The `base → developer → backend` pattern is too useful for sharing project-wide conventions.

### Consequences

- Profile resolution is a simple loop: start with an empty map, iterate from root ancestor to leaf, shallow-merge each profile's fields.
- Profile authors can reason about the resolved profile by reading the leaf profile and its ancestors in order. The last definition of any field wins.
- The system does not need merge-conflict detection or resolution logic — there are no conflicts, only replacements.
- Documentation should include a "resolved profile" view (the fully flattened output) so authors can verify what agents will actually receive.

### Follow-up Needed

- Implement the resolution algorithm in the context assembly layer
- Add a `kbz context resolve --profile <name>` command (or MCP equivalent) that shows the fully resolved profile for debugging
- Document the inheritance model in user-facing docs with examples

---

## `P2-DEC-003: Batch document import for project bootstrap`

- Status: accepted
- Date: 2026-03-24
- Scope: phase-2b, bootstrap, document-management
- Related:
  - `work/design/machine-context-design.md` §14
  - `work/spec/phase-2-specification.md` §11
  - `work/plan/phase-2-scope.md` §9.4

### Decision

Phase 2b includes a **batch document import** capability for onboarding existing projects. This is a thin orchestration layer over the existing `doc_record_submit` primitive.

The batch import operation:

1. Accepts a directory path and an optional glob pattern (e.g., `work/**/*.md`).
2. For each matching file, calls the equivalent of `doc_record_submit` — creating a document record in `draft` status, computing the content hash, and running Layers 1–2 (structural parsing).
3. Infers document type from file path conventions where possible (e.g., files under `design/` default to `design`, files under `spec/` default to `specification`). Falls back to a caller-provided default or requires explicit mapping.
4. Uses the auto-resolved `created_by` value (see P2-DEC-004).
5. Skips files that already have document records (idempotent).
6. Returns a summary: files imported, files skipped, errors encountered.

Classification (Layer 3) remains incremental — after batch import, an agent uses `doc_pending` to list unclassified documents and `doc_classify` to classify them. Batch import does not attempt automated classification.

The operation is exposed as both a CLI command (`kbz import`) and an MCP tool (`batch_import_documents`).

Knowledge extraction from existing code (option 3 from the machine-context design §14.2) is **not** included in Phase 2b. It is a separate capability that can follow in Phase 3 once the knowledge system is proven.

### Rationale

Any project with existing design documents needs a practical way to bring them into the system. Submitting 30+ documents one at a time through individual `doc_record_submit` calls is tedious and error-prone. The batch operation is mechanically simple — it's a loop with path matching and error collection — but essential for adoption.

Classification is deliberately excluded from the batch path because it requires agent judgment (Layer 3 is AI-assisted). The existing incremental workflow (`doc_pending` → `doc_classify`) is already built and works well for this. Forcing classification during import would either require a connected agent or produce low-quality automated classifications.

### Alternatives Considered

- **No batch operation — agents import documents one at a time.** Rejected — impractical for projects with more than a handful of existing documents. The Kanbanzai project itself has 30+ files in `work/` that would need individual submission.
- **Batch import with automatic classification.** Rejected — classification requires agent judgment. Mixing mechanical import with AI-assisted classification in a single operation conflates two concerns with different reliability profiles.
- **Full bootstrap wizard.** A guided interactive flow that walks the user through import, classification, profile creation, and initial knowledge seeding. Rejected as over-engineering for Phase 2b — the primitives should work well individually before composing them into a guided flow.

### Consequences

- Phase 2b adds one new CLI command (`kbz import`) and one new MCP tool (`batch_import_documents`)
- The import operation depends on document type inference heuristics, which need to be defined (path-based defaults, override mappings)
- Post-import, the `doc_pending` list will contain all newly imported documents awaiting classification — agents should expect a potentially large backlog on first run
- Idempotency means the import can be re-run safely after adding new files to the project

### Follow-up Needed

- Define the document type inference heuristics (which path patterns map to which document types)
- Decide whether `kbz import` should auto-detect the project's document directories or require explicit paths
- Consider whether import should assign `owner` (parent Plan or Feature) based on directory structure or leave it unset for manual assignment

---

## `P2-DEC-004: created_by auto-resolution from git config with local override`

- Status: accepted
- Date: 2026-03-24
- Scope: phase-2b, identity, configuration
- Related:
  - `work/spec/phase-2-specification.md` §8
  - `work/plan/phase-2-scope.md`

### Decision

The `created_by` field on entities and document records is auto-resolved when the caller does not provide an explicit value. The resolution order is:

1. **Explicit value** — if the caller passes `created_by`, that value is used. Agents may have legitimate reasons to attribute work to a specific identity.
2. **Local config** — if `.kbz/local.yaml` exists and contains `user.name`, that value is used.
3. **Git config** — the output of `git config user.name` is used.
4. **Error** — if none of the above produces a value, the operation fails with a clear error message. The system does not silently default to `"human"` or any other placeholder.

The `.kbz/local.yaml` file is added to `.gitignore`. It holds per-machine settings that must not be shared across collaborators. Initial schema:

```yaml
user:
  name: sambeau
```

The `created_by` field remains a required stored field on every entity and document record. What changes is that it becomes **optional on input** — callers can omit it and the system fills it in. This applies to all MCP tools and CLI commands that accept `created_by`.

### Rationale

Every entity and document record in the system currently says `created_by: human` because there is no automatic resolution. This is uninformative and will become actively harmful when multiple people contribute to a project — all records look identical regardless of who created them.

Git config is the right automatic default because it is universally available, already configured on every developer's machine, requires zero setup, and reflects the identity the person uses for their commits. For this project, `git config user.name` returns `sambeau`, which is immediately more useful than `human`.

The local config override exists for cases where the git username isn't what someone wants in entity records (e.g., their git name is a full name like "Sam Phillips" but they prefer `sam` in entity records). It also supports CI/CD environments where git config may not be set or may be set to a service account name.

The `.kbz/local.yaml` file is gitignored because it contains per-machine preferences. If it were committed, it would be overwritten every time a different collaborator pushes.

### Alternatives Considered

- **Global setting in `.kbz/config.yaml`.** Rejected — config.yaml is committed to git. A `user.name` field there would be overwritten by every collaborator's push. This is the problem the user identified.
- **Environment variable (`KBZ_USER`).** Viable as an additional resolution step but insufficient on its own — environment variables are not persistent across shell sessions without explicit shell config. Could be added later as a step between local config and git config if needed.
- **Prompt on first use.** Rejected — the system should work without interactive setup. A sensible automatic default (git config) is better than requiring every new user to answer a prompt.
- **Keep `created_by` required on input.** Rejected — this is the status quo and it produces `created_by: human` everywhere. Agents don't know the user's identity, so they pass a placeholder.

### Consequences

- All MCP tools and CLI commands that accept `created_by` must be updated to treat it as optional, falling back to the resolution chain.
- A new `internal/config/user.go` (or similar) module implements the resolution chain: check local config → shell out to `git config user.name` → error.
- `.kbz/local.yaml` is added to `.gitignore`.
- Existing entity records with `created_by: human` are not automatically updated — they retain their original value. A future migration or manual edit can correct them.
- This change should be implemented before any batch import (P2-DEC-003) to avoid creating a large batch of records with placeholder attribution.

### Follow-up Needed

- Implement the resolution chain in a shared module usable by both MCP and CLI layers
- Update all MCP tool handlers and CLI commands to use auto-resolution when `created_by` is not provided
- Add `.kbz/local.yaml` to `.gitignore`
- Consider whether `approved_by` should use the same resolution chain (it likely should)
- Document the local config file in user-facing docs
- Consider implementing this before Phase 2b proper, as a standalone improvement

---

## Summary

This log records Phase 2 architectural and scope decisions. It is a companion to `work/plan/phase-1-decision-log.md`, which covers Phase 1 decisions P1-DEC-001 through P1-DEC-021.

| ID | Title | Status | Date |
|---|---|---|---|
| P2-DEC-001 | Knowledge lifecycle MVP scope for Phase 2b | accepted | 2026-03-24 |
| P2-DEC-002 | Context profile inheritance uses leaf-level replace | accepted | 2026-03-24 |
| P2-DEC-003 | Batch document import for project bootstrap | accepted | 2026-03-24 |
| P2-DEC-004 | created_by auto-resolution from git config with local override | accepted | 2026-03-24 |