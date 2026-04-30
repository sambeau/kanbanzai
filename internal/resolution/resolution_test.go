package resolution

import (
	"testing"
)

func TestDisambiguate_FilePath_Slash(t *testing.T) {
	result := Disambiguate("work/design/foo.md")
	if result != ResolvePath {
		t.Errorf("Disambiguate(%q) = %v, want ResolvePath", "work/design/foo.md", result)
	}
}

func TestDisambiguate_FilePath_TxtExtension(t *testing.T) {
	result := Disambiguate("notes.txt")
	if result != ResolvePath {
		t.Errorf("Disambiguate(%q) = %v, want ResolvePath", "notes.txt", result)
	}
}

func TestDisambiguate_FilePath_MdExtension(t *testing.T) {
	result := Disambiguate("README.md")
	if result != ResolvePath {
		t.Errorf("Disambiguate(%q) = %v, want ResolvePath", "README.md", result)
	}
}

func TestDisambiguate_FilePath_LeadingDotSlash(t *testing.T) {
	result := Disambiguate("./work/design/foo.md")
	if result != ResolvePath {
		t.Errorf("Disambiguate(%q) = %v, want ResolvePath", "./work/design/foo.md", result)
	}
}

func TestDisambiguate_EntityID_DisplayFormat(t *testing.T) {
	result := Disambiguate("FEAT-042")
	if result != ResolveEntity {
		t.Errorf("Disambiguate(%q) = %v, want ResolveEntity", "FEAT-042", result)
	}
}

func TestDisambiguate_EntityID_PlanWithSlug(t *testing.T) {
	result := Disambiguate("P1-my-plan")
	if result != ResolveEntity {
		t.Errorf("Disambiguate(%q) = %v, want ResolveEntity", "P1-my-plan", result)
	}
}

func TestDisambiguate_EntityID_BatchWithSlug(t *testing.T) {
	result := Disambiguate("B24-auth-system")
	if result != ResolveEntity {
		t.Errorf("Disambiguate(%q) = %v, want ResolveEntity", "B24-auth-system", result)
	}
}

func TestDisambiguate_EntityID_FullTSID(t *testing.T) {
	result := Disambiguate("FEAT-01KMKA278DFNV")
	if result != ResolveEntity {
		t.Errorf("Disambiguate(%q) = %v, want ResolveEntity", "FEAT-01KMKA278DFNV", result)
	}
}

func TestDisambiguate_EntityID_TaskDisplay(t *testing.T) {
	result := Disambiguate("TASK-001")
	if result != ResolveEntity {
		t.Errorf("Disambiguate(%q) = %v, want ResolveEntity", "TASK-001", result)
	}
}

func TestDisambiguate_EntityID_TaskFull(t *testing.T) {
	result := Disambiguate("T-01KMKA278DFNV")
	if result != ResolveEntity {
		t.Errorf("Disambiguate(%q) = %v, want ResolveEntity", "T-01KMKA278DFNV", result)
	}
}

func TestDisambiguate_EntityID_BugDisplay(t *testing.T) {
	result := Disambiguate("BUG-007")
	if result != ResolveEntity {
		t.Errorf("Disambiguate(%q) = %v, want ResolveEntity", "BUG-007", result)
	}
}

func TestDisambiguate_EntityID_IncidentDisplay(t *testing.T) {
	result := Disambiguate("INC-003")
	if result != ResolveEntity {
		t.Errorf("Disambiguate(%q) = %v, want ResolveEntity", "INC-003", result)
	}
}

func TestDisambiguate_BarePlanPrefix(t *testing.T) {
	result := Disambiguate("P1")
	if result != ResolvePlanPrefix {
		t.Errorf("Disambiguate(%q) = %v, want ResolvePlanPrefix", "P1", result)
	}
}

func TestDisambiguate_BarePlanPrefix_MultiDigit(t *testing.T) {
	result := Disambiguate("P42")
	if result != ResolvePlanPrefix {
		t.Errorf("Disambiguate(%q) = %v, want ResolvePlanPrefix", "P42", result)
	}
}

func TestDisambiguate_BarePlanPrefix_Batch(t *testing.T) {
	result := Disambiguate("B7")
	if result != ResolvePlanPrefix {
		t.Errorf("Disambiguate(%q) = %v, want ResolvePlanPrefix", "B7", result)
	}
}

func TestDisambiguate_None_GenericToken(t *testing.T) {
	result := Disambiguate("sometoken")
	if result != ResolveNone {
		t.Errorf("Disambiguate(%q) = %v, want ResolveNone", "sometoken", result)
	}
}

func TestDisambiguate_None_EmptyString(t *testing.T) {
	result := Disambiguate("")
	if result != ResolveNone {
		t.Errorf("Disambiguate(%q) = %v, want ResolveNone", "", result)
	}
}

func TestDisambiguate_None_LowercasePrefix(t *testing.T) {
	result := Disambiguate("p1")
	if result != ResolveNone {
		t.Errorf("Disambiguate(%q) = %v, want ResolveNone", "p1", result)
	}
}

func TestDisambiguate_None_SingleLetter(t *testing.T) {
	result := Disambiguate("P")
	if result != ResolveNone {
		t.Errorf("Disambiguate(%q) = %v, want ResolveNone", "P", result)
	}
}

func TestDisambiguate_None_OnlyDigits(t *testing.T) {
	result := Disambiguate("123")
	if result != ResolveNone {
		t.Errorf("Disambiguate(%q) = %v, want ResolveNone", "123", result)
	}
}

func TestDisambiguate_None_SpecialChars(t *testing.T) {
	result := Disambiguate("hello_world")
	if result != ResolveNone {
		t.Errorf("Disambiguate(%q) = %v, want ResolveNone", "hello_world", result)
	}
}

func TestDisambiguate_FilePath_DeepPath(t *testing.T) {
	result := Disambiguate("work/spec/B36-F2-spec-status-argument-resolution.md")
	if result != ResolvePath {
		t.Errorf("Disambiguate(%q) = %v, want ResolvePath", "work/spec/B36-F2-spec-status-argument-resolution.md", result)
	}
}

func TestDisambiguate_PathOverEntity(t *testing.T) {
	result := Disambiguate("path/FEAT-042")
	if result != ResolvePath {
		t.Errorf("Disambiguate(%q) = %v, want ResolvePath", "path/FEAT-042", result)
	}
}

func TestResolutionKind_String(t *testing.T) {
	tests := []struct {
		kind ResolutionKind
		want string
	}{
		{ResolvePath, "ResolvePath"},
		{ResolveEntity, "ResolveEntity"},
		{ResolvePlanPrefix, "ResolvePlanPrefix"},
		{ResolveNone, "ResolveNone"},
		{ResolutionKind(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.kind.String()
		if got != tt.want {
			t.Errorf("ResolutionKind(%d).String() = %q, want %q", tt.kind, got, tt.want)
		}
	}
}

func TestDisambiguate_FilePath_OnlySlash(t *testing.T) {
	result := Disambiguate("foo/bar")
	if result != ResolvePath {
		t.Errorf("Disambiguate(%q) = %v, want ResolvePath", "foo/bar", result)
	}
}

func TestDisambiguate_FilePath_HomeDirStyle(t *testing.T) {
	result := Disambiguate("/absolute/path/foo.md")
	if result != ResolvePath {
		t.Errorf("Disambiguate(%q) = %v, want ResolvePath", "/absolute/path/foo.md", result)
	}
}

func TestDisambiguate_None_LeadingDigitPrefix(t *testing.T) {
	result := Disambiguate("1FEAT")
	if result != ResolveNone {
		t.Errorf("Disambiguate(%q) = %v, want ResolveNone", "1FEAT", result)
	}
}

func TestDisambiguate_EntityID_CaseInsensitivePrefix(t *testing.T) {
	result := Disambiguate("feat-01KMKA278DFNV")
	if result != ResolveEntity {
		t.Errorf("Disambiguate(%q) = %v, want ResolveEntity", "feat-01KMKA278DFNV", result)
	}
}
