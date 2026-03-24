# Phase 2b Specification: Context Management and Agent Capabilities

This document is the binding specification for Phase 2b of the Kanbanzai project. It supersedes the high-level requirements in `work/spec/phase-2-specification.md` §19–20 and §22.12–22.13, which served as the initial scope definition.

Phase 2b builds on the foundation delivered by Phase 2a (entity model evolution, document management, document intelligence) to add context management, knowledge persistence, and agent capabilities.

**Reading prerequisites:**

- `work/spec/phase-2-specification.md` — Phase 2a specification (implemented and complete)
- `work/design/machine-context-design.md` — design basis for context management
- `work/plan/phase-2-decision-log.md` — P2-DEC-001 through P2-DEC-004

---

## 1. Purpose

Phase 2b makes the system assemble targeted context for agents and persist knowledge across sessions. Where Phase 2a taught the system to understand documents, Phase 2b teaches it to understand what each agent needs to know.

The core deliverables are:

1. **Context profiles** — named role definitions that scope what knowledge each agent receives
2. **Knowledge entries** — persistent records of project knowledge contributed by agents during work, with lifecycle and confidence tracking
3. **Context assembly** — composing design context (from document intelligence), implementation context (from knowledge entries), and project conventions (from profiles) into targeted packets per agent per task
4. **Agent capabilities** — link resolution and duplicate detection that leverage the document intelligence and entity layers
5. **Batch document import** — practical onboarding for existing projects
6. **User identity resolution** — automatic attribution from git config with local override

---

## 2. Goals

1. Every agent session starts with the right context for its role and task — no more, no less.
2. Knowledge produced during agent work persists and improves over time through a confidence feedback loop.
3. The system remains simple for small projects (one profile, a handful of knowledge entries) and scales to teams with multiple specialist roles.
4. Existing projects with documents can be onboarded practically, not one file at a time.
5. Entity attribution reflects the actual contributor, not a placeholder.

---

## 3. Scope

### 3.1 In scope for Phase 2b

- Context profile definition with inheritance
- Context assembly with byte-based budgeting
- Knowledge contribution with deduplication on write
- KnowledgeEntry entity with lifecycle (contributed → confirmed → retired)
- Wilson confidence scoring with usage reporting
- Tier-dependent confidence filtering during assembly
- Link resolution (entity link suggestions from free-text)
- Duplicate detection (candidate surfacing at entity creation time)
- Document-to-entity extraction guidance
- Batch document import for project bootstrap
- User identity auto-resolution (`created_by` from git config / local config)

### 3.2 Deferred to Phase 3

Per P2-DEC-001, the following knowledge lifecycle features are deferred:

- Git anchoring and automatic staleness detection
- TTL-based automatic pruning
- Automatic promotion triggers (Tier 3 → Tier 2)
- Post-merge compaction
- Knowledge extraction from existing code (automated codebase analysis)

### 3.3 Excluded from Phase 2

- Orchestration or agent delegation (Phase 4)
- Git worktree management (Phase 3)
- Cross-project knowledge sharing (Phase 3+)
- Embedding-based semantic similarity for deduplication (Phase 3+ if needed)
- Automatic context assembly optimisation (Phase 3+)

---

## 4. Design Principles

### 4.1 The tool is a database; agents provide intelligence

The system stores, indexes, and retrieves knowledge. It does not generate knowledge, infer meaning, or make autonomous decisions about what is true. Agents contribute knowledge through explicit operations; the system validates structure and manages lifecycle.

### 4.2 Context is scoped, not broadcast

Every piece of context belongs to a scope — a role, a project, a tier. Context assembly uses these scopes to deliver only what is relevant. An agent working on the backend does not receive frontend conventions.

### 4.3 Confidence is earned, not declared

New knowledge starts uncertain (confidence 0.5). Confidence increases through successful use and decreases when knowledge is found wrong. The system never treats a new contribution as authoritative, regardless of the contributor's identity.

### 4.4 Graceful degradation by layer

A project with no context profiles still works — agents use the MCP tools directly. A project with profiles but no knowledge entries still works — agents get role conventions without accumulated knowledge. Each layer adds value independently.

### 4.5 Leaf-level replace for composition

Per P2-DEC-002, when profiles inherit from parent profiles, child values completely replace parent values at each key. No deep merge, no list concatenation. If a child wants the parent's entries plus its own, it includes them explicitly. This follows the CSS cascade model.

---

## 5. Context Tiers

Implementation context is structured into three tiers, distinguished by scope and stability. The tier model determines how knowledge is stored, filtered, and eventually pruned.

### 5.1 Tier 1: Project conventions

Relatively stable knowledge that applies across the entire project: language choices, code organisation patterns, naming conventions, error handling patterns, test patterns, build conventions.

Tier 1 knowledge is represented by context profiles — specifically, the root profile (typically `base`) and its conventions. Tier 1 is human-curated and authoritative. It is not stored as KnowledgeEntry records.

### 5.2 Tier 2: Architecture knowledge

Moderately stable knowledge about how the codebase is structured and how its parts interact: module boundaries, key abstractions, data flow patterns, integration patterns, known constraints.

Tier 2 knowledge is stored as KnowledgeEntry records with `tier: 2`. It is contributed by agents and confirmed through review or repeated successful use.

### 5.3 Tier 3: Session knowledge

Ephemeral but shareable knowledge produced during specific agent work sessions: failed approaches, non-obvious interactions, clarifications from humans, workarounds.

Tier 3 knowledge is stored as KnowledgeEntry records with `tier: 3`. It is contributed by agents and has a shorter expected lifespan than Tier 2.

---

## 6. KnowledgeEntry Entity

### 6.1 Purpose

A KnowledgeEntry is a persistent record of project knowledge contributed by an agent during work. It represents a single fact, convention, pattern, or constraint that future agents working in the same area should know.

KnowledgeEntry is a new entity type, managed through MCP operations and stored as Git-tracked YAML files.

### 6.2 ID format

KnowledgeEntry IDs use the TSID13 format established in P1-DEC-021:

```
KE-{TSID13}
```

Examples: `KE-01J3K7MXP3RT5`, `KE-01J4AR7WHN4F2`

The `KE-` prefix is added to the entity type pattern table:

| Pattern | Entity type |
|---------|-------------|
| `FEAT-{id}` | Feature |
| `TASK-{id}` | Task |
| `BUG-{id}` | Bug |
| `DEC-{id}` | Decision |
| `KE-{id}` | KnowledgeEntry |
| `{X}{n}-{slug}` | Plan |

### 6.3 Required fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | yes | Format: `KE-{TSID13}` |
| `tier` | enum | yes | `2` or `3` |
| `topic` | string | yes | Normalised topic key (lowercase, hyphenated) |
| `scope` | string | yes | Profile name (e.g., `backend`) or `project` for project-wide |
| `content` | string | yes | The knowledge content — a concise, actionable statement |
| `learned_from` | string | no | Task ID or other provenance reference |
| `status` | enum | yes | Current lifecycle state |
| `created` | timestamp | yes | Creation time |
| `created_by` | string | yes | Contributor identity (auto-resolved per P2-DEC-004) |
| `updated` | timestamp | yes | Last modification time |
| `last_used` | timestamp | no | Last time this entry was retrieved and used by an agent |
| `use_count` | integer | yes | Number of times used successfully (default: 0) |
| `miss_count` | integer | yes | Number of times found wrong or unhelpful (default: 0) |
| `confidence` | float | yes | Wilson score lower bound (default: 0.5) |
| `ttl_days` | integer | no | Tier-dependent TTL in days (informational in Phase 2b — not enforced by automated pruning) |
| `promoted_from` | string | no | ID of the Tier 3 entry this was promoted from (Phase 3 automation; manual in Phase 2b) |
| `merged_from` | list of strings | no | IDs of entries merged during compaction (Phase 3) |
| `deprecated_reason` | string | no | Reason for retirement, if status is `retired` |
| `git_anchors` | list of strings | no | File paths for staleness detection (Phase 3 — stored but not processed in Phase 2b) |
| `tags` | list of strings | no | Freeform tags for cross-cutting organisation |

### 6.4 Field order for deterministic YAML

Identity → tier/scope → content → lifecycle → retention → provenance → tags → timestamps → supersession.

```
id, tier, topic, scope, content, learned_from, status,
use_count, miss_count, confidence, last_used, ttl_days,
promoted_from, merged_from, deprecated_reason, git_anchors,
tags, created, created_by, updated
```

### 6.5 Default values

| Field | Default |
|-------|---------|
| `status` | `contributed` |
| `use_count` | `0` |
| `miss_count` | `0` |
| `confidence` | `0.5` |
| `tier` | `3` (if not specified on contribution) |
| `ttl_days` | `30` for Tier 3, `90` for Tier 2 |

---

## 7. Knowledge Lifecycle

### 7.1 States

| State | Description |
|-------|-------------|
| `contributed` | Proposed by an agent, not yet confirmed |
| `confirmed` | Accepted — by human review, coordinator review, or repeated successful use |
| `disputed` | Contradicts another entry; both flagged for resolution |
| `stale` | Flagged because the knowledge may be outdated |
| `retired` | Explicitly marked as no longer applicable |

### 7.2 Transitions

| From | To | Trigger |
|------|----|---------|
| `contributed` | `confirmed` | Manual confirmation via `knowledge_confirm`, or automated when `use_count ≥ 3` and `miss_count = 0` |
| `contributed` | `disputed` | Manual — flagged as contradicting another entry |
| `contributed` | `retired` | Manual retirement, or `miss_count ≥ 2` |
| `confirmed` | `disputed` | Manual — new contradictory evidence |
| `confirmed` | `stale` | Manual — flagged as potentially outdated |
| `confirmed` | `retired` | Manual retirement, or `miss_count ≥ 2` after re-evaluation |
| `disputed` | `confirmed` | Manual — conflict resolved in favour of this entry |
| `disputed` | `retired` | Manual — conflict resolved against this entry |
| `stale` | `confirmed` | Manual — re-confirmed as still accurate |
| `stale` | `retired` | Manual — confirmed as outdated |

**Terminal state:** `retired`. Retired entries are retained in storage for audit purposes but excluded from context assembly and query results by default.

### 7.3 Transition enforcement

The system must reject lifecycle transitions that are not in the transition table above.

The auto-confirmation transition (`contributed → confirmed` when `use_count ≥ 3, miss_count = 0`) must occur as a side effect of usage reporting. It must not be triggered by direct `use_count` manipulation.

The auto-retirement transition (`miss_count ≥ 2`) must set `deprecated_reason` to a system-generated message indicating retirement due to negative feedback.

### 7.4 Phase 2b lifecycle scope

Per P2-DEC-001, the following lifecycle features are functional in Phase 2b:

- All states and transitions listed above
- Auto-confirmation at `use_count ≥ 3`
- Auto-retirement at `miss_count ≥ 2`
- Manual transitions via MCP tools

The following are deferred to Phase 3:

- Automatic TTL-based pruning (the `ttl_days` field is stored but not acted upon)
- Automatic promotion from Tier 3 to Tier 2 (manual promotion is supported)
- Git-anchor-based staleness detection (the `git_anchors` field is stored but not processed)
- Post-merge compaction

---

## 8. Confidence Scoring

### 8.1 Wilson score lower bound

Each KnowledgeEntry carries a `confidence` score computed from `(use_count, miss_count)` using the Wilson score lower bound for a binomial proportion:

```
p̂ = use_count / (use_count + miss_count)
n = use_count + miss_count
z = 1.96 (95% confidence interval)

confidence = (p̂ + z²/2n − z × √(p̂(1−p̂)/n + z²/4n²)) / (1 + z²/n)
```

When `n = 0` (no observations), `confidence` is set to 0.5 (uncertain prior).

The Wilson score naturally penalises low sample sizes: an entry used once successfully (1/1 = 100%) scores lower than an entry used 50 times with 2 misses (48/50 = 96%).

### 8.2 Confidence updates

The `confidence` field must be recomputed whenever `use_count` or `miss_count` changes. This occurs:

- When usage reporting increments `use_count` for a successfully used entry
- When usage reporting increments `miss_count` for an entry found wrong
- When an agent manually flags an entry via `knowledge_flag`

### 8.3 Tier-dependent confidence filtering

During context assembly, knowledge entries are filtered by a minimum confidence threshold that varies by tier:

| Tier | Minimum confidence | Rationale |
|------|-------------------|-----------|
| Tier 1 (conventions) | No filter | Human-curated, authoritative (not KnowledgeEntry records) |
| Tier 2 (architecture) | 0.3 | Exclude entries that are mostly wrong |
| Tier 3 (session) | 0.5 | Higher bar — unproven entries excluded from assembled context |

Entries below their tier's confidence threshold are excluded from `context_assemble` results but remain queryable via `knowledge_list` and `knowledge_get`.

---

## 9. Knowledge Contribution

### 9.1 Contribution mechanism

Agents contribute knowledge through the `knowledge_contribute` MCP tool. Contribution is deliberate and opt-in — agents contribute when they discover a convention, pattern, or constraint that future agents should know.

The contributor provides: `topic`, `content`, `scope`, and optionally `tier` and `learned_from`. The system assigns the ID, sets initial lifecycle fields, and performs deduplication checks.

### 9.2 Deduplication on write

When an agent calls `knowledge_contribute`, the system checks for existing entries before creating a new one:

1. **Exact topic match** — if an entry with the same `topic` key already exists in the same `scope`, the contribution is rejected. The response includes a pointer to the existing entry so the agent can update it instead.

2. **Near-duplicate detection** — the system computes Jaccard similarity over normalised word-sets (lowercase, stop words removed) between the new `content` and existing entries in the same `scope`. If similarity exceeds 0.65, the contribution is rejected with a pointer to the near-duplicate entry.

The system must not silently discard contributions. Rejections must include the ID and topic of the conflicting entry so the agent can take appropriate action (update the existing entry, or contribute with a distinct topic).

### 9.3 Topic normalisation

Topic keys must be normalised on contribution:

- Lowercase
- Spaces and underscores replaced with hyphens
- Consecutive hyphens collapsed
- Leading and trailing hyphens stripped

Example: `"API JSON Naming Convention"` → `"api-json-naming-convention"`

---

## 10. Usage Reporting

### 10.1 Reporting mechanism

When an agent completes a task, it reports which knowledge entries were used and whether they were accurate. This is done through the `context_report` MCP tool.

### 10.2 Report structure

A usage report contains:

- `task_id` — the completed task
- `used` — list of KnowledgeEntry IDs that were retrieved and found accurate
- `flagged` — (optional) list of entries found wrong or unhelpful, each with `entry_id` and `reason`

### 10.3 Report processing

For each entry in `used`:
- Increment `use_count` by 1
- Update `last_used` to the current timestamp
- Recompute `confidence`
- If the entry is in `contributed` status and `use_count ≥ 3` and `miss_count = 0`, transition to `confirmed`

For each entry in `flagged`:
- Increment `miss_count` by 1
- Recompute `confidence`
- If `miss_count ≥ 2`, transition to `retired` with `deprecated_reason` set from the flag reason
- Store the flag reason for human review

### 10.4 Reporting cost

Reporting is designed to be lightweight — approximately 50–100 tokens per report. The default report ("all entries were fine") requires only the `task_id` and the `used` list. Only negatives require additional detail.

---

## 11. Context Profiles

### 11.1 Purpose

A context profile is a named bundle of knowledge that gets loaded into an agent's context at the start of a session. It defines what the agent needs to know for its role.

Context profiles are not job titles — they are scoping mechanisms. The same underlying model receives a different context profile depending on what it is doing.

### 11.2 Profile storage

Context profiles are YAML files stored in `.kbz/context/roles/`, one file per profile. They are human-authored (or agent-drafted for human review), Git-tracked, and validated by the system on read.

Profiles are not entity records — they do not have lifecycle states, TSID13 IDs, or timestamps. They use human-readable names as identifiers.

### 11.3 Required fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | yes | Profile name (lowercase alphanumeric and hyphens, 2–30 characters) |
| `inherits` | string | no | Parent profile name (must resolve, no cycles) |
| `description` | string | yes | Human-readable description of what this role covers |
| `packages` | list of strings | no | Directories or packages this role owns or operates within |
| `conventions` | list of strings | no | Role-specific conventions, patterns, and constraints |
| `architecture` | object | no | Architecture context for this role |
| `architecture.summary` | string | no | Brief architecture overview relevant to this role |
| `architecture.key_interfaces` | list of strings | no | Key interfaces and abstractions this role works with |

### 11.4 Field order for deterministic YAML

```
id, inherits, description, packages, conventions, architecture
```

### 11.5 Example profile

```yaml
id: backend
inherits: developer
description: "Backend development — core logic, MCP service layer, CLI"
packages:
  - internal/core
  - internal/mcp
  - cmd/kanbanzai
conventions:
  - "Error handling: use fmt.Errorf with %w wrapping, never bare errors.New for contextual errors"
  - "Tests: table-driven, no mocks for pure functions"
  - "Logging: structured via slog, never fmt.Println"
architecture:
  summary: "MCP layer is a thin adapter; all business logic lives in internal packages"
  key_interfaces:
    - "storage.EntityStore — canonical read/write interface for entities"
    - "validate.Validator — schema and transition validation"
```

### 11.6 Inheritance and resolution

Per P2-DEC-002, context profiles use **leaf-level replace** semantics.

When the system resolves a profile, it walks the inheritance chain from root ancestor to leaf and layers each profile's fields in order:

1. Start with an empty resolved profile.
2. For each profile in the chain (root ancestor first, leaf last):
   - For each field the profile defines, replace the corresponding field in the resolved profile.
   - Fields the profile does not define are inherited unchanged from the ancestor.

**Merge rules:**

- **Scalar fields** (`description`, `architecture.summary`): child wins.
- **List fields** (`packages`, `conventions`, `architecture.key_interfaces`): child's list replaces the parent's list entirely. No concatenation or deduplication.
- **Map fields** (`architecture`): child's map replaces the parent's map. Individual keys within the map are not merged from parent and child.
- **Absent fields**: parent's value is inherited unchanged.

The `id` and `inherits` fields are never inherited — they belong to the profile definition, not the resolved output.

### 11.7 Profile validation

The system must validate profiles on read:

1. The `inherits` reference must resolve to an existing profile file.
2. The inheritance graph must not contain cycles.
3. The `id` field must match the filename (without `.yaml` extension).
4. All required fields must be present.

Validation errors must be reported clearly, identifying the profile and the specific violation.

### 11.8 Suggested initial profiles

A project should define at minimum a `base` profile (project-wide conventions inherited by all roles). Typical profiles:

| Profile | Inherits | Purpose |
|---------|----------|---------|
| `base` | — | Project-wide conventions, all agents inherit this |
| `developer` | `base` | General development conventions |
| `backend` | `developer` | Backend-specific scope and patterns |
| `frontend` | `developer` | Frontend-specific scope and patterns |
| `testing` | `base` | Test strategy and test implementation |

Projects define their own profile trees. A small project might have only `base`. The system does not enforce a specific hierarchy.

---

## 12. Context Assembly

### 12.1 Assembly model

When an agent needs context for a task, the system assembles a **context packet** — a single coherent bundle drawn from multiple sources in a defined order:

1. **Role profile** — the resolved profile (after inheritance) for the agent's role
2. **Design context** — task-specific design fragments from the document intelligence layer, scoped by entity hierarchy (Plan goals → Feature spec → Task requirements → relevant decisions)
3. **Implementation context** — Tier 2 KnowledgeEntry records matching the role's scope, filtered by confidence ≥ 0.3
4. **Session context** — Tier 3 KnowledgeEntry records matching the role's scope, filtered by confidence ≥ 0.5
5. **Task instructions** — acceptance criteria, constraints, verification requirements from the task entity (when a `task_id` is provided)

### 12.2 Byte-based budgeting

Context packets are assembled with a configurable byte ceiling:

- **Default ceiling:** 30,720 bytes (30 KB), approximately 8–10K tokens across most tokenizers.
- **Configurable:** via the `max_bytes` parameter on `context_assemble`, or via a project-level default in `.kbz/config.yaml`.

The system budgets in bytes, not tokens. The MCP protocol does not expose the calling model's identity, context window size, or tokenizer. Bytes are universal and observable.

### 12.3 Priority and trimming

When assembled content exceeds the byte ceiling, entries are trimmed in reverse priority order:

1. Tier 3 entries with the lowest confidence are trimmed first.
2. Then Tier 2 entries with the lowest confidence.
3. Design context fragments are trimmed last (they are task-specific and highest priority).
4. The role profile is never trimmed.
5. Task instructions are never trimmed.

Context entries include a priority indicator (`high`, `normal`, `low`) in the assembled output, using the MCP `Annotations.priority` field where supported.

### 12.4 Tiered retrieval

Rather than assembling maximum context in one call, the system supports tiered retrieval:

| Operation | Returns | Typical size |
|-----------|---------|-------------|
| `get_entity` (task) | Task requirements, acceptance criteria, constraints | 2–4 KB |
| `context_assemble` | Full context packet: profile + design + knowledge + task | 10–30 KB |
| `knowledge_get` / `profile_get` | Individual entries on demand | < 1 KB each |

Agents request more detail when they need it. This pull-based model avoids over-stuffing context and works across models with different window sizes.

### 12.5 Assembly without a task

When `context_assemble` is called without a `task_id`, the assembly omits design context and task instructions. The result contains:

1. Resolved role profile
2. Tier 2 knowledge entries matching the role's scope
3. Tier 3 knowledge entries matching the role's scope

This is useful for general-purpose agent sessions where no specific task is assigned.

---

## 13. Agent Capabilities

### 13.1 Link resolution

The system must support suggesting entity links from free-text references in documents and entity fields.

**Mechanism:**

1. The caller provides a block of text.
2. The system scans for patterns that resemble entity references: ID patterns (`FEAT-*`, `TASK-*`, `KE-*`, Plan ID patterns), slugs, and title fragments.
3. The system matches candidate references against known entities in the entity store.
4. The system returns a list of suggested links, each with: the matched text span, the candidate entity ID, the entity title, and a match quality indicator (`exact`, `prefix`, `fuzzy`).

This builds on the Phase 2a entity reference extraction (Layer 2 of document intelligence) but makes it interactive — agents can request link suggestions for arbitrary text, not just registered documents.

Link resolution is tool-assisted: the system provides candidates, the agent confirms. The system must not automatically create links.

### 13.2 Duplicate detection

The system must support detecting potential duplicate entities at creation time.

**Mechanism:**

1. Before creating an entity, the caller provides the entity type, title, and summary.
2. The system searches existing entities of the same type for similar titles and summaries using normalised word-set Jaccard similarity.
3. If any existing entity exceeds a similarity threshold of 0.5, the system returns the candidate duplicates with their IDs, titles, and similarity scores.
4. The caller decides whether to proceed with creation or use an existing entity.

Duplicate detection is advisory. The system must not block entity creation based on duplicate detection — it surfaces candidates and the agent (or human) decides.

### 13.3 Document-to-entity extraction guidance

The system must provide structured support that helps agents extract entities from approved documents more reliably.

**Mechanism:**

1. The caller provides a document ID.
2. The system retrieves the document's structural analysis (Layer 1–2 from Phase 2a) and classification (Layer 3, if available).
3. The system returns an extraction guide containing:
   - The document's structural outline with section roles (from classification)
   - For each section classified with an entity-relevant role (e.g., `requirements`, `acceptance-criteria`, `task-breakdown`), a prompt indicating what entity type might be extracted and which fields map to the section's content
   - The list of entity references already found in the document (from Layer 2)

This is guidance, not automation. The agent uses the guide to make extraction decisions. The system does not extract entities automatically.

---

## 14. Batch Document Import

Per P2-DEC-003, Phase 2b includes a batch document import capability for onboarding existing projects.

### 14.1 Import operation

The batch import operation:

1. Accepts a directory path and an optional glob pattern (e.g., `work/**/*.md`).
2. For each matching file, creates a document record in `draft` status — computing the content hash and running Layers 1–2 (structural parsing).
3. Infers document type from file path conventions where possible (see §14.2).
4. Uses the auto-resolved `created_by` value (see §15).
5. Skips files that already have document records (idempotent by path).
6. Returns a summary: files imported, files skipped (with reason), errors encountered.

### 14.2 Document type inference

The system infers document type from directory path using configurable conventions. Default path-to-type mapping:

| Path pattern | Inferred type |
|-------------|---------------|
| `*/design/*` | `design` |
| `*/spec/*` | `specification` |
| `*/plan/*` | `report` |
| `*/research/*` | `research` |

When the path does not match any pattern, the caller must provide a `default_type` parameter. If neither inference nor default produces a type, the file is skipped with an error.

The path-to-type mapping is configurable via `.kbz/config.yaml` under an `import` key. The system provides the defaults above; projects can override or extend them.

### 14.3 Import constraints

- Import does not attempt Layer 3 classification. Classification remains incremental via `doc_pending` + `doc_classify`.
- Import does not assign `owner` (parent Plan or Feature) automatically. Ownership is set manually or via a follow-up operation.
- Import is idempotent — re-running on the same directory skips already-imported files.
- Import operates on the file system as it exists at invocation time. It does not scan git history.

---

## 15. User Identity Resolution

Per P2-DEC-004, the `created_by` field is auto-resolved when the caller does not provide an explicit value.

### 15.1 Resolution order

1. **Explicit value** — if the caller passes `created_by`, that value is used.
2. **Local config** — if `.kbz/local.yaml` exists and contains a `user.name` field, that value is used.
3. **Git config** — the output of `git config user.name` is used.
4. **Error** — if none of the above produces a value, the operation fails with a clear error message. The system must not silently default to `"human"` or any other placeholder.

### 15.2 Local config file

The `.kbz/local.yaml` file holds per-machine settings that must not be shared across collaborators. It is added to `.gitignore`.

Schema:

```yaml
user:
  name: sambeau
```

### 15.3 Scope of auto-resolution

Auto-resolution applies to all MCP tools and CLI commands that accept `created_by`:

- Entity creation tools (`create_plan`, `create_feature`, `create_task`, `create_bug`, `create_decision`, `knowledge_contribute`)
- Document management tools (`doc_record_submit`)
- Batch import (`batch_import_documents`)

The `approved_by` field on document approval should use the same resolution chain when the caller does not provide an explicit value.

### 15.4 Existing records

Existing entity records with `created_by: human` are not automatically updated. They retain their original value. Correction is a manual operation.

---

## 16. Storage and File Requirements

### 16.1 KnowledgeEntry storage

All KnowledgeEntry files must be stored in `.kbz/state/knowledge/`, one YAML file per entry. Filename format: `{id}.yaml` (e.g., `KE-01J3K7MXP3RT5.yaml`).

### 16.2 Context profile storage

Context profiles must be stored in `.kbz/context/roles/`, one YAML file per profile. Filename format: `{id}.yaml` (e.g., `backend.yaml`, `base.yaml`).

### 16.3 Context directory structure

```
.kbz/
├── state/
│   ├── plans/            # existing (Phase 2a)
│   ├── features/         # existing (Phase 1)
│   ├── tasks/            # existing (Phase 1)
│   ├── bugs/             # existing (Phase 1)
│   ├── decisions/        # existing (Phase 1)
│   ├── documents/        # existing (Phase 2a)
│   └── knowledge/        # NEW — KnowledgeEntry records
├── context/
│   └── roles/            # NEW — context profile YAML files
├── index/                # existing (Phase 2a)
├── cache/                # existing (Phase 1)
├── config.yaml           # existing
└── local.yaml            # NEW — per-machine settings (gitignored)
```

### 16.4 Deterministic formatting

KnowledgeEntry YAML files must follow the same deterministic serialisation rules as all other entity files (P1-DEC-008): block style for mappings and sequences, double-quoted strings only when required by YAML syntax, deterministic field order (§6.4), UTF-8, LF line endings, trailing newline, no YAML tags/anchors/aliases.

Context profile YAML files must follow the same formatting rules. Field order per §11.4.

### 16.5 Gitignore additions

The following must be added to `.gitignore`:

- `.kbz/local.yaml`

The `.kbz/context/` directory is Git-tracked (profiles are shared across collaborators). The `.kbz/state/knowledge/` directory is Git-tracked (knowledge entries are shared).

---

## 17. MCP Interface Requirements

### 17.1 Knowledge operations

| Tool | Required parameters | Optional parameters | Description |
|------|---------------------|---------------------|-------------|
| `knowledge_contribute` | `topic`, `content`, `scope` | `tier`, `learned_from`, `created_by`, `tags` | Create a KnowledgeEntry. Performs deduplication check. Returns the new entry or a rejection with pointer to the conflicting entry. Default `tier` is `3`. |
| `knowledge_get` | `id` | — | Get a KnowledgeEntry by ID |
| `knowledge_list` | — | `tier`, `scope`, `status`, `topic`, `tags`, `min_confidence` | List knowledge entries with filtering. Excludes `retired` entries by default. |
| `knowledge_update` | `id`, `content` | — | Update the content of an existing entry. Resets `use_count` and `miss_count` to 0 and `confidence` to 0.5 (content changed, prior usage data no longer applies). |
| `knowledge_confirm` | `id` | — | Transition to `confirmed`. Updates `updated` timestamp. |
| `knowledge_flag` | `id`, `reason` | — | Increment `miss_count`, recompute confidence. If `miss_count ≥ 2`, transition to `retired`. |
| `knowledge_retire` | `id`, `reason` | — | Transition to `retired`. Sets `deprecated_reason`. |
| `knowledge_promote` | `id` | — | Change `tier` from `3` to `2`. Sets `promoted_from` to the entry's own ID (for provenance tracking). Must fail if entry is not Tier 3. |

### 17.2 Context profile operations

| Tool | Required parameters | Optional parameters | Description |
|------|---------------------|---------------------|-------------|
| `profile_get` | `id` | `resolved` (boolean, default: `true`) | Get a context profile. If `resolved` is true, returns the fully resolved profile after walking the inheritance chain. If false, returns the raw profile as defined. |
| `profile_list` | — | — | List available context profiles with their `id`, `inherits`, and `description`. |

Profile creation and editing is a file-system operation (human-authored YAML). The MCP surface for profiles is read-only.

### 17.3 Context assembly operations

| Tool | Required parameters | Optional parameters | Description |
|------|---------------------|---------------------|-------------|
| `context_assemble` | `role` | `task_id`, `max_bytes` (default: 30720) | Assemble a context packet for a role, optionally scoped to a specific task. Returns the assembled packet with source annotations. |
| `context_report` | `task_id`, `used` (list of entry IDs) | `flagged` (list of `{entry_id, reason}`) | Report usage of knowledge entries after task completion. Updates use/miss counts and confidence scores. |

### 17.4 Agent capability operations

| Tool | Required parameters | Optional parameters | Description |
|------|---------------------|---------------------|-------------|
| `suggest_links` | `text` | `scope` (entity type filter) | Suggest entity links from free-text. Returns candidate matches with text span, entity ID, title, and match quality. |
| `check_duplicates` | `entity_type`, `title` | `summary` | Check for potential duplicate entities. Returns candidates with IDs, titles, and similarity scores. |
| `doc_extraction_guide` | `document_id` | — | Return a structured extraction guide for a document, combining structural analysis, classification, and entity reference data. |

### 17.5 Batch import operations

| Tool | Required parameters | Optional parameters | Description |
|------|---------------------|---------------------|-------------|
| `batch_import_documents` | `path` | `glob`, `default_type`, `owner`, `created_by` | Import existing documents from a directory. Returns summary of files imported, skipped, and errors. |

### 17.6 MCP output requirements

All new MCP operations must follow the same output conventions as Phase 1 and Phase 2a:

- Clear success/failure indication
- Useful error information with actionable detail
- Structured machine-readable output (JSON-serialised in `mcp.CallToolResult`)
- Enough detail for an AI agent to interpret the outcome and decide next steps

### 17.7 Strict validation

All new MCP operations must reject invalid input rather than silently repairing it:

- Invalid knowledge entry status transitions
- Duplicate topic/scope on contribution (with pointer to existing entry)
- Unresolvable profile inheritance references
- Cyclic profile inheritance
- Invalid tier values
- Missing required fields

---

## 18. Validation and Health Requirements

### 18.1 Extended health check coverage

The `health` operation must be extended to cover Phase 2b state:

| Check | Description |
|-------|-------------|
| Knowledge entry schema | All KnowledgeEntry files conform to the required field schema |
| Knowledge entry lifecycle | No entries in invalid states or with impossible transitions |
| Knowledge confidence consistency | `confidence` matches the Wilson score computed from `use_count` and `miss_count` |
| Profile inheritance resolution | All `inherits` references resolve and no cycles exist |
| Profile schema | All profile files conform to the required field schema |
| Knowledge scope validity | All `scope` values reference an existing profile `id` or are `project` |

### 18.2 Validation timing

Health checks run on demand via the `health` MCP tool or `kbz health` CLI command. They do not run automatically on every operation.

---

## 19. CLI Requirements

### 19.1 New CLI commands

| Command | Description |
|---------|-------------|
| `kbz import <path> [--glob <pattern>] [--type <default-type>] [--owner <id>]` | Batch document import |
| `kbz knowledge list [--tier <n>] [--scope <name>] [--status <status>]` | List knowledge entries |
| `kbz knowledge get <id>` | Get a knowledge entry |
| `kbz profile list` | List context profiles |
| `kbz profile get <id> [--raw]` | Get a resolved (or raw) context profile |
| `kbz context assemble --role <name> [--task <id>] [--max-bytes <n>]` | Assemble a context packet |

CLI commands mirror the MCP tool surface. They are secondary to MCP tools (per P1-DEC-003) but provide human access to the same operations.

---

## 20. Acceptance Criteria

### 20.1 Knowledge contribution and retrieval

It must be possible to:

- [ ] Contribute a knowledge entry with topic, content, and scope via `knowledge_contribute`
- [ ] Retrieve a knowledge entry by ID via `knowledge_get`
- [ ] List knowledge entries filtered by tier, scope, status, topic, and tags via `knowledge_list`
- [ ] Reject a contribution that duplicates an existing topic in the same scope, returning a pointer to the existing entry
- [ ] Reject a contribution whose content has Jaccard similarity > 0.65 with an existing entry in the same scope
- [ ] Update a knowledge entry's content via `knowledge_update`, resetting usage counts
- [ ] Contribute entries with auto-resolved `created_by` when not explicitly provided

### 20.2 Knowledge lifecycle

It must be possible to:

- [ ] Confirm a contributed entry via `knowledge_confirm`
- [ ] Flag an entry as wrong via `knowledge_flag`, incrementing `miss_count` and recomputing confidence
- [ ] Retire an entry via `knowledge_retire` with a reason
- [ ] Promote a Tier 3 entry to Tier 2 via `knowledge_promote`
- [ ] Observe auto-confirmation when `use_count ≥ 3` and `miss_count = 0` (triggered by `context_report`)
- [ ] Observe auto-retirement when `miss_count ≥ 2` (triggered by `knowledge_flag` or `context_report`)
- [ ] Reject invalid lifecycle transitions

### 20.3 Confidence scoring

It must be possible to:

- [ ] Observe confidence of 0.5 on a newly created entry
- [ ] Observe confidence increasing after successful usage reports
- [ ] Observe confidence decreasing after negative usage reports
- [ ] Verify confidence matches the Wilson score lower bound formula for given `use_count` and `miss_count`

### 20.4 Usage reporting

It must be possible to:

- [ ] Submit a usage report via `context_report` with `task_id`, `used` list, and optional `flagged` list
- [ ] Verify that `use_count` increments for entries in the `used` list
- [ ] Verify that `miss_count` increments for entries in the `flagged` list
- [ ] Verify that `last_used` timestamps are updated for used entries
- [ ] Verify that `confidence` is recomputed after each report

### 20.5 Context profiles

It must be possible to:

- [ ] Define a context profile as a YAML file in `.kbz/context/roles/`
- [ ] Retrieve a resolved profile via `profile_get` with inheritance applied
- [ ] Retrieve a raw (unresolved) profile via `profile_get` with `resolved: false`
- [ ] List available profiles via `profile_list`
- [ ] Verify leaf-level replace semantics: a child profile's list field replaces (not concatenates) the parent's
- [ ] Detect and report cyclic inheritance
- [ ] Detect and report unresolvable `inherits` references

### 20.6 Context assembly

It must be possible to:

- [ ] Assemble a context packet for a role via `context_assemble`
- [ ] Assemble a context packet for a role + task via `context_assemble` with `task_id`
- [ ] Verify that the assembled packet includes the resolved role profile
- [ ] Verify that the assembled packet includes Tier 2 knowledge entries matching the role's scope with confidence ≥ 0.3
- [ ] Verify that the assembled packet includes Tier 3 knowledge entries matching the role's scope with confidence ≥ 0.5
- [ ] Verify that the assembled packet respects the byte ceiling (entries trimmed when over budget)
- [ ] Verify that low-confidence Tier 3 entries are trimmed before Tier 2 entries

### 20.7 Agent capabilities

It must be possible to:

- [ ] Suggest entity links from free-text via `suggest_links`
- [ ] Detect potential duplicate entities at creation time via `check_duplicates`
- [ ] Retrieve a document extraction guide via `doc_extraction_guide`

### 20.8 Batch document import

It must be possible to:

- [ ] Import documents from a directory via `batch_import_documents`
- [ ] Verify that imported documents appear as `draft` document records
- [ ] Verify that Layers 1–2 analysis runs for each imported document
- [ ] Verify that document type is inferred from path conventions
- [ ] Verify that already-imported files are skipped on re-run (idempotent)
- [ ] Verify that errors for individual files do not abort the entire import

### 20.9 User identity resolution

It must be possible to:

- [ ] Create an entity without providing `created_by` and verify it is auto-resolved from `git config user.name`
- [ ] Create an entity without providing `created_by` with a `.kbz/local.yaml` present and verify it uses the local config value
- [ ] Create an entity with an explicit `created_by` and verify the explicit value takes precedence
- [ ] Verify that the operation fails with a clear error when no identity source is available

### 20.10 Deterministic storage

- [ ] KnowledgeEntry YAML files produce deterministic output (write → read → write → compare)
- [ ] Context profile YAML files produce deterministic output
- [ ] Field order matches the canonical order defined in §6.4 and §11.4

### 20.11 Health checks

- [ ] `health` detects invalid KnowledgeEntry schema
- [ ] `health` detects confidence/count inconsistencies
- [ ] `health` detects broken profile inheritance

---

## 21. Open Questions for Planning

These are implementation-level questions to be resolved during planning or early implementation. They do not affect the specification's requirements.

1. **Cache schema for knowledge entries** — should KnowledgeEntry records be indexed in the SQLite cache for query performance, following the pattern established (but not yet implemented) for documents in Phase 2a? If so, what columns and indexes are needed?

2. **Context assembly performance** — at what knowledge store size does linear scanning of YAML files become a bottleneck for `context_assemble`? Should the cache be used for assembly queries from the start, or only when performance demands it?

3. **Stop word list for Jaccard deduplication** — what stop word list should be used for the normalised word-set comparison? A minimal English stop word list, or a domain-aware list?

4. **Import path configuration** — should the document type inference mapping in `.kbz/config.yaml` use glob patterns or prefix matching? Should it be part of the existing `import` config section or a new top-level key?

5. **Profile creation tooling** — should Phase 2b include a `kbz profile init` command that scaffolds a basic profile tree (`base` + one role), or is file creation sufficient?

---

## 22. Summary

Phase 2b extends the Kanbanzai system with context management, knowledge persistence, and agent capabilities.

**Knowledge entries** give agents a way to contribute and retrieve project knowledge, with a confidence feedback loop that makes the knowledge store self-correcting. The Wilson score ensures that proven knowledge rises and wrong knowledge is retired.

**Context profiles** scope what each agent knows, using a simple inheritance model with predictable override semantics. Projects define their own role hierarchies, from a single `base` profile for small projects to deep specialisation trees for large ones.

**Context assembly** composes design context (from document intelligence), implementation context (from knowledge entries), and role conventions (from profiles) into targeted packets. Byte-based budgeting and tiered retrieval ensure the system works across models with different context window sizes.

**Agent capabilities** — link resolution, duplicate detection, and extraction guidance — leverage the document intelligence and entity layers built in Phase 2a to make agents more effective at structured work.

**Batch import** and **user identity resolution** address practical adoption concerns: existing projects can onboard their documents efficiently, and every record carries meaningful attribution.

The design-to-delivery pipeline — design → specify → plan → implement → verify — remains the structural backbone. Phase 2b adds the context layer that makes each step in the pipeline more efficient than the last, because agents accumulate and share knowledge rather than starting fresh every session.