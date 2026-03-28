package service

import (
	"strings"
	"testing"
)

// ─── ValidateRetroSignal ──────────────────────────────────────────────────────

func TestValidateRetroSignal_Valid(t *testing.T) {
	t.Parallel()
	cases := []RetroSignalInput{
		{Category: "workflow-friction", Observation: "Step was confusing", Severity: "minor"},
		{Category: "tool-gap", Observation: "No tool for X", Severity: "moderate"},
		{Category: "tool-friction", Observation: "Tool required too many params", Severity: "significant"},
		{Category: "spec-ambiguity", Observation: "Spec unclear on error format", Severity: "moderate"},
		{Category: "context-gap", Observation: "Convention not in context packet", Severity: "minor"},
		{Category: "decomposition-issue", Observation: "Task too large", Severity: "significant"},
		{Category: "design-gap", Observation: "Design missed edge case", Severity: "moderate"},
		{Category: "worked-well", Observation: "Vertical slicing worked great", Severity: "minor"},
		// optional fields populated
		{
			Category:        "spec-ambiguity",
			Observation:     "Error format undefined",
			Severity:        "moderate",
			Suggestion:      "Add error format section",
			RelatedDecision: "DEC-042",
		},
	}
	for _, s := range cases {
		if err := ValidateRetroSignal(s); err != nil {
			t.Errorf("ValidateRetroSignal(%+v) = %v, want nil", s, err)
		}
	}
}

func TestValidateRetroSignal_MissingCategory(t *testing.T) {
	t.Parallel()
	err := ValidateRetroSignal(RetroSignalInput{
		Observation: "Something happened",
		Severity:    "minor",
	})
	if err == nil {
		t.Fatal("expected error for missing category, got nil")
	}
	if !strings.Contains(err.Error(), "category is required") {
		t.Errorf("error = %q, want to contain \"category is required\"", err.Error())
	}
}

func TestValidateRetroSignal_UnknownCategory(t *testing.T) {
	t.Parallel()
	err := ValidateRetroSignal(RetroSignalInput{
		Category:    "something-made-up",
		Observation: "Something happened",
		Severity:    "minor",
	})
	if err == nil {
		t.Fatal("expected error for unknown category, got nil")
	}
	if !strings.Contains(err.Error(), "unknown category") {
		t.Errorf("error = %q, want to contain \"unknown category\"", err.Error())
	}
	// Error message should list the valid categories.
	for cat := range ValidRetroCategories {
		if !strings.Contains(err.Error(), cat) {
			t.Errorf("error message missing valid category %q: %s", cat, err.Error())
		}
	}
}

func TestValidateRetroSignal_MissingObservation(t *testing.T) {
	t.Parallel()
	err := ValidateRetroSignal(RetroSignalInput{
		Category: "tool-gap",
		Severity: "minor",
	})
	if err == nil {
		t.Fatal("expected error for missing observation, got nil")
	}
	if !strings.Contains(err.Error(), "observation is required") {
		t.Errorf("error = %q, want to contain \"observation is required\"", err.Error())
	}
}

func TestValidateRetroSignal_MissingSeverity(t *testing.T) {
	t.Parallel()
	err := ValidateRetroSignal(RetroSignalInput{
		Category:    "tool-gap",
		Observation: "No tool for X",
	})
	if err == nil {
		t.Fatal("expected error for missing severity, got nil")
	}
	if !strings.Contains(err.Error(), "severity is required") {
		t.Errorf("error = %q, want to contain \"severity is required\"", err.Error())
	}
}

func TestValidateRetroSignal_UnknownSeverity(t *testing.T) {
	t.Parallel()
	err := ValidateRetroSignal(RetroSignalInput{
		Category:    "tool-gap",
		Observation: "No tool for X",
		Severity:    "catastrophic",
	})
	if err == nil {
		t.Fatal("expected error for unknown severity, got nil")
	}
	if !strings.Contains(err.Error(), "unknown severity") {
		t.Errorf("error = %q, want to contain \"unknown severity\"", err.Error())
	}
	if !strings.Contains(err.Error(), "minor") || !strings.Contains(err.Error(), "moderate") || !strings.Contains(err.Error(), "significant") {
		t.Errorf("error message should list valid severities: %s", err.Error())
	}
}

func TestValidateRetroSignal_ErrorCarriesSignal(t *testing.T) {
	t.Parallel()
	input := RetroSignalInput{
		Category:    "bad-category",
		Observation: "Something",
		Severity:    "minor",
	}
	err := ValidateRetroSignal(input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	ve, ok := err.(*RetroSignalValidationError)
	if !ok {
		t.Fatalf("expected *RetroSignalValidationError, got %T", err)
	}
	if ve.Signal.Category != input.Category {
		t.Errorf("error.Signal.Category = %q, want %q", ve.Signal.Category, input.Category)
	}
}

// ─── EncodeRetroContent ───────────────────────────────────────────────────────

func TestEncodeRetroContent_NoOptionals(t *testing.T) {
	t.Parallel()
	s := RetroSignalInput{
		Category:    "spec-ambiguity",
		Observation: "Spec says handle errors appropriately without defining format",
		Severity:    "moderate",
	}
	got := EncodeRetroContent(s)
	want := "[moderate] spec-ambiguity: Spec says handle errors appropriately without defining format"
	if got != want {
		t.Errorf("EncodeRetroContent() = %q, want %q", got, want)
	}
}

func TestEncodeRetroContent_WithSuggestion(t *testing.T) {
	t.Parallel()
	s := RetroSignalInput{
		Category:    "spec-ambiguity",
		Observation: "Error format undefined",
		Severity:    "moderate",
		Suggestion:  "Add error format section to spec template",
	}
	got := EncodeRetroContent(s)
	want := "[moderate] spec-ambiguity: Error format undefined Suggestion: Add error format section to spec template"
	if got != want {
		t.Errorf("EncodeRetroContent() = %q, want %q", got, want)
	}
}

func TestEncodeRetroContent_WithRelatedDecision(t *testing.T) {
	t.Parallel()
	s := RetroSignalInput{
		Category:        "worked-well",
		Observation:     "Error format spec section eliminated guesswork",
		Severity:        "minor",
		RelatedDecision: "DEC-042",
	}
	got := EncodeRetroContent(s)
	want := "[minor] worked-well: Error format spec section eliminated guesswork Related: DEC-042"
	if got != want {
		t.Errorf("EncodeRetroContent() = %q, want %q", got, want)
	}
}

func TestEncodeRetroContent_WithSuggestionAndRelatedDecision(t *testing.T) {
	t.Parallel()
	s := RetroSignalInput{
		Category:        "spec-ambiguity",
		Observation:     "Error format undefined",
		Severity:        "significant",
		Suggestion:      "Add error format section",
		RelatedDecision: "DEC-042",
	}
	got := EncodeRetroContent(s)
	want := "[significant] spec-ambiguity: Error format undefined Suggestion: Add error format section Related: DEC-042"
	if got != want {
		t.Errorf("EncodeRetroContent() = %q, want %q", got, want)
	}
}

func TestEncodeRetroContent_BlankSuggestionOmitted(t *testing.T) {
	t.Parallel()
	s := RetroSignalInput{
		Category:    "tool-friction",
		Observation: "Required too many params",
		Severity:    "minor",
		Suggestion:  "   ", // whitespace only — should be omitted
	}
	got := EncodeRetroContent(s)
	if strings.Contains(got, "Suggestion:") {
		t.Errorf("blank suggestion should be omitted, got: %q", got)
	}
}

func TestEncodeRetroContent_AllSeverities(t *testing.T) {
	t.Parallel()
	for sev := range ValidRetroSeverities {
		s := RetroSignalInput{
			Category:    "tool-gap",
			Observation: "No tool for X",
			Severity:    sev,
		}
		got := EncodeRetroContent(s)
		if !strings.HasPrefix(got, "["+sev+"]") {
			t.Errorf("severity %q: got %q, want prefix \"[%s]\"", sev, got, sev)
		}
	}
}

// ─── RetroSignalTopic ─────────────────────────────────────────────────────────

func TestRetroSignalTopic_First(t *testing.T) {
	t.Parallel()
	got := RetroSignalTopic("TASK-01ABC", 1)
	want := "retro-TASK-01ABC"
	if got != want {
		t.Errorf("RetroSignalTopic(n=1) = %q, want %q", got, want)
	}
}

func TestRetroSignalTopic_Second(t *testing.T) {
	t.Parallel()
	got := RetroSignalTopic("TASK-01ABC", 2)
	want := "retro-TASK-01ABC-2"
	if got != want {
		t.Errorf("RetroSignalTopic(n=2) = %q, want %q", got, want)
	}
}

func TestRetroSignalTopic_Subsequent(t *testing.T) {
	t.Parallel()
	for n, want := range map[int]string{
		3: "retro-TASK-XYZ-3",
		5: "retro-TASK-XYZ-5",
		9: "retro-TASK-XYZ-9",
	} {
		if got := RetroSignalTopic("TASK-XYZ", n); got != want {
			t.Errorf("RetroSignalTopic(n=%d) = %q, want %q", n, got, want)
		}
	}
}

func TestRetroSignalTopic_ZeroTreatedAsFirst(t *testing.T) {
	t.Parallel()
	// n=0 is out-of-range but should not panic; treated same as n=1.
	got := RetroSignalTopic("TASK-01ABC", 0)
	want := "retro-TASK-01ABC"
	if got != want {
		t.Errorf("RetroSignalTopic(n=0) = %q, want %q", got, want)
	}
}

func TestRetroSignalTopic_UniquePerSequence(t *testing.T) {
	t.Parallel()
	taskID := "TASK-UNIQUE"
	seen := make(map[string]bool)
	for n := 1; n <= 10; n++ {
		topic := RetroSignalTopic(taskID, n)
		if seen[topic] {
			t.Errorf("duplicate topic at n=%d: %q", n, topic)
		}
		seen[topic] = true
	}
}
