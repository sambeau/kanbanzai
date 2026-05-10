# P62-F3 Review Report — Runtime Discovery Surfaces

| Field    | Value                                                |
|----------|------------------------------------------------------|
| Feature  | FEAT-01KR7BKXMK3B6 (runtime-surfaces)               |
| Batch    | B64-install-skill-quality                            |
| Plan     | P62-install-skill-quality-remediation                |
| Reviewer | reviewer-conformance                                 |
| Verdict  | **APPROVED**                                         |
| Date     | 2026-05-10                                           |

## Scope

This is a **conformance review** — verifying the implementation against the
specification's acceptance criteria (AC-001 through AC-010) and the five
verification points specified for this review.

The implementation consists of 8 commits across 20 files (+1006/−3 lines):

| # | Commit | Task |
|---|--------|------|
| 1 | `b29772be` | T1: Embed F3 source files into binary |
| 2 | `2db6b748` | T2: Add F3 Artifact entries to Manifest |
| 3 | `1b771222` | T3: Install CLAUDE.md and OPENAI.md |
| 4 | `e529c200` | T4: Install .claude/skills/ wrappers |
| 5 | `aa6c45eb` | T5: Conditional cursorRule and --enable-cursor |
| 6 | `df224efd` | T6: Wire --skip-instructions suppression |
| 7 | `d193319b` | T7: Extend TestEmbeddedCorpus |
| 8 | `14af50d7` | T8: Integration/e2e tests for AC-001 through AC-009 |

## Verification Points

### 1. CLAUDE.md and OPENAI.md installed unconditionally ✅

**Evidence:** `internal/kbzinit/init.go` lines in `runNewProject` and
`runExistingProject` call `installClaudeMd(baseDir)` and
`installOpenAiRedirect(baseDir)` within the `!opts.SkipAgentsMD` block.
These calls have no other conditional gates.

`claude_md.go` and `openai_redirect.go` each embed their source, resolve
the artifact from Manifest via `manifestByKind()`, and delegate to
`installArtifact()` with the standard `MarkerSpec` comparator.

Both embedded files carry `<!-- kanbanzai-managed: v3 -->` on line 1
and reference `AGENTS.md`, satisfying REQ-001 and REQ-002.

### 2. .claude/skills/ wrappers installed ✅

**Evidence:** `claude_skills.go` iterates over all `ClaudeWrapper` entries
in the Manifest (7 wrappers), reads embedded content, applies
`transformSkillContent()`, and installs via `installArtifact()`.

Manifest entries (confirmed in `manifest.go`):
- `kanbanzai-getting-started` (`.claude/skills/kanbanzai-getting-started/SKILL.md`)
- `kanbanzai-workflow` (`.claude/skills/kanbanzai-workflow/SKILL.md`)
- `write-design` (`.claude/skills/write-design/SKILL.md`)
- `implement-task` (`.claude/skills/implement-task/SKILL.md`)
- `orchestrate-development` (`.claude/skills/orchestrate-development/SKILL.md`)
- `review-code` (`.claude/skills/review-code/SKILL.md`)
- `write-spec` (`.claude/skills/write-spec/SKILL.md`)

Every wrapper contains `# kanbanzai-managed:` and `# kanbanzai-version:`
in YAML frontmatter, satisfying REQ-003.

REQ-004 naming convention: `kanbanzai-` prefix is used for wrappers targeting
`.agents/skills/kanbanzai-*/` (getting-started, workflow); bare names are used
for `.kbz/skills/<name>/` targets (write-design, implement-task, etc.). Fully
consistent with the Manifest's `InstallPath` field.

### 3. .cursor/rules/kanbanzai.mdc conditional on --enable-cursor ✅

**Evidence:** `cursor_rule.go` implements a dual-condition gate (REQ-005/006):
- Condition (a): `.cursor/` directory exists on disk → install
- Condition (b): `enableCursor` parameter is `true` → install
- Neither holds → silent skip (`return nil`)

`cmd/kbz/init_cmd.go` parses `--enable-cursor` and sets `opts.EnableCursor = true`.
The flag is documented in `initUsageText`.

The embedded `cursor_rules/kanbanzai.mdc` has valid MDC frontmatter with
`description:` field, satisfying the Cursor MDC format constraint.

### 4. --skip-agents-md (--skip-instructions) suppresses F3 surfaces ✅

**Evidence:** In both `runNewProject` and `runExistingProject`, all four F3
install functions (`installClaudeMd`, `installOpenAiRedirect`,
`installClaudeWrappers`, `installCursorRule`) are nested within:
```go
if !opts.SkipAgentsMD {
    // ... AGENTS.md, copilot-instructions ...
    // F3: runtime discovery surfaces — suppressed by --skip-agents-md / --skip-instructions.
    if err := i.installClaudeMd(baseDir); err != nil { ... }
    if err := i.installOpenAiRedirect(baseDir); err != nil { ... }
    if err := i.installClaudeWrappers(baseDir); err != nil { ... }
    if err := i.installCursorRule(baseDir, opts.EnableCursor); err != nil { ... }
}
```

No per-runtime skip flags exist, matching the constraint.

### 5. TestEmbeddedCorpus extended, e2e tests for AC-001 through AC-009 ✅

**Evidence:** All tests pass (20/20, 0.756s).

`corpus_test.go` adds `testF3SurfaceContent` to `TestEmbeddedCorpus` verifying:
- CLAUDE.md references AGENTS.md
- OPENAI.md references AGENTS.md
- `.cursor/rules/kanbanzai.mdc` has `description:` frontmatter
- All ClaudeWrapper entries have `# kanbanzai-managed:` markers

Plus negative fixtures for AC-010 (missing Manifest entries detectable).

`f3_e2e_test.go` has dedicated tests for every acceptance criterion:

| Test | Criteria | Status |
|------|----------|--------|
| `TestF3_AC001_ClaudeMdCreatedWithMarker` | AC-001 | PASS |
| `TestF3_AC002_OpenAiRedirectCreated` | AC-002 | PASS |
| `TestF3_AC003_ClaudeWrappersInstalled` | AC-003 | PASS |
| `TestF3_AC004_DirectoryNamingConvention` | AC-004 | PASS |
| `TestF3_AC005_NoCursorDirWithoutFlag` | AC-005 | PASS |
| `TestF3_AC006_PreCreatedCursorDirInstallsRule` | AC-006 | PASS |
| `TestF3_AC007_EnableCursorFlagCreatesDir` | AC-007 | PASS |
| `TestF3_AC008_UnmanagedCLAUDE_PreservedWithWarning` | AC-008 | PASS |
| `TestF3_AC009_NewerMarkerNoOp` | AC-009 | PASS |
| `TestEmbeddedCorpus_F3ManifestCompleteness` et al. | AC-010 | PASS |

## Conformance Matrix

| Criterion | Requirement | Verdict | Evidence |
|-----------|-------------|---------|----------|
| AC-001 | REQ-001 | ✅ PASS | `TestF3_AC001_ClaudeMdCreatedWithMarker` |
| AC-002 | REQ-002 | ✅ PASS | `TestF3_AC002_OpenAiRedirectCreated` |
| AC-003 | REQ-003 | ✅ PASS | `TestF3_AC003_ClaudeWrappersInstalled` |
| AC-004 | REQ-004 | ✅ PASS | `TestF3_AC004_DirectoryNamingConvention` |
| AC-005 | REQ-005a | ✅ PASS | `TestF3_AC005_NoCursorDirWithoutFlag` |
| AC-006 | REQ-005b | ✅ PASS | `TestF3_AC006_PreCreatedCursorDirInstallsRule` |
| AC-007 | REQ-006 | ✅ PASS | `TestF3_AC007_EnableCursorFlagCreatesDir` |
| AC-008 | REQ-008 | ✅ PASS | `TestF3_AC008_UnmanagedCLAUDE_PreservedWithWarning` |
| AC-009 | REQ-008 | ✅ PASS | `TestF3_AC009_NewerMarkerNoOp` |
| AC-010 | REQ-007 | ✅ PASS | `TestEmbeddedCorpus_F3ManifestCompleteness` |
| REQ-NF-002 | Size <50KB | ✅ PASS | Embedded surfaces are ~200 lines total, well under limit |

## Findings

None. The implementation is fully conformant with the specification. All ten
acceptance criteria are satisfied with automated test coverage. The Manifest
is the single authority for artifact registration. All artifacts use the
`MarkerSpec` comparator from F2. No unmanaged-file-skip violations exist.

## Recommendations

- **Spec-gap observation (non-blocking):** The spec mentions `--skip-instructions`
  as the renamed flag from F4, but the current flag is `--skip-agents-md`. The
  code comments reference both names (`--skip-agents-md / --skip-instructions`)
  which provides clear transition documentation. This is informational only;
  F4 will handle the rename.
