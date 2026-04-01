package service

// DefaultMaxReviewCycles is the maximum number of review cycles before a
// human decision is required. A named constant to facilitate future binding
// registry integration (NFR-004).
const DefaultMaxReviewCycles = 3

// IncrementFeatureReviewCycle increments the review_cycle field on the feature
// entity by 1 and persists the change to disk. Called each time a feature
// transitions into the reviewing status (FR-002).
func (s *EntityService) IncrementFeatureReviewCycle(featureID, slug string) error {
	if slug == "" {
		_, resolvedSlug, err := s.ResolvePrefix("feature", featureID)
		if err != nil {
			return err
		}
		slug = resolvedSlug
	}

	record, err := s.store.Load("feature", featureID, slug)
	if err != nil {
		return err
	}

	current, _ := record.Fields["review_cycle"].(int)
	record.Fields["review_cycle"] = current + 1

	_, err = s.store.Write(record)
	return err
}
