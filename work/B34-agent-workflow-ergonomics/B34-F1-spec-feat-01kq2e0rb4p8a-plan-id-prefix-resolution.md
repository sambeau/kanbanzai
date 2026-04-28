# Specification: Plan ID Prefix Resolution

| Field   | Value                                           |
|---------|-------------------------------------------------|
| Date    | 2026-04-25                                      |
| Status  | Draft                                           |
| Author  | Spec Author                                     |
| Feature | FEAT-01KQ2E0RB4P8A                              |
| Plan    | P34-agent-workflow-ergonomics                   |

---

## 1. Related Work

**Design:** `work/design/design-p34-agent-workflow-ergonomics.md`
(`P34-agent-workflow-ergonomics/design-design-p34-agent-workflow-ergonomics`) —
§1 · Plan prefix shorthand resolution defines the predicate, service method, and
tool integration specified here.

**Prior decisions and designs consulted:**

| Document | Relevance to this specification |
|----------|---------------------------------|
| P29 — State Store Read Path Performance | `ResolvePlanByNumber` delegates to `ListPlanIDs`, which is cache-backed and O(1) after P29. The design notes the method degrades gracefully to O(n) without P29. |
| P7 — Developer Experience (`server_info`) | Established the project pattern of purely additive quality-of-life improvements to existing tools with no structural change. This specification follows the same pattern. |

The design's Related Work section attests: "No directly related prior work was
found on plan ID prefix shorthand resolution." No prior specification covers this
topic.

---

## 2. Overview

This specification covers short plan reference syntax (`P30`, `M5`) in the
`entity` and `status` MCP tools. A new predicate `ParseShortPlanRef` detects
short-form plan references. A new service method `ResolvePlanByNumber` resolves a
detected short reference to the canonical full plan ID by validating the prefix
against the project config registry and scanning the plan store. Both tools
perform this pre-resolution before dispatching to the service layer. All existing
full-ID code paths are unchanged.

---

## 3. Scope

**In scope:**

- A new `model.ParseShortPlanRef` predicate that detects and extracts the prefix
  and number from a short plan reference.
- A new `EntityService.ResolvePlanByNumber` service method that resolves a short
  plan reference to a full canonical plan ID.
- Integration of short-ref pre-resolution into the `entity` and `status` MCP tool
  handlers at the ID normalisation step.

**Out of scope:**

- Changes to `IsPlanID`, `ParsePlanID`, `ValidatePrefix`, or any other existing
  model predicate or function.
- Short-form resolution for feature IDs (ULIDs; the existing `ResolvePrefix`
  mechanism already handles ULID prefix matching and is unchanged).
- Any change to the canonical plan ID format or storage format.
- Short-ref resolution in any tool other than `entity` and `status`.
- Support for short-form references in human-facing CLI output or documentation.

---

## 4. Functional Requirements

### Model predicate

**FR-001** — The system MUST expose a new function
`model.ParseShortPlanRef(s string) (prefix, number string, ok bool)` that returns
`ok = true` when `s` consists of exactly one non-digit Unicode rune followed by
one or more ASCII digit characters and nothing else.

**FR-002** — `ParseShortPlanRef` MUST return `ok = false` for any input that
contains a hyphen, contains trailing non-digit characters after the number,
contains no leading non-digit rune, or is the empty string. Examples that MUST
return `ok = false`: `"P30-slug"`, `"30"`, `""`, `"P30X"`, `"P"`.

**FR-003** — `ParseShortPlanRef` MUST treat any single non-digit Unicode rune as a
valid prefix character, consistent with the existing `ValidatePrefix` semantics.
It MUST NOT restrict the prefix to uppercase ASCII letters only.

**FR-004** — `ParseShortPlanRef` MUST be a pure lexical function with no file
system or network I/O.

### Service method

**FR-005** — The system MUST expose a new method
`EntityService.ResolvePlanByNumber(cfg config.Config, prefix, number string) (id, slug string, err error)`.

**FR-006** — `ResolvePlanByNumber` MUST call `cfg.IsActivePrefix(prefix)` as its
first step. When `IsActivePrefix` returns false, the method MUST return an error
whose message includes the unknown prefix and lists the currently active prefixes,
for example: `"unknown plan prefix %q — valid prefixes are: [P, M]"`.

**FR-007** — When the prefix is recognised, `ResolvePlanByNumber` MUST scan the
plan store via `ListPlanIDs` and return the `id` and `slug` of the plan whose
`ParsePlanID` decomposition matches the given `prefix` and `number`.

**FR-008** — `ResolvePlanByNumber` MUST return a non-nil error when no plan in the
store matches the given prefix and number.

### Tool integration

**FR-009** — The `entity` MCP tool handler MUST, before dispatching to the service
layer, apply the following pre-resolution step to any input ID: call
`ParseShortPlanRef`; when it returns `ok = true`, call `ResolvePlanByNumber`
(passing the loaded project config); substitute the returned full canonical plan ID
for the original input before proceeding. This step MUST occur at the same point
in the handler where `ResolvePrefix` is currently applied for ULID-style inputs.

**FR-010** — The `status` MCP tool handler MUST apply the identical short-ref
pre-resolution step as described in FR-009.

**FR-011** — When `ParseShortPlanRef` returns `ok = false` for an input, the
pre-resolution step MUST be skipped and the input passed through unchanged. No
existing full-ID code path in either tool may be altered.

**FR-012** — When `ResolvePlanByNumber` returns an error, the `entity` or `status`
tool MUST surface that error to the caller and MUST NOT attempt to dispatch the
original unresolved short reference to the service layer.

---

## 5. Non-Functional Requirements

**NFR-001** — `ParseShortPlanRef` MUST execute entirely in memory with no I/O. It
is called synchronously in the tool handler and MUST NOT add measurable latency.

**NFR-002** — `ResolvePlanByNumber` MUST delegate plan listing to the
cache-backed `ListPlanIDs` operation from P29. It MUST NOT introduce a new O(n)
direct file-scan code path.

**NFR-003** — The change MUST be purely additive with respect to the existing model
package, service layer, and tool handlers. No existing exported function signature,
no existing test, and no existing caller of `entity` or `status` may require
modification to accommodate this feature.

---

## 6. Acceptance Criteria

**AC-001 (FR-009)** — Given a plan exists with full canonical ID
`P30-handoff-skill-assembly-prompt-hygiene`, when
`entity(action: "get", id: "P30")` is called, then the tool returns the plan's
details without error, identical to calling with the full ID.

**AC-002 (FR-010)** — Given the same plan, when `status(id: "P30")` is called,
then the tool returns the plan's status dashboard without error.

**AC-003 (FR-006)** — Given no plan prefix `X` is registered in the project config,
when `entity(action: "get", id: "X30")` is called, then the tool returns an error
whose message contains `"unknown plan prefix"` and names at least one valid prefix.

**AC-004 (FR-011)** — Given a plan exists with full canonical ID
`P30-handoff-skill-assembly-prompt-hygiene`, when
`entity(action: "get", id: "P30-handoff-skill-assembly-prompt-hygiene")` is called,
then the tool behaves exactly as it did before this change.

**AC-005 (FR-011)** — Given a feature with ULID-based ID `FEAT-01KQxxxxxx`, when
`entity(action: "get", id: "FEAT-01KQxxxxxx")` is called, then the tool behaves
exactly as before (FEAT ID resolution is unaffected).

**AC-006 (FR-001, FR-004)** — Unit test: `ParseShortPlanRef("P30")` returns
`prefix="P"`, `number="30"`, `ok=true`.

**AC-007 (FR-002)** — Unit test: `ParseShortPlanRef("P30-foo")` returns `ok=false`.

**AC-008 (FR-002)** — Unit test: `ParseShortPlanRef("30")` returns `ok=false`.

**AC-009 (FR-002)** — Unit test: `ParseShortPlanRef("")` returns `ok=false`.

**AC-010 (FR-003)** — Unit test: `ParseShortPlanRef("ñ5")` returns
`prefix="ñ"`, `number="5"`, `ok=true`, confirming non-ASCII prefix support.

**AC-011 (FR-008)** — Given a valid prefix `P` is registered but no plan numbered
`99` exists, when `ResolvePlanByNumber` is called with `prefix="P"`, `number="99"`,
then it returns a non-nil error.

---

## 7. Dependencies and Assumptions

**DEP-001** — `config.Config.IsActivePrefix` must accept a single-rune string as
the prefix argument; `ParseShortPlanRef` extracts exactly one rune.

**DEP-002** — `ListPlanIDs` must be available via the cache-backed path introduced
in P29. `ResolvePlanByNumber` delegates to this operation and inherits its
performance characteristics.

**DEP-003** — Both the `entity` and `status` tool handlers already load the project
config for other purposes. Passing the loaded config to `ResolvePlanByNumber` does
not introduce a new config-loading dependency.

**ASM-001** — Sequential plan numbers within a given prefix are unique. Two plans
with the same prefix and number cannot exist in the same project, so
`ResolvePlanByNumber` cannot encounter an ambiguous match.

**ASM-002** — A plan's prefix and number are encoded in its full canonical ID and
are recoverable via `ParsePlanID`. No additional metadata field is required.