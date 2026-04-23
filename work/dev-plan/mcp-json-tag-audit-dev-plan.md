# Dev Plan: MCP Parameter Struct JSON Tag Audit

**Status:** draft
**Feature:** FEAT-01KPX-5CW4R82P
**Spec:** work/spec/p32-mcp-json-tag-audit.md

## Scope

Audit all MCP parameter structs in `internal/mcp/` that carry `yaml:` tags and are decoded from JSON tool parameters via `json.Unmarshal`. Add explicit `json:` tags to any exported field missing one. Add a regression test to prevent future regressions.

`Classification` and `ConceptIntroEntry` are already fixed and excluded from scope.

## Task Breakdown

| # | Task | Effort |
|---|------|--------|
| T1 | Audit yaml-tagged structs for missing json tags (`TASK-01KPXE3VRGBQF`) | S |
| T2 | Write JSON round-trip regression test (`TASK-01KPXE45XV5A5`) | S |
| T3 | Verify existing MCP tool calls unaffected (`TASK-01KPXE4CR77A7`) | XS |

T1 and T2 can run in parallel. T3 depends on T1 and T2.

## Dependency Graph

```
T1 ──┐
     ├──▶ T3
T2 ──┘
```

## Risk Assessment

- **Low risk**: purely additive change — adding json tags to structs that already have yaml tags. No behavioural changes.
- The regression test (T2) is the main safety net; T3 provides end-to-end confidence.

## Verification Approach

- `go test ./internal/mcp/...` must pass after T1 and T2.
- The reflection-based check in T2 must fail if any audited struct gains a new untagged field in future.
- T3 confirms no existing tool call behaviour is altered.