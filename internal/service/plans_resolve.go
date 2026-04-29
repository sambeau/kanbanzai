package service

import (
	"fmt"
	"strings"

	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/model"
)

// ResolvePlanByNumber returns the full canonical Plan ID and slug for the plan whose
// prefix and number match the given arguments. cfg is used to validate the prefix
// against the active prefix registry (FR-005, FR-006, FR-007, FR-008).
func (s *EntityService) ResolvePlanByNumber(cfg config.Config, prefix, number string) (id, slug string, err error) {
	if !cfg.IsActivePrefix(prefix) {
		active := cfg.ActivePrefixes()
		names := make([]string, len(active))
		for i, entry := range active {
			names[i] = entry.Prefix
		}
		return "", "", fmt.Errorf("unknown plan prefix %q — valid prefixes are: [%s]", prefix, strings.Join(names, ", "))
	}

	ids, listErr := s.listPlanIDs()
	if listErr != nil {
		return "", "", fmt.Errorf("list plan IDs: %w", listErr)
	}

	for _, planID := range ids {
		p, num, planSlug := model.ParsePlanID(planID)
		if p == prefix && num == number {
			return planID, planSlug, nil
		}
	}

	return "", "", fmt.Errorf("no plan found with prefix %q and number %q", prefix, number)
}
