# Specification: Fix Empty Task Names in `decompose propose`

**Feature:** FEAT-01KPQ08Y71A8V
**Plan:** P25 — Agent Tooling and Pipeline Quality
**Design:** work/design/p25-fix-decompose-empty-names.md
**Status:** Draft

---

## 1. Overview

`decompose propose` generates a `Proposal` containing `ProposedTask` entries. Each entry's
`Name` field is currently never populated, defaulting to an empty string. When `decompose apply`
passes these names to `CreateTask`, the call fails with a validation error. This specification
covers the fix: every `ProposedTask` returned by `decompose propose` MUST carry a non-empty
`Name` that satisfies `validate.ValidateName`, derived from the acceptance criterion text or a
deterministic positional fallback.

---

## 2. Scope

**In scope:**

- Populating the `Name` field on every `ProposedTask` produced by `generateProposal` in
  `internal/service/decompose.go`.
- A `deriveTaskName` helper that strips bold-identifier prefixes, truncates to the 60-character
  limit, and falls back to a positional identifier when derivation yields an empty string.
- Correct handling of all three AC formats: checkbox, numbered list, and bold-identifier
  (`**AC-NN.** text`).
- The test task appended unconditionally at the end of a proposal.
- Test coverage for the `Name` field on proposals and the end-to-end `apply` path.

**Out of scope:**

- Changes to `Slug` or `Summary` derivation.
- Changes to `validate.ValidateName` rules.
- Changes to the MCP layer (`internal/mcp/decompose_tool.go`).
- Changes to `internal/service/entities.go` or `CreateTask`.
- AC-based grouping heuristics or dev-plan-aware grouping (tracked separately).

---

## 3. Functional Requirements

**FR-001:** Every `ProposedTask` in a proposal returned by `decompose propose` MUST have a
non-empty `Name` field.

**Acceptance criteria:**
- Calling `decompose propose` on any feature with a valid spec produces a proposal where
  `task.Name != ""` for every task, including grouped tasks and the appended test task.
- Calling `decompose apply` with the unmodified proposal (as returned by `propose`) completes
  without a `"name must not be empty"` error.

---

**FR-002:** The `Name` field MUST satisfy `validate.ValidateName`: non-empty, at most 60
characters, no colon character, no phase prefix.

**Acceptance criteria:**
- `validate.ValidateName(task.Name)` returns nil for every task name in every proposal.
- No task name in a proposal contains a colon character.
- No task name exceeds 60 characters.

---

**FR-003:** Name derivation for checkbox and numbered-list AC formats MUST use the AC's text
directly as the candidate name (after truncation).

**Acceptance criteria:**
- A checkbox AC with text `"Users can log in with email and password"` produces a task whose
  `Name` is `"Users can log in with email and password"` (or truncated to ≤60 characters).

---

**FR-004:** Name derivation for bold-identifier AC format (`**AC-NN.** description`) MUST strip
the `[A-Z]+-\d+: ` prefix before using the text, so that the resulting name contains no colon.

**Acceptance criteria:**
- A bold-ident AC stored as `"AC-01: The service MUST accept JSON input"` produces a task whose
  `Name` is `"The service MUST accept JSON input"` (no `"AC-01:"` prefix, no colon).
- `validate.ValidateName` returns nil for this name.
- A plain-prose AC that contains a colon in its body (e.g. `"Login: users can authenticate"`)
  but does NOT begin with the strict `[A-Z]+-\d+: ` pattern is NOT stripped — the full text
  (truncated if needed) is used as the name candidate.

---

**FR-005:** When the derived candidate name is empty after prefix stripping, the implementation
MUST fall back to a deterministic positional name of the form `"Implement AC-NNN"` (zero-padded
to three digits).

**Acceptance criteria:**
- An AC whose text is an empty string after parsing produces a task name matching
  `"Implement AC-\d{3}"`.
- The fallback name passes `validate.ValidateName`.

---

**FR-006:** The grouped-task path (2–4 ACs in one L2 section) MUST derive a name from the
section title using the pattern `"Implement " + sectionTitle`. When `sectionTitle` is empty, the
fixed fallback `"Implement grouped tasks"` MUST be used.

**Acceptance criteria:**
- A grouped task for a section titled `"Authentication"` produces `Name = "Implement Authentication"`.
- A grouped task for a section with an empty title produces `Name = "Implement grouped tasks"`.
- Both names pass `validate.ValidateName`.

---

**FR-007:** The unconditionally appended test task MUST have the fixed name `"Write tests"`.

**Acceptance criteria:**
- The test task appended by `generateProposal` has `Name == "Write tests"`.
- `validate.ValidateName("Write tests")` returns nil.

---

**FR-008:** The `Proposal` struct's shape MUST remain unchanged. No new fields are added; only
previously-empty `Name` fields are now populated.

**Acceptance criteria:**
- `decompose apply` accepts a proposal from `decompose propose` without schema changes.
- The JSON serialisation of a proposal contains non-empty `"name"` values for all tasks.

---

**FR-009:** `deriveTaskName` MUST be implemented as an unexported package-level helper in
`internal/service/decompose.go`.

**Acceptance criteria:**
- The helper is callable from within `generateProposal` for all task-construction code paths.
- The helper is not exported (lowercase function name).

---

## 4. Non-Functional Requirements

**NFR-001:** All changes MUST be confined to `internal/service/decompose.go` and
`internal/service/decompose_test.go`. No changes to the MCP layer, model layer, or validate
package are permitted.

**NFR-002:** Existing tests in `decompose_test.go` MUST continue to pass without modification
(except where they require updating to assert the new `Name` field presence).

**NFR-003:** The fix MUST NOT introduce a new error return from `generateProposal`. Name
derivation failures MUST fall back silently to the positional fallback name; they MUST NOT
propagate as errors.

---

## 5. Acceptance Criteria

**AC-01:** `TestDecomposeFeature_ProposalProduced` asserts that `task.Name != ""` for every task
in the returned proposal.

**AC-02:** A new test `TestDecomposeFeature_BoldACSpec_NameHasNoColon` verifies that a spec using
bold-identifier AC format produces task names containing no colons, all of which pass
`validate.ValidateName`.

**AC-03:** A new end-to-end test `TestDecomposeApply_SucceedsWithProposedNames` calls
`decompose propose` on a feature with a valid spec, passes the unmodified proposal to
`decomposeApply`, and asserts that all tasks are created without error.

**AC-04:** A test verifies that a spec with an AC whose text is empty produces a task name
matching `"Implement AC-\d{3}"`.

**AC-05:** A test verifies that a spec with a plain-prose AC containing a colon in the body
(but not beginning with `[A-Z]+-\d+: `) produces a task name equal to that prose text (truncated
if needed), not stripped.

---

## 6. Dependencies and Assumptions

**DEP-001:** `validate.ValidateName` in `internal/validate/entity.go` is the authoritative
source of name validity rules. The derivation logic MUST be consistent with those rules. This
file is read-only for this feature.

**DEP-002:** `internal/service/entities.go` — `CreateTask` calls `validate.ValidateName` and
is unaffected by this change. Read-only dependency.

**DEP-003:** FEAT-01KPQ08YBJ5AK (dev-plan-aware grouping) also modifies `decompose.go`. These
two features MUST be sequenced or their branches coordinated to avoid merge conflicts. This
feature (fix empty names) is the lower-effort change and SHOULD land first.

**ASM-001:** The three AC formats (checkbox, numbered list, bold-identifier) are the complete
set of formats produced by `parseSpecStructure`. No other format requires special handling.

**ASM-002:** The `ac.text` field populated by `parseSpecStructure` for bold-identifier format
is of the form `"AC-NN: description"` (identifier + colon + space + prose). The strict regex
`^[A-Z]+-\d+: ` reliably identifies this format without false positives on plain prose.