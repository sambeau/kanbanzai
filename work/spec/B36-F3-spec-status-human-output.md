# Specification: Status Command Human Output

| Field  | Value                            |
|--------|----------------------------------|
| Date   | 2026-04-30                       |
| Status | Draft                            |
| Author | spec-author                      |

---

## Related Work

- **Design:** `work/_project/design-kbz-cli-and-binary-rename.md` — §5.4 (`human` format),
  §5.5 (five output views), §5.6 (visual conventions). That document is the authoritative
  source for the visual language and layout rules applied here.
- **B36-F2 (argument resolution)** — F2 owns the logic that determines which entity or
  document a target string refers to and routes it to the renderer. This specification begins
  where F2 ends: it defines what the renderer produces for each resolved route. F3 has a hard
  dependency on F2's routing interface being available.
- **B36-F4 (machine output formats)** — Covers `--format plain` and `--format json`. Those
  formats are out of scope here. This spec covers the `human` format only.

---

## Overview

`kbz status` currently outputs a thin stub: a health check followed by a work-queue count
line. This feature replaces that stub with a properly formatted human-readable renderer that
covers five distinct views (unregistered document, registered document with owner entity,
direct feature lookup, plan lookup, and project overview) and adapts its presentation
automatically to whether stdout is a TTY or a pipe.

---

## Scope

**In scope:**

- TTY detection and automatic selection of Unicode symbols + ANSI colour (TTY) vs. ASCII
  fallbacks and no colour (non-TTY).
- All five output views defined in §5.5 of the design.
- Boundary and edge cases for each view (empty collections, missing optional data, etc.).
- Exit behaviour: the command exits 0 on any successful query (including "not found" answers
  such as an unregistered document). Exit non-zero only on runtime errors (I/O failure,
  corrupt state, etc.).

**Out of scope:**

- `--format plain` and `--format json` — covered by B36-F4.
- Argument parsing and target resolution — covered by B36-F2.
- MCP `status` tool — unchanged by this feature.
- Any write operations or state mutations.

---

## Functional Requirements

### FR-1: TTY Detection

**FR-1.1** The renderer MUST check, at render time, whether stdout is connected to a TTY.

**FR-1.2** When stdout IS a TTY:
- Unicode symbols MUST be used: `✓ ✗ ⚠ ● ○ ·`
- ANSI colour MUST be applied according to the colour table in §5.6 (green = done/approved/
  healthy; yellow = active/warning; red = errors/missing required; default = ready/neutral).

**FR-1.3** When stdout is NOT a TTY (piped, redirected, or in a CI environment where no TTY
is allocated):
- Colour MUST be suppressed entirely (no ANSI escape codes in the output stream).
- Each Unicode symbol MUST be replaced with its ASCII fallback:

  | Unicode | ASCII fallback |
  |---------|---------------|
  | `✓`     | `[ok]`        |
  | `✗`     | `[missing]`   |
  | `⚠`     | `[warn]`      |
  | `●`     | `[*]`         |
  | `○`     | `[ ]`         |
  | `·`     | `-`           |

**FR-1.4** No flag or environment variable is needed for the user to obtain clean output
when piping. Detection is fully automatic.

---

### FR-2: View — Unregistered Document

Triggered when the target resolves to a file path that exists on disk but has no matching
document record in the document store.

**FR-2.1** The first line of output MUST be the file path as supplied by the user (not
normalised or resolved to an absolute path).

**FR-2.2** The second line MUST be blank, followed by an indented line reading exactly:
`Not registered with Kanbanzai.`

**FR-2.3** A blank line MUST follow, then an indented `To register:` label, then an
indented suggested `kbz doc register` command. The suggested command MUST include the
file path, `--type` and `--title` placeholders (or inferred values if the type can be
guessed from the path or filename).

**FR-2.4** No entity data, task counts, or attention items are shown. The view is strictly
scoped to the document registration prompt.

**FR-2.5 (Edge case — file not found):** When the target looks like a file path but does
not exist on disk, the renderer MUST output a message indicating the file was not found,
followed by exit 0. (The command successfully determined that the file does not exist; this
is not a runtime error.)

---

### FR-3: View — Registered Document with Owner Entity

Triggered when the target resolves to a registered document that belongs to a feature entity.

**FR-3.1** A **document block** MUST appear first. It MUST contain:
- Line 1: the document file path (as recorded in the document store).
- Indented fields: `Title:`, `Type:`, `Status:`. Values are aligned to a common column
  within this block.

**FR-3.2** A blank line MUST separate the document block from the feature block.

**FR-3.3** The feature block MUST be rendered identically to a direct feature lookup
(FR-4), with all the same sub-sections (Plan, Documents, Tasks, Attention).

**FR-3.4 (Edge case — document with no owner):** When a registered document has no owning
feature (e.g., an orphan doc), only the document block is shown. No feature block appears.
The document block MUST include the `Status:` field; if status is `draft` or `pending`, an
attention item MUST be appended below the document block noting it is not yet approved.

---

### FR-4: View — Direct Feature Lookup

Triggered when the target resolves directly to a feature entity.

**FR-4.1** The header line MUST contain, in order:
- The word `Feature` followed by two spaces.
- The short display ID (e.g., `F-042`), a space-separator symbol (`·`/`-`), and the slug.
- The feature's lifecycle status right-aligned on the same line.

**FR-4.2** Immediately below the header, indented, a `Plan:` line MUST show the plan's
display ID and slug in the format `P1-my-plan · main-plan` (or ASCII equivalent). If the
feature has no plan, this line MUST be omitted entirely.

**FR-4.3** A `Documents` sub-section MUST follow, headed by the label `Documents` (no
colon). Under it, one row per document type in a fixed order: Design, Spec, Dev plan.
Each row MUST contain:
- The document type label (right-padded so values align across rows in the block).
- A status symbol: `✓`/`[ok]` for present and approved; `✗`/`[missing]` for absent;
  `⚠`/`[warn]` for present but not yet approved (draft/pending).
- For present documents: the file path, then the status word (`approved`, `draft`, etc.)
  right-padded so status values align across rows.
- For absent documents: the word `missing` in place of path and status.

**FR-4.4** A `Tasks` summary line MUST follow. Format:
`Tasks  ● {active} active · {ready} ready · {done} done  ({total} total)`
(or ASCII equivalent). If there are no tasks at all, the line MUST still appear showing
`0` counts.

**FR-4.5** An `Attention` block MUST follow if there are any attention items. Each item
MUST be prefixed with `⚠`/`[warn]`. Multi-line attention items (where a suggested command
is included) MUST indent the continuation line(s) by three spaces relative to the `⚠`
marker.

**FR-4.6 (Edge case — feature with no documents of any type):** The Documents sub-section
MUST still render. All three document type rows MUST show `✗ missing`/`[missing] missing`.
An attention item MUST appear noting that no documents have been registered.

**FR-4.7 (Edge case — feature with no tasks):** The Tasks line MUST show all-zero counts.
No additional warning is generated solely from the zero-task state (task absence may be
normal early in the lifecycle).

**FR-4.8 (Edge case — feature with no plan):** The `Plan:` line is omitted. No warning or
attention item is generated solely from the absence of a plan.

---

### FR-5: View — Plan Lookup

Triggered when the target resolves to a plan entity.

**FR-5.1** The header line MUST contain, in order:
- The word `Plan` followed by two spaces.
- The full plan ID with slug (e.g., `P1-main-plan · main-plan`), or ASCII equivalent.
- The plan's lifecycle status right-aligned on the same line.

**FR-5.2** A `Features ({n})` sub-section MUST list every feature belonging to the plan,
one per line. Each row MUST contain:
- The feature's short display ID.
- The feature slug.
- A status symbol and the status word. Green `✓` for done, yellow `●` for active/
  developing, default `○` for ready (or ASCII equivalents).

**FR-5.3** Rows in the Features list MUST be column-aligned (slug column, status symbol
column, status word column) within the block.

**FR-5.4** A `Tasks` summary line (same format as FR-4.4) MUST show aggregate counts
across all features in the plan.

**FR-5.5** An `Attention` block MUST follow if there are any attention items affecting any
feature in the plan, using the same format as FR-4.5. Each item MUST be prefixed with the
short feature reference (e.g., `F-042 my-feature:`).

**FR-5.6 (Edge case — plan with no features):** The `Features (0)` heading MUST appear
with no rows beneath it. The Tasks line MUST show all-zero counts. No synthetic warning is
generated from the zero-feature state.

**FR-5.7 (Edge case — plan where all features are done):** The Tasks line MUST still show
the correct historical counts. Done features remain listed with the green `✓ done`
indicator.

---

### FR-6: View — Project Overview (No Target)

Triggered when `kbz status` is invoked with no target argument.

**FR-6.1** The header line MUST read `Kanbanzai  {project-name}` where `{project-name}`
is the name from the project configuration.

**FR-6.2** A `Plans ({n})` sub-section MUST list every plan, one per line. Each row MUST
contain: the plan display ID and slug, a status symbol and status word, and a brief
activity summary in the form `{n} features active` or `{n} features started`
(0 when none are active). Rows MUST be column-aligned within the block.

**FR-6.3** A `Health` line MUST follow. Format:
`Health  ✓ no errors · {n} warnings` (or ASCII equivalent with colour). If there are
errors: `Health  ✗ {n} errors · {n} warnings` coloured red.

**FR-6.4** An `Attention` block MUST list all project-level attention items, prefixed by
`⚠`/`[warn]`. Each item references the affected entity or document by ID and short label.

**FR-6.5** A `Work queue` line MUST appear last:
`Work queue  {n} ready · {n} active` (or ASCII equivalent).

**FR-6.6 (Edge case — no plans):** The `Plans (0)` heading MUST appear with no rows.
The Health, Attention, and Work queue sections MUST still be rendered.

**FR-6.7 (Edge case — no attention items):** The Attention block is omitted entirely.
The Work queue line follows directly after Health.

**FR-6.8 (Edge case — no ready or active tasks):** The Work queue line MUST still appear
showing `0 ready · 0 active`.

---

### FR-7: Alignment and Layout

**FR-7.1** Within a single block (e.g., the Documents sub-section of a feature view), all
value columns MUST align to a common character position. The column width is determined by
the longest label in that block.

**FR-7.2** Alignment MUST NOT carry across blocks. Each block calculates its own column
widths independently.

**FR-7.3** Lines MUST NOT be artificially wrapped or truncated. The terminal handles
wrapping. Content MUST be authored to be readable at 80 columns but the renderer does not
enforce a maximum width.

**FR-7.4** Each major block (header, sub-section, attention) MUST be separated by a blank
line in the output.

---

## Non-Functional Requirements

**NFR-1 (Performance):** The renderer MUST produce output within 100 ms for any single
view given normal project sizes (up to 100 features, 500 tasks). Data retrieval latency is
excluded from this budget; the rendering step itself must not introduce observable delay.

**NFR-2 (No state mutation):** The renderer MUST be a pure read operation. It MUST NOT
modify any entity, document record, or index as a side effect of rendering.

**NFR-3 (Robustness):** The renderer MUST NOT panic on missing optional fields (e.g., a
feature with no plan, a document with no title). It MUST render gracefully with empty
strings or omitted lines as specified in the edge-case requirements above.

**NFR-4 (Test surface):** The TTY-detection logic MUST be injectable (e.g., via a boolean
parameter or a renderer option) so that unit tests can exercise both TTY and non-TTY
rendering paths without spawning a real TTY.

---

## Acceptance Criteria

### AC-1: TTY rendering

| # | Scenario | Expected |
|---|----------|----------|
| 1.1 | `kbz status` run in an interactive terminal | Output contains Unicode symbols and ANSI colour codes |
| 1.2 | `kbz status \| cat` | Output contains ASCII fallbacks; no ANSI escape sequences in the byte stream |
| 1.3 | `kbz status > file.txt` | Same as 1.2 |

### AC-2: Unregistered document view

| # | Scenario | Expected |
|---|----------|----------|
| 2.1 | `kbz status work/design/new-draft.md` where file exists but is not registered | Output starts with the path; second section reads "Not registered with Kanbanzai."; a `kbz doc register` command is shown |
| 2.2 | File path given does not exist on disk | Output states file not found; exit 0 |

### AC-3: Registered document with owner

| # | Scenario | Expected |
|---|----------|----------|
| 3.1 | `kbz status work/design/my-feature.md` for an approved design belonging to a feature | Document block (path, Title, Type, Status) appears before the feature block; feature block matches AC-4 expectations |
| 3.2 | Document is registered but in `draft` status | Status field shows `draft`; attention item noting not yet approved appears below the document block |
| 3.3 | Document has no owning feature (orphan) | Only document block rendered; no feature block |

### AC-4: Direct feature lookup

| # | Scenario | Expected |
|---|----------|----------|
| 4.1 | Feature with plan, all three docs, some tasks | Header, Plan line, Documents section (three rows, aligned), Tasks line, no Attention block |
| 4.2 | Feature with a missing dev-plan | Dev-plan row shows `✗ missing`; Attention block shows warning with suggested `kbz doc register` command |
| 4.3 | Feature with no plan | `Plan:` line absent; no warning generated for the missing plan |
| 4.4 | Feature with no documents at all | All three doc rows show `✗ missing`; Attention item noting no documents registered |
| 4.5 | Feature with no tasks | Tasks line shows `0 active · 0 ready · 0 done  (0 total)` |

### AC-5: Plan lookup

| # | Scenario | Expected |
|---|----------|----------|
| 5.1 | Plan with 5 features in mixed states | All five listed, column-aligned; task counts aggregated across all features |
| 5.2 | Plan with no features | `Features (0)` heading; Tasks line with zero counts |
| 5.3 | Plan where a feature has no dev-plan | Attention block references that feature by short ID and slug |

### AC-6: Project overview

| # | Scenario | Expected |
|---|----------|----------|
| 6.1 | Project with 2 plans, 2 attention items, tasks in queue | Plans section, Health line, Attention block with 2 items, Work queue line |
| 6.2 | Project with no attention items | Attention block absent; Work queue line directly follows Health line |
| 6.3 | Project with no plans | `Plans (0)` heading; Health, Work queue still rendered |
| 6.4 | Health check finds errors | Health line shows red `✗ {n} errors · {n} warnings` |

### AC-7: Edge cases

| # | Scenario | Expected |
|---|----------|----------|
| 7.1 | Runtime I/O error reading state | Command exits non-zero with an error message to stderr |
| 7.2 | Target is a valid entity ID that does not exist | Informational message; exit 0 |

---

## Verification Plan

1. **Unit tests — rendering logic:** Exercise each of the five view functions with synthetic
   data. Cover TTY and non-TTY paths for each view. Verify symbol substitution, column
   alignment, and blank-line separation. All edge cases listed in FR-2 through FR-6 MUST
   have corresponding test cases.

2. **Unit tests — TTY injection:** Confirm that passing a `isTTY=false` option produces
   output free of ANSI escape sequences and uses only ASCII fallback symbols.

3. **Integration test — stub replacement:** Run `kbz status` against a fixture project
   state. Confirm the output no longer matches the old stub format (health dump + work-queue
   count line) and instead matches the project overview format specified in FR-6.

4. **Pipe test:** Confirm `kbz status | cat` produces clean output with no ANSI codes on a
   developer workstation where stdout of the test process is not a TTY.

5. **Manual review:** A reviewer with the feature in `developing` state MUST run each of the
   five `kbz status` invocation forms against a real project and verify the output against
   the annotated examples in §5.5 of the design document.

---

## Dependencies and Assumptions

**DEP-1:** B36-F2 (argument resolution) MUST be merged and its routing interface stable
before the renderer can be integrated. F3 consumes F2's resolved result type; the contract
between them (what data F2 hands to the renderer) MUST be agreed before implementation
begins.

**DEP-2:** The service-layer calls used by the existing MCP `status` tool (entity lookup,
document lookup, task counts, health check, work queue) are assumed to be available and
correct. F3 reuses those calls and does not reimplement data retrieval.

**DEP-3:** The existing `runStatus` stub in `cmd/kanbanzai/workflow_cmd.go` (which calls
`runHealth` then prints a work-queue count) is the implementation being replaced. The new
implementation MUST cover the same project-overview case (no target) plus the four
additional cases added by this feature.

**ASSUME-1:** ANSI colour support is assumed to be present on any TTY stdout. No attempt is
made to detect specific terminal capabilities beyond the TTY check.

**ASSUME-2:** The document store distinguishes between `design`, `spec`, and `dev-plan`
types using the type field recorded at registration time. The renderer uses these type
strings to populate the fixed document rows. If a feature has multiple documents of the
same type, all are shown (rows are added, not merged).

**ASSUME-3:** Project name is available from the project configuration without additional
I/O beyond what is already loaded for other `kbz` commands.
