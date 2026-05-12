package registry

import (
	"fmt"
	"slices"
	"strings"
)

const skillsBasePath = ".kbz/skills"

// RolesAndSkillsContent returns the Markdown content (excluding region markers)
// for the roles-and-skills generated region. Stages appear in declaration order
// from the RegistryModel. The output opens with a warning comment identifying
// .kbz/stage-bindings.yaml as the canonical source. Output is deterministic:
// given the same model, identical bytes are always produced (REQ-NF-003).
// Full skill procedures, examples, and anti-pattern bodies are not included
// (REQ-NF-004).
func RolesAndSkillsContent(model *RegistryModel) string {
	var sb strings.Builder

	sb.WriteString("> **Generated** — canonical source: `.kbz/stage-bindings.yaml`.")
	sb.WriteString(" Do not hand-edit this section; run `make registry-sync` to update.\n")
	sb.WriteByte('\n')
	sb.WriteString("| Stage | Description | Roles | Skills | Gate | Doc Type |\n")
	sb.WriteString("|-------|-------------|-------|--------|------|----------|\n")

	for _, s := range model.Stages {
		gate := "auto"
		if s.HumanGate {
			gate = "human"
		}
		docType := s.DocumentType
		if docType == "" {
			docType = "—"
		}
		fmt.Fprintf(&sb, "| %s | %s | %s | %s | %s | %s |\n",
			s.Name,
			escapeCell(s.Description),
			formatRoles(s.Roles),
			formatSkillLinks(s.Skills),
			gate,
			docType,
		)
	}

	return sb.String()
}

// RoleIndexContent returns the Markdown content (excluding region markers) for
// the role-index generated region. Roles are sorted lexicographically by role
// ID. The output opens with a warning comment identifying .kbz/roles/*.yaml as
// the canonical source. Output is deterministic (REQ-NF-003). Full role
// vocabulary, anti-pattern bodies, and procedures are not included (REQ-NF-004).
func RoleIndexContent(model *RegistryModel) string {
	var sb strings.Builder

	sb.WriteString("> **Generated** — canonical source: `.kbz/roles/*.yaml`.")
	sb.WriteString(" Do not hand-edit this section; run `make registry-sync` to update.\n")
	sb.WriteByte('\n')
	sb.WriteString("| Role | Identity | Inherits | Source |\n")
	sb.WriteString("|------|----------|----------|--------|\n")

	ids := make([]string, 0, len(model.Roles))
	for id := range model.Roles {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	for _, id := range ids {
		r := model.Roles[id]
		inherits := "—"
		if r.Inherits != "" {
			inherits = "`" + r.Inherits + "`"
		}
		fmt.Fprintf(&sb, "| `%s` | %s | %s | `%s` |\n",
			r.ID,
			escapeCell(r.Identity),
			inherits,
			r.SourcePath,
		)
	}

	return sb.String()
}

// formatRoles formats a slice of role names as comma-separated backtick-quoted
// identifiers. Returns "—" if the slice is empty.
func formatRoles(roles []string) string {
	if len(roles) == 0 {
		return "—"
	}
	parts := make([]string, len(roles))
	for i, r := range roles {
		parts[i] = "`" + r + "`"
	}
	return strings.Join(parts, ", ")
}

// formatSkillLinks formats a slice of skill names as comma-separated Markdown
// links pointing to the canonical skill file at .kbz/skills/{name}/SKILL.md.
// Returns "—" if the slice is empty.
func formatSkillLinks(skills []string) string {
	if len(skills) == 0 {
		return "—"
	}
	parts := make([]string, len(skills))
	for i, s := range skills {
		path := skillsBasePath + "/" + s + "/SKILL.md"
		parts[i] = "[" + s + "](" + path + ")"
	}
	return strings.Join(parts, ", ")
}

// escapeCell replaces pipe characters within a Markdown table cell value to
// prevent them from breaking the table structure.
func escapeCell(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}
