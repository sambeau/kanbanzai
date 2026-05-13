# Bug Specification: Handoff does not emit Code Graph section for sub-agents

## Observed Behaviour
1. `handoff` prompt never contains `## Code Graph` — the pipeline (`internal/context/pipeline.go`) has zero references to `GraphProject`, `graph_project`, or `Code Graph`. The `stepAssembleSections` method builds sections for identity, role, vocabulary, anti-patterns, available tools, procedure, orchestration, dispatch contract, and worktree directive — but nothing graph-related.

2. `assembledContext` in `internal/mcp/assembly.go` loads `graphProject` from the worktree record (L314-317) but no rendering code ever reads it. The field is only consumed by `nextContextToMap` (for the orchestrator's structured output) and `synthesiseFeature` (for `missing_graph_index` attention items).

3. The dev-plan (`work/B21-codebase-memory-integration/B21-F2-dev-plan-worktree-graph-context.md`) specified a three-state handoff logic: project set → examples, project empty → index instruction, no worktree → omit. This was never implemented.

4. Tests for the Code Graph section exist only as comment stubs in `internal/mcp/assembly_test.go` L542-556 (`TestRenderHandoffPrompt_CodeGraphSection_ProjectSet`, etc.) — none are actual test functions.

5. `internal/kbzinit/agents_md_test.go` L623-627 confirms the Code Graph section was deliberately removed: "The implementation removed two false claims (context packet definition and Code Graph section)."

6. AGENTS.md L344 documents the gap: "(`handoff` renders a Markdown prompt and does not include graph project metadata.)"

7. The local config IS set: `.kbz/local.yaml` has `codebase_memory.graph_project: Users-samphillips-Dev-kanbanzai`. Worktree records do have `GraphProject` populated. The data is present; the rendering is absent.

## Expected Behaviour
When `handoff` is called for a task whose parent feature has a worktree with `GraphProject` set to a non-empty string, the generated prompt must include a `## Code Graph` section placed after `## Available Tools` containing:

1. The project name (e.g. `Users-samphillips-Dev-kanbanzai`)
2. Example tool calls using that project name:
   - `search_graph(project: "Users-samphillips-Dev-kanbanzai", query: "...")`
   - `trace_path(function_name: "...", project: "Users-samphillips-Dev-kanbanzai")`
   - `query_graph(query: "...", project: "Users-samphillips-Dev-kanbanzai")`
   - `get_code_snippet(qualified_name: "...", project: "Users-samphillips-Dev-kanbanzai")`
3. A note: "Prefer graph tools over grep for structural questions. Read `.github/skills/codebase-memory-*/SKILL.md` before using any graph tool."
4. A re-indexing instruction: "If the graph is stale, run `index_repository(repo_path: "<worktree-path>")`."

When `GraphProject` is empty but a worktree exists, the section should contain only the `index_repository` instruction using the worktree path.

When no worktree exists, the section must be omitted entirely.

The section must not exceed ~500 bytes when GraphProject is set (NFR-003 from the dev-plan).

## Severity
medium | Priority: medium | Type: implementation-defect
