# Bootstrap Self-Hosting Specification

- Status: specification draft
- Purpose: define concrete requirements and acceptance criteria for bootstrap self-hosting of the Kanbanzai workflow kernel
- Date: 2026-07-25
- Based on:
  - `work/spec/phase-1-specification.md` §22, §24.10
  - P1-DEC-005 (Phase 1 must support limited bootstrap self-use)
  - P1-DEC-015 (Bootstrap introduction strategy)

## §1 Purpose

This specification makes the Phase 1 bootstrap requirement (Phase 1 spec §22) concrete. It defines what "usable to track limited development of the tool itself" means in testable terms.

The Phase 1 spec states that the kernel "must be usable to track limited development of the workflow tool itself" (§22) and that it "must be sufficient to begin tracking the workflow tool's own work in a limited way without requiring Phase 2+ systems" (§24.10). Neither statement provides concrete criteria. This document supplies them.

## §2 Scope

This specification covers:

- Initial records required for bootstrap activation
- Document registration of existing project documents
- Tool health on its own state
- Coexistence with the bootstrap workflow during transition
- Retirement criteria for the bootstrap workflow

This specification does not cover:

- Full project management automation
- Multi-project or multi-instance support
- Agent orchestration or delegation
- Backfill of historical tasks or bugs
- Synchronisation between bootstrap workflow documents and tool state

## §3 Prerequisites

Bootstrap activation requires all of the following:

1. All Phase 1 entity CRUD operations are functional — create, get, update, and list for all five entity types (Epic, Feature, Task, Bug, Decision) and Documents.
2. Lifecycle enforcement is operational — invalid state transitions are rejected.
3. Health check and validation pass on an empty `.kbz/` instance.
4. The ID allocator is operational. Bootstrap does not depend on which allocator is active (sequential or TSID13); either is acceptable.

Do not attempt bootstrap activation until all four prerequisites are met.

## §4 Initial Records

The following records must be created through the tool's own interfaces (MCP or CLI), not by writing YAML files directly.

### §4.1 Epic

One Epic representing Phase 1 work. The Epic must have at minimum: `id`, `slug`, `summary`, and `status` set to `active`.

### §4.2 Features

At least one Feature for each remaining area of Phase 1 work, linked to the Epic. Each Feature must reference the Epic via the `epic` field. Features should link to relevant design or spec documents where applicable.

### §4.3 Documents

All existing documents in the following directories must be registered through the tool's document operations:

- `work/design/`
- `work/spec/`
- `work/plan/`
- `work/research/`
- `work/bootstrap/`

Each registered document must have: `id`, `slug`, `path`, `doc_type`, and `status`. The tool must be able to retrieve each registered document by ID.

### §4.4 Tasks, Bugs, Decisions

No backfill of historical tasks, bugs, or decisions is required. New tasks, bugs, and decisions created after bootstrap activation must be recorded through the tool.

## §5 State Integrity

After initial records are created:

1. `kbz health` (or the MCP health check operation) must pass with no errors on the `.kbz/` directory.
2. `kbz validate` must report no validation errors.
3. All cross-references must be resolvable — Feature→Epic links, Task→Feature links, and Document registrations must all resolve to existing records.

## §6 Committed State

The `.kbz/` directory must be committed to the Git repository. It is version-controlled alongside code and documents. The `.kbz/` directory must not appear in `.gitignore`.

## §7 Coexistence with Bootstrap Workflow

During the transition period:

1. Both the bootstrap workflow (`work/bootstrap/bootstrap-workflow.md`) and the tool operate simultaneously.
2. There is no requirement for the two systems to be synchronised — they are parallel records during transition.
3. The bootstrap workflow remains the fallback if the tool encounters issues.

Neither system is subordinate to the other during transition. If they disagree, the bootstrap workflow is authoritative until retirement.

## §8 Retirement Criteria

The bootstrap workflow may be retired when ALL of the following are true:

1. The tool has been used to track at least 3 complete feature cycles (created → in-progress → done or verified).
2. At least 5 tasks have been created, worked, and completed through the tool.
3. At least 1 bug has been recorded and resolved through the tool.
4. At least 1 decision has been recorded through the tool.
5. The health check has passed consistently for at least 2 weeks of active development.
6. The human has explicitly approved retirement of the bootstrap workflow.

Criterion 6 is a hard gate — the other criteria are necessary but not sufficient. Even if criteria 1–5 are met, the bootstrap workflow is not retired until the human says so.

## §9 Acceptance Criteria

Bootstrap self-hosting is acceptable only if all of the following are true:

**§9.1** — The `.kbz/state/` directory exists and contains at least one Epic with status `active`.

**§9.2** — At least one Feature exists, linked to the Epic, with status `open` or `in-progress`.

**§9.3** — All documents in `work/design/`, `work/spec/`, `work/plan/`, `work/research/`, and `work/bootstrap/` are registered in the tool and retrievable by ID.

**§9.4** — `kbz health` passes with no errors on the project's own `.kbz/` state.

**§9.5** — `kbz validate` reports no validation errors on the project's own state.

**§9.6** — The `.kbz/` directory is committed to Git — not ignored, not untracked.

**§9.7** — It is possible to create a new Task, Bug, or Decision through the tool and have it correctly written to `.kbz/state/` with proper cross-references.

**§9.8** — The bootstrap workflow retirement criteria (§8) are documented and trackable.

## §10 Out-of-Scope Clarifications

The following are explicitly not required for bootstrap self-hosting:

- Full automation of the development process.
- Migration of historical tasks or bugs from before bootstrap activation.
- Synchronisation between bootstrap workflow documents and tool state.
- Multi-project or multi-instance support.
- Agent orchestration or delegation through the tool.