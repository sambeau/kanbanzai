# SKILLs

## What are SKILLs?

SKILLs are **focused, step-by-step procedures** for specific repeatable tasks that agents perform when working on the kanbanzai project.

Each SKILL is a standalone markdown file that provides:
- Clear purpose and scope
- When to use the procedure
- Numbered step-by-step instructions
- Examples and code snippets
- Troubleshooting for common issues
- Verification steps
- Related documentation references

## Why SKILLs exist

SKILLs solve a specific problem: `AGENTS.md` contains broad policies, conventions, and project context, but detailed procedures can make it unwieldy. SKILLs extract specific "how to do X" procedures into separate, reusable files.

**SKILLs are for procedures. AGENTS.md is for policy.**

| Use a SKILL for... | Use AGENTS.md for... |
|-------------------|----------------------|
| Step-by-step procedures | Broad project conventions |
| Repeatable tasks | Decision-making rules |
| Tool usage patterns | Git commit policies |
| Specific workflows | Reading order and orientation |
| Troubleshooting guides | Project status and context |

## When to create a new SKILL

Create a SKILL when:
1. A procedure is **repeatable** — it will be done multiple times
2. A procedure is **specific** — clear start, clear end, clear steps
3. A procedure is **substantial** — more than 3-4 steps, or has edge cases
4. A procedure would **bloat AGENTS.md** — adding 50+ lines of detail to AGENTS.md for one task

Do not create a SKILL for:
- One-off tasks
- Trivial operations (1-2 steps)
- Policy or philosophy (those belong in design documents or AGENTS.md)

## How to use SKILLs

### For agents

When `AGENTS.md` references a SKILL:

> "Follow the `document-creation` SKILL in `.skills/document-creation.md`"

Read the referenced SKILL file and execute its procedure step by step.

### For humans

SKILLs are equally useful for humans performing manual workflow operations via CLI. They serve as clear, tested recipes for common tasks.

## Existing SKILLs

| SKILL | Purpose | When to use |
|-------|---------|-------------|
| `document-creation` | Register documents with the kanbanzai system | After creating any new `.md` file in `work/` |

## SKILL format

Each SKILL follows this structure:

```markdown
# SKILL: {Name}

## Purpose
Brief statement of what this procedure accomplishes.

## When to Use
- Bullet list of scenarios where this SKILL applies

---

## Procedure: {Main Path}

### Step 1: {Action}
Description and examples.

### Step 2: {Action}
Description and examples.

---

## Procedure: {Alternate Path}
(Optional: for variants of the main procedure)

---

## Common Issues

### Issue: {Problem}
**Symptom:** ...
**Cause:** ...
**Solution:** ...

---

## Verification
Checklist to confirm the procedure completed successfully.

---

## Related
- Links to relevant documentation
```

## Maintenance

SKILLs should be:
- **Kept up to date** — when tools or procedures change, update the SKILL
- **Versioned in Git** — SKILLs are part of the project and follow the same commit discipline as code
- **Referenced, not duplicated** — AGENTS.md and other docs should reference SKILLs by name, not copy their content

## Relationship to other documentation

```
AGENTS.md
  ├─ Broad project conventions, policies, orientation
  ├─ References to design documents and specs
  └─ References to SKILLs (by name)

.skills/
  ├─ Focused, step-by-step procedures
  ├─ Standalone and reusable
  └─ Updated as tools and processes evolve

work/bootstrap/bootstrap-workflow.md
  ├─ Current workflow process during development
  └─ May reference SKILLs for specific operations

work/design/
  └─ Design vision and architectural decisions
      (SKILLs implement procedures aligned with design)
```

## Contributing new SKILLs

When you identify a procedure that should be a SKILL:

1. Create the SKILL file in `.skills/{name}.md`
2. Follow the SKILL format template above
3. Update this README to list the new SKILL
4. Update `AGENTS.md` or other docs to reference the SKILL by name
5. Commit all changes together

Example commit message:
```
feat(skills): add {name} SKILL

- Create .skills/{name}.md with procedure for {purpose}
- Update AGENTS.md to reference the SKILL
- Add entry to .skills/README.md
```
