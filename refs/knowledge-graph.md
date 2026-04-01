# Codebase Knowledge Graph (`codebase-memory-mcp`)

Full guide for using the knowledge graph. Referenced from [AGENTS.md](../AGENTS.md).

This project is indexed in `codebase-memory-mcp` under the project name **`Users-samphillips-Dev-kanbanzai`** with root path `/Users/samphillips/Dev/kanbanzai`.

The graph is the preferred way to explore code structure. Use it **instead of** `grep` or `find_path` whenever you need to understand definitions, relationships, callers, callees, dependencies, or architecture.

## When to use graph tools (preferred)

| Question | Tool | Example |
|----------|------|---------|
| What does a function/type look like? | `get_code_snippet` | `get_code_snippet(qualified_name="EntityService.Get")` |
| Who calls this function? | `trace_call_path` | `trace_call_path(function_name="ResolvePrefix", direction="inbound")` |
| What does this function call? | `trace_call_path` | `trace_call_path(function_name="Get", direction="outbound")` |
| Find a function/class/type by name | `search_graph` | `search_graph(name_pattern="Allocat")` |
| Understand package structure | `get_architecture` | `get_architecture(project="Users-samphillips-Dev-kanbanzai")` |
| Complex cross-package queries | `query_graph` | Cypher queries for multi-hop analysis |

## When to use text search (fallback)

Use `grep` only for content that is not structural:

- String literals and error messages
- Config values and magic constants
- YAML field names in test fixtures
- Comments and documentation text
- Broad "does this string appear anywhere?" sweeps

Use `find_path` only when searching by filename pattern, not by code content.

## Keeping the graph current

The graph auto-syncs after the initial index. If results seem stale or the project is missing from `list_projects`, force a refresh:

```
index_repository(repo_path="/Users/samphillips/Dev/kanbanzai")
```

## Fallback policy

1. Use graph queries first for structural questions.
2. Use `search_graph` to discover exact qualified names before `trace_call_path` or `get_code_snippet`.
3. Fall back to `grep` only for non-structural content searches.
4. Fall back to `read_file` only when you need to see exact file content that the graph doesn't cover (e.g., full test bodies, YAML fixtures).