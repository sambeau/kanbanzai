| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-07                     |
| Author | spec-author                     |
| Status | draft                          |
| Plan   | P56-bug-lifecycle-hardening    |
| Feature | FEAT-01KR12RE99QPT             |

# Specification: Bug Spec and Document Infrastructure

## Related Work

- **P56-design-bug-lifecycle-hardening.md** (P56-bug-lifecycle-hardening/design-p56-design-bug-lifecycle-hardening) — Design document. This spec implements Components B and F.
- **internal/model/entities.go** — Existing bug entity model. This spec extends `CreateBugInput` with `fix_plan`.
- **internal/service/entities.go** — Existing `CreateBug` function. This spec adds spec auto-generation to the creation path.
- **internal/service/documents.go** — Existing `CanonicalDocPath`. This spec extends it for bug entities.
- **internal/config/config.go** — Existing `DefaultFastTrackConfig`. This spec changes the `bug_fix` tier's `Spec` gate from `human` to `auto`.

**Constraining decisions:**
- P56 Decision 1: Bugs get inline specs, not separate documents.
- P56 Decision 3: `bug_fix` Spec gate mode changes from `human` to `auto`.
- P56 Decision 4: One reviewer for bugs, not a panel — enabled by the auto-generated spec.

## Overview

Bugs currently have no canonical document paths and no mechanism to generate a specification. The `bug_fix` tier has `Spec: human` but bugs cannot satisfy this gate because there is no spec document path or auto-generation. This specification adds auto-generated specs from bug reports, a `fix_plan` field for inline dev-plan content, canonical document paths for bugs, and changes the `bug_fix` Spec gate to `auto`.

## Scope

**In scope:**
- Auto-generate a specification document from `observed`/`expected` on bug creation
- Add `fix_plan` field to `CreateBugInput` and the bug entity model
- Add canonical document path resolution for bug entities
- Change `bug_fix` tier `Spec` gate from `human` to `auto`
- Register the auto-generated spec and `fix_plan` (if present) as documents owned by the bug

**Out of scope:**
- Editing auto-generated specs after creation (deferred to Open Question 2)
- Generating specs for existing bugs (deferred to Open Question 5)
- The review report document path (defined in F1 gate enforcement, registered by the review process)
- Close-out verification (F4)

## Functional Requirements

### Pillar A — Auto-Generated Specification

**FR-101:** When a bug is created via `CreateBug`, the system MUST auto-generate a specification document at `work/bugs/<slug>/spec.md` with the following structure:

```markdown
# Bug Specification: <bug name>

## Observed Behaviour
<bug.observed>

## Expected Behaviour
<bug.expected>

## Severity
<bug.severity> | Priority: <bug.priority> | Type: <bug.type>
```

**FR-102:** The auto-generated spec MUST be registered as a document of type `specification`, owned by the bug entity, and auto-approved. The document title MUST be `"Bug Specification: <bug name>"`.

**FR-103:** If the directory `work/bugs/<slug>/` does not exist, it MUST be created before writing the spec file.

**FR-104:** If a spec file already exists at `work/bugs/<slug>/spec.md`, the system MUST NOT overwrite it. The existing document MUST be re-registered (idempotent).

**FR-105:** Auto-generation MUST happen after the bug entity is persisted to disk, so the bug ID and slug are available for path construction.

**FR-106:** If spec auto-generation fails (e.g., disk write error), the bug creation MUST still succeed. The failure MUST be logged and returned as a warning in the creation response.

**Acceptance criteria:**
- Creating a bug with `observed` and `expected` fields produces a spec file at `work/bugs/<slug>/spec.md`
- The spec document is registered with type `specification`, owner set to the bug ID, and status `approved`
- Creating a second bug with the same slug does not overwrite the existing spec
- Bug creation succeeds even if the spec file cannot be written (e.g., permission error)
- The spec file contains the bug's observed behaviour, expected behaviour, severity, priority, and type

### Pillar B — fix_plan Field

**FR-107:** `CreateBugInput` in `internal/service/entities.go` MUST gain a `FixPlan` field of type `string`.

**FR-108:** The bug entity model in `internal/model/entities.go` MUST gain a `FixPlan` field.

**FR-109:** When `FixPlan` is non-empty at bug creation time, the system MUST write the content to `work/bugs/<slug>/fix-plan.md` and register it as a document of type `dev-plan`, owned by the bug entity, and auto-approved.

**FR-110:** When `FixPlan` is empty, no `fix-plan.md` file is created and no dev-plan document is registered. This is not an error.

**FR-111:** The `fix_plan` field MUST be stored in the bug's state file (`.kbz/state/bugs/<id>.yaml`) alongside other bug fields.

**Acceptance criteria:**
- Creating a bug with `fix_plan: "Change the sort order in status.go"` produces a file at `work/bugs/<slug>/fix-plan.md` containing that text
- The dev-plan document is registered with type `dev-plan`, owner set to the bug ID, and status `approved`
- Creating a bug without `fix_plan` does not create a `fix-plan.md` file and does not error
- The `fix_plan` field appears in the bug's state YAML file

### Pillar C — Canonical Document Paths

**FR-112:** `CanonicalDocPath` in `internal/service/entities.go` MUST be extended to resolve paths for bug entities. For a bug with slug `<slug>`:

- `specification` → `work/bugs/<slug>/spec.md`
- `dev-plan` → `work/bugs/<slug>/fix-plan.md`
- `report` → `work/reviews/review-<bug-id>-<slug>.md`

**FR-113:** `CanonicalDocPath` MUST return an error for unsupported document types (e.g., `design` is not valid for bugs).

**FR-114:** The canonical path resolution MUST work for both `BUG-<TSID>` and `BUG-<TSID>-<slug>` ID formats.

**Acceptance criteria:**
- `CanonicalDocPath("specification", "BUG-01KR12RE970R8")` returns `"work/bugs/<slug>/spec.md"` (after resolving the bug's slug)
- `CanonicalDocPath("dev-plan", "BUG-01KR12RE970R8")` returns `"work/bugs/<slug>/fix-plan.md"`
- `CanonicalDocPath("report", "BUG-01KR12RE970R8")` returns `"work/reviews/review-BUG-01KR12RE970R8-<slug>.md"`
- `CanonicalDocPath("design", "BUG-...")` returns an error

### Pillar D — Tier Configuration Change

**FR-115:** In `DefaultFastTrackConfig()` in `internal/config/config.go`, the `TierBugFix` entry MUST change `Spec` from `string(GateModeHuman)` to `string(GateModeAuto)`.

**FR-116:** The `TierBugFix` entry after the change MUST be:

```go
TierBugFix: {
    Design:    string(GateModeHuman),
    Spec:      string(GateModeAuto),
    DevPlan:   string(GateModeAuto),
    Review:    string(GateModeAuto),
    MaxCycles: 2,
},
```

**FR-117:** Existing tests that assert `bug_fix` tier has `Spec: human` MUST be updated to expect `Spec: auto`.

**Acceptance criteria:**
- `DefaultFastTrackConfig().Tiers["bug_fix"].Spec` equals `"auto"`
- The `TestFastTrack_TierMatrix_BugFixTierHumanSpec` test in `fast_track_integration_test.go` is updated to expect `Spec: auto` and renamed accordingly
- All fast-track integration tests pass

## Non-Functional Requirements

**NFR-101:** Spec auto-generation MUST complete within 100ms of bug creation (dominated by file I/O).

**NFR-102:** The auto-generated spec and fix-plan files MUST use UTF-8 encoding with LF line endings.

**NFR-103:** Bug creation with spec generation MUST NOT introduce a circular dependency: the bug must be persisted before the spec document references the bug ID.

## Acceptance Criteria (Cross-Cutting)

**AC-101:** End-to-end: create a bug with `observed`, `expected`, and `fix_plan` — a spec file and fix-plan file are created, registered, and approved. Both documents are owned by the bug. The `bug_fix` tier's Spec gate is `auto`.

**AC-102:** A reviewer can find the bug's spec by calling `doc(action: "get", id: "<bug-id>/spec-...")` and the fix-plan by calling `doc(action: "get", id: "<bug-id>/dev-plan-...")`.

**AC-103:** The canonical path tool returns correct paths for bug entities without requiring a feature parent.

## Dependencies and Assumptions

**Dependencies:**
- F1 (Bug Lifecycle Gate Enforcement) — The `needs-review → verifying` gate checks for a review report at the canonical path defined in FR-112.
- `internal/service/documents.go` — `SubmitDocument` is called to register the auto-generated spec and fix-plan.

**Assumptions:**
1. The bug entity has been persisted and has a valid `ID` and `Slug` before spec generation runs.
2. The `work/bugs/` directory tree is writable by the process.
3. The `auto_approve` flag on `SubmitDocument` works for `specification` type documents. Currently it does not (spec is in `gatedDocTypes`). This spec requires lifting that restriction for auto-generated bug specs, or using a separate approval path that bypasses the concept-tagging gate for auto-generated documents.
