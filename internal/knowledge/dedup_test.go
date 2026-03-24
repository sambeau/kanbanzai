package knowledge

import (
	"testing"
)

func TestNormalizeTopic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{
			input: "API JSON Naming Convention",
			want:  "api-json-naming-convention",
		},
		{
			input: "go_error_handling",
			want:  "go-error-handling",
		},
		{
			input: "already-hyphenated",
			want:  "already-hyphenated",
		},
		{
			input: "  leading and trailing spaces  ",
			want:  "leading-and-trailing-spaces",
		},
		{
			input: "multiple---hyphens",
			want:  "multiple-hyphens",
		},
		{
			input: "mixed_spaces and_underscores",
			want:  "mixed-spaces-and-underscores",
		},
		{
			input: "UPPERCASE",
			want:  "uppercase",
		},
		{
			input: "-leading-hyphen",
			want:  "leading-hyphen",
		},
		{
			input: "trailing-hyphen-",
			want:  "trailing-hyphen",
		},
		{
			input: "space _ underscore - hyphen",
			want:  "space-underscore-hyphen",
		},
		{
			input: "",
			want:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got := NormalizeTopic(tc.input)
			if got != tc.want {
				t.Errorf("NormalizeTopic(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestContentWords(t *testing.T) {
	t.Parallel()

	t.Run("removes stop words", func(t *testing.T) {
		t.Parallel()
		words := ContentWords("the API is a service")
		// "the", "is", "a" are stop words
		if _, ok := words["the"]; ok {
			t.Error("ContentWords should remove stop word 'the'")
		}
		if _, ok := words["is"]; ok {
			t.Error("ContentWords should remove stop word 'is'")
		}
		if _, ok := words["a"]; ok {
			t.Error("ContentWords should remove stop word 'a'")
		}
		if _, ok := words["api"]; !ok {
			t.Error("ContentWords should include 'api'")
		}
		if _, ok := words["service"]; !ok {
			t.Error("ContentWords should include 'service'")
		}
	})

	t.Run("lowercases all words", func(t *testing.T) {
		t.Parallel()
		words := ContentWords("JSON REST API")
		if _, ok := words["json"]; !ok {
			t.Error("ContentWords should lowercase 'JSON' to 'json'")
		}
		if _, ok := words["rest"]; !ok {
			t.Error("ContentWords should lowercase 'REST' to 'rest'")
		}
		if _, ok := words["api"]; !ok {
			t.Error("ContentWords should lowercase 'API' to 'api'")
		}
	})

	t.Run("splits on punctuation", func(t *testing.T) {
		t.Parallel()
		words := ContentWords("use camelCase: always")
		if _, ok := words["use"]; !ok {
			t.Error("ContentWords should include 'use'")
		}
		if _, ok := words["camelcase"]; !ok {
			t.Error("ContentWords should include 'camelcase' (punctuation split)")
		}
		if _, ok := words["always"]; !ok {
			t.Error("ContentWords should include 'always'")
		}
	})

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()
		words := ContentWords("")
		if len(words) != 0 {
			t.Errorf("ContentWords('') = %v, want empty", words)
		}
	})

	t.Run("only stop words", func(t *testing.T) {
		t.Parallel()
		words := ContentWords("the a an is are")
		if len(words) != 0 {
			t.Errorf("ContentWords(all stop words) = %v, want empty", words)
		}
	})
}

func TestJaccardSimilarity(t *testing.T) {
	t.Parallel()

	t.Run("identical sets", func(t *testing.T) {
		t.Parallel()
		a := map[string]struct{}{"foo": {}, "bar": {}, "baz": {}}
		b := map[string]struct{}{"foo": {}, "bar": {}, "baz": {}}
		got := JaccardSimilarity(a, b)
		if got != 1.0 {
			t.Errorf("JaccardSimilarity(identical) = %f, want 1.0", got)
		}
	})

	t.Run("completely disjoint sets", func(t *testing.T) {
		t.Parallel()
		a := map[string]struct{}{"foo": {}, "bar": {}}
		b := map[string]struct{}{"baz": {}, "qux": {}}
		got := JaccardSimilarity(a, b)
		if got != 0.0 {
			t.Errorf("JaccardSimilarity(disjoint) = %f, want 0.0", got)
		}
	})

	t.Run("partial overlap", func(t *testing.T) {
		t.Parallel()
		// intersection = {b}, union = {a, b, c, d} → 1/4 = 0.25
		a := map[string]struct{}{"a": {}, "b": {}}
		b := map[string]struct{}{"b": {}, "c": {}, "d": {}}
		got := JaccardSimilarity(a, b)
		want := 1.0 / 4.0
		if abs(got-want) > 0.001 {
			t.Errorf("JaccardSimilarity(partial) = %f, want %f", got, want)
		}
	})

	t.Run("both empty sets", func(t *testing.T) {
		t.Parallel()
		a := map[string]struct{}{}
		b := map[string]struct{}{}
		got := JaccardSimilarity(a, b)
		if got != 1.0 {
			t.Errorf("JaccardSimilarity(both empty) = %f, want 1.0", got)
		}
	})

	t.Run("one empty set", func(t *testing.T) {
		t.Parallel()
		a := map[string]struct{}{"foo": {}}
		b := map[string]struct{}{}
		got := JaccardSimilarity(a, b)
		if got != 0.0 {
			t.Errorf("JaccardSimilarity(one empty) = %f, want 0.0", got)
		}
	})

	t.Run("high similarity above 0.65 threshold", func(t *testing.T) {
		t.Parallel()
		// Simulate near-duplicate content
		a := ContentWords("Use camelCase for all JSON API field names in responses")
		b := ContentWords("Use camelCase for JSON API field names in HTTP responses")
		sim := JaccardSimilarity(a, b)
		if sim <= 0.65 {
			t.Errorf("JaccardSimilarity(near-duplicate) = %f, expected > 0.65", sim)
		}
	})

	t.Run("low similarity below 0.65 threshold", func(t *testing.T) {
		t.Parallel()
		a := ContentWords("Always wrap errors with context using fmt.Errorf wrapping")
		b := ContentWords("Use camelCase for JSON API field names in HTTP responses")
		sim := JaccardSimilarity(a, b)
		if sim > 0.65 {
			t.Errorf("JaccardSimilarity(distinct content) = %f, expected <= 0.65", sim)
		}
	})
}

func TestExactTopicDedup(t *testing.T) {
	t.Parallel()

	// Two topics that normalise to the same string should be treated as duplicates.
	topic1 := NormalizeTopic("API JSON Naming Convention")
	topic2 := NormalizeTopic("api-json-naming-convention")

	if topic1 != topic2 {
		t.Errorf("NormalizeTopic should produce same result: %q vs %q", topic1, topic2)
	}
}

func TestNearDuplicateContentAboveThreshold(t *testing.T) {
	t.Parallel()

	content1 := "Always use camelCase for JSON API field names in REST responses"
	content2 := "Always use camelCase for JSON API field names in HTTP responses"

	words1 := ContentWords(content1)
	words2 := ContentWords(content2)
	sim := JaccardSimilarity(words1, words2)

	if sim <= 0.65 {
		t.Errorf("Similar content should have Jaccard > 0.65, got %f", sim)
	}
}

func TestBelowThresholdContent(t *testing.T) {
	t.Parallel()

	content1 := "Always wrap errors using fmt.Errorf with %w for proper unwrapping"
	content2 := "Prefer table-driven tests for multiple related test scenarios"

	words1 := ContentWords(content1)
	words2 := ContentWords(content2)
	sim := JaccardSimilarity(words1, words2)

	if sim > 0.65 {
		t.Errorf("Distinct content should have Jaccard <= 0.65, got %f", sim)
	}
}

func TestCrossScopeIndependence(t *testing.T) {
	t.Parallel()

	// Same topic normalises to the same string regardless of scope.
	// Scope-based dedup is enforced at the service layer, not in this package.
	// This test verifies topic normalisation is consistent across different inputs.
	topic := "api-error-handling"
	if NormalizeTopic("API Error Handling") != topic {
		t.Errorf("NormalizeTopic('API Error Handling') != %q", topic)
	}
	if NormalizeTopic("api_error_handling") != topic {
		t.Errorf("NormalizeTopic('api_error_handling') != %q", topic)
	}
	if NormalizeTopic("api-error-handling") != topic {
		t.Errorf("NormalizeTopic('api-error-handling') != %q", topic)
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
