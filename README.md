# Kanbanzai
Git-native workflow system for human-AI collaborative software development. Coordinates AI agent teams to turn designs into working software, with a document-centric human interface and structured internal state.

## Two workflows

This project has two distinct workflows. Do not confuse them.

- **kbz-workflow** — the workflow process the Kanbanzai tool will implement and enforce. Described in `work/design/` and `work/spec/`. This is what we are *building*.
- **bootstrap-workflow** — the simplified process we use right now to build Kanbanzai. Described in `work/bootstrap/bootstrap-workflow.md`. This is what we *follow*.

## Document map

The repository is organised into two major document areas:

- `work/`
  - active design, specification, planning, research, and bootstrap process documents
- `docs/`
  - user-facing and reference documentation for the reusable workflow system (reserved for later)

## Current planning basis

The main documents to use for planning are:

1. `work/bootstrap/bootstrap-workflow.md`
   - the process we follow right now (bootstrap-workflow)
2. `work/design/workflow-design-basis.md`
   - consolidated design vision (kbz-workflow)
3. `work/spec/phase-1-specification.md`
   - phase 1 scope and verification basis (kbz-workflow)
4. `work/design/agent-interaction-protocol.md`
   - agent behavior and normalization protocol
5. `work/design/quality-gates-and-review-policy.md`
   - review expectations and quality gates
6. `work/design/git-commit-policy.md`
   - required policy for creating and describing commits

## Recommended reading order

If you are new to the project, read in this order:

1. `work/bootstrap/bootstrap-workflow.md`
2. `work/design/workflow-design-basis.md`
3. `work/design/document-centric-interface.md`
4. `work/spec/phase-1-specification.md`
5. `work/design/agent-interaction-protocol.md`
6. `work/design/quality-gates-and-review-policy.md`
7. `work/design/git-commit-policy.md`

Then refer to these supporting documents as needed:

- `work/design/workflow-system-design.md`
- `work/design/product-instance-boundary.md`
- `work/research/initial-workflow-analysis.md`
- `work/research/initial-workflow-analysis-review.md`
- `work/research/workflow-system-design-review.md`
- `work/plan/phase-1-implementation-plan.md`
- `work/plan/phase-1-decision-log.md`

## Directory guide

### `work/bootstrap/`
Bootstrap-workflow process documents. Defines the simplified workflow we follow right now to build Kanbanzai, before the tool exists.

### `work/design/`
Kbz-workflow design documents and policy documents. Describes how the workflow system should work, including the document-centric human interface model, agent behavior, review expectations, Git commit policy, and product/instance boundaries.

### `work/spec/`
Kbz-workflow formal specifications used to constrain and verify implementation work.

### `work/plan/`
Implementation planning documents that translate approved design and specification documents into concrete execution plans and decisions.

### `work/research/`
Background analysis, comparative thinking, and review memos retained as the research trail and decision history.

### `docs/`
User-facing and reference documentation for the reusable workflow system, such as manuals, setup guides, reference docs, and future product documentation. Currently empty; reserved for later.

## Implementation status

Phase 1 implementation has started.

Current status:
- design, specification, and planning documents are in place
- initial Go module scaffolding has begun
- the implementation work is being started from the Phase 1 plan in `work/plan/phase-1-implementation-plan.md`

Near-term implementation focus:
- core entity model for `Epic`, `Feature`, `Task`, `Bug`, and `Decision`
- deterministic YAML-backed storage under `.kbz/`
- ID allocation and lifecycle validation
- MCP-first workflow operations with a thin CLI layer
- health checks and bootstrap self-use support
