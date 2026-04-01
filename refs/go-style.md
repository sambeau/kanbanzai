# Go Style Reference

Full Go code conventions for the Kanbanzai project. Referenced from [AGENTS.md](../AGENTS.md).

## YAML Serialisation Rules

Entity state and documents are stored as YAML. Deterministic, canonical serialisation is a core requirement — not a nice-to-have. The accepted decision P1-DEC-008 in the decision log defines the exact rules:

- Block style for mappings and sequences (no flow style)
- Double-quoted strings only when required by YAML syntax
- Deterministic field order (defined per entity type)
- UTF-8, LF line endings, trailing newline
- No YAML tags, anchors, or aliases
- No multi-document streams

Do not rely on Go's default YAML marshaller to produce correct output. The serialisation must be explicit and tested with round-trip tests (write → read → write → compare).

## Formatting

- Write idiomatic Go
- Run `go fmt` before committing
- Use `goimports` for import organisation
- Maximum line length: 100 characters (soft limit)

## Naming

- Use camelCase for unexported identifiers
- Use PascalCase for exported identifiers
- Acronyms should be consistent case: `URL`, `HTTP`, `ID` (not `Url`, `Http`, `Id`)
- Package names: lowercase, single word, no underscores

## Error Handling

- Always check errors; never use `_` to ignore them
- Wrap errors with context: `fmt.Errorf("doing X: %w", err)`
- Return errors, don't panic (except for truly unrecoverable situations)
- Define sentinel errors with `errors.New` for errors that callers need to check

## Comments

- Exported functions must have doc comments
- Doc comments start with the function name: `// FunctionName does...`
- Use `// TODO:` for planned improvements
- Use `// FIXME:` for known issues

## Interfaces

- Accept interfaces, return structs
- Define interfaces at the consumer, not the provider
- Keep interfaces small — one or two methods is ideal
- Do not define interfaces preemptively; extract them when a second implementation or a test double is needed

## Concurrency

- Do not use goroutines unless there is a demonstrated need
- This is a request-response system — no concurrent workflows
- If goroutines are needed, pass `context.Context` and use it for cancellation

## Package Design

- Keep packages small and focused on a single responsibility
- No circular imports — if two packages need each other, extract shared types into a third
- The `internal/` directory is not importable from outside this module
- No `init()` functions — they create hidden coupling and make testing harder

## File Organisation

```
cmd/kanbanzai/    # binary entry point
internal/         # all private packages (core logic, MCP server, CLI)
```

This is not a library. There is no `pkg/` directory.

## Dependencies

- Prefer the standard library when reasonable
- Run `go mod tidy` after adding/removing dependencies
- Commit `go.sum` with `go.mod`
