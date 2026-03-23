package docint

import "strings"

// BuildGraphEdges constructs graph edges from a document's index.
// It creates edges for:
// - CONTAINS: Document → Section, Section → Section (parent → child)
// - REFERENCES: Section → EntityRef
// - LINKS_TO: Section → target path (cross-document links)
// - INTRODUCES: Fragment → Concept (concepts_intro from classifications)
// - USES: Fragment → Concept (concepts_used from classifications)
func BuildGraphEdges(index *DocumentIndex) []GraphEdge {
	var edges []GraphEdge
	docID := index.DocumentID

	// CONTAINS edges: document → top-level sections, sections → children
	for _, s := range index.Sections {
		edges = appendContainsEdges(edges, docID, docID, "document", s)
	}

	// REFERENCES edges: section → entity ref
	for _, ref := range index.EntityRefs {
		edges = append(edges, GraphEdge{
			From:     docID + "#" + ref.SectionPath,
			FromType: "section",
			To:       ref.EntityID,
			ToType:   "entity_ref",
			EdgeType: "REFERENCES",
		})
	}

	// LINKS_TO edges: section → cross-document target
	for _, link := range index.CrossDocLinks {
		edges = append(edges, GraphEdge{
			From:     docID + "#" + link.SectionPath,
			FromType: "section",
			To:       link.TargetPath,
			ToType:   "document",
			EdgeType: "LINKS_TO",
		})
	}

	// INTRODUCES and USES edges: fragment → concept
	for _, c := range index.Classifications {
		fragmentID := docID + "#" + c.SectionPath
		for _, name := range c.ConceptsIntro {
			edges = append(edges, GraphEdge{
				From:     fragmentID,
				FromType: "fragment",
				To:       NormalizeConcept(name),
				ToType:   "concept",
				EdgeType: "INTRODUCES",
			})
		}
		for _, name := range c.ConceptsUsed {
			edges = append(edges, GraphEdge{
				From:     fragmentID,
				FromType: "fragment",
				To:       NormalizeConcept(name),
				ToType:   "concept",
				EdgeType: "USES",
			})
		}
	}

	return edges
}

// appendContainsEdges recursively adds CONTAINS edges for a section and its children.
func appendContainsEdges(edges []GraphEdge, docID, parentID, parentType string, section Section) []GraphEdge {
	sectionID := docID + "#" + section.Path
	edges = append(edges, GraphEdge{
		From:     parentID,
		FromType: parentType,
		To:       sectionID,
		ToType:   "section",
		EdgeType: "CONTAINS",
	})
	for _, child := range section.Children {
		edges = appendContainsEdges(edges, docID, sectionID, "section", child)
	}
	return edges
}

// MergeGraphEdges merges edges from multiple documents into a single graph.
// It removes all existing edges from the specified document and adds the new ones.
func MergeGraphEdges(existing []GraphEdge, docID string, newEdges []GraphEdge) []GraphEdge {
	var kept []GraphEdge
	for _, e := range existing {
		if !edgeBelongsToDoc(e, docID) {
			kept = append(kept, e)
		}
	}
	return append(kept, newEdges...)
}

// edgeBelongsToDoc reports whether an edge originates from the given document.
func edgeBelongsToDoc(e GraphEdge, docID string) bool {
	return e.From == docID || strings.HasPrefix(e.From, docID+"#")
}
