package hashvalidate

import (
	"fmt"
	"testing"
)

func TestHashLine_Determinism(t *testing.T) {
	input := "function hello() {"
	first := HashLine(input)
	second := HashLine(input)
	if first != second {
		t.Errorf("HashLine(%q) not deterministic: %s != %s", input, first, second)
	}
}

func TestHashLine_NewlineExclusion(t *testing.T) {
	withoutNL := "  return x;"
	withNL := "  return x;\n"

	hashWithout := HashLine(withoutNL)
	hashWith := HashLine(withNL)

	if hashWithout != hashWith {
		t.Errorf("HashLine should produce same hash with and without trailing newline: %s != %s", hashWithout, hashWith)
	}
}

func TestHashLine_EmptyString(t *testing.T) {
	h := HashLine("")
	if len(h) != HashLength {
		t.Errorf("HashLine(\"\") length = %d, want %d", len(h), HashLength)
	}
}

func TestHashLine_BlankLine(t *testing.T) {
	h := HashLine("\n")
	if len(h) != HashLength {
		t.Errorf("HashLine(\"\\n\") length = %d, want %d", len(h), HashLength)
	}
}

func TestHashLine_Uppercase(t *testing.T) {
	h := HashLine("test content")
	for _, c := range h {
		if c >= 'a' && c <= 'f' {
			t.Errorf("HashLine output %q contains lowercase hex; expected uppercase", h)
		}
	}
}

func TestHashLine_DifferentContentDifferentHash(t *testing.T) {
	a := HashLine("hello")
	b := HashLine("world")
	if a == b {
		t.Errorf("different content should produce different hashes: both = %s", a)
	}
}

func TestHashLine_UniformDistribution(t *testing.T) {
	// Generate 1000 different lines and verify hashes span multiple values.
	seen := make(map[string]int)
	for i := 0; i < 1000; i++ {
		input := fmt.Sprintf("line %d: %x", i, i)
		h := HashLine(input)
		seen[h]++
	}
	// With 1000 inputs and 256 possible values, expect at least 50 unique values.
	if len(seen) < 50 {
		t.Errorf("HashLine distribution too narrow: %d unique values from 1000 inputs (expected >= 50)", len(seen))
	}
}
