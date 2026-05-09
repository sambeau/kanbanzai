# Entity Naming Examples

Good and bad entity name examples by type. Linked from
`.agents/skills/kanbanzai-agents/SKILL.md` `## Entity Names`.

---

## Good names

| Entity | Name | Why |
|--------|------|-----|
| Plan | Kanbanzai 2.0 | Short, identity-oriented |
| Batch | Webhook delivery | ~3 words, describes the grouping |
| Feature | Human-friendly ID display | ~4 words, no prefix, self-contained |
| Feature | Init and skill install | ~4 words, descriptive, no prefix |
| Task | Server info tool | ~3 words, clear |
| Task | Label model and storage | ~4 words |
| Decision | Use TSID for entity IDs | Self-contained, concise |

## Bad names

| Entity | Name | Problem |
|--------|------|---------|
| Plan | P4 Kanbanzai 2.0: MCP Tool Surface Redesign | Phase prefix + colon |
| Feature | P8 — decompose propose Reliability Fixes | Phase prefix + separator dash |
| Feature | The kanbanzai init command: creates .kbz/config.yaml | Colon + far too long — this is a summary |
| Task | Update | Too vague, not self-contained |
