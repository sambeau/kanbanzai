| Field  | Value                                   |
|--------|-----------------------------------------|
| Date   | 2026-04-30                              |
| Status | approved |
| Author | architect                               |

## Overview

This dev-plan covers the rendering layer of the `kbz status` command: converting structured
status data (synthesised by the service layer) into human-readable prose output with TTY-aware
formatting. The four tasks below deliver injectable TTY detection, five output views, column
alignment, and integration into the status command handler.

## Scope

This plan implements the human-readable output renderer for `kbz status` defined in
`work/spec/B36-F3-spec-status-human-output.md` (doc ID: `FEAT-01KQ2VGTRZHPC/spec-b36-f3-spec-status-human-output`)
and the `human` output format described in `work/_project/design-kbz-cli-and-binary-rename.md` §5.4–§5.6.

It covers the rendering layer: TTY detection, Unicode/ANSI colour output (TTY) vs ASCII fallbacks
(piped), and all five human-readable views (unregistered document, registered document with owner,
feature, plan, project overview). It also covers edge-case handling: I/O errors exit non-zero,
non-existent entities produce an informational message and exit 0.

This plan does **not** cover:
- The `--format plain` or `--format json` output modes (F4)
- The argument resolution and routing logic that maps a CLI target to a data payload (F2)
- The binary rename (`kanbanzai` → `kbz`) (F1)

**Dependency on F2:** The renderer consumes the structured data assembled by F2's resolution
logic. F2 must provide an interface (a Go struct or set of structs) representing the resolved
target and its associated data (entity info, document records, task counts, attention items, etc.).
F3 consumes this interface without depending on F2's routing implementation. The tasks below
are written against the data shapes already used by the MCP `status` tool in
`kanbanzai/internal/mcp/status_tool.go`, which are the natural reference for the CLI renderer's
input types.

## Task Breakdown

### Task 1: TTY Detection and Rendering Infrastructure

- **Description:** Create an injectable TTY detector and the renderer option struct. Define
  Unicode-to-ASCII symbol mapping and ANSI colour helpers (green, yellow, red, default). The
  renderer accepts a boolean `isTTY` (or equivalent option) so unit tests can exercise both paths
  without spawning a real TTY. When TTY is true, `✓ ✗ ⚠ ● ○ ·` and ANSI colour codes are emitted.
  When false, `[ok] [missing] [warn] [*] [ ] -` are used and no ANSI escapes appear in output.
- **Deliverable:** New package `kanbanzai/internal/cli/render/` containing:
  - `tty.go` — TTY detection function (`IsTTY(fd uintptr) bool`)
  - `symbols.go` — symbol constants and mapping types
  - `colour.go` — ANSI colour helper functions
  - `renderer.go` — `Renderer` struct with options (TTY flag, stdout io.Writer, colour enabled)
  - Unit tests for TTY and non-TTY symbol selection, colour wrapping
- **Depends on:** None (standalone infrastructure)
- **Effort:** medium
- **Spec requirement:** FR-1 (all sub-requirements), NFR-4

### Task 2: Document Views (Unregistered and Registered)

- **Description:** Implement the two document-centric views:
  1. **Unregistered document** (FR-2): When a file path resolves to a file not in the document
     store, renders the path, "Not registered with Kanbanzai.", and a suggested `kbz doc register`
     command. Edge case: non-existent file path renders "file not found" message (exit 0).
  2. **Registered document with owner** (FR-3): When a file path resolves to a registered document
     owned by a feature, renders the document block (path, Title, Type, Status aligned) followed
     by a blank line and the owning feature's full view (delegates to Task 3's feature view).
     Edge case: orphan document (no owner) renders only the document block with draft/pending
     attention items.
- **Deliverable:** Functions in `kanbanzai/internal/cli/render/`:
  - `documents.go` — `RenderUnregisteredDoc`, `RenderRegisteredDoc`, `docBlock` formatting helpers
  - Unit tests covering all FR-2 and FR-3 scenarios
- **Depends on:** Task 1 (TTY/symbol infrastructure)
- **Effort:** medium
- **Spec requirement:** FR-2 (all sub-requirements), FR-3 (all sub-requirements)

### Task 3: Entity Views (Feature, Plan, Project Overview)

- **Description:** Implement the three entity-centric views:
  1. **Feature view** (FR-4): Header with display ID, slug, status (right-aligned); Plan line
     (omitted if no plan); Documents sub-section with three aligned rows (Design, Spec, Dev plan)
     showing status symbols, paths, and status words; Tasks summary line (`● N active · M ready · K done  (T total)`);
     Attention block with `⚠` prefixed items and indented continuation lines. All FR-4 edge cases.
  2. **Plan view** (FR-5): Header with plan ID, slug, status; `Features (N)` sub-section with
     column-aligned feature rows (display ID, slug, status symbol+word); aggregated Tasks summary;
     Attention block with per-feature references. All FR-5 edge cases.
  3. **Project overview** (FR-6): `Kanbanzai  {project-name}` header; `Plans (N)` sub-section
     with per-plan rows (ID, slug, status symbol, activity summary); Health line (green `✓` or
     red `✗` with counts); Attention block; Work queue line. All FR-6 edge cases.
- **Deliverable:** Functions in `kanbanzai/internal/cli/render/`:
  - `feature.go` — `RenderFeature`
  - `plan.go` — `RenderPlan`
  - `project.go` — `RenderProject`
  - `alignment.go` — block-level column alignment helpers (FR-7)
  - Unit tests covering all FR-4, FR-5, FR-6, FR-7 scenarios
- **Depends on:** Task 1 (TTY/symbol infrastructure)
- **Effort:** large
- **Spec requirement:** FR-4 (all sub-requirements), FR-5 (all sub-requirements), FR-6 (all sub-requirements), FR-7 (all sub-requirements)

### Task 4: Integration and Edge-Case Handling

- **Description:** Wire the renderer into the `kbz status` command handler. Define the input data
  types that F2 will populate (modelled on the MCP status tool's synthesis output — feature detail,
  plan dashboard, project overview, document result). Implement the top-level dispatch that selects
  the correct view based on the resolved target type. Handle all AC-7 edge cases: runtime I/O errors
  write to stderr and exit non-zero; valid but non-existent entity IDs produce an informational
  message and exit 0. Ensure the renderer is a pure read operation (NFR-2) and does not panic on
  missing optional fields (NFR-3). Verify all 7 acceptance criteria pass.
- **Deliverable:** 
  - `kanbanzai/internal/cli/render/dispatch.go` — top-level view dispatch
  - `kanbanzai/internal/cli/render/types.go` — shared input data types (referencing the MCP synthesis types)
  - `kanbanzai/cmd/kanbanzai/status_cmd.go` — `runStatus` function wiring resolution to rendering
  - Integration tests covering all AC-1 through AC-7
- **Depends on:** Task 2, Task 3, and F2's interface contract (the data types can be defined and
  tested in isolation before F2 is complete, but the final wiring requires F2's argument resolution)
- **Effort:** medium
- **Spec requirement:** All FRs (integration), all ACs (verification), NFR-2, NFR-3

## Dependency Graph

```
Task 1 (no dependencies)
Task 2 → depends on Task 1
Task 3 → depends on Task 1
Task 4 → depends on Task 2, Task 3, and F2 interface contract

Parallel groups: [Task 2, Task 3]
Critical path: Task 1 → Task 3 → Task 4
```

Task 2 and Task 3 can proceed in parallel once Task 1 is done. Task 4 requires both Task 2 and
Task 3, plus the F2 interface contract. The input types in Task 4 can be defined independently,
allowing partial work before F2 is complete, but final wiring and integration tests require F2.

## Risk Assessment

### Risk: F2 interface contract mismatch
- **Probability:** medium
- **Impact:** medium
- **Mitigation:** Define the input data types in Task 4 by referencing the existing MCP status
  tool's synthesis output (`featureDetail`, `planDashboard`, `projectOverview`, `DocumentResult`).
  These are already the authoritative data shapes. Confirm the contract with F2's dev-plan before
  final wiring. If F2 diverges, the adapter layer in Task 4 absorbs the difference.
- **Affected tasks:** Task 4

### Risk: TTY detection false negatives in CI
- **Probability:** low
- **Impact:** low
- **Mitigation:** Use Go's `golang.org/x/term.IsTerminal` which is well-tested across platforms.
  The injectable design (NFR-4) ensures tests don't depend on real TTY state.
- **Affected tasks:** Task 1

### Risk: Alignment logic complexity across varying content widths
- **Probability:** low
- **Impact:** low
- **Mitigation:** Keep alignment per-block (FR-7.2) which simplifies the problem. Use
  `fmt.Sprintf` with computed widths. The spec already defines the exact column structure
  for each block.
- **Affected tasks:** Task 3

### Risk: Spec ambiguity around plan-level doc inheritance rendering
- **Probability:** low
- **Impact:** medium
- **Mitigation:** The design §5.5.3 and spec FR-4 show the feature view's Documents section
  reflects the feature's own documents. Plan-level inheritance is a data concern resolved by
  the synthesis layer (already implemented in `status_tool.go`), not the renderer. The renderer
  receives a pre-resolved document list.
- **Affected tasks:** Task 3, Task 4

## Interface Contracts

The renderer consumes structured input types from the resolution layer. These types mirror
the MCP status tool's synthesis output already defined in `kanbanzai/internal/mcp/status_tool.go`.

| Contract | Consumer | Provider | Data Shape Reference |
|----------|----------|----------|---------------------|
| Feature detail input | `RenderFeature` | F2 resolution layer | `featureDetail` struct (status_tool.go L1053+) |
| Plan dashboard input | `RenderPlan` | F2 resolution layer | `planDashboard` struct (status_tool.go L580+) |
| Project overview input | `RenderProject` | F2 resolution layer | `projectOverview` struct (status_tool.go L280+) |
| Document lookup result | `RenderRegisteredDoc` | F2 resolution layer | `DocumentResult` struct (service/documents.go L100+) |
| Attention items | All views | F2 resolution layer | `AttentionItem` struct (status_tool.go L200+) |
| TTY status | All views | `os.Stdout` fd | `golang.org/x/term.IsTerminal` |

All inputs are passed by value or pointer — the renderer never modifies them (NFR-2).
The renderer writes exclusively to an `io.Writer` (stdout or a test buffer).

## Traceability Matrix

| Spec Requirement | Task(s) | Verification |
|-----------------|---------|-------------|
| FR-1: TTY Detection | Task 1 | Unit tests: TTY and non-TTY symbol+colour output |
| FR-2: Unregistered Document View | Task 2 | Unit tests: all sub-requirements and edge cases |
| FR-3: Registered Document with Owner View | Task 2 | Unit tests: with-owner, orphan, draft status |
| FR-4: Direct Feature Lookup View | Task 3 | Unit tests: all sub-requirements and edge cases |
| FR-5: Plan Lookup View | Task 3 | Unit tests: all sub-requirements and edge cases |
| FR-6: Project Overview View | Task 3 | Unit tests: all sub-requirements and edge cases |
| FR-7: Alignment and Layout | Task 3 | Unit tests: per-block column alignment |
| NFR-1: Performance (<100ms render) | Task 3, Task 4 | Benchmark test |
| NFR-2: No State Mutation | Task 4 | Code review: renderer writes only to io.Writer |
| NFR-3: Robustness (no panic) | Task 4 | Unit tests: nil/empty optional fields |
| NFR-4: Injectable TTY | Task 1 | Unit tests: boolean flag injection |

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|---------------------|--------------------| ---------------|
| AC-1: TTY rendering — Unicode+ANSI on TTY, ASCII+no-colour when piped | Unit test (TTY flag injection) | Task 1 |
| AC-2: Unregistered document view — path, "Not registered", suggested command; non-existent → "file not found", exit 0 | Unit test | Task 2 |
| AC-3: Registered document with owner — doc block then feature block; draft docs → attention; orphan → doc block only | Unit test | Task 2 |
| AC-4: Direct feature lookup — header, plan, docs, tasks, attention; all edge cases (no dev-plan, no plan, no docs, no tasks) | Unit test | Task 3 |
| AC-5: Plan lookup — feature list, aggregated tasks, per-feature attention; empty plan, all-done plan | Unit test | Task 3 |
| AC-6: Project overview — plans, health, attention, work queue; no attention, no plans, errors | Unit test | Task 3 |
| AC-7: Edge cases — I/O error → stderr + non-zero exit; non-existent entity → message + exit 0 | Integration test | Task 4 |
| NFR-2: No state mutation | Code review (renderer writes only to io.Writer) | Task 4 |
| NFR-3: No panic on missing optional fields | Unit test (nil/empty input fields) | Task 4 |
| NFR-4: TTY detection injectable | Unit test (boolean flag) | Task 1 |
