package docint

import "strings"

// UpdateConceptRegistry updates the concept registry with concepts from a document's classifications.
// It adds new concepts and updates existing ones with new introduced_in/used_in references.
// For concepts_intro entries that carry aliases, the aliases are stored on the concept.
func UpdateConceptRegistry(registry *ConceptRegistry, docID string, classifications []Classification) {
	for _, c := range classifications {
		sectionRef := docID + "#" + c.SectionPath
		for _, entry := range c.ConceptsIntro {
			upsertConceptRef(registry, entry.Name, sectionRef, true)
			if len(entry.Aliases) > 0 {
				mergeAliases(registry, entry.Name, entry.Aliases)
			}
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

// mergeAliases stores normalised aliases on the named concept.
// Aliases that equal the canonical name, or that are already present, are skipped.
func mergeAliases(registry *ConceptRegistry, canonicalName string, aliases []string) {
	concept := FindConcept(registry, canonicalName)
	if concept == nil {
		return
	}
	canonical := NormalizeConcept(canonicalName)
	for _, a := range aliases {
		norm := NormalizeConcept(a)
		if norm == canonical {
			continue // skip alias identical to canonical name
		}
		if !stringSliceContains(concept.Aliases, norm) {
			concept.Aliases = append(concept.Aliases, norm)
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
// It first checks canonical names, then searches aliases.
func FindConcept(registry *ConceptRegistry, name string) *Concept {
	normalized := NormalizeConcept(name)
	for i := range registry.Concepts {
		if NormalizeConcept(registry.Concepts[i].Name) == normalized {
			return &registry.Concepts[i]
		}
	}
	// No canonical match — scan aliases.
	for i := range registry.Concepts {
		for _, alias := range registry.Concepts[i].Aliases {
			if NormalizeConcept(alias) == normalized {
				return &registry.Concepts[i]
			}
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
