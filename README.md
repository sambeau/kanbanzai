# Kanbanzai

Kanbanzai is a Git-native workflow system for human-AI collaborative software development.

The idea is simple:

- humans work through documents, reviews, and decisions
- AI agents turn that intent into structured work
- the project state is stored in plain files in Git
- the workflow stays visible, reviewable, and recoverable

Kanbanzai is still under active development. This repository currently contains both the implementation work and the design/specification documents that define what the finished system should do.

---

## 1. What Kanbanzai is, and why it might be useful

Software projects often end up with work scattered across chats, tickets, scratch notes, pull requests, and half-finished plans. That gets even messier when AI agents are involved. They may be helpful, but it can become hard to see:

- what was decided
- what work exists
- what depends on what
- what is approved vs still in draft
- what an agent changed, and why

Kanbanzai is meant to solve that problem.

When it is finished, Kanbanzai will provide a structured way to manage software work where:

- **humans own intent**
  - goals, priorities, designs, approvals, tradeoffs
- **AI agents own execution**
  - breaking work down, implementing changes, validating results, updating workflow state
- **Git stores the source of truth**
  - plans, entities, decisions, and state are all tracked as files
- **documents remain the human interface**
  - people do not need to manually maintain internal records one by one

### Why that can be useful

If Kanbanzai works as intended, it should help teams:

- keep AI-assisted development auditable
- make decisions and work status easier to review
- avoid losing context between conversations
- keep structured state in sync with real implementation work
- scale from a small project to more coordinated multi-agent work without needing a heavy external system

It is not trying to replace Git, design docs, or human review. It is trying to connect them into a workflow that is easier to trust.

---

## 2. Quick manual: how to try it now, and how it is intended to work later

This section is written for someone who wants the practical picture first.

### What you can try today

The current implementation includes a Phase 1 workflow kernel and Phase 2a entity model evolution, document management, and prefix registry. It supports a useful subset of the planned system:

- creating and storing workflow entities (including the new Plan entity)
- validating lifecycle state and references
- storing canonical YAML under `.kbz/`
- managing documents through tracked lifecycle with content hash verification
- configuring Plan ID prefixes through a prefix registry
- exposing operations through both:
  - a CLI (Phase 1 entities)
  - an MCP server for AI-agent use (Phase 1 and Phase 2a entities)

The entity types currently implemented are:

- `Plan` (Phase 2a — replaces Epic, uses `{prefix}{number}-{slug}` IDs)
- `Epic` (Phase 1, deprecated — retained for migration compatibility)
- `Feature` (updated in Phase 2a with document ownership and document-driven lifecycle states)
- `Task`
- `Bug`
- `Decision`

Phase 1 document support includes:

- scaffolding
- submission
- approval
- retrieval
- validation
- listing
- extraction support for approved documents

Phase 2a adds document record management — tracked metadata records for documents that remain at their canonical paths:

- submit (register with content hash)
- approve (with hash verification and approver tracking)
- supersede (with bidirectional linking)
- content drift detection
- filtering by type, status, and owner

### Basic CLI usage

Build or run the CLI with Go:

```/dev/null/README.md#L1-20
go run ./cmd/kanbanzai --help
go run ./cmd/kanbanzai version
```

Create a few example entities (CLI currently supports Phase 1 entity types):

```/dev/null/README.md#L1-20
go run ./cmd/kanbanzai create feature \
  --slug audit-2-remediation \
  --summary "Complete audit remediation tracks" \
  --created_by sam
```

Read and inspect state:

```/dev/null/README.md#L1-20
go run ./cmd/kanbanzai list features
go run ./cmd/kanbanzai health
```

Work with documents:

```/dev/null/README.md#L1-20
go run ./cmd/kanbanzai doc scaffold --type proposal --title "Example Proposal"

go run ./cmd/kanbanzai doc submit \
  --type proposal \
  --title "Example Proposal" \
  --created_by sam \
  --body "# Example Proposal

## Summary

Short summary.

## Problem

What needs to change.

## Proposal

What to do."
```

Rebuild the local derived cache:

```/dev/null/README.md#L1-20
go run ./cmd/kanbanzai cache rebuild
```

Start the MCP server (exposes both Phase 1 and Phase 2a tools):

```/dev/null/README.md#L1-20
go run ./cmd/kanbanzai serve
```

Phase 2a entity operations (Plans, document records, prefix registry) are currently available through MCP tools only. CLI support for Phase 2a entities has not yet been added.

### What files it creates

Kanbanzai stores project-local instance state in `.kbz/`.

In the current implementation, that includes things like:

- `.kbz/config.yaml`
  - project configuration including the prefix registry for Plan IDs
- `.kbz/state/`
  - canonical entity records (epics, features, tasks, bugs, decisions)
- `.kbz/state/plans/`
  - Plan entity records (Phase 2a)
- `.kbz/state/documents/`
  - document metadata records (Phase 2a)
- `.kbz/docs/`
  - managed documents (Phase 1 document store)
- `.kbz/cache/`
  - derived local cache data

The goal is that the important state is plain, inspectable, Git-friendly data rather than hidden in a separate service.

### How it is intended to work when finished

In the finished system, the normal experience should look more like this:

1. A person writes or reviews a document
   - proposal
   - design
   - specification
   - implementation plan
   - user documentation

2. The document is reviewed and approved

3. An AI agent uses that approved document to create or update structured workflow state
   - plans
   - features
   - tasks
   - decisions
   - links between them

4. Document approvals automatically drive entity lifecycle transitions
   - approving a specification advances its Feature to `dev-planning`
   - approving a dev plan advances its Feature to `developing`

5. Agents implement and verify work while keeping workflow state consistent

6. Humans review the resulting code, decisions, and progress

So the human-facing workflow stays mostly document- and review-based, while the structured internals keep the project machine-readable and safer to automate.

### Important current limitations

This project is not fully finished yet.

Some important caveats:

- the current implementation covers Phase 1 and Phase 2a, not the full long-term vision
- broader multi-agent orchestration is not the focus yet
- the repository still contains design and planning material alongside implementation
- CLI support for Phase 2a entity types has not yet been added

If you are trying it today, treat it as an evolving workflow kernel rather than a polished end-user product. See `work/plan/phase-2a-progress.md` for detailed status.

---

## 3. Developer details: progress, architecture, and internal behavior

This section is for contributors and technically curious readers.

### Current project status

The repository has moved beyond planning-only work. The Phase 1 implementation kernel is complete and functioning. Phase 2a implementation is complete — all acceptance criteria are met, all audit bugs are fixed, and all tests pass with race detector enabled.

Broadly, the project now includes:

- implementation entrypoint in `cmd/kanbanzai/`
- core internal packages in `internal/`
- design, spec, planning, and research documents in `work/`

Phase 1 implementation covers:

- canonical entity storage
- deterministic YAML serialization
- entity ID allocation
- lifecycle validation
- health checks
- document lifecycle support (Phase 1 document store)
- MCP tool surface
- CLI support
- local derived cache support
- document extraction support for approved documents
- CLI parity for core document operations
- slug validation, ID-format validation

Phase 2a implementation adds:

- Plan entity type replacing Epic, with prefix-based IDs (P1-basic-ui format)
- prefix registry in `.kbz/config.yaml` with validation and retirement support
- document metadata records with SHA-256 content hash tracking and drift detection
- Feature model updates (parent, design, spec, dev_plan, tags fields) at all layers
- Phase 2 Feature lifecycle states (proposed→designing→specifying→dev-planning→developing→done)
- document-driven Feature lifecycle transitions (document approval/supersession auto-transitions Features)
- document intelligence Layers 1–4 (structural parsing, pattern extraction, AI classification, document graph)
- optimistic locking for concurrent writes with end-to-end FileHash propagation
- Epic→Plan migration command (idempotent, with feature field renames)
- rich queries: date range, cross-entity, tag listing across types, `list_entities_filtered`
- Plan and document record MCP tools
- configuration MCP tools for prefix registry management
- deterministic YAML serialization for all entity types and index files
- comprehensive lifecycle validation with backward transitions
- tags on all entity types with filtering support
- extended health checks for documents, plan prefixes, and feature parent references

All tests pass with race detector enabled.

### Repository structure

At a high level:

```/dev/null/README.md#L1-40
kanbanzai/
├── AGENTS.md
├── README.md
├── cmd/kanbanzai/
├── internal/
├── docs/
└── work/
```

Key directories:

- `cmd/kanbanzai/`
  - CLI and MCP server entrypoint
- `internal/service/`
  - entity and document record service logic
- `internal/document/`
  - Phase 1 document store, lifecycle logic, templates, validation
- `internal/docint/`
  - document intelligence: structural parsing, pattern extraction, classification, document graph
- `internal/storage/`
  - canonical YAML entity storage and document record storage
- `internal/config/`
  - project configuration and prefix registry
- `internal/core/`
  - instance paths and root utilities
- `internal/validate/`
  - entity and health validation, lifecycle state machines
- `internal/mcp/`
  - MCP server and tools (Phase 1 and Phase 2a)
- `internal/model/`
  - entity type definitions and ID utilities
- `internal/cache/`
  - local derived cache
- `internal/id/`
  - canonical ID allocation and validation
- `internal/fsutil/`
  - filesystem utilities (atomic write)
- `internal/testutil/`
  - shared test helpers
- `work/`
  - design, spec, planning, bootstrap, and research documents

### Workflow model

The project distinguishes between two workflows:

- **bootstrap-workflow**
  - the lightweight process used to build Kanbanzai right now
- **kbz-workflow**
  - the workflow Kanbanzai itself is intended to implement and enforce

That distinction matters. Much of the repository describes the target workflow, while the code implements the early kernel needed to support it.

### Phase 1 scope

Phase 1 is intentionally limited. It is the workflow kernel, not the whole future system.

Its focus is on:

- local canonical state
- deterministic persistence
- lifecycle correctness
- document support
- MCP and CLI surfaces
- validation and repair/debugging support

It explicitly avoids broader future features such as:

- orchestration-heavy multi-agent coordination
- semantic retrieval / embeddings
- broad GitHub automation
- knowledge graph style context packing as a required runtime dependency
- full worktree automation

### Entity model

The current entity set is:

- `Plan` (Phase 2a — coordinates work, organises Features)
- `Epic` (Phase 1, deprecated — retained for migration)
- `Feature` (updated in Phase 2a with document references and tags)
- `Task`
- `Bug`
- `Decision`
- `DocumentRecord` (Phase 2a — metadata for tracked documents)

These are stored as YAML files under `.kbz/state/` using deterministic ordering rules.

Examples of current ID families:

- plans: `P1-basic-ui`, `X2-infrastructure` (prefix + number + slug)
- epics: `E-001` (deprecated)
- features: `FEAT-001`
- bugs: `BUG-001`
- decisions: `DEC-001`
- tasks: `FEAT-001.1`
- document records: `FEAT-123/design-my-doc`, `PROJECT/policy-security`

Tasks are feature-local IDs rather than global IDs. Plan IDs use a human-assigned prefix from the project's prefix registry.

### Deterministic YAML

A core design constraint is deterministic canonical serialization.

The implementation is expected to preserve stable output so that:

- repeated writes do not churn Git diffs
- round-trip write/read/write behavior is stable
- records remain human-reviewable

The canonical YAML rules include:

- block style mappings/sequences
- deterministic field order by entity type
- LF line endings
- trailing newline
- no anchors, aliases, or tags
- no multi-document streams

This is important enough that the implementation does not simply rely on a default YAML marshaller for canonical output.

### Document lifecycle

There are currently two document subsystems:

**Phase 1 document store** (`internal/document/`) — documents move through a linear lifecycle:

- `draft` → `submitted` → `normalised` → `approved`

This subsystem supports scaffold generation, submission, body update, approval, retrieval, validation, listing, and extraction for approved documents. It stores document content directly in `.kbz/docs/`.

**Phase 2a document records** (`internal/service/documents.go`, `internal/storage/document_store.go`) — metadata records for documents that remain at their canonical paths (e.g., `work/design/foo.md`):

- `draft` → `approved` → `superseded`

This subsystem tracks content hashes (SHA-256) for integrity verification, detects content drift when files are modified outside the system, and supports supersession chains for document versioning. Records are stored in `.kbz/state/documents/`.

The Phase 2a document record system is intended to eventually replace the Phase 1 document store for document lifecycle management, while the Phase 1 store may be retained for scaffolding and template generation.

### MCP and CLI

Kanbanzai is MCP-first, but the CLI is no longer just a placeholder.

Current CLI support includes (Phase 1 entities):

- entity creation, retrieval, listing, status updates, field updates
- document scaffold / submit / approve / retrieve / validate / list
- candidate validation
- health check
- cache rebuild
- MCP server startup

The MCP layer exposes Phase 1 tool operations plus Phase 2a operations:

- Plan tools: `create_plan`, `get_plan`, `list_plans`, `update_plan_status`, `update_plan`
- Document record tools: `doc_record_submit`, `doc_record_approve`, `doc_record_supersede`
- Config tools: `get_project_config`, `get_prefix_registry`, `add_prefix`
- Document intelligence tools: `doc_outline`, `doc_concepts`, `doc_classify`
- Query tools: `list_tags`, `list_by_tag`, `list_entities_filtered`, `query_plan_tasks`, `doc_supersession_chain`
- Migration tools: `migrate_phase2`

CLI support for Phase 2a entity types has not yet been added.

### Validation behavior

The validation layer currently checks things such as:

- required fields
- known lifecycle states
- slug format
- entity ID format
- bug enum values
- cross-reference integrity
- supersession consistency warnings
- document feature-reference validity

Health checks operate over stored canonical state and surface both errors and warnings.

### Bootstrap self-use

Bootstrap self-use has now been exercised locally against `.kbz/state/` to verify that the kernel can manage limited real project state without corruption.

That verification included:

- creating initial instance records
- rebuilding cache
- running health checks
- reading back stored records
- validating legal lifecycle transitions
- validating rejection of illegal transitions

At the moment, this bootstrap state is being treated as local proof rather than final committed project state.

### Recommended reading for contributors

If you need to understand the project deeply, start here:

1. `work/bootstrap/bootstrap-workflow.md`
2. `work/design/workflow-design-basis.md`
3. `work/design/document-centric-interface.md`
4. `work/spec/phase-1-specification.md`
5. `work/spec/phase-2-specification.md`
6. `work/design/agent-interaction-protocol.md`
7. `work/design/quality-gates-and-review-policy.md`
8. `work/design/git-commit-policy.md`

Then use as needed:

- `work/plan/phase-2a-progress.md` — current implementation status
- `work/plan/phase-2-scope.md` — Phase 2 scope and planning
- `work/plan/phase-1-implementation-plan.md`
- `work/plan/phase-1-decision-log.md`
- `work/design/document-intelligence-design.md`
- `work/design/machine-context-design.md`

### Build and test

Typical commands:

```/dev/null/README.md#L1-20
go build ./...
go test ./...
go test -race ./...
go vet ./...
go fmt ./...
```

### In short

Kanbanzai is trying to become a practical workflow kernel for human-AI software delivery:

- document-centered for humans
- structured and enforceable for machines
- Git-native for visibility and control

The codebase now demonstrates the core entity model, document management with integrity tracking, document intelligence, a prefix-based Plan system, and end-to-end lifecycle integration. Phase 2a is complete. See `work/plan/phase-2a-progress.md` for detailed status.