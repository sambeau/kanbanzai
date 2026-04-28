# Implementation Plan: Context Assembly Pipeline

| Field   | Value                                                                 |
|---------|-----------------------------------------------------------------------|
| Feature | FEAT-01KN5-88PE43M6 (context-assembly-pipeline)                      |
| Spec    | `work/spec/3.0-context-assembly-pipeline.md`                          |
| Design  | `work/design/skills-system-redesign-v2.md` §3.4, §6.1, §6.2, §11    |

---

## 1. Overview

This plan decomposes the Context Assembly Pipeline specification into assignable tasks for AI agents. The pipeline is the integration feature that ties roles, skills, stage bindings, and knowledge auto-surfacing together into a single 10-step assembly path invoked by the `handoff` tool.

The existing codebase has:
- `internal/context/assemble.go` — current assembly logic (profile + knowledge tiers + design context + task)
- `internal/context/profile.go` — profile loading from `.kbz/context/roles/` with YAML parsing
- `internal/context/resolve.go` — single-parent inheritance chain resolution with leaf-wins semantics
- `internal/mcp/handoff_tool.go` — MCP tool definition, status validation, prompt rendering
- `internal/service/` — EntityService, KnowledgeService, IntelligenceService, DocumentService

The pipeline replaces the current assembly logic with a 10-step orchestrated pipeline that reads from role files (`.kbz/roles/*.yaml`), skill files (`.kbz/skills/*/SKILL.md`), and a stage binding registry, then outputs attention-curve-ordered Markdown with token budget management.

**Scope boundaries (from spec §2.2 — out of scope):**
- Hard tool filtering (dynamically scoping MCP tool list per session)
- Role/skill/binding schema definitions (specified in their own features)
- Knowledge auto-surfacing matching/cap logic (separate feature; this pipeline calls it at step 7)
- Freshness tracking metadata (separate feature)
- Feature lifecycle state machine changes

---

## 2. Task Breakdown

### Task 1: Pipeline Data Types and Step Interfaces

**Objective:** Define the core data types and step interfaces for the 10-step pipeline. Each step is independently testable behind an interface so that downstream tasks can mock any step.

**Specification references:** FR-001, FR-016, FR-018, NFR-005

**Input context:**
- `internal/context/assemble.go` — existing `AssemblyInput`, `AssemblyResult`, `AssemblyItem` types
- `internal/context/profile.go` — existing `Profile`, `ResolvedProfile`, `ProfileStore` types
- `internal/context/resolve.go` — existing `ResolveProfile`, `ResolveChain` functions
- Spec §3 FR-016 — the 10-position attention-curve table
- Spec §3 FR-018 — four progressive disclosure layers

**Output artifacts:**
- New file `internal/context/pipeline.go` — pipeline orchestrator type, step interfaces, shared data types
- New file `internal/context/pipeline_test.go` — unit tests for pipeline orchestration with mock steps
- New type `PipelineInput` (extends `AssemblyInput` with stage binding, role, skill references)
- New type `PipelineResult` (extends `AssemblyResult` with section-ordered output, token estimate, metadata warnings)
- Interfaces: `RoleLoader`, `SkillLoader`, `BindingLookup`, `KnowledgeSurfacer`

**Dependencies:** None — this is the foundational task.

**Interface contracts (shared with all other tasks):**

```go
// RoleLoader loads and resolves roles with inheritance.
type RoleLoader interface {
    LoadRole(id string) (*Role, error)
    ResolveRole(id string) (*ResolvedRole, error)
}

// Role is the loaded role from .kbz/roles/*.yaml.
// Fields align with the Role System feature's schema.
type Role struct {
    ID           string   `yaml:"id"`
    Inherits     string   `yaml:"inherits,omitempty"`
    Identity     string   `yaml:"identity"`
    Vocabulary   []string `yaml:"vocabulary,omitempty"`
    AntiPatterns []string `yaml:"anti_patterns,omitempty"`
    Tools        []string `yaml:"tools,omitempty"`
    Tags         []string `yaml:"tags,omitempty"`
    LastVerified string   `yaml:"last_verified,omitempty"`
}

// ResolvedRole is the result of inheritance resolution.
type ResolvedRole struct {
    ID           string
    Identity     string
    Vocabulary   []string // parent first, then child (concatenated)
    AntiPatterns []string // parent first, then child (concatenated)
    Tools        []string
    Tags         []string
}

// SkillLoader loads skills from .kbz/skills/*/SKILL.md.
type SkillLoader interface {
    LoadSkill(name string) (*Skill, error)
}

// Skill is the parsed skill from a SKILL.md file.
type Skill struct {
    Name               string
    Vocabulary         []string
    AntiPatterns       []string
    Procedure          string   // numbered steps markdown
    OutputFormat       string
    Examples           string
    EvaluationCriteria string
    RetrievalAnchors   []string
    LastVerified       string
}

// BindingLookup retrieves stage bindings from the binding registry.
type BindingLookup interface {
    LookupBinding(stage string) (*StageBinding, error)
}

// StageBinding maps a lifecycle stage to its context assembly configuration.
type StageBinding struct {
    Stage              string
    RoleID             string
    SkillName          string
    Pattern            string   // e.g. "single-agent", "orchestrator-workers"
    EffortBudget       string
    MaxReviewCycles    int
    Prerequisites      []string
    IncludeCategories  []string // content categories to include
    ExcludeCategories  []string // content categories to exclude
}

// KnowledgeSurfacer retrieves relevant knowledge entries for a task context.
// Implemented by the Knowledge Auto-Surfacing feature.
type KnowledgeSurfacer interface {
    Surface(ctx KnowledgeSurfaceInput) (*KnowledgeSurfaceResult, error)
}

type KnowledgeSurfaceInput struct {
    FilePaths []string
    RoleTags  []string
}

type KnowledgeSurfaceResult struct {
    Entries  []SurfacedEntry
    Excluded []ExcludedEntry // for diagnostic logging
}

type SurfacedEntry struct {
    ID      string
    Content string  // formatted "Always/Never X BECAUSE Y"
    Score   float64 // recency-weighted confidence
}

type ExcludedEntry struct {
    ID    string
    Topic string
}

// PipelineSection represents one section in the attention-curve-ordered output.
type PipelineSection struct {
    Position int    // 1–10
    Label    string // e.g. "identity", "vocabulary", "knowledge"
    Content  string
    Layer    int    // progressive disclosure layer (1–4)
    Tokens   int    // estimated token count
}

// PipelineResult is the output of the 10-step pipeline.
type PipelineResult struct {
    Sections       []PipelineSection
    TotalTokens    int
    TokenWarning   string // non-empty when > 40% of context window
    MetadataWarnings []string // e.g. staleness warnings from Freshness Tracking
    Diagnostics    map[string]any // excluded knowledge entries, etc.
}
```

---

### Task 2: Pipeline Steps 0–4 — Validation, Resolution, and Orchestration Metadata

**Objective:** Implement the first five pipeline steps: lifecycle state validation (step 0), task-to-stage resolution (step 1), stage binding lookup (step 2), stage-specific inclusion/exclusion (step 3), and orchestration metadata extraction (step 4). These steps transform a task ID into a validated context with binding and orchestration metadata ready for role/skill loading.

**Specification references:** FR-002, FR-003, FR-004, FR-005, FR-006, NFR-004

**Input context:**
- `internal/context/pipeline.go` (from Task 1) — `PipelineInput`, `BindingLookup`, `StageBinding` types
- `internal/service/entities.go` — `EntityService.Get()` for task and feature lookups
- `internal/validate/` — existing lifecycle state machines
- `internal/mcp/handoff_tool.go` — existing status validation logic (lines 97–110) as reference for accepted statuses
- Spec §3 FR-002 through FR-006

**Output artifacts:**
- New file `internal/context/steps_early.go` — functions for steps 0–4
- New file `internal/context/steps_early_test.go` — unit tests for each step in isolation
- Each step function takes its inputs and returns either the enriched pipeline state or a structured error (per NFR-004: step name + entity ID + remediation hint)

**Dependencies:** Task 1 (data types and interfaces)

**Interface contract with Task 4 (handoff wiring):** Each step function must have the signature pattern:

```go
func stepValidateLifecycle(featureState map[string]any) error
func stepResolveStage(taskState, featureState map[string]any) (string, error)
func stepLookupBinding(lookup BindingLookup, stage string) (*StageBinding, error)
func stepApplyInclusion(binding *StageBinding) *InclusionStrategy
func stepExtractOrchestration(binding *StageBinding) *OrchestrationMetadata
```

---

### Task 3: Pipeline Steps 5–6 and Merging — Role Resolution, Skill Loading, Vocabulary and Anti-Pattern Merging

**Objective:** Implement role resolution with inheritance using concatenation semantics (step 5), skill loading (step 6), vocabulary merging (FR-009), and anti-pattern merging (FR-010). The key difference from the existing `resolve.go` is that the new role system uses **concatenation** for vocabulary and anti-patterns (parent first, child appended) rather than the existing leaf-replaces-parent semantics.

**Specification references:** FR-007, FR-008, FR-009, FR-010, FR-017

**Input context:**
- `internal/context/pipeline.go` (from Task 1) — `RoleLoader`, `SkillLoader`, `ResolvedRole`, `Skill` types
- `internal/context/resolve.go` — existing inheritance resolution (leaf-replaces semantics; the new pipeline needs concatenation semantics for vocabulary/anti-patterns but identity-override for scalars)
- Spec §3 FR-007 — inheritance: parent vocab first, child appended; child identity overrides parent
- Spec §3 FR-009 — merged vocabulary: role terms before skill terms, no deduplication
- Spec §3 FR-010 — merged anti-patterns: role items before skill items
- Spec §3 FR-017 — within-section ordering: most critical item LAST (recency bias)

**Output artifacts:**
- New file `internal/context/steps_content.go` — role resolution, skill loading, vocabulary merging, anti-pattern merging
- New file `internal/context/steps_content_test.go` — unit tests including:
  - Inherited role produces parent vocab then child vocab
  - Merged vocabulary has role terms then skill terms (4-way concat: grandparent → parent → child role → skill)
  - Duplicate terms preserved (no dedup)
  - Child identity overrides parent identity
  - Missing role/skill returns error with entity name
  - Within-section ordering: most critical item last

**Dependencies:** Task 1 (data types and interfaces)

**Interface contract with Task 2:** This task does not depend on Task 2's step functions, but the pipeline orchestrator (Task 1) calls steps 0–4 before steps 5–6. The step functions here receive the `StageBinding` output from Task 2's `stepLookupBinding`.

**Interface contract with Task 4:** Merged vocabulary and anti-patterns are returned as `[]string` slices ready for rendering into `PipelineSection` structs:

```go
func stepResolveRole(loader RoleLoader, roleID string) (*ResolvedRole, error)
func stepLoadSkill(loader SkillLoader, skillName string) (*Skill, error)
func mergeVocabulary(role *ResolvedRole, skill *Skill) []string
func mergeAntiPatterns(role *ResolvedRole, skill *Skill) []string
func orderByCriticality(items []string) []string // most critical last
```

---

### Task 4: Pipeline Steps 8–10 — Tool Guidance, Token Budget, and Output Assembly

**Objective:** Implement tool subset guidance generation (step 8), token budget estimation with warning/refusal (step 9), progressive disclosure layers (FR-018), and the final attention-curve output ordering (step 10). This task produces the final `PipelineResult` with sections in the correct 10-position order.

**Specification references:** FR-012, FR-013, FR-014, FR-015, FR-016, FR-017, FR-018, NFR-001, NFR-002

**Input context:**
- `internal/context/pipeline.go` (from Task 1) — `PipelineSection`, `PipelineResult` types
- Spec §3 FR-012 — tool guidance from role's `tools` field (soft filtering, no hard removal)
- Spec §3 FR-013 — token estimation: sum all sections, character count / 4 approximation
- Spec §3 FR-014 — warning at 40% of context window
- Spec §3 FR-015 — refusal at 60% of context window
- Spec §3 FR-016 — 10-position attention-curve table
- Spec §3 FR-018 — four progressive disclosure layers with budget-aware loading
- `internal/config/config.go` — for potential context window size configuration

**Output artifacts:**
- New file `internal/context/steps_output.go` — tool guidance, token budget, output ordering
- New file `internal/context/steps_output_test.go` — unit tests:
  - Tool guidance lists role tools; no tool guidance when tools field empty
  - Token estimate is positive integer for any valid assembly
  - Warning emitted when > 40% threshold
  - Refusal (error, not context) when > 60% threshold
  - Output sections in correct 10-position order
  - Optional sections omitted without affecting order
  - Layer 1 always present; Layer 3 absent unless skill procedure requests it
  - Deterministic output (NFR-002): identical inputs → identical output
- Benchmark test for assembly latency (NFR-001: < 2 seconds)

**Dependencies:** Task 1 (data types), Task 2 (steps 0–4 produce orchestration metadata for position 3), Task 3 (steps 5–6 produce vocabulary/anti-patterns for positions 4–5)

**Interface contract:** The output assembler function:

```go
// DefaultContextWindowTokens is the assumed context window size if not configured.
const DefaultContextWindowTokens = 200000

// estimateTokens approximates token count from text (chars / 4, ±10% acceptable).
func estimateTokens(text string) int

// assembleOutput orders sections by the 10-position attention curve and applies
// progressive disclosure layer budgeting.
func assembleOutput(sections []PipelineSection, contextWindowTokens int) (*PipelineResult, error)
```

---

### Task 5: Handoff Tool Integration and Backward Compatibility

**Objective:** Wire the new 10-step pipeline into the existing `handoff` MCP tool, preserving the existing tool contract (`task_id`, `role`, `instructions` parameters). When a stage binding exists for the resolved stage, use the new pipeline. When no binding exists, fall back to the current assembly path. Update the prompt renderer to output attention-curve-ordered sections.

**Specification references:** FR-001, FR-011, NFR-003, NFR-004, NFR-005

**Input context:**
- `internal/mcp/handoff_tool.go` — existing tool definition, handler, and prompt renderer
- `internal/context/assemble.go` — existing `Assemble()` function (the fallback path)
- `internal/context/pipeline.go` (from Task 1) — pipeline orchestrator
- `internal/context/steps_early.go` (from Task 2) — steps 0–4
- `internal/context/steps_content.go` (from Task 3) — steps 5–6
- `internal/context/steps_output.go` (from Task 4) — steps 8–10
- Spec §3 FR-001 — `handoff(task_id=...)` returns structured Markdown via the pipeline
- Spec §3 FR-011 — step 7 invokes `KnowledgeSurfacer` interface (injected; no-op stub until Knowledge Auto-Surfacing feature lands)
- Spec §6 NFR-003 — existing `handoff` calls continue to work unchanged

**Output artifacts:**
- Modified `internal/mcp/handoff_tool.go` — wire pipeline into handler; fallback to `Assemble()` when no binding
- New file `internal/context/pipeline_run.go` — `RunPipeline()` function that executes all 10 steps in sequence
- New file `internal/context/pipeline_run_test.go` — integration test: full pipeline with filesystem-backed test fixtures
- New file `internal/context/surfacer_stub.go` — no-op `KnowledgeSurfacer` implementation (returns empty result) for use until the Knowledge Auto-Surfacing feature is implemented
- Modified `internal/mcp/handoff_tool.go` tests — verify backward compatibility: existing calls produce output without error

**Dependencies:** Task 2, Task 3, Task 4 (all pipeline steps must be implemented)

---

## 3. Dependency Graph

```
Task 1: Pipeline Data Types and Step Interfaces
  ├──→ Task 2: Steps 0–4 (Validation, Resolution, Orchestration)
  ├──→ Task 3: Steps 5–6 (Role, Skill, Merging)
  └──→ Task 4: Steps 8–10 (Tool Guidance, Token Budget, Output)
            │
            ▼
       Task 5: Handoff Tool Integration
       (depends on Tasks 2, 3, 4)
```

**Parallelism:** Tasks 2, 3, and 4 can execute in parallel after Task 1 completes. They share the interface contracts from Task 1 but do not depend on each other's implementations. Task 5 is the serial integration point.

**Execution order:**
1. Task 1 (serial — foundational types)
2. Tasks 2, 3, 4 (parallel — independent step implementations)
3. Task 5 (serial — integration)

---

## 4. Interface Contracts

### 4.1 Pipeline Orchestrator (Task 1 → all tasks)

All step functions operate on a shared `PipelineState` struct that accumulates results as steps execute:

```go
type PipelineState struct {
    Input              PipelineInput
    TaskState          map[string]any
    FeatureState       map[string]any
    Stage              string           // from step 1
    Binding            *StageBinding    // from step 2
    Inclusion          *InclusionStrategy // from step 3
    Orchestration      *OrchestrationMetadata // from step 4
    Role               *ResolvedRole    // from step 5
    Skill              *Skill           // from step 6
    MergedVocabulary   []string         // from step 5+6
    MergedAntiPatterns []string         // from step 5+6
    KnowledgeEntries   []SurfacedEntry  // from step 7
    ToolGuidance       string           // from step 8
    Sections           []PipelineSection // accumulated
    TokenEstimate      int              // from step 9
}
```

### 4.2 Knowledge Integration Point (Task 5 → Knowledge Auto-Surfacing feature)

Step 7 calls `KnowledgeSurfacer.Surface()` with the task's file paths and the resolved role's tags. Until the Knowledge Auto-Surfacing feature lands, Task 5 provides a stub that returns an empty result. The interface is defined in Task 1 and does not need to change when the real implementation arrives.

### 4.3 Freshness Metadata Hook (Task 5 → Freshness Tracking feature)

The pipeline result includes a `MetadataWarnings []string` field. The Freshness Tracking feature will populate this by checking `last_verified` on the loaded role and skill. Until that feature lands, the field remains empty. No stub is needed — the pipeline simply does not add warnings.

### 4.4 Backward-Compatible Handoff (Task 5)

The modified `handoff` handler must detect whether a binding exists:

```go
// Pseudocode for the handler decision:
// 1. Resolve task → feature → stage
// 2. Attempt binding lookup for stage
// 3. If binding found → run new pipeline
// 4. If no binding → fall back to existing Assemble()
```

---

## 5. Traceability Matrix

| Requirement | Task(s) |
|-------------|---------|
| FR-001 (Pipeline entry point) | Task 5 |
| FR-002 (Step 0 — lifecycle validation) | Task 2 |
| FR-003 (Step 1 — task-to-stage resolution) | Task 2 |
| FR-004 (Step 2 — stage binding lookup) | Task 2 |
| FR-005 (Step 3 — inclusion/exclusion) | Task 2 |
| FR-006 (Step 4 — orchestration metadata) | Task 2 |
| FR-007 (Step 5 — role resolution with inheritance) | Task 3 |
| FR-008 (Step 6 — skill loading) | Task 3 |
| FR-009 (Vocabulary merging) | Task 3 |
| FR-010 (Anti-pattern merging) | Task 3 |
| FR-011 (Step 7 — knowledge integration point) | Task 5 (stub) |
| FR-012 (Step 8 — tool subset guidance) | Task 4 |
| FR-013 (Step 9 — token budget estimation) | Task 4 |
| FR-014 (Token budget warning) | Task 4 |
| FR-015 (Token budget refusal) | Task 4 |
| FR-016 (Step 10 — attention-curve ordering) | Task 4 |
| FR-017 (Within-section recency bias) | Task 3, Task 4 |
| FR-018 (Progressive disclosure layers) | Task 1 (types), Task 4 (implementation) |
| NFR-001 (Assembly latency < 2s) | Task 4 (benchmark) |
| NFR-002 (Deterministic output) | Task 4 |
| NFR-003 (Backward compatibility) | Task 5 |
| NFR-004 (Error message quality) | Task 2, Task 3, Task 4 |
| NFR-005 (Testability — injectable interfaces) | Task 1 |