# Design: Fix Empty Task Names in `decompose propose`

**Feature:** FEAT-01KPQ08Y71A8V  
**Plan:** P25 — Agent Tooling and Pipeline Quality  
**Status:** Draft

---

## Overview

`decompose propose` generates a `Proposal` object containing `ProposedTask` entries. Each
`ProposedTask` has a `Name` field that `decompose apply` passes directly to `CreateTask`.
`CreateTask` calls `validate.ValidateName` on this value, which rejects empty strings with
`"name must not be empty"`.

The bug is that `generateProposal` in `internal/service/decompose.go` never sets the `Name`
field on any `ProposedTask`. Because the field is a Go `string`, it defaults to `""`. Every
proposal produced by the current code is therefore silently broken: `apply` will always fail
with a validation error on the first task, regardless of spec content.

**Current behaviour:** `decompose apply` fails immediately for every feature with the error
`"Cannot create task %q for feature %s: name must not be empty"`. The agent must fall back to
manual `entity(action: "create")` calls, which requires knowledge of the internal task schema
and dependency wiring format that the `decompose` tool is supposed to encapsulate.

**Required behaviour:** Every `ProposedTask` returned by `decompose propose` must have a
non-empty `Name` that passes `validate.ValidateName`. The proposal must be safe to pass
unmodified to `decompose apply`.

---

## Goals and Non-Goals

### Goals

- Every `ProposedTask` in a proposal returned by `decompose propose` has a non-empty `Name`
  that satisfies `validate.ValidateName` (non-empty, ≤ 60 characters, no colon, no phase
  prefix).
- Name derivation is correct for all three AC formats the parser recognises: checkbox,
  numbered list, and bold-identifier (`**AC-NN.** text`).
- A deterministic fallback name is produced when derivation yields an empty or invalid string.
- Existing tests continue to pass. New tests cover the `Name` field on proposals and the
  end-to-end `apply` path.

### Non-Goals

- Changing how `Slug` or `Summary` are derived — those fields are correct.
- Improving the quality of names beyond "readable and valid" — this is not a UX polish task.
- Addressing the separate issue of AC-based grouping heuristics (P4 in the proposal).
- Adding dev-plan-aware grouping (also P4).
- Validating `Name` inside `generateProposal` itself — validation remains the responsibility
  of `CreateTask` via `validate.ValidateName`.

---

## Design

### Root cause

`generateProposal` constructs `ProposedTask` structs in two code paths:

1. **Grouped task** (2–4 ACs in one L2 section): creates a task with `Slug`, `Summary`,
   `Rationale`, and `Covers` — `Name` is never set.
2. **Individual task** (1 AC or 5+ ACs in a section): creates a task with `Slug`, `Summary`,
   `Rationale`, and `Covers` — `Name` is never set.

A third path, the auto-appended test task (`featureSlug + "-tests"`), also omits `Name`.

The field exists on the struct and carries a `json:"name"` tag. The JSON round-trip through
`parseProposal` (in `decompose_tool.go`) therefore preserves the empty string faithfully into
`decomposeApply`, where it is passed as-is to `CreateTask`.

### The name derivation challenge

For checkbox and numbered-list ACs, `ac.text` is plain prose (e.g. `"Users can log in with
email and password"`). After truncation to 60 characters this is directly usable as a `Name`.

For bold-identifier ACs, `parseSpecStructure` stores the criterion as
`identifier + ": " + description` (e.g. `"AC-01: The service MUST accept JSON input"`).
This string contains a colon. `validate.ValidateName` rejects names containing colons (Rule 3).
Using `ac.text` directly as `Name` for this format would cause `CreateTask` to fail —
replacing one empty-name error with a colon-name error.

A name derivation helper must therefore handle this format explicitly.

### Fix: `deriveTaskName` helper

Introduce a package-private helper `deriveTaskName(text, fallback string) string` in
`decompose.go`:

1. **Strip bold-ident prefix.** If `text` matches `<IDENT>: <rest>` (i.e. contains `": "`
   with a non-empty prefix that has no spaces), discard the prefix and use `<rest>` as the
   candidate name. This handles the `"AC-01: description"` format produced by
   `parseSpecStructure` for bold-identifier criteria.

2. **Trim to 60 characters.** Truncate the candidate at a word boundary where possible; never
   truncate mid-word if a shorter clean boundary exists within the limit.

3. **Validate the result.** If after stripping and truncating the candidate is still empty,
   return `fallback`.

4. **Return the candidate.** The caller is responsible for ensuring the fallback is also
   non-empty and colon-free.

### Applying `deriveTaskName` in `generateProposal`

**Grouped task path:** `Name = deriveTaskName("Implement "+sectionTitle, "Implement grouped tasks")`. If `sectionTitle` is empty, the fallback fires directly.

**Individual task path:** `Name = deriveTaskName(ac.text, "Implement AC-"+fmt.Sprintf("%03d", taskIndex+i+1))`. The fallback uses a positional AC identifier.

**Test task:** `Name = "Write tests"` (constant; always valid).

### Where the fix lives

All changes are confined to `internal/service/decompose.go`. No changes are required in
`internal/mcp/decompose_tool.go`, `internal/service/entities.go`, or
`internal/validate/entity.go`. The interface contract between `propose` and `apply` is
unchanged — the proposal struct gains populated `Name` fields; its shape does not change.

### Failure modes

| Scenario | Outcome after fix |
|---|---|
| AC text is plain prose, ≤ 60 chars | Name = AC text directly |
| AC text is plain prose, > 60 chars | Name = truncated AC text |
| AC text is bold-ident format (`"AC-01: desc"`) | Name = description part only |
| AC text is empty string (parser defect) | Name = positional fallback `"Implement AC-NNN"` |
| Section title is empty for grouped task | Name = `"Implement grouped tasks"` fallback |

No scenario produces an empty name. No scenario returns an error from `generateProposal` —
the fallback name ensures `apply` can always proceed.

### Test coverage gaps to close

- `TestDecomposeFeature_ProposalProduced`: add assertion that `task.Name != ""` for every task.
- Add `TestDecomposeFeature_BoldACSpec_NameHasNoColon`: assert that a bold-ident spec produces
  proposals whose task names contain no colons and pass `validate.ValidateName`.
- Add `TestDecomposeApply_SucceedsWithProposedNames`: end-to-end test that calls `propose` then
  feeds the unmodified proposal to `decomposeApply` and confirms all tasks are created without
  error.

---

## Alternatives Considered

### Alternative 1: Fallback to AC identifier only (e.g. `"Implement AC-001"`)

Use a positional identifier as the name for every task, ignoring AC text entirely.

**What it makes easier:** The derivation logic is trivial — one line per task path.

**What it makes harder:** Names carry no semantic content. The agent reviewing the proposal
sees `"Implement AC-001"` through `"Implement AC-007"` with no indication of what each task
involves. This degrades proposal readability and forces the agent to cross-reference the
`covers` field for every task before deciding whether to accept the proposal.

**Why rejected:** The AC text is available and — for the majority of specs using checkbox or
numbered formats — is directly usable as a readable name. Discarding it when it is present is
an unnecessary loss of signal. The identifier fallback is appropriate only when the derived
text is empty.

### Alternative 2: Fail explicitly from `generateProposal` with a diagnostic

If a task name cannot be derived, return an error from `generateProposal` (and therefore from
`DecomposeFeature`) with a message identifying which AC could not produce a name.

**What it makes easier:** The contract is explicit — either the proposal is fully valid or the
call fails cleanly. No silent degradation to fallback names.

**What it makes harder:** Any spec that produces a derivation edge case now blocks the entire
`propose` action with an error. The agent must inspect and fix the spec before decomposition
can proceed. For empty AC text (which would only arise from a parser defect, not a spec
authoring error), this forces the agent to debug an internal tool failure rather than
continuing with a reasonable fallback.

**Why rejected:** The failure cases that reach the fallback path are either parser defects
(empty `ac.text` after a regex match, which should not occur in normal operation) or edge
cases in spec formatting that do not warrant blocking the entire decomposition. A fallback name
is recoverable — the agent can rename the task after `apply`. An error from `propose` cannot
be recovered without changing the spec. The marginal safety of explicit failure does not
justify the interruption to the workflow in cases where a valid fallback exists. Explicit
failure remains appropriate if a future change introduces a code path where no fallback can
be constructed, but that does not arise with this fix.

### Alternative 3: Strip proposal items with empty names before returning

Before returning from `generateProposal`, filter out any `ProposedTask` where `Name == ""`.

**What it makes easier:** The code change is minimal — a post-processing filter.

**What it makes harder:** Work is silently dropped. An AC that exists in the spec produces no
task in the proposal, and nothing in the output indicates that any task was omitted. The
agent proceeds to `apply`, the feature's tasks cover fewer acceptance criteria than the spec
requires, and the gap is invisible unless the agent manually counts tasks against ACs. This
is worse than the current bug: the current bug fails loudly; silent omission corrupts the
decomposition result without signalling that anything went wrong.

**Why rejected:** A proposal that does not cover all spec ACs is invalid output. Silently
dropping tasks to avoid an error is the wrong trade-off. The design must ensure every AC
contributes a task, not that only ACs producing valid names are included.

---

## Dependencies

- **`internal/service/decompose.go`** — the only file that requires code changes.
- **`internal/service/decompose_test.go`** — new and updated tests.
- **`internal/validate/entity.go`** — read-only dependency; `ValidateName` rules constrain the
  name derivation logic but are not changed.
- **`internal/service/entities.go`** — read-only dependency; `CreateTask` validates `Name` but
  is not changed.
- No changes to the MCP layer (`internal/mcp/decompose_tool.go`), the model layer, or any
  configuration.
```

Now let me register this document: