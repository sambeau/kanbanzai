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

The current implementation is a Phase 1 workflow kernel. It already supports a useful subset of the planned system:

- creating and storing workflow entities
- validating lifecycle state and references
- storing canonical YAML under `.kbz/`
- managing documents through a simple lifecycle
- exposing operations through both:
  - a CLI
  - an MCP server for AI-agent use

The main entity types currently implemented are:

- `Epic`
- `Feature`
- `Task`
- `Bug`
- `Decision`

The current document support includes:

- scaffolding
- submission
- approval
- retrieval
- validation
- listing
- extraction support for approved documents

### Basic CLI usage

Build or run the CLI with Go:

```/dev/null/README.md#L1-20
go run ./cmd/kanbanzai --help
go run ./cmd/kanbanzai version
```

Create a few example entities:

```/dev/null/README.md#L1-20
go run ./cmd/kanbanzai create epic \
  --slug phase-1-completion \
  --title "Phase 1 Completion" \
  --summary "Track remaining Phase 1 work" \
  --created_by sam

go run ./cmd/kanbanzai create feature \
  --slug audit-2-remediation \
  --epic E-001 \
  --summary "Complete audit remediation tracks" \
  --created_by sam
```

Read and inspect state:

```/dev/null/README.md#L1-20
go run ./cmd/kanbanzai get epic --id E-001 --slug phase-1-completion
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

Start the MCP server:

```/dev/null/README.md#L1-20
go run ./cmd/kanbanzai serve
```

### What files it creates

Kanbanzai stores project-local instance state in `.kbz/`.

In the current implementation, that includes things like:

- `.kbz/state/`
  - canonical entity records
- `.kbz/docs/`
  - managed documents
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
   - epics
   - features
   - tasks
   - decisions
   - links between them

4. Agents implement and verify work while keeping workflow state consistent

5. Humans review the resulting code, decisions, and progress

So the human-facing workflow stays mostly document- and review-based, while the structured internals keep the project machine-readable and safer to automate.

### Important current limitations

This project is not fully finished yet.

Some important caveats:

- the current implementation is Phase 1, not the full long-term vision
- broader multi-agent orchestration is not the focus yet
- the repository still contains design and planning material alongside implementation
- bootstrap self-use has been exercised locally, but project-local instance state is not yet being treated as final committed product state

If you are trying it today, treat it as an evolving workflow kernel rather than a polished end-user product.

---

## 3. Developer details: progress, architecture, and internal behavior

This section is for contributors and technically curious readers.

### Current project status

The repository has moved beyond planning-only work. The Phase 1 implementation kernel exists and is functioning.

Broadly, the project now includes:

- implementation entrypoint in `cmd/kanbanzai/`
- core internal packages in `internal/`
- design, spec, planning, and research documents in `work/`

The implementation currently covers:

- canonical entity storage
- deterministic YAML serialization
- entity ID allocation
- lifecycle validation
- health checks
- document lifecycle support
- MCP tool surface
- CLI support
- local derived cache support

Recent progress has also added:

- document extraction support for approved documents
- CLI parity for core document operations
- CLI health and candidate validation commands
- slug validation
- ID-format validation
- document feature-reference validation
- local bootstrap self-use verification

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
  - entity application/service logic
- `internal/document/`
  - document store, lifecycle logic, templates, validation
- `internal/storage/`
  - canonical YAML entity storage
- `internal/validate/`
  - entity and health validation
- `internal/mcp/`
  - MCP server and tools
- `internal/cache/`
  - local derived cache
- `internal/id/`
  - canonical ID allocation and validation
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

- `Epic`
- `Feature`
- `Task`
- `Bug`
- `Decision`

These are stored as YAML files under `.kbz/state/` using deterministic ordering rules.

Examples of current ID families:

- epics: `E-001`
- features: `FEAT-001`
- bugs: `BUG-001`
- decisions: `DEC-001`
- tasks: `FEAT-001.1`

Tasks are feature-local IDs rather than global IDs.

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

Documents currently move through a linear lifecycle:

- `draft`
- `submitted`
- `normalised`
- `approved`

The document subsystem currently supports:

- scaffold generation
- submission
- body update / normalisation step
- approval
- retrieval
- validation
- listing
- extraction surface for approved documents

The extraction support is intentionally minimal in Phase 1: it exposes approved document content in a structured way so an agent can create entities through existing tools. It does not try to fully automate semantic extraction.

### MCP and CLI

Kanbanzai is MCP-first, but the CLI is no longer just a placeholder.

Current CLI support includes:

- entity creation
- entity retrieval and listing
- entity status updates
- field updates
- document scaffold / submit / approve / retrieve / validate / list
- candidate validation
- health check
- cache rebuild
- MCP server startup

The MCP layer exposes corresponding tool operations for agent use.

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
5. `work/design/agent-interaction-protocol.md`
6. `work/design/quality-gates-and-review-policy.md`
7. `work/design/git-commit-policy.md`

Then use as needed:

- `work/plan/phase-1-implementation-plan.md`
- `work/plan/phase-1-decision-log.md`
- `work/plan/phase-1-audit-remediation.md`
- `work/plan/phase-1-audit-2-remediation.md`

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

The codebase is now far enough along to demonstrate the core model, but the larger vision is still being finished.