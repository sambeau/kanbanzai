package knowledge

import "strings"

// stopWords is the set of words removed before computing Jaccard similarity per spec §9.
var stopWords = map[string]struct{}{
	"a": {}, "an": {}, "the": {}, "is": {}, "are": {}, "was": {}, "were": {},
	"be": {}, "been": {}, "being": {}, "have": {}, "has": {}, "had": {},
	"do": {}, "does": {}, "did": {}, "will": {}, "would": {}, "shall": {},
	"should": {}, "may": {}, "might": {}, "can": {}, "could": {},
	"in": {}, "on": {}, "of": {}, "for": {}, "to": {}, "and": {}, "or": {},
	"but": {}, "not": {}, "with": {}, "at": {}, "by": {}, "from": {}, "as": {},
	"it": {}, "this": {}, "that": {},
}

// NormalizeTopic normalises a topic string per spec §9:
// lowercase, spaces and underscores replaced with hyphens,
// consecutive hyphens collapsed, leading/trailing hyphens stripped.
func NormalizeTopic(topic string) string {
	topic = strings.ToLower(topic)

	var b strings.Builder
	b.Grow(len(topic))
	prevHyphen := false
	for _, r := range topic {
		switch r {
		case ' ', '_', '-':
			if !prevHyphen {
				b.WriteRune('-')
				prevHyphen = true
			}
		default:
			b.WriteRune(r)
			prevHyphen = false
		}
	}

	return strings.Trim(b.String(), "-")
}

// ContentWords returns the normalised word set for Jaccard similarity.
// It lowercases, splits on non-alphanumeric characters, and removes stop words.
func ContentWords(content string) map[string]struct{} {
	content = strings.ToLower(content)

	var words []string
	var cur strings.Builder
	for _, r := range content {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			cur.WriteRune(r)
		} else {
			if cur.Len() > 0 {
				words = append(words, cur.String())
				cur.Reset()
			}
		}
	}
	if cur.Len() > 0 {
		words = append(words, cur.String())
	}

	result := make(map[string]struct{})
	for _, w := range words {
		if _, isStop := stopWords[w]; !isStop && len(w) > 0 {
			result[w] = struct{}{}
		}
	}
	return result
}

// JaccardSimilarity computes the Jaccard similarity coefficient between two word sets:
// |intersection(A,B)| / |union(A,B)|.
// Returns 1.0 when both sets are empty.
func JaccardSimilarity(a, b map[string]struct{}) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1.0
	}

	intersection := 0
	for w := range a {
		if _, ok := b[w]; ok {
			intersection++
		}
	}

	union := len(a) + len(b) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}
