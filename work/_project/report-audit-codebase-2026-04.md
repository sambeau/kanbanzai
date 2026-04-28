# Quality Audit: Kanbanzai Codebase
Date: 2026-04-22
Scope: ./...
Index state: current (32,413 nodes, 12,371 edges)
Tools run: go vet, go test -race, knowledge graph (staticcheck: not installed — see static_analysis)

Overall: has_critical_findings (resolved during audit)

---

## Dimensions

### test_health: has_findings → clean (fixed during audit)

Summary: 9 tests failing pre-audit across internal/kbzinit; 0 failing post-fix.
Race detector: no races detected. All other packages: passing (cached).

Findings (all resolved):

- [critical] TestRun_NewProject_SkillFrontmatter: embedded skill files for
    `agents`, `getting-started`, and `workflow` used the old YAML-style managed
    marker (`metadata: kanbanzai-managed: "true"`) instead of the comment-style
    markers (`# kanbanzai-managed: true` / `# kanbanzai-version: dev`) expected
    by `transformSkillContent`.
    (location: internal/kbzinit/skills/{agents,getting-started,workflow}/SKILL.md)
    Resolution: replaced YAML metadata blocks with comment-style markers in all
    three source files. Tests now pass.

- [critical] TestRun_Idempotency, TestRun_Idempotency_Skills,
    TestRun_Skills_CurrentVersion_NoOp, TestRun_Skills_OlderVersion_Overwritten,
    TestRun_UpdateSkills_OnlySkills, TestRun_ReviewerRole_CurrentVersion_NoOp,
    TestInit_ManagedMcpJson_CurrentVersion_NoOp: idempotency checks failed because
    `hasLine(existing, "# kanbanzai-managed:")` returned false on installed skill
    files produced from the old-format source. The second `kbz init` run exited
    non-zero with "exists but is not managed by Kanbanzai".
    (location: internal/kbzinit/skills.go:78, root cause same as above)
    Resolution: same frontmatter fix resolved all seven tests.

- [critical] TestP12_Integration_NewProject (AC-C1): `getting-started` skill
    missing `edit_file` reference required by acceptance criterion AC-C1 (MCP
    write-tools rule).
    (location: internal/kbzinit/skills/getting-started/SKILL.md)
    Resolution: added "Direct File Writes Bypassing MCP Tools" anti-pattern
    section referencing `edit_file`.

- [critical] TestP12_Integration_NewProject (AC-C2): `workflow` skill emergency
    brake section missing `.kbz/state/` reference required by acceptance
    criterion AC-C2 (direct-write condition).
    (location: internal/kbzinit/skills/workflow/SKILL.md)
    Resolution: added direct-write bullet to the emergency brake section.

---

### static_analysis: has_findings

Summary: go vet — 0 findings. staticcheck — 260 findings total after install.
Triaged into: 3 production unused-code (U1000), 4 test unused-code (U1000),
6 code-quality improvements (S1005/S1016/S1017/S1039), and 245 systematic
capitalized-error-string violations (ST1005) across internal/mcp/.

Findings:

- [major] U1000: guidanceRules — var declared but never used in production
    decompose service. Non-trivial: the variable is populated with rule data
    that is silently discarded.
    (location: internal/service/decompose.go:566)
    Recommendation: either wire guidanceRules into the decompose logic or
    delete it and its population code.

- [minor] U1000: rebuildIndexUsageText — unexported const declared but never
    referenced in production code.
    (location: cmd/kanbanzai/rebuild.go:12)
    Recommendation: delete.

- [minor] U1000: path field — struct field declared but never read or written
    in production code.
    (location: internal/docint/parser.go:163)
    Recommendation: delete the field; verify no JSON/YAML deserialisation
    depends on it.

- [minor] S1039: unnecessary fmt.Sprintf — plain string literal passed to
    fmt.Sprintf with no format verbs.
    (locations: internal/mcp/handoff_tool.go:110,
     internal/service/gate_errors.go:81, :91, :110)
    Recommendation: replace with the string literal directly.

- [minor] S1017: conditional TrimPrefix/TrimSuffix should be unconditional —
    staticcheck recommends replacing `if strings.HasPrefix(s, p) { s =
    strings.TrimPrefix(s, p) }` with a single `strings.TrimPrefix` call.
    (locations: internal/service/decompose.go:1167, :1170)
    Recommendation: apply the simpler unconditional form.

- [minor] S1005: unnecessary blank identifier assignment.
    (location: internal/kbzinit/init.go:370)
    Recommendation: remove the assignment.

- [minor] S1016: should use type conversion instead of struct literal when
    converting AntiPattern to AntiPatternEntry.
    (location: internal/context/pipeline.go:384)
    Recommendation: replace struct literal with type conversion.

- [minor] U1000 (test helpers): four unused test helper functions.
    (locations: cmd/kanbanzai/main_test.go:666 captureStdout,
     internal/git/branch_test.go:39 addCommitsToMain,
     internal/mcp/doc_intel_tool_test.go:69 writeKnowledgeFile,
     internal/mcp/status_tool_test.go:89 callStatus)
    Recommendation: delete or wire up to a test case.

- [minor] ST1005 (systematic): 245 error strings start with a capital letter
    or end with punctuation across all internal/mcp/ tool files. Go convention
    (ST1005) requires lowercase, unpunctuated error strings. However, MCP tool
    error messages are user-facing responses returned directly to AI agents;
    capitalisation may be intentional for readability at that boundary.
    (location: internal/mcp/*.go — pervasive)
    Recommendation: decide once whether MCP error strings are "Go errors" (fix
    to lowercase) or "user-facing messages" (suppress ST1005 for internal/mcp/
    via a staticcheck config). Either way, make it consistent and document the
    decision. Do not fix piecemeal.

---

### structural_quality: has_findings

Summary: 341 zero-inbound-degree function candidates returned by graph query.
After verification with trace_call_path and USAGE-edge checks: 7 confirmed dead
code functions; 0 high fan-out SRP violations (2 incidental fan-out cases noted);
1 change-coupling pair in documentation (not source code, not a concern).

Findings:

- [major] dead_code: AllStates — exported function, zero callers confirmed,
    no USAGE edges. Returns all known states for an EntityKind by iterating
    entryStates, allowedTransitions, and terminalStates maps. Non-trivial (38
    lines, complexity 9). Was likely written in anticipation of schema validation
    that was not yet wired up.
    (location: internal/validate/lifecycle.go:413)
    Recommendation: either wire up a caller (e.g. health check or schema
    validator) or delete. If retained for future use, add a TODO comment
    explaining the intended caller.

- [major] dead_code: CheckProfileHealth — exported function, 54 lines, zero
    callers confirmed, no USAGE edges. Checks role-inheritance health (missing
    inherits references, cycle detection) by accepting two function parameters.
    Health system has since grown its own path via internal/health/; this
    function appears to be a superseded phase-2b implementation.
    (location: internal/validate/phase2b_health.go:127)
    Recommendation: verify whether the health system covers profile health via
    another path; if so, delete this function.

- [minor] dead_code: FindGitRoot — exported function, zero callers confirmed,
    no USAGE edges. Traverses the directory tree upward to find .git. The
    internal/kbzinit package uses git shell invocations for its own init logic;
    this function is unused alongside it.
    (location: internal/kbzinit/git.go:12)
    Recommendation: delete, or unexport and wire up if it fills a genuine need.

- [minor] dead_code: HasCommits — exported function in same file as FindGitRoot,
    zero callers confirmed. Checks whether a git repo has any commits by running
    `git rev-parse HEAD`.
    (location: internal/kbzinit/git.go)
    Recommendation: delete alongside FindGitRoot review.

- [minor] dead_code: DefaultCheckOptions — exported zero-argument constructor,
    zero callers confirmed, no USAGE edges. Returns CheckOptions with
    DefaultBranchThresholds and IncludeOK: false.
    (location: internal/health/check.go:38)
    Recommendation: delete if CheckOptions is always constructed inline at call
    sites, or wire up where options are currently constructed manually.

- [minor] dead_code: DeriveGraphProject — exported 9-line utility, zero callers
    confirmed, no USAGE edges. Converts an absolute repo path to a
    hyphen-separated graph project name (mirrors the codebase-memory naming
    convention).
    (location: internal/config/user.go:151)
    Recommendation: verify whether the graph project name is derived elsewhere
    (e.g. hardcoded at call sites); if so, delete and centralise on this
    function, or delete if the derivation is intentionally inline.

- [minor] dead_code: NewCompositeTransitionHook — exported constructor, 3
    lines, zero callers confirmed, no USAGE edges. Wraps multiple
    StatusTransitionHook values into a CompositeTransitionHook. The composite
    pattern was built but never wired into the service layer.
    (location: internal/service/status_transition_hook.go:65)
    Recommendation: delete if single-hook dispatch is sufficient; otherwise wire
    up at the service wiring site.

Notes (not findings):

- docTool (16 outbound calls) and knowledgeTool (12 outbound calls): both are
  large MCP tool handlers that dispatch over action parameters. High outbound
  degree is incidental to the dispatch-table pattern, not an SRP violation.
  No action required.

- Change coupling: entity-structure-and-document-pipeline.md <->
  phase-2-scope.md (coupling_score: 0.80). Both are work/ documentation files,
  not source code. Expected co-evolution. No action required.

---

### style_conformance: clean_with_notes

Summary: 1 minor observation. No acronym-casing violations found in exported
identifiers. No unguarded error discards in production code; all `_ =` usages
are either: (a) explicitly commented as best-effort with a spec reference
(FR-018 in actionlog), (b) cleanup/close operations where the primary error path
is already handled, or (c) test helpers.

Findings:

- [minor] dead-assignment: CheckDependencyCycles assigns `parent` variable then
    immediately discards it with `_ = parent`. Comment reads "parent is used in
    cycle path reconstruction (future enhancement)." This is a suppressor for a
    not-yet-implemented feature, not a logic error.
    (location: internal/health/phase4a.go:69)
    Recommendation: either implement the cycle path reconstruction and use
    `parent`, or remove the variable and its assignment until the feature is
    needed. Carrying suppressed assignments as TODOs degrades signal-to-noise.

---

## Finding Summary

  Critical: 4  (all resolved during audit — see test_health)
  Major:    3  (AllStates, CheckProfileHealth, guidanceRules)
  Minor:    13
  Total:    20

---

## Next Actions

  1. [RESOLVED] Fix 9 test failures in internal/kbzinit (embedded skill
     frontmatter format mismatch + two missing acceptance-criterion content
     requirements). All tests now pass.

  2. [major] Investigate and resolve AllStates dead code
     (internal/validate/lifecycle.go:413) — wire up or delete.

  3. [major] Investigate and resolve CheckProfileHealth dead code
     (internal/validate/phase2b_health.go:127) — verify superseded by health
     system, then delete.

  4. [major] Investigate guidanceRules unused var in decompose service
     (internal/service/decompose.go:566) — wire up or delete along with its
     population code.

  5. [minor] Add staticcheck to CI as a pre-merge gate. It is now installed
     at ~/go/bin/staticcheck.

  6. [minor] Decide and document the ST1005 policy for internal/mcp/ error
     strings (245 violations): either lowercase them (Go convention) or add a
     staticcheck config suppressing ST1005 for that package (user-facing
     messages). Do not fix piecemeal.

  7. [minor] Delete minor production dead code: rebuildIndexUsageText
     (cmd/kanbanzai/rebuild.go:12), path field
     (internal/docint/parser.go:163).

  8. [minor] Apply small staticcheck fixes: S1039 fmt.Sprintf (4 locations),
     S1017 TrimPrefix/TrimSuffix (decompose.go:1167-1170), S1005
     (kbzinit/init.go:370), S1016 (context/pipeline.go:384).

  9. [minor] Delete 4 unused test helpers: captureStdout, addCommitsToMain,
     writeKnowledgeFile, callStatus.

  10. [minor] Review and delete confirmed minor dead code cluster:
      FindGitRoot, HasCommits (internal/kbzinit/git.go),
      DefaultCheckOptions (internal/health/check.go:38),
      DeriveGraphProject (internal/config/user.go:151),
      NewCompositeTransitionHook (internal/service/status_transition_hook.go:65).

  11. [minor] Resolve CheckDependencyCycles dead assignment
      (internal/health/phase4a.go:69) — implement or remove.