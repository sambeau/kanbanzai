# Testing Conventions

Full testing conventions for the Kanbanzai project. Referenced from [`AGENTS.md`](../AGENTS.md).

## Conventions

- Test files: `*_test.go` in the same package
- Test functions: `TestFunctionName_Scenario`
- Use table-driven tests for multiple cases
- Aim for meaningful coverage, not 100%

## Test isolation

- Tests must not depend on external services or network calls
- Use `t.TempDir()` for filesystem tests — never write to the working directory
- Test fixtures live in `testdata/` directories alongside the test files
- Test helpers must call `t.Helper()` so failures report the caller's line number

## What to test

- Core validation logic (field validation, lifecycle transitions, referential integrity)
- Serialisation and deterministic formatting (round-trip: write → read → write → compare)
- ID allocation edge cases
- Document validation (valid and invalid cases)
- MCP operations (integration tests where practical)
- CLI behaviour (integration tests where practical)

## What not to test

- Do not test the standard library
- Do not write tests that only assert that a mock was called — test behaviour, not wiring
- Do not test unexported functions directly unless they contain complex logic worth isolating