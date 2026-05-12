# Bug Specification: Replace custom canonical YAML parser with yaml.v3

## Observed Behaviour
Entity listing fails with "unexpected indentation at line 13" for features using inline list format like `- from_status: dev-planning`. The hotfix patches the custom parser but hand-rolled parsers will always lag behind standard libraries.

## Expected Behaviour
All valid YAML formats parse correctly, including inline list items, quoted strings with colons, and multi-line scalars, via a battle-tested standard library.

## Severity
medium | Priority: medium | Type: implementation-defect
