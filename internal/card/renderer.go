package card

import (
	"fmt"
	"strings"

	"github.com/sambeau/kanbanzai/internal/binding"
	kbzctx "github.com/sambeau/kanbanzai/internal/context"
)

const (
	maxNonEmptyLines = 25
	maxBytes         = 2500

	toolRoutingLine = "**Tool routing:** Sub-agent prompts must come from `handoff`, not manual composition."
	unknownStageMsg = "> **UNKNOWN STAGE** — no stage binding found. Load `.kbz/stage-bindings.yaml` manually."
)

// Render composes a compact markdown constraint card from typed inputs.
//
// role must be non-nil and have a non-empty Identity field (REQ-007). A
// missing required role field causes an immediate error naming that field.
//
// When b is nil the stage is treated as unknown: the card includes the
// UNKNOWN STAGE warning and manual-load instruction (REQ-008).
//
// entries are pre-selected constraints for this role+stage pair. They are
// rendered verbatim; the renderer does not read SKILL.md files.
//
// The card is enforced to ≤25 non-empty lines and ≤2500 bytes
// (REQ-NF-001, REQ-NF-002). An error is returned if either limit is exceeded.
func Render(role *kbzctx.ResolvedRole, stage string, b *binding.StageBinding, entries []ConstraintEntry) (string, error) {
	if role == nil {
		return "", fmt.Errorf("constraint card: missing required input \"role\"")
	}
	if role.Identity == "" {
		return "", fmt.Errorf("constraint card: role %q missing required field \"identity\"", role.ID)
	}

	var sb strings.Builder

	sb.WriteString("---\n")

	if role.ID != "" {
		sb.WriteString(fmt.Sprintf("**Role:** %s — %s\n", role.ID, role.Identity))
	} else {
		sb.WriteString(fmt.Sprintf("**Role:** %s\n", role.Identity))
	}

	if b == nil {
		// Unknown stage (REQ-008).
		sb.WriteString(unknownStageMsg + "\n")
	} else {
		sb.WriteString(fmt.Sprintf("**Stage:** %s\n", stage))
		if len(b.Skills) > 0 {
			sb.WriteString(fmt.Sprintf("**Skills:** %s\n", strings.Join(b.Skills, ", ")))
		}
		if len(entries) > 0 {
			sb.WriteString("**Constraints:**\n")
			for _, e := range entries {
				sb.WriteString(fmt.Sprintf("- %s\n", e.Rule))
			}
		}
		sb.WriteString(toolRoutingLine + "\n")
	}

	sb.WriteString("---\n")

	card := sb.String()

	if err := checkLimits(card); err != nil {
		return "", err
	}
	return card, nil
}

// checkLimits returns an error if card exceeds the non-empty-line or byte budget.
func checkLimits(card string) error {
	nonEmpty := 0
	for _, line := range strings.Split(card, "\n") {
		if strings.TrimSpace(line) != "" {
			nonEmpty++
		}
	}
	if nonEmpty > maxNonEmptyLines {
		return fmt.Errorf("constraint card: exceeded max non-empty lines (%d > %d)", nonEmpty, maxNonEmptyLines)
	}
	if len(card) > maxBytes {
		return fmt.Errorf("constraint card: exceeded max bytes (%d > %d)", len(card), maxBytes)
	}
	return nil
}
