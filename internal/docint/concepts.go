package docint

import "strings"

// UpdateConceptRegistry updates the concept registry with concepts from a document's classifications.
// It adds new concepts and updates existing ones with new introduced_in/used_in references.
func UpdateConceptRegistry(registry *ConceptRegistry, docID string, classifications []Classification) {
	for _, c := range classifications {
		sectionRef := docID + "#" + c.SectionPath
		for _, name := range c.ConceptsIntro {
			upsertConceptRef(registry, name, sectionRef, true)
		}
		for _, name := range c.ConceptsUsed {
			upsertConceptRef(registry, name, sectionRef, false)
		}
	}
}

// upsertConceptRef adds a section reference to a concept, creating the concept if needed.
func upsertConceptRef(registry *ConceptRegistry, name, sectionRef string, isIntro bool) {
	concept := FindConcept(registry, name)
	if concept == nil {
		newConcept := Concept{Name: NormalizeConcept(name)}
		if isIntro {
			newConcept.IntroducedIn = []string{sectionRef}
		} else {
			newConcept.UsedIn = []string{sectionRef}
		}
		registry.Concepts = append(registry.Concepts, newConcept)
		return
	}
	if isIntro {
		if !stringSliceContains(concept.IntroducedIn, sectionRef) {
			concept.IntroducedIn = append(concept.IntroducedIn, sectionRef)
		}
	} else {
		if !stringSliceContains(concept.UsedIn, sectionRef) {
			concept.UsedIn = append(concept.UsedIn, sectionRef)
		}
	}
}

// RemoveDocumentFromRegistry removes all references to a document from the concept registry.
// If a concept has no remaining references after removal, it is deleted from the registry.
func RemoveDocumentFromRegistry(registry *ConceptRegistry, docID string) {
	prefix := docID + "#"
	var kept []Concept
	for _, c := range registry.Concepts {
		c.IntroducedIn = removeByPrefix(c.IntroducedIn, prefix)
		c.UsedIn = removeByPrefix(c.UsedIn, prefix)
		if len(c.IntroducedIn) > 0 || len(c.UsedIn) > 0 {
			kept = append(kept, c)
		}
	}
	registry.Concepts = kept
}

// NormalizeConcept normalizes a concept name for deduplication.
func NormalizeConcept(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	// Collapse consecutive hyphens
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}
	return name
}

// FindConcept looks up a concept by name (case-insensitive, normalized).
func FindConcept(registry *ConceptRegistry, name string) *Concept {
	normalized := NormalizeConcept(name)
	for i := range registry.Concepts {
		if NormalizeConcept(registry.Concepts[i].Name) == normalized {
			return &registry.Concepts[i]
		}
	}
	return nil
}

func removeByPrefix(refs []string, prefix string) []string {
	var result []string
	for _, r := range refs {
		if !strings.HasPrefix(r, prefix) {
			result = append(result, r)
		}
	}
	return result
}

func stringSliceContains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
