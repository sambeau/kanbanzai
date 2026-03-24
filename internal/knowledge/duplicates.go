package knowledge

// ExistingEntity represents a candidate entity for duplicate checking.
type ExistingEntity struct {
	ID      string
	Type    string
	Title   string
	Summary string
}

// DuplicateCandidate is a candidate entity whose title+summary is similar to the input.
type DuplicateCandidate struct {
	EntityID   string
	EntityType string
	Title      string
	Similarity float64
}

// FindDuplicateCandidates returns entities whose (title + summary) word-set has
// Jaccard similarity >= threshold against the given title+summary.
// threshold is typically 0.5 per spec §13.
func FindDuplicateCandidates(title, summary string, existing []ExistingEntity, threshold float64) []DuplicateCandidate {
	inputWords := ContentWords(title + " " + summary)

	var candidates []DuplicateCandidate
	for _, e := range existing {
		existingWords := ContentWords(e.Title + " " + e.Summary)
		sim := JaccardSimilarity(inputWords, existingWords)
		if sim >= threshold {
			candidates = append(candidates, DuplicateCandidate{
				EntityID:   e.ID,
				EntityType: e.Type,
				Title:      e.Title,
				Similarity: sim,
			})
		}
	}

	return candidates
}
