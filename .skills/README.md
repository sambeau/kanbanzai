# Legacy Skills (Deprecated)

The skill files that were previously in this directory have been retired.

## Where skills live now

- **Agent workflow skills** (for Claude Code / MCP agent discovery):
  `.agents/skills/kanbanzai-*/SKILL.md`

- **Context assembly skills** (consumed by the 3.0 context assembly pipeline):
  `.kbz/skills/*/SKILL.md`

## Migration mapping

| Old file | New location |
|----------|-------------|
| `code-review.md` | `.kbz/skills/review-code/SKILL.md` + `.kbz/skills/orchestrate-review/SKILL.md` |
| `plan-review.md` | `.kbz/skills/review-plan/SKILL.md` |
| `document-creation.md` | `.kbz/skills/write-design/`, `write-spec/`, `write-dev-plan/`, `write-research/`, `update-docs/` |